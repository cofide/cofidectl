package main

import (
	"log"
	"os"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	hclog "github.com/hashicorp/go-hclog"
	go_plugin "github.com/hashicorp/go-plugin"
)

func main() {
	/*
		logger := hclog.New(&hclog.LoggerOptions{
			Name:   "plugin",
			Output: os.Stdout,
			Level:  hclog.Error,
		})
	*/

	var ds cofidectl_plugin.DataSource

	// default to the local data source
	ds, err := cofidectl_plugin.NewLocalDataSource("cofide.yaml")
	if err != nil {
		log.Fatal(err)
	}

	/*
		// determine plugins to be loaded
		plugins, err := ds.(*cofidectl_plugin.LocalDataSource).GetPlugins()
		if err != nil {
			log.Fatal(err)
		}

		// if the Connect plugin is enabled use it in place of the local data source
		if len(plugins) > 0 && plugins[0] == "cofidectl-connect-plugin" {
			ds, err = loadConnectPlugin(logger)
			if err != nil {
				log.Fatal(err)
			}
		}
	*/

	rootCmd, err := cmd.NewRootCommand(ds, os.Args[1:]).GetRootCommand()
	if err != nil {
		log.Fatal(err)
	}

	if err = rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
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
		log.Fatal("cannot create interface to plugin", "error", err)
	}

	if err = grpcClient.Ping(); err != nil {
		log.Fatal("failed to ping the gRPC client", "error", err)
	}

	raw, err := grpcClient.Dispense("connect_data_source")
	if err != nil {
		log.Fatal("failed to dispense an instance of the plugin", "error", err)
	}

	plugin := raw.(cofidectl_plugin.DataSource)
	return plugin, nil
}
