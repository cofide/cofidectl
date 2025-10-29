// Copyright 2025 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_provider/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/trustprovider"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
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
		Use:   "cluster add|del|list [ARGS]",
		Short: "Manage clusters",
		Long:  clusterListCmdDesc,
	}

	cmd.AddCommand(
		c.getAddCommand(),
		c.getListClustersCommand(),
		c.getDelCommand(),
	)

	return cmd
}

var clusterAddCmdDesc = `
This command will add a cluster to the Cofide configuration state.
`

type addOpts struct {
	name                           string
	trustZone                      string
	kubernetesClusterOIDCIssuerURL string
	kubernetesClusterCACert        string
	context                        string
	profile                        string
	externalServer                 bool
}

func (c *ClusterCommand) getAddCommand() *cobra.Command {
	opts := addOpts{}
	cmd := &cobra.Command{
		Use:   "add [NAME]",
		Short: "Add a cluster",
		Long:  clusterAddCmdDesc,
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
			return c.addCluster(cmd.Context(), opts, ds)
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "Name of the trust zone to add the cluster to")
	f.StringVar(&opts.kubernetesClusterOIDCIssuerURL, "kubernetes-oidc-issuer", "", "OIDC issuer URL for the Kubernetes cluster")
	f.StringVar(&opts.kubernetesClusterCACert, "kubernetes-ca-cert", "", "Path to the CA certificate of the Kubernetes cluster, used for TLS during OIDC validation")
	f.StringVar(&opts.context, "kubernetes-context", "", "Kubernetes context to use for this cluster")
	f.StringVar(&opts.profile, "profile", "kubernetes", "Cofide profile used in the installation (e.g. kubernetes, istio)")
	f.BoolVar(&opts.externalServer, "external-server", false, "If the SPIRE server runs externally")

	cobra.CheckErr(cmd.MarkFlagRequired("trust-zone"))
	return cmd
}

func (c *ClusterCommand) addCluster(ctx context.Context, opts addOpts, ds datasource.DataSource) error {
	tz, err := ds.GetTrustZoneByName(opts.trustZone)
	if err != nil {
		return fmt.Errorf("failed to get trust zone %s: %w", opts.trustZone, err)
	}

	trustProviderKind, err := trustprovider.GetTrustProviderKindFromProfile(opts.profile)
	if err != nil {
		return err
	}

	var caBytes []byte
	if opts.kubernetesClusterCACert != "" {
		caBytes, err = parseKubernetesCACertFromPath(opts.kubernetesClusterCACert)
		if err != nil {
			return fmt.Errorf("failed to create cluster with CA cert %w", err)
		}
	}

	newCluster := &clusterpb.Cluster{
		Name:              &opts.name,
		TrustZoneId:       tz.Id,
		KubernetesContext: &opts.context,
		TrustProvider:     &trust_provider_proto.TrustProvider{Kind: &trustProviderKind},
		Profile:           &opts.profile,
		ExternalServer:    &opts.externalServer,
		OidcIssuerUrl:     &opts.kubernetesClusterOIDCIssuerURL,
	}

	if caBytes != nil {
		newCluster.OidcIssuerCaCert = caBytes
	}

	_, err = ds.AddCluster(newCluster)
	return err
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
	trustZone string
	force     bool
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
			return c.deleteCluster(cmd.Context(), args[0], opts.trustZone, kubeConfig, opts.force)
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "Name of the cluster's trust zone")
	f.BoolVar(&opts.force, "force", false, "Skip pre-delete checks")

	cobra.CheckErr(cmd.MarkFlagRequired("trust-zone"))
	return cmd
}

func (c *ClusterCommand) deleteCluster(ctx context.Context, name, trustZoneName, kubeConfig string, force bool) error {
	ds, err := c.cmdCtx.PluginManager.GetDataSource(ctx)
	if err != nil {
		return err
	}

	tz, err := ds.GetTrustZoneByName(trustZoneName)
	if err != nil {
		return fmt.Errorf("failed to get trust zone %s: %w", trustZoneName, err)
	}

	cluster, err := ds.GetClusterByName(name, tz.GetId())
	if err != nil {
		return err
	}

	if !force {
		// Fail if the cluster is reachable and SPIRE is deployed.
		if deployed, err := helmprovider.IsClusterDeployed(ctx, cluster, kubeConfig); err != nil {
			return err
		} else if deployed {
			return fmt.Errorf("cluster %s in trust zone %s cannot be deleted while it is up", name, trustZoneName)
		}
	}

	return ds.DestroyCluster(cluster.GetId())
}

func validateOpts(opts addOpts) error {
	normalisedURL, err := validateAndParseOIDCIssuerURL(opts.kubernetesClusterOIDCIssuerURL)
	if err != nil {
		return fmt.Errorf("invalid --kubernetes-oidc-issuer: %w", err)
	}
	opts.kubernetesClusterOIDCIssuerURL = normalisedURL

	return nil
}

func validateAndParseOIDCIssuerURL(oidcIssuerURL string) (string, error) {
	// It's an optional flag, so if it's empty, it's valid.
	if oidcIssuerURL == "" {
		return "", nil
	}

	u, err := url.ParseRequestURI(oidcIssuerURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	if u.Scheme != "https" {
		return "", fmt.Errorf("URL scheme must be 'https', but got '%s'", u.Scheme)
	}

	if u.Host == "" {
		return "", fmt.Errorf("URL must include a host")
	}

	if u.RawQuery != "" {
		return "", fmt.Errorf("URL must not have a query component")
	}

	if u.Fragment != "" {
		return "", fmt.Errorf("URL must not have a fragment component")
	}

	u.Path = strings.TrimRight(u.Path, "/")
	return u.String(), nil
}

func parseKubernetesCACertFromPath(path string) ([]byte, error) {
	return os.ReadFile(path)
}
