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
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/cofide/cofidectl/pkg/plugin/provision"
	"github.com/cofide/cofidectl/pkg/plugin/provision/spirehelm"
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
	configLoader   config.Loader
	loadGrpcPlugin func(hclog.Logger, string) (*go_plugin.Client, cofidectl_plugin.DataSource, provision.Provision, error)
	source         cofidectl_plugin.DataSource
	provision      provision.Provision
	clients        map[string]*go_plugin.Client
}

func NewManager(configLoader config.Loader) *PluginManager {
	return &PluginManager{
		configLoader:   configLoader,
		loadGrpcPlugin: loadGrpcPlugin,
		clients:        map[string]*go_plugin.Client{},
	}
}

// Init initialises the configuration for the specified plugins.
func (pm *PluginManager) Init(plugins *pluginspb.Plugins, pluginConfig map[string]*structpb.Struct) (cofidectl_plugin.DataSource, error) {
	if plugins == nil {
		plugins = GetDefaultPlugins()
	}

	if exists, _ := pm.configLoader.Exists(); exists {
		// Check that existing plugin config matches.
		cfg, err := pm.configLoader.Read()
		if err != nil {
			return nil, err
		}
		ds := plugins.GetDataSource()
		provision := plugins.GetProvision()
		if ds != cfg.Plugins.GetDataSource() {
			return nil, fmt.Errorf("existing config file uses a different data source plugin: %s vs %s", cfg.Plugins.GetDataSource(), ds)
		}
		if cfg.Plugins.GetProvision() != provision {
			return nil, fmt.Errorf("existing config file uses a different provision plugin: %s vs %s", cfg.Plugins.GetProvision(), provision)
		}
		if !maps.EqualFunc(cfg.PluginConfig, pluginConfig, proto.StructsEqual) {
			return nil, fmt.Errorf("existing config file has different plugin config:\n%v\nvs\n\n%v", cfg.PluginConfig, pluginConfig)
		}
		fmt.Println("the config file already exists")
	} else {
		cfg := config.NewConfig()
		plugins, err := proto.ClonePlugins(plugins)
		if err != nil {
			return nil, err
		}
		cfg.Plugins = plugins
		if pluginConfig != nil {
			cfg.PluginConfig = pluginConfig
		}
		if err := pm.configLoader.Write(cfg); err != nil {
			return nil, err
		}
	}

	return pm.loadDataSource()
}

func (pm *PluginManager) GetDataSource() (cofidectl_plugin.DataSource, error) {
	if pm.source != nil {
		return pm.source, nil
	}

	return pm.loadDataSource()
}

func (pm *PluginManager) loadDataSource() (cofidectl_plugin.DataSource, error) {
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

	var ds cofidectl_plugin.DataSource
	var client *go_plugin.Client
	switch dsName {
	case LocalDSPluginName:
		ds, err = local.NewLocalDataSource(pm.configLoader)
		if err != nil {
			return nil, err
		}
	default:
		logger := hclog.New(&hclog.LoggerOptions{
			Name:   "plugin",
			Output: os.Stdout,
			Level:  hclog.Error,
		})

		client, ds, _, err = pm.loadGrpcPlugin(logger, dsName)
		if err != nil {
			return nil, err
		}
	}

	if err := ds.Validate(); err != nil {
		if client != nil {
			client.Kill()
		}
		return nil, err
	}
	pm.source = ds
	pm.clients[dsName] = client
	return ds, nil
}

func (pm *PluginManager) GetProvision() (provision.Provision, error) {
	if pm.provision != nil {
		return pm.provision, nil
	}
	return pm.loadProvision()
}

func (pm *PluginManager) loadProvision() (provision.Provision, error) {
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

	var provision provision.Provision
	var client *go_plugin.Client
	switch provisionName {
	case SpireHelmProvisionPluginName:
		return spirehelm.NewSpireHelm(nil), nil
	default:
		logger := hclog.New(&hclog.LoggerOptions{
			Name:   "plugin",
			Output: os.Stdout,
			Level:  hclog.Error,
		})

		client, _, provision, err = pm.loadGrpcPlugin(logger, provisionName)
		if err != nil {
			return nil, err
		}
	}

	if err := provision.Validate(context.TODO()); err != nil {
		if client != nil {
			client.Kill()
		}
		return nil, err
	}
	pm.provision = provision
	pm.clients[provisionName] = client
	return provision, nil
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

func loadGrpcPlugin(logger hclog.Logger, pluginName string) (*go_plugin.Client, cofidectl_plugin.DataSource, provision.Provision, error) {
	pluginPath, err := cofidectl_plugin.GetPluginPath(pluginName)
	if err != nil {
		return nil, nil, nil, err
	}

	cmd := exec.Command(pluginPath, cofidectl_plugin.DataSourcePluginArgs...)
	client := go_plugin.NewClient(&go_plugin.ClientConfig{
		Cmd:             cmd,
		HandshakeConfig: cofidectl_plugin.HandshakeConfig,
		Plugins: map[string]go_plugin.Plugin{
			cofidectl_plugin.DataSourcePluginName: &cofidectl_plugin.DataSourcePlugin{},
			provision.ProvisionPluginName:         &provision.ProvisionPlugin{},
		},
		AllowedProtocols: []go_plugin.Protocol{go_plugin.ProtocolGRPC},
		Logger:           logger,
	})

	source, provision, err := startGrpcPlugin(client, pluginName)
	if err != nil {
		client.Kill()
		return nil, nil, nil, err
	}
	return client, source, provision, nil
}

func startGrpcPlugin(client *go_plugin.Client, pluginName string) (cofidectl_plugin.DataSource, provision.Provision, error) {
	grpcClient, err := client.Client()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create interface to plugin: %w", err)
	}

	if err = grpcClient.Ping(); err != nil {
		return nil, nil, fmt.Errorf("failed to ping the gRPC client: %w", err)
	}

	raw, err := grpcClient.Dispense(cofidectl_plugin.DataSourcePluginName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dispense an instance of the plugin: %w", err)
	}

	source, ok := raw.(cofidectl_plugin.DataSource)
	if !ok {
		return nil, nil, fmt.Errorf("gRPC data source plugin %s does not implement plugin interface", pluginName)
	}

	raw, err = grpcClient.Dispense(provision.ProvisionPluginName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dispense an instance of the plugin: %w", err)
	}

	provision, ok := raw.(provision.Provision)
	if !ok {
		return nil, nil, fmt.Errorf("gRPC data source plugin %s does not implement plugin interface", pluginName)
	}
	return source, provision, nil
}

func (pm *PluginManager) Shutdown() {
	for name, client := range pm.clients {
		if client != nil {
			client.Kill()
		}
		delete(pm.clients, name)
	}
	pm.source = nil
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

func GetDefaultPlugins() *pluginspb.Plugins {
	ds := LocalDSPluginName
	provision := SpireHelmProvisionPluginName
	return &pluginspb.Plugins{
		DataSource: &ds,
		Provision:  &provision,
	}
}
