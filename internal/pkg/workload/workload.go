// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package workload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/provision_plugin/v1alpha2"
	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	"github.com/cofide/cofidectl/pkg/plugin/provision"
	"github.com/cofide/cofidectl/pkg/spire"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

const debugContainerNamePrefix = "cofidectl-debug"
const debugContainerImage = "ghcr.io/cofide/cofidectl-debug-container:v0.2.1"

type Workload struct {
	Name             string
	Namespace        string
	SPIFFEID         string
	Status           string
	Type             string
	NumSecrets       int
	NumSecretsAtRisk int
}

type WorkloadSecretMetadata struct {
	AtRisk bool
	Age    time.Duration
}

// GetRegisteredWorkloads will find all workloads that are registered
func GetRegisteredWorkloads(ctx context.Context, kubeConfig string, kubeContext string) ([]Workload, error) {
	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeConfig, kubeContext)
	if err != nil {
		return nil, err
	}

	registeredEntries, err := spire.GetRegistrationEntries(ctx, client)
	if err != nil {
		return nil, err
	}

	pods, err := client.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	registeredWorkloads := []Workload{}

	for _, pod := range pods.Items {
		registeredEntry, ok := registeredEntries[string(pod.UID)]
		if ok {
			registeredWorkload := &Workload{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				SPIFFEID:  registeredEntry.Id,
				Status:    string(pod.Status.Phase),
				Type:      "Pod",
			}

			registeredWorkloads = append(registeredWorkloads, *registeredWorkload)
		}
	}

	return registeredWorkloads, nil
}

// GetUnregisteredWorkloads will discover workloads in a Kubernetes cluster that are not (yet) registered
func GetUnregisteredWorkloads(ctx context.Context, kubeCfgFile string, kubeContext string, secretDiscovery bool, checkSpire bool) ([]Workload, error) {
	// Includes the initial Kubernetes namespaces.
	ignoredNamespaces := map[string]bool{
		"kube-node-lease":    true,
		"kube-public":        true,
		"kube-system":        true,
		"local-path-storage": true,
		"spire":              true,
		"spire-server":       true,
		"spire-system":       true,
		"spire-mgmt":         true,
	}

	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, kubeContext)
	if err != nil {
		return nil, err
	}

	var registeredEntries map[string]*spire.RegisteredEntry
	if checkSpire {
		registeredEntries, err = spire.GetRegistrationEntries(ctx, client)
		if err != nil {
			return nil, err
		}
	}

	pods, err := client.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var secretsMap map[string]WorkloadSecretMetadata
	var secrets *v1.SecretList
	if secretDiscovery {
		secrets, err = client.Clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		secretsMap = analyseSecrets(secrets)
	}

	unregisteredWorkloads := []Workload{}

	for _, pod := range pods.Items {
		_, ok := ignoredNamespaces[pod.Namespace]
		if ok {
			continue
		}

		_, ok = registeredEntries[string(pod.UID)]
		if !ok {
			unregisteredWorkload := &Workload{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Status:    string(pod.Status.Phase),
				Type:      "Pod",
			}

			// Add related secrets (metadata) if secret discovery is enabled
			if secretDiscovery {
				associateSecrets(&pod, unregisteredWorkload, secretsMap)
			}

			unregisteredWorkloads = append(unregisteredWorkloads, *unregisteredWorkload)
		}
	}

	return unregisteredWorkloads, nil
}

func GetStatus(ctx context.Context, statusCh chan<- *provisionpb.Status, dataCh chan string, client *kubeutil.Client, podName, namespace, wlVolumeMount string) {
	debugContainerName := fmt.Sprintf("%s-%s", debugContainerNamePrefix, rand.String(5))

	statusCh <- provision.StatusOk(
		"Creating",
		fmt.Sprintf("Waiting for ephemeral debug container to be created in %s", podName),
	)

	if err := createDebugContainer(ctx, client, podName, namespace, wlVolumeMount, debugContainerName); err != nil {
		statusCh <- provision.StatusError(
			"Creating",
			fmt.Sprintf("Failed waiting for ephemeral debug container to be created in %s", podName),
			err,
		)
		return
	}

	statusCh <- provision.StatusOk(
		"Waiting",
		"Waiting for ephemeral debug container to complete",
	)

	if err := waitForDebugContainer(ctx, client, podName, namespace, debugContainerName); err != nil {
		statusCh <- provision.StatusError(
			"Waiting",
			"Error waiting for ephemeral debug container to complete",
			err,
		)
		return
	}

	logs, err := getDebugContainerLogs(ctx, client, podName, namespace, debugContainerName)
	if err != nil {
		statusCh <- provision.StatusError(
			"Waiting",
			"Error waiting for ephemeral debug container logs",
			err,
		)
		return
	}

	dataCh <- logs
	statusCh <- provision.StatusDone(
		"Complete",
		fmt.Sprintf("Successfully executed emphemeral debug container in %s", podName),
	)
}

