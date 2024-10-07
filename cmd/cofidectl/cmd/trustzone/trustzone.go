package trustzone

import (
	"os"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/trust_zone/v1"
	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/gobeam/stringy"
	"github.com/olekukonko/tablewriter"
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

var trustZoneDesc = `
This command consists of multiple sub-commands to administer Cofide trust zones.
`

func (c *TrustZoneCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust-zone add|list [ARGS]",
		Short: "add, list trust zones",
		Long:  trustZoneDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(c.GetListCommand())
	cmd.AddCommand(c.GetAddCommand())

	return cmd
}

var trustZoneListDesc = `
This command will list trust zones in the Cofide configuration state.
`

func (c *TrustZoneCommand) GetListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "List trust-zones",
		Long:  trustZoneListDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			trustZones, err := c.source.GetTrustZones()
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

var trustZoneAddDesc = `
This command will add a new trust zone to the Cofide configuration state.
`

type Opts struct {
	name         string
	trust_domain string
}

func (c *TrustZoneCommand) GetAddCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "add [NAME]",
		Short: "Add a new trust zone",
		Long:  trustZoneAddDesc,
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			str := stringy.New(args[0])
			opts.name = str.KebabCase().ToLower()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			newTrustZone := &trust_zone_proto.TrustZone{
				Name:        opts.name,
				TrustDomain: opts.trust_domain,
			}
			return c.source.AddTrustZone(newTrustZone)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trust_domain, "trust-domain", "", "Trust domain to use for this trust zone")
	cmd.MarkFlagRequired("trust-domain")

	return cmd
}
