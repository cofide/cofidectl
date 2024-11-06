// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"fmt"
	"os"

	"github.com/cofide/cofidectl/internal/pkg/config"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/local"

	hclog "github.com/hashicorp/go-hclog"
	go_plugin "github.com/hashicorp/go-plugin"
)

const (
	LocalPluginName   = "local"
	ConnectPluginName = "cofidectl-connect-plugin"
)

// PluginManager provides an interface for loading and managing `DataSource` plugins based on configuration.
type PluginManager struct {
	configLoader      config.Loader
	loadConnectPlugin func(logger hclog.Logger) (cofidectl_plugin.DataSource, error)
}

func NewManager(configLoader config.Loader) *PluginManager {
	return &PluginManager{
		configLoader:      configLoader,
		loadConnectPlugin: loadConnectPlugin,
	}
}

func (pm *PluginManager) Init(pluginName string) (cofidectl_plugin.DataSource, error) {
	if exists, _ := pm.configLoader.Exists(); exists {
		// Check that existing plugin config matches.
		cfg, err := pm.configLoader.Read()
		if err != nil {
			return nil, err
		}
		if cfg.DataSource != pluginName {
			return nil, fmt.Errorf("existing config file uses a different plugin: %s vs %s", cfg.DataSource, pluginName)
		}
		fmt.Println("the config file already exists")
	} else {
		cfg := config.NewConfig()
		cfg.DataSource = pluginName
		if err := pm.configLoader.Write(cfg); err != nil {
			return nil, err
		}
	}

	return pm.GetPlugin()
}

func (pm *PluginManager) GetPlugin() (cofidectl_plugin.DataSource, error) {
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

	var ds cofidectl_plugin.DataSource
	switch cfg.DataSource {
	case ConnectPluginName:
		logger := hclog.New(&hclog.LoggerOptions{
			Name:   "plugin",
			Output: os.Stdout,
			Level:  hclog.Error,
		})

		ds, err = pm.loadConnectPlugin(logger)
		if err != nil {
			return nil, err
		}
	case LocalPluginName:
		ds, err = local.NewLocalDataSource(pm.configLoader)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("only %s and %s plugins are currently supported", LocalPluginName, ConnectPluginName)
	}

	if err := ds.Validate(); err != nil {
		return nil, err
	}
	return ds, nil
}

func loadConnectPlugin(logger hclog.Logger) (cofidectl_plugin.DataSource, error) {
	client := go_plugin.NewClient(&go_plugin.ClientConfig{
		HandshakeConfig: cofidectl_plugin.HandshakeConfig,
		Plugins: map[string]go_plugin.Plugin{
			"connect_data_source": &cofidectl_plugin.DataSourcePlugin{},
		},
		AllowedProtocols: []go_plugin.Protocol{go_plugin.ProtocolGRPC},
		Logger:           logger,
	})

	defer client.Kill()

	grpcClient, err := client.Client()
	if err != nil {
		return nil, fmt.Errorf("cannot create interface to plugin: %w", err)
	}

	if err = grpcClient.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping the gRPC client: %w", err)
	}

	raw, err := grpcClient.Dispense("connect_data_source")
	if err != nil {
		return nil, fmt.Errorf("failed to dispense an instance of the plugin: %w", err)
	}

	plugin := raw.(cofidectl_plugin.DataSource)
	return plugin, nil
}
