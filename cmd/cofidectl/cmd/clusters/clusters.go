// Copyright 2025 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package clusters

import (
	"context"
	"fmt"
	"os"

	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var clustersListCmdDesc = `
This command consists of multiple sub-commands to interact with clusters
`

type ClustersCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewClustersCommand(cmdCtx *cmdcontext.CommandContext) *ClustersCommand {
	return &ClustersCommand{
		cmdCtx: cmdCtx,
	}
}

func (c *ClustersCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clusters",
		Short: "Manage clusters",
		Long:  clustersListCmdDesc,
	}

	cmd.AddCommand(c.getListClustersCommand())

	return cmd
}

func (c *ClustersCommand) getListClustersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List clusters",
		Long:  clustersListCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.ListClusters(cmd.Context())
		},
	}

	return cmd
}

func (c *ClustersCommand) ListClusters(ctx context.Context) error {
	ds, err := c.cmdCtx.PluginManager.GetDataSource(ctx)
	if err != nil {
		return err
	}
	zones, err := ds.ListTrustZones()
	if err != nil {
		return fmt.Errorf("failed to list trust zones: %v", err)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Trust Zone", "Profile"})
	table.SetBorder(false)

	for _, zone := range zones {
		clusters, err := ds.ListClusters(zone.GetName())
		if err != nil {
			return err
		}
		if len(clusters) == 0 {
			continue
		}
		for _, cluster := range clusters {
			table.Append([]string{
				cluster.GetName(),
				zone.GetName(),
				cluster.GetProfile(),
			})
		}
	}

	table.Render()
	return nil
}
