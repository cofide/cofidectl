package cmd

import (
	"fmt"
	"log"

	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/hashicorp/go-hclog"
	go_plugin "github.com/hashicorp/go-plugin"
	"github.com/spf13/cobra"
)

type InitCommand struct {
	source cofidectl_plugin.DataSource
}

func NewInitCommand(source cofidectl_plugin.DataSource) *InitCommand {
	return &InitCommand{
		source: source,
	}
}

var initRootCmdDesc = `
This command initialises a new Cofide config file in the current working
directory
`

type Opts struct {
	enableConnect bool
}

func (i *InitCommand) GetRootCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "init [ARGS]",
		Short: "Initialises the Cofide config file",
		Long:  initRootCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := i.source.Init()
			if err != nil {
				log.Fatal(err)
			}

			if opts.enableConnect {
				fmt.Println("👀 get in touch with us at hello@cofide.io to find out more")
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&opts.enableConnect, "enable-connect", false, "Enables Cofide Connect")

	return cmd
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
