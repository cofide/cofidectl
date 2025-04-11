// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package workload

import (
	"context"
	"errors"
	"fmt"
	"os"

	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/provision_plugin/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/statusspinner"
	"github.com/cofide/cofidectl/internal/pkg/trustzone"
	"github.com/cofide/cofidectl/internal/pkg/workload"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/provider/helm"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type WorkloadCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewWorkloadCommand(cmdCtx *cmdcontext.CommandContext) *WorkloadCommand {
	return &WorkloadCommand{
		cmdCtx: cmdCtx,
	}
}

var workloadRootCmdDesc = `
This command consists of multiple sub-commands to interact with workloads.
`

func (c *WorkloadCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workload list|discover|status [ARGS]",
		Short: "List or introspect the status of workloads in a trust zone or discover candidate workloads",
		Long:  workloadRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		c.GetStatusCommand(),
		c.GetListCommand(),
		c.GetDiscoverCommand(),
	)

	return cmd
}

var workloadListCmdDesc = `
This command will list all of the registered workloads.
`

type ListOpts struct {
	trustZone string
}

func (w *WorkloadCommand) GetListCommand() *cobra.Command {
	opts := ListOpts{}
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "List workloads",
		Long:  workloadListCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			ds, err := w.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			var trustZones []*trust_zone_proto.TrustZone

			if opts.trustZone != "" {
				trustZone, err := ds.GetTrustZone(opts.trustZone)
				if err != nil {
					return err
				}

				trustZones = append(trustZones, trustZone)
			} else {
				trustZones, err = ds.ListTrustZones()
				if err != nil {
					return err
				}
			}

			if len(trustZones) == 0 {
				return fmt.Errorf("no trust zones have been configured")
			}

			kubeConfig, err := cmd.Flags().GetString("kube-config")
			if err != nil {
				return fmt.Errorf("failed to retrieve the kubeconfig file location")
			}

			err = renderRegisteredWorkloads(cmd.Context(), ds, kubeConfig, trustZones)
			if err != nil {
				return err
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "list the registered workloads in a specific trust zone")

	return cmd
}

var workloadStatusCmdDesc = `
This command will display the status of workloads in the Cofide configuration state.
`

type StatusOpts struct {
	podName   string
	namespace string
	trustZone string
}

func (w *WorkloadCommand) GetStatusCommand() *cobra.Command {
	opts := StatusOpts{}
	cmd := &cobra.Command{
		Use:   "status [ARGS]",
		Short: "Display workload status",
		Long:  workloadStatusCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := w.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			kubeConfig, err := cmd.Flags().GetString("kube-config")
			if err != nil {
				return fmt.Errorf("failed to retrieve the kubeconfig file location")
			}

			return w.status(cmd.Context(), ds, kubeConfig, opts)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.podName, "pod-name", "", "Pod name for the workload")
	f.StringVar(&opts.namespace, "namespace", "", "Namespace for the workload")
	f.StringVar(&opts.trustZone, "trust-zone", "", "Trust zone for the workload")

	cobra.CheckErr(cmd.MarkFlagRequired("pod-name"))
	cobra.CheckErr(cmd.MarkFlagRequired("namespace"))
	cobra.CheckErr(cmd.MarkFlagRequired("trust-zone"))

	return cmd
}

func (w *WorkloadCommand) status(ctx context.Context, ds datasource.DataSource, kubeConfig string, opts StatusOpts) error {
	trustZone, err := ds.GetTrustZone(opts.trustZone)
	if err != nil {
		return err
	}

	cluster, err := trustzone.GetClusterFromTrustZone(trustZone, ds)
	if err != nil {
		return err
	}

	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeConfig, cluster.GetKubernetesContext())
	if err != nil {
		return err
	}

	statusCh, dataCh := getWorkloadStatus(ctx, client, opts.podName, opts.namespace)

	// Create a spinner to display whilst the debug container is created and executed and logs retrieved
	if err := statusspinner.WatchProvisionStatus(ctx, statusCh, false); err != nil {
		return fmt.Errorf("retrieving workload status failed: %w", err)
	}

	result := <-dataCh
	if result == "" {
		return fmt.Errorf("retrieving workload status failed")
	}

	fmt.Println(result)
	return nil
}

