// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"

	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/proto"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"google.golang.org/protobuf/types/known/structpb"

	hclog "github.com/hashicorp/go-hclog"
	go_plugin "github.com/hashicorp/go-plugin"
)

const (
	LocalPluginName = "local"
)

// PluginManager provides an interface for loading and managing `DataSource` plugins based on configuration.
type PluginManager struct {
	configLoader   config.Loader
	loadGrpcPlugin func(hclog.Logger, string) (*go_plugin.Client, cofidectl_plugin.DataSource, error)
	source         cofidectl_plugin.DataSource
	client         *go_plugin.Client
}

func NewManager(configLoader config.Loader) *PluginManager {
	return &PluginManager{
		configLoader:   configLoader,
		loadGrpcPlugin: loadGrpcPlugin,
	}
}

// Init initialises the configuration for the specified data source plugin.
func (pm *PluginManager) Init(dsName string, pluginConfig map[string]*structpb.Struct) (cofidectl_plugin.DataSource, error) {
	if exists, _ := pm.configLoader.Exists(); exists {
		// Check that existing plugin config matches.
		cfg, err := pm.configLoader.Read()
		if err != nil {
			return nil, err
		}
		if cfg.DataSource != dsName {
			return nil, fmt.Errorf("existing config file uses a different plugin: %s vs %s", cfg.DataSource, dsName)
		}
		if !maps.EqualFunc(cfg.PluginConfig, pluginConfig, proto.StructsEqual) {
			return nil, fmt.Errorf("existing config file has different plugin config:\n%v\nvs\n\n%v", cfg.PluginConfig, pluginConfig)
		}
		fmt.Println("the config file already exists")
	} else {
		cfg := config.NewConfig()
		cfg.DataSource = dsName
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

	exists, err := pm.configLoader.Exists()
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("the config file doesn't exist. Please run cofidectl init")
	}

	cfg, err := pm.configLoader.Read()
	if err != nil {
		return nil, err
	}

	if cfg.DataSource == "" {
		return nil, errors.New("plugin name cannot be empty")
	}

	var ds cofidectl_plugin.DataSource
	var client *go_plugin.Client
	switch cfg.DataSource {
	case LocalPluginName:
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

		client, ds, err = pm.loadGrpcPlugin(logger, cfg.DataSource)
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
	pm.client = client
	return ds, nil
}

func loadGrpcPlugin(logger hclog.Logger, pluginName string) (*go_plugin.Client, cofidectl_plugin.DataSource, error) {
	pluginPath, err := cofidectl_plugin.GetPluginPath(pluginName)
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.Command(pluginPath, cofidectl_plugin.DataSourcePluginArgs...)
	client := go_plugin.NewClient(&go_plugin.ClientConfig{
		Cmd:             cmd,
		HandshakeConfig: cofidectl_plugin.HandshakeConfig,
		Plugins: map[string]go_plugin.Plugin{
			cofidectl_plugin.DataSourcePluginName: &cofidectl_plugin.DataSourcePlugin{},
		},
		AllowedProtocols: []go_plugin.Protocol{go_plugin.ProtocolGRPC},
		Logger:           logger,
	})

	source, err := startGrpcPlugin(client, pluginName)
	if err != nil {
		client.Kill()
		return nil, nil, err
	}
	return client, source, nil
}

func startGrpcPlugin(client *go_plugin.Client, pluginName string) (cofidectl_plugin.DataSource, error) {
	grpcClient, err := client.Client()
	if err != nil {
		return nil, fmt.Errorf("cannot create interface to plugin: %w", err)
	}

	if err = grpcClient.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping the gRPC client: %w", err)
	}

	raw, err := grpcClient.Dispense(cofidectl_plugin.DataSourcePluginName)
	if err != nil {
		return nil, fmt.Errorf("failed to dispense an instance of the plugin: %w", err)
	}

	source, ok := raw.(cofidectl_plugin.DataSource)
	if !ok {
		return nil, fmt.Errorf("gRPC data source plugin %s does not implement plugin interface", pluginName)
	}
	return source, nil
}

func (pm *PluginManager) Shutdown() {
	if pm.client != nil {
		pm.client.Kill()
		pm.client = nil
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
