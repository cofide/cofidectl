package loader

import (
	"errors"
	"fmt"
	"os"

	"github.com/cofide/cofidectl/internal/pkg/config"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/local"

	hclog "github.com/hashicorp/go-hclog"
	go_plugin "github.com/hashicorp/go-plugin"
)

// Loader provides an interface for loading `DataSource` plugins based on configuration.
type Loader struct {
	configLoader      config.Loader
	loadConnectPlugin func(logger hclog.Logger) (cofidectl_plugin.DataSource, error)
}

func NewLoader(configLoader config.Loader) *Loader {
	return &Loader{
		configLoader:      configLoader,
		loadConnectPlugin: loadConnectPlugin,
	}
}

func (l *Loader) GetPlugins() ([]cofidectl_plugin.DataSource, error) {
	exists, err := l.configLoader.Exists()
	if err != nil {
		return nil, err
	}

	var pluginNames []string
	if exists {
		cfg, err := l.configLoader.Read()
		if err != nil {
			return nil, err
		}
		pluginNames = cfg.Plugins
	}

	plugins := []cofidectl_plugin.DataSource{}

	// If the Connect plugin is enabled use it in place of the local data source
	for _, plugin := range pluginNames {
		if plugin == "cofidectl-connect-plugin" {
			logger := hclog.New(&hclog.LoggerOptions{
				Name:   "plugin",
				Output: os.Stdout,
				Level:  hclog.Error,
			})

			ds, err := l.loadConnectPlugin(logger)
			if err != nil {
				return nil, err
			}

			plugins = append(plugins, ds)
		} else {
			return nil, errors.New("only the cofidectl-connect-plugin is currently supported")
		}
	}

	// If no plugins have been loaded, fall back to the local data source plugin.
	if len(plugins) == 0 {
		lds, err := local.NewLocalDataSource(l.configLoader)
		if err != nil {
			return nil, err
		}

		plugins = append(plugins, lds)
	}

	return plugins, nil
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
