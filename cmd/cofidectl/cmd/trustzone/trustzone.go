package trustzone

import (
	"fmt"
	"log/slog"

	go_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/spf13/cobra"
)

type TrustZoneCommand struct {
	source go_plugin.DataSource
}

func NewTrustZoneCommand(source go_plugin.DataSource) *TrustZoneCommand {
	return &TrustZoneCommand{
		source: source,
	}
}

func (c *TrustZoneCommand) ListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists existing trust zones (if any)",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			//potentially good place to init the grpc client (lazily)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			trustZones, err := c.source.ListTrustZones()
			if err != nil {
				return fmt.Errorf("failed to list trust zones")
			}
			slog.Info("retrieved trust zones", "trust_zones", trustZones)
			return nil
		},
	}

	return cmd
}
