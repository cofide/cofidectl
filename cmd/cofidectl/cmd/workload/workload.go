package workload

import (
	"context"
	"fmt"
	"log"

	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
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

type Opts struct {
	workload_name string
	pod_name      string
	namespace     string
	trust_zone    string
}

func (w *WorkloadCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workload status",
		Short: "status workloads",
		Long:  workloadRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		w.GetStatusCommand(),
	)

	return cmd
}

var workloadStatusCmdDesc = `
This command will display the status of workloads in the Cofide configuration state.
`

func (w *WorkloadCommand) GetStatusCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "status [NAME]",
		Short: "Display workload status",
		Long:  workloadStatusCmdDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeConfig, err := cmd.Flags().GetString("kube-config")
			if err != nil {
				return fmt.Errorf("failed to retrieve the kubeconfig file location")
			}
			opts.workload_name = args[0]
			return w.status(cmd.Context(), kubeConfig, opts)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.pod_name, "pod-name", "", "Pod name for the workload")
	f.StringVar(&opts.namespace, "namespace", "", "Namespace for the workload")
	f.StringVar(&opts.trust_zone, "trust-zone", "", "Trust zone for the workload")

	return cmd
}

func (w *WorkloadCommand) status(ctx context.Context, kubeConfig string, opts Opts) error {
	trustZone, err := w.source.GetTrustZone(opts.trust_zone)
	if err != nil {
		return err
	}

	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeConfig, trustZone.KubernetesContext)
	if err != nil {
		return err
	}

	if err := createDebugContainer(ctx, client); err != nil {
		log.Fatalf("Error creating debug container: %v", err)
	}

	workload, err := getWorkloadStatus(ctx, client)
	if err != nil {
		return err
	}

	fmt.Println(workload)

	return nil
}

func createDebugContainer(ctx context.Context, client *kubeutil.Client) error {
	// TODO
	return nil
}

func getWorkloadStatus(ctx context.Context, client *kubeutil.Client) (string, error) {
	// TODO
	return "", nil
}
