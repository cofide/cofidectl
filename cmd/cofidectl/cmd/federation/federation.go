// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package federation

import (
	"context"
	"os"

	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"

	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	"github.com/cofide/cofidectl/pkg/provider/helm"
	"github.com/cofide/cofidectl/pkg/spire"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const (
	FederationStatus_HEALTHY   string = "Healthy"
	FederationStatus_UNHEALTHY string = "Unhealthy"

	FederationStatusReason_NO_BUNDLE_FOUND      string = "No bundle found"
	FederationStatusReason_BUNDLES_DO_NOT_MATCH string = "Bundles do not match"
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
				from, err := ds.GetTrustZone(federation.From)
				if err != nil {
					return err
				}

				to, err := ds.GetTrustZone(federation.To)
				if err != nil {
					return err
				}

				status, reason, err := checkFederationStatus(cmd.Context(), kubeConfig, from, to)
				if err != nil {
					return err
				}

				data[i] = []string{
					federation.From,
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
func checkFederationStatus(ctx context.Context, kubeConfig string, from *trust_zone_proto.TrustZone, to *trust_zone_proto.TrustZone) (string, string, error) {
	compare := make(map[*trust_zone_proto.TrustZone]bundles)

	for _, tz := range []*trust_zone_proto.TrustZone{from, to} {
		if deployed, err := isTrustZoneDeployed(ctx, tz); err != nil {
			return "", "", err
		} else if !deployed {
			return "Inactive", "", nil
		}

		client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeConfig, tz.GetKubernetesContext())
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
		return FederationStatus_UNHEALTHY, FederationStatusReason_NO_BUNDLE_FOUND, nil
	}

	// Bundle does not match entry on opposite trust domain
	if compare[from].federatedBundles[to.TrustDomain] != compare[to].serverCABundle {
		return FederationStatus_UNHEALTHY, FederationStatusReason_BUNDLES_DO_NOT_MATCH, nil
	}

	return FederationStatus_HEALTHY, "", nil
}

// isTrustZoneDeployed returns whether a trust zone has been deployed, i.e. whether a SPIRE Helm release has been installed.
func isTrustZoneDeployed(ctx context.Context, trustZone *trust_zone_proto.TrustZone) (bool, error) {
	prov, err := helm.NewHelmSPIREProvider(ctx, trustZone, nil, nil)
	if err != nil {
		return false, err
	}
	return prov.CheckIfAlreadyInstalled()
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
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			newFederation := &federation_proto.Federation{
				From: opts.from,
				To:   opts.to,
			}
			_, err = ds.AddFederation(newFederation)
			return err
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.from, "from", "", "Trust zone to federate from")
	f.StringVar(&opts.to, "to", "", "Trust zone to federate to")

	cobra.CheckErr(cmd.MarkFlagRequired("from"))
	cobra.CheckErr(cmd.MarkFlagRequired("to"))

	return cmd
}
