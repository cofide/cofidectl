package trustzone

import (
	"fmt"
	"log/slog"

	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/spf13/cobra"
)

type TrustZoneCommand struct {
	source plugin.DataSource
}

func NewTrustZoneCommand(source plugin.DataSource) *TrustZoneCommand {
	return &TrustZoneCommand{
		source: source,
	}
}

func (c *TrustZoneCommand) GetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists existing trust zones (if any)",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			//potentially good place to init the grpc client (lazily)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("getting here - it shouldn't though!")
			trustZones, err := c.source.GetTrustZones()
			if err != nil {
				return fmt.Errorf("failed to get trust zones")
			}
			slog.Info("retrieved trust zones", "trust_zones", trustZones)
			return nil
		},
	}

	return cmd
}
