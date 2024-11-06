package federation

import (
	"os"

	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
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
			if err := c.source.Validate(); err != nil {
				return err
			}

			federations, err := c.source.ListFederations()
			if err != nil {
				return err
			}

			data := make([][]string, len(federations))
			for i, federation := range federations {
				data[i] = []string{
					federation.From,
					federation.To,
					"Healthy", // TODO
				}
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"From Trust Zone", "To Trust Zone", "Status"})
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
	from string
	to   string
}

func (c *FederationCommand) GetAddCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new federation",
		Long:  federationAddCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.source.Validate(); err != nil {
				return err
			}

			newFederation := &federation_proto.Federation{
				From: opts.from,
				To:   opts.to,
			}
			return c.source.AddFederation(newFederation)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.from, "from", "", "Trust zone to federate from")
	f.StringVar(&opts.to, "to", "", "Trust zone to federate to")

	cobra.CheckErr(cmd.MarkFlagRequired("from"))
	cobra.CheckErr(cmd.MarkFlagRequired("to"))

	return cmd
}
