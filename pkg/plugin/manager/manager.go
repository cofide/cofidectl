// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"

	pluginspb "github.com/cofide/cofide-api-sdk/gen/go/proto/plugins/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/proto"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/cofide/cofidectl/pkg/plugin/provision"
	"github.com/cofide/cofidectl/pkg/plugin/provision/spirehelm"
	"github.com/cofide/cofidectl/pkg/plugin/validator"
	"google.golang.org/protobuf/types/known/structpb"

	hclog "github.com/hashicorp/go-hclog"
	go_plugin "github.com/hashicorp/go-plugin"
)

const (
	LocalDSPluginName            = "local"
	SpireHelmProvisionPluginName = "spire-helm"
)

// PluginManager provides an interface for loading and managing `DataSource` plugins based on configuration.
type PluginManager struct {
	configLoader     config.Loader
	grpcPluginLoader grpcPluginLoader
	source           datasource.DataSource
	provision        provision.Provision
	clients          map[string]*go_plugin.Client
}

// grpcPluginLoader is a function that loads a gRPC plugin. The function should load a single
// plugin that implements all cofidectl plugins with the specified name in pluginCfg.
// All cofidectl plugins in the returned grpcPlugin object should be validated using the
// Validate RPC before returing.
// It is primarily used for mocking in unit tests.
type grpcPluginLoader func(ctx context.Context, logger hclog.Logger, pluginName string, pluginCfg *pluginspb.Plugins) (*grpcPlugin, error)

// grpcPlugin is used when loading loading gRPC plugins. It collects the client and any cofidectl
// plugins that the gRPC plugin implements.
type grpcPlugin struct {
	client    *go_plugin.Client
	source    datasource.DataSource
	provision provision.Provision
}

func NewManager(configLoader config.Loader) *PluginManager {
	return &PluginManager{
		configLoader:     configLoader,
		grpcPluginLoader: loadGrpcPlugin,
		clients:          map[string]*go_plugin.Client{},
	}
}

// Init initialises the configuration for the specified plugins.
func (pm *PluginManager) Init(ctx context.Context, plugins *pluginspb.Plugins, pluginConfig map[string]*structpb.Struct) error {
	if plugins == nil {
		plugins = GetDefaultPlugins()
	}

	if exists, _ := pm.configLoader.Exists(); exists {
		// Check that existing plugin config matches.
		cfg, err := pm.configLoader.Read()
		if err != nil {
			return err
		}
		ds := plugins.GetDataSource()
		if ds != cfg.Plugins.GetDataSource() {
			return fmt.Errorf("existing config file uses a different data source plugin: %s vs %s", cfg.Plugins.GetDataSource(), ds)
		}
		provision := plugins.GetProvision()
		if cfg.Plugins.GetProvision() != provision {
			return fmt.Errorf("existing config file uses a different provision plugin: %s vs %s", cfg.Plugins.GetProvision(), provision)
		}
		if !maps.EqualFunc(cfg.PluginConfig, pluginConfig, proto.StructsEqual) {
			return fmt.Errorf("existing config file has different plugin config:\n%v\nvs\n\n%v", cfg.PluginConfig, pluginConfig)
		}
		fmt.Println("the config file already exists")
	} else {
		cfg := config.NewConfig()
		plugins, err := proto.ClonePlugins(plugins)
		if err != nil {
			return err
		}
		cfg.Plugins = plugins
		if pluginConfig != nil {
			cfg.PluginConfig = pluginConfig
		}
		if err := pm.configLoader.Write(cfg); err != nil {
			return err
		}
	}

	return nil
}

// GetDataSource returns the data source plugin, loading it if necessary.
func (pm *PluginManager) GetDataSource(ctx context.Context) (cofidectl_plugin.DataSource, error) {
	if pm.source != nil {
		return pm.source, nil
	}
	return pm.loadDataSource(ctx)
}

