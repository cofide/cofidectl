package trustzone

import (
	"fmt"
	"log/slog"
	"os"

	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type TrustZoneCommand struct {
	source cofidectl_plugin.DataSource
}

func NewTrustZoneCommand(source cofidectl_plugin.DataSource) *TrustZoneCommand {
	return &TrustZoneCommand{
		source: source,
	}
}

var trustZoneDesc = `
This command consists of multiple sub-commands to administer Cofide trust zones.
`

func (c *TrustZoneCommand) ListRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust-zone list",
		Short: "list trust-zones",
		Long:  trustZoneDesc,
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			// TODO: potentially good place to init the grpc client (lazily)
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

	cmd.AddCommand(c.GetListCommand())

	return cmd
}

var trustZoneListDesc = `
This command will list trust zones in the Cofide configuration state.
`

func (c *TrustZoneCommand) GetListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [NAME]",
		Short: "List trust zones",
		Long:  trustZoneListDesc,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			trustZones, err := c.source.ListTrustZones()
			if err != nil {
				return err
			}

			data := make([][]string, len(trustZones))
			for i, trustZone := range trustZones {
				data[i] = []string{
					trustZone.Name,
					trustZone.TrustDomain,
				}
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Name", "Trust Domain"})
			table.SetBorder(false)
			table.AppendBulk(data)
			table.Render()
			return nil
		},
	}

	return cmd
}
