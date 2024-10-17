package workloads

import (
	"os"
	"path"

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
This command consists of multiple sub-commands to interact with Cofide trust zone workloads.
`

var kubeCfgFile string

func (c *WorkloadsCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workloads list [ARGS]",
		Short: "list trust zone workloads",
		Long:  workloadsRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	cmd.PersistentFlags().StringVar(&kubeCfgFile, "kube-config", path.Join(home, ".kube/config"), "kubeconfig file location")

	cmd.AddCommand(c.GetListCommand())

	return cmd
}

var workloadsListCmdDesc = `
This command will list the workloads in a trust zone.
`

func (w *WorkloadsCommand) GetListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "list workloads",
		Long:  workloadsListCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			trustZones, err := w.source.ListTrustZones()
			if err != nil {
				return err
			}

			data := make([][]string, 0, 2)

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
			table.SetHeader([]string{"Name", "Trust Zone", "Type", "Status", "SPIFFE ID"})
			table.SetBorder(false)
			table.AppendBulk(data)
			table.Render()

			return nil
		},
	}

	return cmd
}
