// Copyright 2025 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"os"

	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	helmprovider "github.com/cofide/cofidectl/pkg/provider/helm"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var clusterListCmdDesc = `
This command consists of multiple sub-commands to interact with clusters
`

type ClusterCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewClusterCommand(cmdCtx *cmdcontext.CommandContext) *ClusterCommand {
	return &ClusterCommand{
		cmdCtx: cmdCtx,
	}
}

func (c *ClusterCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster del|list [ARGS]",
		Short: "Manage clusters",
		Long:  clusterListCmdDesc,
	}

	cmd.AddCommand(
		c.getListClustersCommand(),
		c.getDelCommand(),
	)

	return cmd
}

func (c *ClusterCommand) getListClustersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List clusters",
		Long:  clusterListCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.ListClusters(cmd.Context())
		},
	}

	return cmd
}

func (c *ClusterCommand) ListClusters(ctx context.Context) error {
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

var clusterDelCmdDesc = `
This command will delete a cluster from the Cofide configuration state.
`

type delOpts struct {
	trustZone string
}

func (c *ClusterCommand) getDelCommand() *cobra.Command {
	opts := delOpts{}
	cmd := &cobra.Command{
		Use:   "del [NAME]",
		Short: "Delete a cluster",
		Long:  clusterDelCmdDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.deleteCluster(cmd.Context(), args[0], opts.trustZone)
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "Name of the cluster's trust zone")

	cobra.CheckErr(cmd.MarkFlagRequired("trust-zone"))
	return cmd
}

func (c *ClusterCommand) deleteCluster(ctx context.Context, name, trustZoneName string) error {
	ds, err := c.cmdCtx.PluginManager.GetDataSource(ctx)
	if err != nil {
		return err
	}

	cluster, err := ds.GetCluster(name, trustZoneName)
	if err != nil {
		return err
	}

	// Fail if the cluster is up.
	if deployed, err := helmprovider.IsClusterDeployed(ctx, cluster); err != nil {
		return err
	} else if deployed {
		return fmt.Errorf("cluster %s in trust zone %s cannot be deleted while it is up", name, trustZoneName)
	}

	return ds.DestroyCluster(name, trustZoneName)
}