// loadDataSource loads the data source plugin, which may be an in-process or gRPC plugin.
func (pm *PluginManager) loadDataSource(ctx context.Context) (cofidectl_plugin.DataSource, error) {
	if pm.source != nil {
		return nil, errors.New("data source has already been loaded")
	}

	cfg, err := pm.readConfig()
	if err != nil {
		return nil, err
	}

	dsName := cfg.Plugins.GetDataSource()
	if dsName == "" {
		return nil, errors.New("plugin name cannot be empty")
	}

	// Check if an in-process data source implementation has been requested.
	if dsName == LocalDSPluginName {
		ds, err := local.NewLocalDataSource(pm.configLoader)
		if err != nil {
			return nil, err
		}
		if err := ds.Validate(ctx); err != nil {
			return nil, err
		}
		pm.source = ds
		return pm.source, nil
	}

	if err := pm.loadGrpcPlugin(ctx, dsName, cfg.Plugins); err != nil {
		return nil, err
	}
	return pm.source, nil
}

// GetProvision returns the provision plugin, loading it if necessary.
func (pm *PluginManager) GetProvision(ctx context.Context) (provision.Provision, error) {
	if pm.provision != nil {
		return pm.provision, nil
	}
	return pm.loadProvision(ctx)
}

// loadProvision loads the provision plugin, which may be an in-process or gRPC plugin.
func (pm *PluginManager) loadProvision(ctx context.Context) (provision.Provision, error) {
	if pm.provision != nil {
		return nil, errors.New("provision plugin has already been loaded")
	}

	cfg, err := pm.readConfig()
	if err != nil {
		return nil, err
	}

	provisionName := cfg.Plugins.GetProvision()
	if provisionName == "" {
		return nil, errors.New("provision plugin name cannot be empty")
	}

	// Check if an in-process provision implementation has been requested.
	if provisionName == SpireHelmProvisionPluginName {
		spireHelm := spirehelm.NewSpireHelm(nil)
		if err := spireHelm.Validate(ctx); err != nil {
			return nil, err
		}
		pm.provision = spireHelm
		return pm.provision, nil
	}

	if err := pm.loadGrpcPlugin(ctx, provisionName, cfg.Plugins); err != nil {
		return nil, err
	}
	return pm.provision, nil
}

// loadGrpcPlugin loads a gRPC plugin.
// The gRPC plugin may provide one or more cofidectl plugins (e.g. data source, provision), and
// all cofidectl plugins configured to use this gRPC plugin will be loaded in a single plugin
// client and server process.
func (pm *PluginManager) loadGrpcPlugin(ctx context.Context, pluginName string, pluginCfg *pluginspb.Plugins) error {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin",
		Output: os.Stdout,
		Level:  hclog.Error,
	})

	grpcPlugin, err := pm.grpcPluginLoader(ctx, logger, pluginName, pluginCfg)
	if err != nil {
		return err
	}

	if grpcPlugin.source != nil {
		pm.source = grpcPlugin.source
	}
	if grpcPlugin.provision != nil {
		pm.provision = grpcPlugin.provision
	}
	pm.clients[pluginName] = grpcPlugin.client
	return nil
}

// loadGrpcPlugin is the default grpcPluginLoader.
func loadGrpcPlugin(ctx context.Context, logger hclog.Logger, pluginName string, plugins *pluginspb.Plugins) (*grpcPlugin, error) {
	pluginPath, err := cofidectl_plugin.GetPluginPath(pluginName)
	if err != nil {
		return nil, err
	}

	pluginSet := map[string]go_plugin.Plugin{}
	if plugins.GetDataSource() == pluginName {
		pluginSet[cofidectl_plugin.DataSourcePluginName] = &cofidectl_plugin.DataSourcePlugin{}
	}
	if plugins.GetProvision() == pluginName {
		pluginSet[provision.ProvisionPluginName] = &provision.ProvisionPlugin{}
	}

	cmd := exec.Command(pluginPath, cofidectl_plugin.DataSourcePluginArgs...)
	client := go_plugin.NewClient(&go_plugin.ClientConfig{
		Cmd:              cmd,
		HandshakeConfig:  cofidectl_plugin.HandshakeConfig,
		Plugins:          pluginSet,
		AllowedProtocols: []go_plugin.Protocol{go_plugin.ProtocolGRPC},
		Logger:           logger,
	})

	grpcPlugin, err := startGrpcPlugin(ctx, client, pluginName, plugins)
	if err != nil {
		client.Kill()
		return nil, err
	}
	return grpcPlugin, nil
}

