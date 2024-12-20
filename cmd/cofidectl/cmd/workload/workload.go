// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package workload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/briandowns/spinner"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/workload"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	"github.com/cofide/cofidectl/pkg/provider/helm"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
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

			err = renderRegisteredWorkloads(cmd.Context(), kubeConfig, trustZones)
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
		Use:   "status [NAME]",
		Short: "Display workload status",
		Long:  workloadStatusCmdDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeConfig, err := cmd.Flags().GetString("kube-config")
			if err != nil {
				return fmt.Errorf("failed to retrieve the kubeconfig file location")
			}

			return w.status(cmd.Context(), kubeConfig, opts)
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

const debugContainerNamePrefix = "cofidectl-debug"
const debugContainerImage = "ghcr.io/cofide/cofidectl-debug-container/cmd:v0.1.0"

func (w *WorkloadCommand) status(ctx context.Context, kubeConfig string, opts StatusOpts) error {
	ds, err := w.cmdCtx.PluginManager.GetDataSource(ctx)
	if err != nil {
		return err
	}

	trustZone, err := ds.GetTrustZone(opts.trustZone)
	if err != nil {
		return err
	}

	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeConfig, *trustZone.KubernetesContext)
	if err != nil {
		return err
	}

	// Create a spinner to display whilst the debug container is created and executed and logs retrieved
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Start()
	defer s.Stop()
	s.Suffix = "Starting debug container"

	pod, container, err := createDebugContainer(ctx, client, opts.podName, opts.namespace)
	if err != nil {
		return fmt.Errorf("could not create ephemeral debug container: %s", err)
	}

	s.Suffix = "Retrieving workload status"

	workload, err := getWorkloadStatus(ctx, client, pod, container)
	if err != nil {
		return fmt.Errorf("could not retrieve logs of the ephemeral debug container: %w", err)
	}

	fmt.Println(workload)

	return nil
}

func createDebugContainer(ctx context.Context, client *kubeutil.Client, podName string, namespace string) (*corev1.Pod, string, error) {
	pod, err := client.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("error getting pod: %v", err)
	}

	debugContainerName := fmt.Sprintf("%s-%s", debugContainerNamePrefix, rand.String(5))

	debugContainer := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:            debugContainerName,
			Image:           debugContainerImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			TTY:             true,
			Stdin:           true,
			VolumeMounts: []corev1.VolumeMount{
				{
					ReadOnly:  true,
					Name:      "spiffe-workload-api",
					MountPath: "/spiffe-workload-api",
				}},
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
		return nil, "", fmt.Errorf("error creating debug container: %v", err)
	}

	// Wait for the debug container to complete
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	for {
		pod, err := client.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return nil, "", fmt.Errorf("error getting pod status: %v", err)
		}

		for _, status := range pod.Status.EphemeralContainerStatuses {
			if status.Name == debugContainerName && status.State.Terminated != nil {
				return pod, debugContainerName, nil
			}
		}

		select {
		case <-waitCtx.Done():
			return nil, "", fmt.Errorf("timeout waiting for debug container to complete")
		default:
			continue
		}
	}
}

func renderRegisteredWorkloads(ctx context.Context, kubeConfig string, trustZones []*trust_zone_proto.TrustZone) error {
	data := make([][]string, 0, len(trustZones))

	for _, trustZone := range trustZones {
		if deployed, err := isTrustZoneDeployed(ctx, trustZone); err != nil {
			return err
		} else if !deployed {
			return fmt.Errorf("trust zone %s has not been deployed", trustZone.Name)
		}

		registeredWorkloads, err := workload.GetRegisteredWorkloads(ctx, kubeConfig, trustZone.GetKubernetesContext())
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

func getWorkloadStatus(ctx context.Context, client *kubeutil.Client, pod *corev1.Pod, container string) (string, error) {
	logs, err := client.Clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: container,
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

			err = renderUnregisteredWorkloads(cmd.Context(), kubeConfig, trustZones, opts.includeSecrets)
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

func renderUnregisteredWorkloads(ctx context.Context, kubeConfig string, trustZones []*trust_zone_proto.TrustZone, includeSecrets bool) error {
	data := make([][]string, 0, len(trustZones))

	for _, trustZone := range trustZones {
		deployed, err := isTrustZoneDeployed(ctx, trustZone)
		if err != nil {
			return err
		}

		registeredWorkloads, err := workload.GetUnregisteredWorkloads(ctx, kubeConfig, trustZone.GetKubernetesContext(), includeSecrets, deployed)
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

// isTrustZoneDeployed returns whether a trust zone has been deployed, i.e. whether a SPIRE Helm release has been installed.
func isTrustZoneDeployed(ctx context.Context, trustZone *trust_zone_proto.TrustZone) (bool, error) {
	prov, err := helm.NewHelmSPIREProvider(ctx, trustZone, nil, nil)
	if err != nil {
		return false, err
	}
	return prov.CheckIfAlreadyInstalled()
}
