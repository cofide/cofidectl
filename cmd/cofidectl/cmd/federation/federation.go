// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package federation

import (
	"os"

	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	cmdcontext "github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type FederationCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewFederationCommand(cmdCtx *cmdcontext.CommandContext) *FederationCommand {
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
			ds, err := c.cmdCtx.PluginManager.GetDataSource()
			if err != nil {
				return err
			}

			federations, err := ds.ListFederations()
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
			ds, err := c.cmdCtx.PluginManager.GetDataSource()
			if err != nil {
				return err
			}

			newFederation := &federation_proto.Federation{
				From: opts.from,
				To:   opts.to,
			}
			return ds.AddFederation(newFederation)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.from, "from", "", "Trust zone to federate from")
	f.StringVar(&opts.to, "to", "", "Trust zone to federate to")

	cobra.CheckErr(cmd.MarkFlagRequired("from"))
	cobra.CheckErr(cmd.MarkFlagRequired("to"))

	return cmd
}
