package trustzone

import (
	"fmt"
	"log/slog"
	"os/exec"

	go_plugin "github.com/hashicorp/go-plugin"

	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"

	"github.com/spf13/cobra"
)

func TrustZoneListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists existing trust zones (if any)",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return listTrustZones()
		},
	}

	return cmd
}

func listTrustZones() error {
	client := go_plugin.NewClient(&go_plugin.ClientConfig{
		HandshakeConfig: cofidectl_plugin.HandshakeConfig,
		Plugins: map[string]go_plugin.Plugin{
			"connect_data_source": &cofidectl_plugin.DataSourcePlugin{},
		},
		Cmd:              exec.Command("sh", "-c", "./cofidectl-connect-plugin"),
		AllowedProtocols: []go_plugin.Protocol{go_plugin.ProtocolNetRPC, go_plugin.ProtocolGRPC},
	})

	defer client.Kill()

	grpcClient, err := client.Client()
	if err != nil {
		return err
	}

	if err = grpcClient.Ping(); err != nil {
		slog.Error("failed to ping the gRPC client", "error", err)
		return err
	}

	raw, err := grpcClient.Dispense("connect_data_source")
	if err != nil {
		slog.Error("failed to dispense an instance of the plugin", "error", err)
		return err
	}

	plugin := raw.(cofidectl_plugin.DataSource)

	trustZones, err := plugin.GetTrustZones()
	if err != nil {
		slog.Error("failed to get trust zones", "error", err)
		return err
	}

	fmt.Println(trustZones)

	return nil
}
