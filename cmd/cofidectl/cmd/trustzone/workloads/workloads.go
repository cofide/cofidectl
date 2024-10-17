package workloads

import (
	"os"
	"path"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/cofide/cofidectl/internal/pkg/trustzone/workloads"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type WorkloadsCommand struct {
	source cofidectl_plugin.DataSource
}

func NewWorkloadsCommand(source cofidectl_plugin.DataSource) *WorkloadsCommand {
	return &WorkloadsCommand{
		source: source,
	}
}

var workloadsRootCmdDesc = `
This command consists of multiple sub-commands to interact with workloads.
`

var kubeCfgFile string

func (c *WorkloadsCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workloads list|discover [ARGS]",
		Short: "list or discover trust zone workloads",
		Long:  workloadsRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	cmd.PersistentFlags().StringVar(&kubeCfgFile, "kube-config", path.Join(home, ".kube/config"), "kubeconfig file location")

	cmd.AddCommand(
		c.GetListCommand(),
		c.GetDiscoverCommand(),
	)

	return cmd
}

var workloadsListCmdDesc = `
This command will list all of the registered workloads in every trust zone.
`

type Opts struct {
	trust_zone string
}

func (w *WorkloadsCommand) GetListCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "list workloads",
		Long:  workloadsListCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var trustZones []*trust_zone_proto.TrustZone

			if opts.trust_zone != "" {
				trustZone, err := w.source.GetTrustZone(opts.trust_zone)
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

			data := make([][]string, 0, len(trustZones))

			for _, trustZone := range trustZones {
				registeredWorkloads, err := workloads.GetRegisteredWorkloads(kubeCfgFile, trustZone.KubernetesContext)
				if err != nil {
					return err
				}

				for _, workload := range registeredWorkloads {
					data = append(data, []string{
						workload.Name,
						trustZone.Name,
						workload.Type,
						workload.Status,
						workload.SPIFFEID,
					})
				}
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Name", "Trust Zone", "Type", "Status", "Workload ID"})
			table.SetBorder(false)
			table.AppendBulk(data)
			table.Render()

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trust_zone, "trust-zone", "", "list the registered workloads in a specific trust zone")

	return cmd
}

var workloadsDiscoverCmdDesc = `
This command will discover all of the unregistered workloads in every trust zone.
`

func (w *WorkloadsCommand) GetDiscoverCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "discover [ARGS]",
		Short: "discover workloads",
		Long:  workloadsDiscoverCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var trustZones []*trust_zone_proto.TrustZone

			if opts.trust_zone != "" {
				trustZone, err := w.source.GetTrustZone(opts.trust_zone)
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

			data := make([][]string, 0, len(trustZones))

			for _, trustZone := range trustZones {
				registeredWorkloads, err := workloads.GetUnregisteredWorkloads(kubeCfgFile, trustZone.KubernetesContext)
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
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trust_zone, "trust-zone", "", "list the registered workloads in a specific trust zone")

	return cmd
}