func renderRegisteredWorkloads(ctx context.Context, ds datasource.DataSource, kubeConfig string, trustZones []*trust_zone_proto.TrustZone) error {
	data := make([][]string, 0, len(trustZones))

	for _, trustZone := range trustZones {
		cluster, err := trustzone.GetClusterFromTrustZone(trustZone, ds)
		if err != nil {
			if errors.Is(err, trustzone.ErrNoClustersInTrustZone) {
				continue
			}
			return err
		}

		if deployed, err := helm.IsClusterDeployed(ctx, cluster, kubeConfig); err != nil {
			return err
		} else if !deployed {
			return fmt.Errorf("trust zone %s has not been deployed", trustZone.Name)
		}

		registeredWorkloads, err := workload.GetRegisteredWorkloads(ctx, kubeConfig, cluster.GetKubernetesContext())
		if err != nil {
			return err
		}

		for _, workload := range registeredWorkloads {
			data = append(data, []string{
				workload.Name,
				trustZone.Name,
				workload.Type,
				workload.Status,
				workload.Namespace,
				workload.SPIFFEID,
			})
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Trust Zone", "Type", "Status", "Namespace", "Workload ID"})
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()

	return nil
}

func getWorkloadStatus(ctx context.Context, client *kubeutil.Client, podName string, namespace string) (<-chan *provisionpb.Status, chan string) {
	statusCh := make(chan *provisionpb.Status)
	dataCh := make(chan string, 1)

	go func() {
		defer close(statusCh)
		defer close(dataCh)
		workload.GetStatus(ctx, statusCh, dataCh, client, podName, namespace)
	}()

	return statusCh, dataCh
}

var workloadDiscoverCmdDesc = `
This command will discover all of the unregistered workloads.
`

type DiscoverOpts struct {
	trustZone      string
	includeSecrets bool
}

func (w *WorkloadCommand) GetDiscoverCommand() *cobra.Command {
	opts := DiscoverOpts{}
	cmd := &cobra.Command{
		Use:   "discover [ARGS]",
		Short: "Discover workloads",
		Long:  workloadDiscoverCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			ds, err := w.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			var trustZones []*trust_zone_proto.TrustZone

			if opts.trustZone != "" {
				trustZone, err := ds.GetTrustZone(opts.trustZone)
				if err != nil {
					return err
				}

				trustZones = append(trustZones, trustZone)
			} else {
				trustZones, err = ds.ListTrustZones()
				if err != nil {
					return err
				}
			}

			if len(trustZones) == 0 {
				return fmt.Errorf("no trust zones have been configured")
			}

			kubeConfig, err := cmd.Flags().GetString("kube-config")
			if err != nil {
				return fmt.Errorf("failed to retrieve the kubeconfig file location")
			}

			err = renderUnregisteredWorkloads(cmd.Context(), ds, kubeConfig, trustZones, opts.includeSecrets)
			if err != nil {
				return err
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "list the unregistered workloads in a specific trust zone")
	f.BoolVar(&opts.includeSecrets, "include-secrets", false, "discover workload secrets and analyse for risk")

	return cmd
}

func renderUnregisteredWorkloads(ctx context.Context, ds datasource.DataSource, kubeConfig string, trustZones []*trust_zone_proto.TrustZone, includeSecrets bool) error {
	data := make([][]string, 0, len(trustZones))

	for _, trustZone := range trustZones {
		cluster, err := trustzone.GetClusterFromTrustZone(trustZone, ds)
		if err != nil {
			if errors.Is(err, trustzone.ErrNoClustersInTrustZone) {
				continue
			}
			return err
		}

		deployed, err := helm.IsClusterDeployed(ctx, cluster, kubeConfig)
		if err != nil {
			return err
		}

		registeredWorkloads, err := workload.GetUnregisteredWorkloads(ctx, kubeConfig, cluster.GetKubernetesContext(), includeSecrets, deployed)
		if err != nil {
			return err
		}

		for _, workload := range registeredWorkloads {
			rows := []string{
				workload.Name,
				trustZone.Name,
				workload.Type,
				workload.Status,
				workload.Namespace,
			}
			if includeSecrets {
				rows = append(rows, fmt.Sprintf("%d (%d at risk)", workload.NumSecrets, workload.NumSecretsAtRisk))
			}
			data = append(data, rows)
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	headers := []string{"Name", "Trust Zone", "Type", "Status", "Namespace"}
	if includeSecrets {
		headers = append(headers, "Secrets")
	}
	table.SetHeader(headers)
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()

	return nil
}
