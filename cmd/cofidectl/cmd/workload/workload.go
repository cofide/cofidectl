package workload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/briandowns/spinner"
	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	cmd.MarkFlagRequired("pod-name")
	cmd.MarkFlagRequired("namespace")
	cmd.MarkFlagRequired("trust-zone")

	return cmd
}

const debugContainerName = "cofidectl-debug-container"
const debugContainerImage = "cofidectl-debug:latest"

func (w *WorkloadCommand) status(ctx context.Context, kubeConfig string, opts Opts) error {
	trustZone, err := w.source.GetTrustZone(opts.trust_zone)
	if err != nil {
		return err
	}

	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeConfig, trustZone.KubernetesContext)
	if err != nil {
		return err
	}

	// Create a spinner to display whilst installation is underway
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Start()
	s.Suffix = "Starting debug container"

	pod, err := createDebugContainer(ctx, client, opts.pod_name, opts.namespace)
	if err != nil {
		return err
	}

	s.Suffix = "Retrieving workload status"

	workload, err := getWorkloadStatus(ctx, client, pod)
	if err != nil {
		return err
	}

	s.Stop()

	fmt.Println(workload)

	return nil
}

func createDebugContainer(ctx context.Context, client *kubeutil.Client, podName string, namespace string) (*corev1.Pod, error) {
	pod, err := client.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting pod: %v", err)
	}

	// Check if debug container already exists
	for _, ec := range pod.Spec.EphemeralContainers {
		if ec.Name == debugContainerName {
			return pod, nil // Debug container already exists
		}
	}

	debugContainer := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:            debugContainerName,
			Image:           debugContainerImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			TTY:             true,
			Stdin:           true,
		},
		TargetContainerName: pod.Spec.Containers[0].Name,
	}

	pod.Spec.EphemeralContainers = append(pod.Spec.EphemeralContainers, debugContainer)
	_, err = client.Clientset.CoreV1().Pods(namespace).UpdateEphemeralContainers(
		ctx,
		pod.Name,
		pod,
		metav1.UpdateOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating debug container: %v", err)
	}

	// Wait for debug container to be ready
	for {
		pod, err := client.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting pod status: %v", err)
		}

		for _, status := range pod.Status.EphemeralContainerStatuses {
			if status.Name == debugContainerName && status.State.Running != nil {
				return pod, nil
			}
		}
	}
}

func getWorkloadStatus(ctx context.Context, client *kubeutil.Client, pod *corev1.Pod) (string, error) {
	for _, status := range pod.Status.EphemeralContainerStatuses {
		if status.Name == debugContainerName {

			// If container has terminated, get its logs
			if status.State.Terminated != nil {
				logs, err := client.Clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
					Container: debugContainerName,
				}).Stream(ctx)
				if err != nil {
					return "", err
				}
				defer logs.Close()

				// Read the logs
				buf := new(bytes.Buffer)
				_, err = io.Copy(buf, logs)
				if err != nil {
					return "", err
				}

				return buf.String(), nil
			}
		}
	}
	return "", fmt.Errorf("could not determine status")
}