func createDebugContainer(ctx context.Context, client *kubeutil.Client, podName string, namespace string, wlVolumeMount string, debugContainerName string) error {
	pod, err := client.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	debugContainer := v1.EphemeralContainer{
		EphemeralContainerCommon: v1.EphemeralContainerCommon{
			Name:            debugContainerName,
			Image:           debugContainerImage,
			ImagePullPolicy: v1.PullIfNotPresent,
			TTY:             true,
			Stdin:           true,
			VolumeMounts: []v1.VolumeMount{
				{
					ReadOnly:  true,
					Name:      wlVolumeMount,
					MountPath: fmt.Sprintf("/%s", wlVolumeMount),
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
		return err
	}
	return nil
}

func waitForDebugContainer(ctx context.Context, client *kubeutil.Client, podName string, namespace string, debugContainerName string) error {
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	for {
		pod, err := client.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		for _, status := range pod.Status.EphemeralContainerStatuses {
			if status.Name == debugContainerName && status.State.Terminated != nil {
				return nil
			}
		}

		select {
		case <-waitCtx.Done():
			return err
		default:
			time.Sleep(time.Second)
			continue
		}
	}
}

func getDebugContainerLogs(ctx context.Context, client *kubeutil.Client, podName string, namespace string, debugContainerName string) (string, error) {
	logs, err := client.Clientset.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{
		Container: debugContainerName,
	}).Stream(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = logs.Close()
	}()

	// Read the logs
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, logs)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func isAtRisk(creationTS time.Time) (time.Duration, bool) {
	// Consider secrets older than 30 days as long-lived and a source for potential risk
	age := time.Since(creationTS)
	if age > 30*24*time.Hour {
		return age, true
	}
	return age, false
}

func analyseSecrets(secrets *v1.SecretList) map[string]WorkloadSecretMetadata {
	// Analyse the metadata for all secrets and determine the age
	workloadSecrets := make(map[string]WorkloadSecretMetadata)
	for _, secret := range secrets.Items {
		key := fmt.Sprintf("%s/%s", secret.Namespace, secret.Name)
		age, atRisk := isAtRisk(secret.CreationTimestamp.Time)
		workloadSecrets[key] = WorkloadSecretMetadata{
			AtRisk: atRisk,
			Age:    age,
		}
	}
	return workloadSecrets
}

func associateSecrets(pod *v1.Pod, workload *Workload, secrets map[string]WorkloadSecretMetadata) {
	// Check secrets mounted in pod volumes
	for _, volume := range pod.Spec.Volumes {
		if volume.Secret != nil {
			key := fmt.Sprintf("%s/%s", pod.Namespace, volume.Secret.SecretName)
			if secret, exists := secrets[key]; exists {
				workload.NumSecrets++
				if secret.AtRisk {
					workload.NumSecretsAtRisk++
				}
			}
		}
	}

	// Check secrets used in environment variables
	for _, container := range pod.Spec.Containers {
		for _, env := range container.EnvFrom {
			if env.SecretRef != nil {
				key := fmt.Sprintf("%s/%s", pod.Namespace, env.SecretRef.Name)
				if secret, exists := secrets[key]; exists {
					workload.NumSecrets++
					if secret.AtRisk {
						workload.NumSecretsAtRisk++
					}
				}
			}
		}
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				key := fmt.Sprintf("%s/%s", pod.Namespace, env.ValueFrom.SecretKeyRef.Name)
				if secret, exists := secrets[key]; exists {
					workload.NumSecrets++
					if secret.AtRisk {
						workload.NumSecretsAtRisk++
					}
				}
			}
		}
	}
}
