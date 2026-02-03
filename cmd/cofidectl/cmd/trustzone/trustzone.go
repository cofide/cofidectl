// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustzone

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/trustzone/helm"
	"github.com/cofide/cofidectl/internal/pkg/trustzone"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/renderer"
	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	helmprovider "github.com/cofide/cofidectl/pkg/provider/helm"
	"github.com/cofide/cofidectl/pkg/spire"
	"github.com/spf13/cobra"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

type TrustZoneCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewTrustZoneCommand(cmdCtx *cmdcontext.CommandContext) *TrustZoneCommand {
	return &TrustZoneCommand{
		cmdCtx: cmdCtx,
	}
}

var trustZoneRootCmdDesc = `
This command consists of multiple sub-commands to administer Cofide trust zones.
`

func (c *TrustZoneCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust-zone add|del|list|status [ARGS]",
		Short: "Manage trust zones",
		Long:  trustZoneRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	helmCmd := helm.NewHelmCommand(c.cmdCtx)

	cmd.AddCommand(
		c.GetListCommand(),
		c.GetAddCommand(),
		c.GetDelCommand(),
		c.GetStatusCommand(),
		helmCmd.GetRootCommand(),
	)

	return cmd
}

var trustZoneListCmdDesc = `
This command will list trust zones in the Cofide configuration state.
`

func (c *TrustZoneCommand) GetListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "List trust zones",
		Long:  trustZoneListCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			trustZones, err := ds.ListTrustZones()
			if err != nil {
				return err
			}

			data := make([][]string, len(trustZones))
			for i, trustZone := range trustZones {
				cluster, err := trustzone.GetClusterFromTrustZone(trustZone, ds)
				if err != nil && !errors.Is(err, trustzone.ErrNoClustersInTrustZone) {
					return err
				}

				clusterName := "N/A"
				if cluster != nil {
					clusterName = cluster.GetName()
				}

				data[i] = []string{
					trustZone.Name,
					trustZone.TrustDomain,
					clusterName,
				}
			}

			tr := renderer.NewTableRenderer(os.Stdout)
			table := renderer.Table{
				Header: []string{"Name", "Trust Domain", "Cluster"},
				Data:   data,
			}
			_, err = tr.RenderTables(table)
			return err
		},
	}

	return cmd
}

var trustZoneAddCmdDesc = `
This command will add a new trust zone to the Cofide configuration state.
`

type addOpts struct {
	name           string
	trustDomain    string
	jwtIssuer      string
	externalServer bool
}

func (c *TrustZoneCommand) GetAddCommand() *cobra.Command {
	opts := addOpts{}
	cmd := &cobra.Command{
		Use:   "add [NAME]",
		Short: "Add a new trust zone",
		Long:  trustZoneAddCmdDesc,
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			if err := validateOpts(opts); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}
			return c.addTrustZone(cmd.Context(), opts, ds)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustDomain, "trust-domain", "", "Trust domain to use for this trust zone")
	f.StringVar(&opts.jwtIssuer, "jwt-issuer", "", "JWT issuer to use for this trust zone")
	f.BoolVar(&opts.externalServer, "external-server", false, "If the SPIRE server runs externally")

	cobra.CheckErr(cmd.MarkFlagRequired("trust-domain"))

	return cmd
}

func (c *TrustZoneCommand) addTrustZone(ctx context.Context, opts addOpts, ds datasource.DataSource) error {
	bundleEndpointProfile := trust_zone_proto.BundleEndpointProfile_BUNDLE_ENDPOINT_PROFILE_HTTPS_SPIFFE

	newTrustZone := &trust_zone_proto.TrustZone{
		Name:                  opts.name,
		TrustDomain:           opts.trustDomain,
		JwtIssuer:             &opts.jwtIssuer,
		BundleEndpointProfile: &bundleEndpointProfile,
	}

	_, err := ds.AddTrustZone(newTrustZone)
	if err != nil {
		return fmt.Errorf("failed to create trust zone %s: %w", newTrustZone.GetName(), err)
	}

	return nil
}

var trustZoneDelCmdDesc = `
This command will delete a trust zone from the Cofide configuration state.
`

type delOpts struct {
	force bool
}

func (c *TrustZoneCommand) GetDelCommand() *cobra.Command {
	opts := &delOpts{}
	cmd := &cobra.Command{
		Use:   "del [NAME]",
		Short: "Delete a trust zone",
		Long:  trustZoneDelCmdDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			kubeConfig, err := cmd.Flags().GetString("kube-config")
			if err != nil {
				return err
			}

			return deleteTrustZone(cmd.Context(), args[0], ds, kubeConfig, opts.force)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&opts.force, "force", false, "Skip pre-delete checks")

	return cmd
}

