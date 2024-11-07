// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/cofide/cofidectl/internal/pkg/config"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/local"

	hclog "github.com/hashicorp/go-hclog"
	go_plugin "github.com/hashicorp/go-plugin"
)

const (
	LocalPluginName      = "local"
	DataSourcePluginName = "data_source"
)

// PluginManager provides an interface for loading and managing `DataSource` plugins based on configuration.
type PluginManager struct {
	configLoader   config.Loader
	loadGrpcPlugin func(hclog.Logger, string) (cofidectl_plugin.DataSource, error)
	source         cofidectl_plugin.DataSource
}

func NewManager(configLoader config.Loader) *PluginManager {
	return &PluginManager{
		configLoader:   configLoader,
		loadGrpcPlugin: loadGrpcPlugin,
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

		ds, err = pm.loadGrpcPlugin(logger, cfg.DataSource)
		if err != nil {
			return nil, err
		}
	}

	if err := ds.Validate(); err != nil {
		return nil, err
	}
	pm.source = ds
	return ds, nil
}

func loadGrpcPlugin(logger hclog.Logger, pluginName string) (cofidectl_plugin.DataSource, error) {
	pluginPath, err := cofidectl_plugin.GetPluginPath(pluginName)
	if err != nil {
		return nil, err
	}

	dsServeArgs := []string{"data-source", "serve"}
	cmd := exec.Command(pluginPath, dsServeArgs...)
	client := go_plugin.NewClient(&go_plugin.ClientConfig{
		Cmd:             cmd,
		HandshakeConfig: cofidectl_plugin.HandshakeConfig,
		Plugins: map[string]go_plugin.Plugin{
			DataSourcePluginName: &cofidectl_plugin.DataSourcePlugin{},
		},
		AllowedProtocols: []go_plugin.Protocol{go_plugin.ProtocolGRPC},
		Logger:           logger,
	})

	grpcClient, err := client.Client()
	if err != nil {
		return nil, fmt.Errorf("cannot create interface to plugin: %w", err)
	}

	if err = grpcClient.Ping(); err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to ping the gRPC client: %w", err)
	}

	raw, err := grpcClient.Dispense(DataSourcePluginName)
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to dispense an instance of the plugin: %w", err)
	}

	plugin, ok := raw.(cofidectl_plugin.DataSource)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("gRPC data source plugin %s does not implement plugin interface", pluginName)
	}
	return plugin, nil
}
