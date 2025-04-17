// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package federation

import (
	"context"
	"errors"
	"fmt"
	"os"

	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/trustzone"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"

	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	"github.com/cofide/cofidectl/pkg/provider/helm"
	"github.com/cofide/cofidectl/pkg/spire"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const (
	FederationStatusHealthy   string = "Healthy"
	FederationStatusUnhealthy string = "Unhealthy"

	FederationStatusReasonNoBundleFound     string = "No bundle found"
	FederationStatusReasonBundlesDoNotMatch string = "Bundles do not match"
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
		Use:   "federation add|del|list [ARGS]",
		Short: "Manage federations",
		Long:  federationRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		c.GetListCommand(),
		c.GetAddCommand(),
		c.getDelCommand(),
	)

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
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			kubeConfig, err := cmd.Flags().GetString("kube-config")
			if err != nil {
				return err
			}

			federations, err := ds.ListFederations()
			if err != nil {
				return err
			}

			data := make([][]string, len(federations))
			for i, federation := range federations {
				// nolint:staticcheck
				from, err := ds.GetTrustZone(federation.From)
				if err != nil {
					return err
				}

				// nolint:staticcheck
				to, err := ds.GetTrustZone(federation.To)
				if err != nil {
					return err
				}

				status, reason, err := checkFederationStatus(cmd.Context(), ds, kubeConfig, from, to)
				if err != nil {
					return err
				}

				data[i] = []string{
					// nolint:staticcheck
					federation.From,
					// nolint:staticcheck
					federation.To,
					status,
					reason,
				}
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"From Trust Zone", "To Trust Zone", "Status", "Reason"})
			table.SetBorder(false)
			table.AppendBulk(data)
			table.Render()
			return nil
		},
	}

	return cmd
}

type bundles struct {
	serverCABundle   string
	federatedBundles map[string]string
}

// checkFederationStatus builds a comparison map between two trust domains, retrieves there server CA bundle and any federated bundles available
// locally from the SPIRE server, and then compares the bundles on each to verify SPIRE has the correct bundles on each side of the federation
func checkFederationStatus(ctx context.Context, ds datasource.DataSource, kubeConfig string, from *trust_zone_proto.TrustZone, to *trust_zone_proto.TrustZone) (string, string, error) {
	compare := make(map[*trust_zone_proto.TrustZone]bundles)

	for _, tz := range []*trust_zone_proto.TrustZone{from, to} {
		cluster, err := trustzone.GetClusterFromTrustZone(tz, ds)
		if err != nil {
			if errors.Is(err, trustzone.ErrNoClustersInTrustZone) {
				return "No cluster", "N/A", nil
			}
			return "", "", err
		}

		if deployed, err := helm.IsClusterDeployed(ctx, cluster); err != nil {
			return "", "", err
		} else if !deployed {
			return "Inactive", "", nil
		}

		client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeConfig, cluster.GetKubernetesContext())
		if err != nil {
			return "", "", err
		}

		serverCABundle, federatedBundles, err := spire.GetServerCABundleAndFederatedBundles(ctx, client)
		if err != nil {
			return "", "", err
		}

		compare[tz] = bundles{
			serverCABundle:   serverCABundle,
			federatedBundles: federatedBundles,
		}
	}

	// Bundle does not exist at all on opposite trust domain
	_, ok := compare[from].federatedBundles[to.TrustDomain]
	if !ok {
		return FederationStatusUnhealthy, FederationStatusReasonNoBundleFound, nil
	}

	// Bundle does not match entry on opposite trust domain
	if compare[from].federatedBundles[to.TrustDomain] != compare[to].serverCABundle {
		return FederationStatusUnhealthy, FederationStatusReasonBundlesDoNotMatch, nil
	}

	return FederationStatusHealthy, "", nil
}

var federationAddCmdDesc = `
This command will add a new federation to the Cofide configuration state.
`

type Opts struct {
	trustZone       string
	remoteTrustZone string
}

func (c *FederationCommand) GetAddCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new federation",
		Long:  federationAddCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Remove these checks when from/to have been deprecated.
			if opts.trustZone == "" {
				return fmt.Errorf(`Error: required flag(s) "trust-zone" not set`)
			}
			if opts.remoteTrustZone == "" {
				return fmt.Errorf(`Error: required flag(s) "remote-trust-zone" not set`)
			}

			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			newFederation := &federation_proto.Federation{
				From: opts.trustZone,
				To:   opts.remoteTrustZone,
			}
			_, err = ds.AddFederation(newFederation)
			return err
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "Local trust zone")
	f.StringVar(&opts.remoteTrustZone, "remote-trust-zone", "", "Remote trust zone to federate with")

	// TODO: Remove the following arguments after a suitable period.
	f.StringVar(&opts.trustZone, "from", "", "Local trust zone")
	f.StringVar(&opts.remoteTrustZone, "to", "", "Remote trust zone to federate with")

	// TODO: Uncomment this when from/to have been deprecated.
	// cobra.CheckErr(cmd.MarkFlagRequired("trust-zone"))
	// cobra.CheckErr(cmd.MarkFlagRequired("remote-trust-zone"))

	cmd.MarkFlagsMutuallyExclusive("from", "trust-zone")
	cmd.MarkFlagsMutuallyExclusive("to", "remote-trust-zone")

	return cmd
}

var federationDelCmdDesc = `
This command will delete a federation from the Cofide configuration state.
`

func (c *FederationCommand) getDelCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "del",
		Short: "Delete a federation",
		Long:  federationDelCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.deleteFederation(cmd.Context(), opts.trustZone, opts.remoteTrustZone)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "Local trust zone")
	f.StringVar(&opts.remoteTrustZone, "remote-trust-zone", "", "Remote trust zone to federate with")

	cobra.CheckErr(cmd.MarkFlagRequired("trust-zone"))
	cobra.CheckErr(cmd.MarkFlagRequired("remote-trust-zone"))

	return cmd
}

func (c *FederationCommand) deleteFederation(ctx context.Context, trustZone, remoteTrustZone string) error {
	ds, err := c.cmdCtx.PluginManager.GetDataSource(ctx)
	if err != nil {
		return err
	}

	federation := &federation_proto.Federation{
		From: trustZone,
		To:   remoteTrustZone,
	}
	return ds.DestroyFederation(federation)
}
