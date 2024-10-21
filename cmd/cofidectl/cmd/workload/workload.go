package workload

import (
	"fmt"
	"os"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/cofide/cofidectl/internal/pkg/workload"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type WorkloadCommand struct {
	source cofidectl_plugin.DataSource
}

func NewWorkloadCommand(source cofidectl_plugin.DataSource) *WorkloadCommand {
	return &WorkloadCommand{
		source: source,
	}
}

var workloadRootCmdDesc = `
This command consists of multiple sub-commands to interact with workloads.
`

func (c *WorkloadCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workload list|discover [ARGS]",
		Short: "List or discover trust zone workloads",
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

type Opts struct {
	trustZone string
}

func (w *WorkloadCommand) GetListCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "List workloads",
		Long:  workloadListCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var trustZones []*trust_zone_proto.TrustZone

			if opts.trustZone != "" {
				trustZone, err := w.source.GetTrustZone(opts.trustZone)
				if err != nil {
					return err
				}

				trustZones = append(trustZones, trustZone)
			} else {
				trustZones, err = w.source.ListTrustZones()
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
		registeredWorkloads, err := workload.GetRegisteredWorkloads(kubeConfig, trustZone.KubernetesContext)
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

func (w *WorkloadCommand) GetDiscoverCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "discover [ARGS]",
		Short: "Discover workloads",
		Long:  workloadDiscoverCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var trustZones []*trust_zone_proto.TrustZone

			if opts.trustZone != "" {
				trustZone, err := w.source.GetTrustZone(opts.trustZone)
				if err != nil {
					return err
				}

				trustZones = append(trustZones, trustZone)
			} else {
				trustZones, err = w.source.ListTrustZones()
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

			err = renderUnregisteredWorkloads(kubeConfig, trustZones)
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

func renderUnregisteredWorkloads(kubeConfig string, trustZones []*trust_zone_proto.TrustZone) error {
	data := make([][]string, 0, len(trustZones))

	for _, trustZone := range trustZones {
		registeredWorkloads, err := workload.GetUnregisteredWorkloads(kubeConfig, trustZone.KubernetesContext)
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
			})
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Trust Zone", "Type", "Status", "Namespace"})
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()

	return nil
}
