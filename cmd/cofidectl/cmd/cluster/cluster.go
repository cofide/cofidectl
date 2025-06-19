// Copyright 2025 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"os"

	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
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
		clusters, err := ds.ListClusters(&datasourcepb.ListClustersRequest_Filter{
			TrustZoneId: zone.Id,
		})
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
	trustZoneName string
	trustZoneID   string
	force         bool
}

func (c *ClusterCommand) getDelCommand() *cobra.Command {
	opts := delOpts{}
	cmd := &cobra.Command{
		Use:   "del [NAME]",
		Short: "Delete a cluster",
		Long:  clusterDelCmdDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeConfig, err := cmd.Flags().GetString("kube-config")
			if err != nil {
				return err
			}
			return c.deleteCluster(cmd.Context(), args[0], kubeConfig, opts)
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.trustZoneName, "trust-zone-name", "", "Name of the cluster's trust zone")
	f.StringVar(&opts.trustZoneID, "trust-zone-id", "", "ID of the cluster's trust zone")
	f.BoolVar(&opts.force, "force", false, "Skip pre-delete checks")

	cmd.MarkFlagsOneRequired("trust-zone-name", "trust-zone-id")
	return cmd
}

func (c *ClusterCommand) deleteCluster(ctx context.Context, name, kubeConfig string, opts delOpts) error {
	ds, err := c.cmdCtx.PluginManager.GetDataSource(ctx)
	if err != nil {
		return err
	}

	tzId := opts.trustZoneID
	if opts.trustZoneName != "" {
		tz, err := ds.GetTrustZoneByName(opts.trustZoneName)
		if err != nil {
			return fmt.Errorf("failed to get trust zone %s: %v", opts.trustZoneName, err)
		}
		tzId = tz.GetId()
	}

	// TODO: Support delete by ID?
	cluster, err := ds.GetClusterByName(name, tzId)
	if err != nil {
		return err
	}

	if !opts.force {
		// Fail if the cluster is reachable and SPIRE is deployed.
		if deployed, err := helmprovider.IsClusterDeployed(ctx, cluster, kubeConfig); err != nil {
			return err
		} else if deployed {
			return fmt.Errorf("cluster %s in trust zone %s cannot be deleted while it is up", name, tzId)
		}
	}

	return ds.DestroyCluster(cluster.GetId())
}