func deleteTrustZone(ctx context.Context, name string, ds datasource.DataSource, kubeConfig string, force bool) error {
	tz, err := ds.GetTrustZoneByName(name)
	if err != nil {
		return err
	}
	id := tz.GetId()

	clusters, err := ds.ListClusters(&datasourcepb.ListClustersRequest_Filter{
		TrustZoneId: &id,
	})
	if err != nil {
		return err
	}

	// TODO: Add IsClusterDeployed to ProvisionPlugin interface and mock in tests.
	if !force {
		// Fail if any clusters in the trust zone are reachable and SPIRE is deployed.
		for _, cluster := range clusters {
			if deployed, err := helmprovider.IsClusterDeployed(ctx, cluster, kubeConfig); err != nil {
				return err
			} else if deployed {
				return fmt.Errorf("cluster %s in trust zone %s cannot be deleted while it is up", cluster.GetName(), name)
			}
		}
	}

	for i, cluster := range clusters {
		err = ds.DestroyCluster(cluster.GetId())
		if err != nil {
			for _, rollbackCluster := range clusters[:i] {
				if _, err := ds.AddCluster(rollbackCluster); err != nil {
					slog.Error("Failed recreating cluster during rollback", "error", err)
				}
			}
			return fmt.Errorf("failed to destroy cluster %s: %w", cluster.GetName(), err)
		}
	}

	err = ds.DestroyTrustZone(id)
	if err != nil {
		return fmt.Errorf("failed to destroy trust zone %s: %w", name, err)
	}

	return nil
}

var trustZoneStatusCmdDesc = `
This command will display the status of trust zones in the Cofide configuration state.

NOTE: This command relies on privileged access to execute SPIRE server CLI commands within the SPIRE server container, which may not be suitable for production environments.
`

func (c *TrustZoneCommand) GetStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [NAME]",
		Short: "Display trust zone status",
		Long:  trustZoneStatusCmdDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			kubeConfig, err := cmd.Flags().GetString("kube-config")
			if err != nil {
				return fmt.Errorf("failed to retrieve the kubeconfig file location")
			}
			return c.status(cmd.Context(), ds, kubeConfig, args[0])
		},
	}

	return cmd
}

func (c *TrustZoneCommand) status(ctx context.Context, source datasource.DataSource, kubeConfig, tzName string) error {
	trustZone, err := source.GetTrustZoneByName(tzName)
	if err != nil {
		return err
	}

	cluster, err := trustzone.GetClusterFromTrustZone(trustZone, source)
	if err != nil {
		return err
	}

	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeConfig, cluster.GetKubernetesContext())
	if err != nil {
		return err
	}

	prov, err := helmprovider.NewHelmSPIREProvider(ctx, trustZone.GetName(), cluster, nil, nil, helmprovider.WithKubeConfig(kubeConfig))
	if err != nil {
		return err
	}

	if installed, err := prov.CheckIfAlreadyInstalled(); err != nil {
		return err
	} else if !installed {
		//nolint:staticcheck // ST1005: error strings should not be capitalized
		return errors.New("Cofide configuration has not been installed. Have you run cofidectl up?")
	}

	server, err := spire.GetServerStatus(ctx, client)
	if err != nil {
		return err
	}

	agents, err := spire.GetAgentStatus(ctx, client)
	if err != nil {
		return err
	}

	return renderStatus(trustZone, server, agents)
}

func renderStatus(trustZone *trust_zone_proto.TrustZone, server *spire.ServerStatus, agents *spire.AgentStatus) error {
	trustZoneData := [][]string{
		{
			"Trust Zone",
			trustZone.Name,
		},
		{
			"SPIRE Servers ready",
			fmt.Sprintf("%d/%d", server.ReadyReplicas, server.Replicas),
		},
		{
			"SPIRE Agents ready",
			fmt.Sprintf("%d/%d", agents.Ready, agents.Expected),
		},
		{
			"Bundle Endpoint",
			trustZone.GetBundleEndpointUrl(),
		},
	}

	serverData := make([][]string, 0)
	for _, container := range server.Containers {
		serverData = append(serverData, []string{
			container.Name,
			strconv.FormatBool(container.Ready),
		})
	}

	scmData := make([][]string, 0)
	for _, scm := range server.SCMs {
		scmData = append(scmData, []string{
			scm.Name,
			strconv.FormatBool(scm.Ready),
		})
	}

	agentData := make([][]string, 0)
	for _, agent := range agents.Agents {
		agentData = append(agentData, []string{
			agent.Name,
			agent.Status,
			agent.AttestationType,
			agent.ExpirationTime.String(),
			strconv.FormatBool(agent.CanReattest),
		})
	}

	agentIdData := make([][]string, 0)
	for _, agent := range agents.Agents {
		agentIdData = append(agentIdData, []string{
			agent.Name,
			agent.Id,
		})
	}

	tr := renderer.NewTableRenderer(os.Stdout)
	_, err := tr.RenderTables(
		renderer.Table{
			Title:  "Trust Zone",
			Header: []string{"Item", "Value"},
			Data:   trustZoneData,
		},
		renderer.Table{
			Title:  "SPIRE Servers",
			Header: []string{"Pod", "Ready"},
			Data:   serverData,
		},
		renderer.Table{
			Title:  "SPIRE Controller Managers",
			Header: []string{"Pod", "Ready"},
			Data:   scmData,
		},
		renderer.Table{
			Title:  "SPIRE Agents",
			Header: []string{"Pod", "Status", "Attestation type", "Expiration time", "Can re-attest"},
			Data:   agentData,
		},
		renderer.Table{
			Title:  "SPIRE Agents SPIFFE IDs",
			Header: []string{"Pod", "SPIFFE ID"},
			Data:   agentIdData,
		},
	)
	return err
}

func validateOpts(opts addOpts) error {
	_, err := spiffeid.TrustDomainFromString(opts.trustDomain)
	if err != nil {
		return err
	}

	return nil
}
