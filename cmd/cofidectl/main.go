package main

import (
	"log"
	"os"
	"os/exec"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	go_plugin "github.com/hashicorp/go-plugin"
)

func main() {
	client := go_plugin.NewClient(&go_plugin.ClientConfig{
		HandshakeConfig: cofidectl_plugin.HandshakeConfig,
		Plugins: map[string]go_plugin.Plugin{
			"connect_data_source": &cofidectl_plugin.DataSourcePlugin{},
		},
		Cmd:              exec.Command("sh", "-c", "./cofidectl-connect-plugin"),
		AllowedProtocols: []go_plugin.Protocol{go_plugin.ProtocolGRPC},
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

	rootCmd, err := cmd.NewRootCmd(os.Args[1:], plugin)
	if err != nil {
		log.Fatal(err)
	}

	if err = rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
