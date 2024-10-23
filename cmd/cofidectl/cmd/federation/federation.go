package federation

import (
	"fmt"
	"os"

	"helm.sh/helm/v3/cmd/helm/require"

	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type FederationCommand struct {
	source cofidectl_plugin.DataSource
}

func NewFederationCommand(source cofidectl_plugin.DataSource) *FederationCommand {
	return &FederationCommand{
		source: source,
	}
}

var federationRootCmdDesc = `
This command consists of multiple sub-commands to administer Cofide trust zone federations.
`

func (c *FederationCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "federation add|list [ARGS]",
		Short: "add, list federation",
		Long:  federationRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(c.GetListCommand())
	cmd.AddCommand(c.GetAddCommand())

	return cmd
}

var federationListCmdDesc = `
This command will list federations in the Cofide configuration state.
`

func (c *FederationCommand) GetListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "List federations",
		Long:  federationListCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.source.Validate(); err != nil {
				return err
			}

			federations, err := c.source.ListFederation()
			if err != nil {
				return err
			}

			data := make([][]string, len(federations))
			for i, federation := range federations {
				data[i] = []string{
					fmt.Sprintf("%s", federation.Left),
					fmt.Sprintf("%s", federation.Right),
					"Healthy", //TODO
				}
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Trust Zone", "Trust Zone", "Status"})
			table.SetBorder(false)
			table.AppendBulk(data)
			table.Render()
			return nil
		},
	}

	return cmd
}

var federationAddCmdDesc = `
This command will add a new federation to the Cofide configuration state.
`

type Opts struct {
	left  string
	right string
}

func (c *FederationCommand) GetAddCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "add [NAME]",
		Short: "Add a new federation",
		Long:  federationAddCmdDesc,
		Args:  require.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.source.Validate(); err != nil {
				return err
			}

			newFederation := &federation_proto.Federation{
				Left:  opts.left,
				Right: opts.right,
			}
			return c.source.AddFederation(newFederation)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.left, "left", "", "Trust zone to federate")
	f.StringVar(&opts.right, "right", "", "Trust zone to federate")
	cmd.MarkFlagRequired("left")
	cmd.MarkFlagRequired("right")

	return cmd
}
