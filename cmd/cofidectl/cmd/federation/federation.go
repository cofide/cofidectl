package federation

import (
	"os"

	"helm.sh/helm/v3/cmd/helm/require"

	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
	cmd_context "github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type FederationCommand struct {
	cmdCtx *cmd_context.CommandContext
}

func NewFederationCommand(cmdCtx *cmd_context.CommandContext) *FederationCommand {
	return &FederationCommand{
		cmdCtx: cmdCtx,
	}
}

var federationRootCmdDesc = `
This command consists of multiple sub-commands to administer Cofide trust zone federations.
`

func (c *FederationCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "federation add|list [ARGS]",
		Short: "Add, list federations",
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
			ds, err := c.cmdCtx.PluginManager.GetPlugin()
			if err != nil {
				return err
			}

			if err := ds.Validate(); err != nil {
				return err
			}

			federations, err := ds.ListFederations()
			if err != nil {
				return err
			}

			data := make([][]string, len(federations))
			for i, federation := range federations {
				data[i] = []string{
					federation.Left,
					federation.Right,
					"Healthy", // TODO
				}
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Source Trust Zone", "Destination Trust Zone", "Status"})
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
		Use:   "add",
		Short: "Add a new federation",
		Long:  federationAddCmdDesc,
		Args:  require.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetPlugin()
			if err != nil {
				return err
			}

			if err := ds.Validate(); err != nil {
				return err
			}

			newFederation := &federation_proto.Federation{
				Left:  opts.left,
				Right: opts.right,
			}
			return ds.AddFederation(newFederation)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.left, "left", "", "Trust zone to federate")
	f.StringVar(&opts.right, "right", "", "Trust zone to federate")

	cobra.CheckErr(cmd.MarkFlagRequired("left"))
	cobra.CheckErr(cmd.MarkFlagRequired("right"))

	return cmd
}
