// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package workload

import (
	"fmt"
	"os"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	cmdcontext "github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"
	"github.com/cofide/cofidectl/internal/pkg/workload"
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
		Use:   "workload list|discover [ARGS]",
		Short: "List workloads in a trust zone or discover candidate workloads",
		Long:  workloadRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
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
			ds, err := w.cmdCtx.PluginManager.GetDataSource()
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

			err = renderRegisteredWorkloads(kubeConfig, trustZones)
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

func renderRegisteredWorkloads(kubeConfig string, trustZones []*trust_zone_proto.TrustZone) error {
	data := make([][]string, 0, len(trustZones))

	for _, trustZone := range trustZones {
		registeredWorkloads, err := workload.GetRegisteredWorkloads(kubeConfig, trustZone.GetKubernetesContext())
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
			ds, err := w.cmdCtx.PluginManager.GetDataSource()
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

			err = renderUnregisteredWorkloads(kubeConfig, trustZones, opts.includeSecrets)
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

func renderUnregisteredWorkloads(kubeConfig string, trustZones []*trust_zone_proto.TrustZone, includeSecrets bool) error {
	data := make([][]string, 0, len(trustZones))

	for _, trustZone := range trustZones {
		registeredWorkloads, err := workload.GetUnregisteredWorkloads(kubeConfig, trustZone.GetKubernetesContext(), includeSecrets)
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