func startGrpcPlugin(ctx context.Context, client *go_plugin.Client, pluginName string, plugins *pluginspb.Plugins) (*grpcPlugin, error) {
	grpcClient, err := client.Client()
	if err != nil {
		return nil, fmt.Errorf("cannot create interface to plugin: %w", err)
	}

	if err = grpcClient.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping the gRPC client: %w", err)
	}

	grpcPlugin := &grpcPlugin{client: client}
	if plugins.GetDataSource() == pluginName {
		source, err := dispensePlugin[datasource.DataSource](ctx, grpcClient, cofidectl_plugin.DataSourcePluginName)
		if err != nil {
			return nil, err
		}
		grpcPlugin.source = source
	}

	if plugins.GetProvision() == pluginName {
		provision, err := dispensePlugin[provision.Provision](ctx, grpcClient, provision.ProvisionPluginName)
		if err != nil {
			return nil, err
		}
		grpcPlugin.provision = provision
	}
	return grpcPlugin, nil
}

// dispensePlugin dispenses a gRPC plugin from a client, ensuring that it implements the specified interface T.
func dispensePlugin[T validator.Validator](ctx context.Context, grpcClient go_plugin.ClientProtocol, name string) (T, error) {
	var zero T
	raw, err := grpcClient.Dispense(name)
	if err != nil {
		return zero, fmt.Errorf("failed to dispense an instance of the gRPC %s plugin: %w", name, err)
	}

	plugin, ok := raw.(T)
	if !ok {
		return zero, fmt.Errorf("gRPC %s plugin (%T) does not implement plugin interface ", name, plugin)
	}

	if err := plugin.Validate(ctx); err != nil {
		return zero, err
	}
	return plugin, nil
}

func (pm *PluginManager) readConfig() (*config.Config, error) {
	exists, err := pm.configLoader.Exists()
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("the config file doesn't exist. Please run cofidectl init")
	}

	return pm.configLoader.Read()
}

func (pm *PluginManager) Shutdown() {
	for name, client := range pm.clients {
		if client != nil {
			client.Kill()
		}
		delete(pm.clients, name)
	}
	pm.source = nil
	pm.provision = nil
}

// GetPluginConfig returns a `Struct` message containing per-plugin configuration from the config file.
func (pm *PluginManager) GetPluginConfig(pluginName string) (*structpb.Struct, error) {
	cfg, err := pm.configLoader.Read()
	if err != nil {
		return nil, err
	}
	pluginConfig, ok := cfg.PluginConfig[pluginName]
	if !ok {
		return nil, fmt.Errorf("no plugin configuration found for %s", pluginName)
	}
	pluginConfig, err = proto.CloneStruct(pluginConfig)
	if err != nil {
		return nil, err
	}
	return pluginConfig, nil
}

// SetPluginConfig writes a `Struct` message containing per-plugin configuration to the config file.
func (pm *PluginManager) SetPluginConfig(pluginName string, pluginConfig *structpb.Struct) error {
	cfg, err := pm.configLoader.Read()
	if err != nil {
		return err
	}
	pluginConfig, err = proto.CloneStruct(pluginConfig)
	if err != nil {
		return err
	}
	cfg.PluginConfig[pluginName] = pluginConfig
	return pm.configLoader.Write(cfg)
}

// GetDefaultPlugins returns a `Plugins` message containing the default plugins.
func GetDefaultPlugins() *pluginspb.Plugins {
	ds := LocalDSPluginName
	provision := SpireHelmProvisionPluginName
	return &pluginspb.Plugins{
		DataSource: &ds,
		Provision:  &provision,
	}
}
