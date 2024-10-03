package trustzone

import (
	"github.com/spf13/cobra"
)

func trustZoneAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [NAME]",
		Short: "Add a new trust zone",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return addTrustZone()
		},
	}

	return cmd
}

func addTrustZone() error {
	// client := go_plugin.NewClient(&go_plugin.ClientConfig{
	// 	HandshakeConfig: cofidectl_plugin.HandshakeConfig,
	// 	Plugins:         cofidectl_plugin.PluginMap,
	// 	Cmd:             exec.Command("sh", "-c", os.Getenv("KV_PLUGIN")),
	// 	AllowedProtocols: []plugin.Protocol{
	// 		plugin.ProtocolNetRPC, plugin.ProtocolGRPC},
	// })
	// defer client.Kill()
	return nil
}
