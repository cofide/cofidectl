// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package workload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	"github.com/cofide/cofidectl/internal/pkg/provider"
	"github.com/cofide/cofidectl/internal/pkg/spire"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

const debugContainerNamePrefix = "cofidectl-debug"
const debugContainerImage = "cofidectl-debug:latest"

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
func GetUnregisteredWorkloads(ctx context.Context, kubeCfgFile string, kubeContext string, secretDiscovery bool) ([]Workload, error) {
	// Includes the initial Kubernetes namespaces.
	ignoredNamespaces := map[string]int{
		"kube-node-lease":    1,
		"kube-public":        2,
		"kube-system":        3,
		"local-path-storage": 4,
		"spire":              5,
	}

	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, kubeContext)
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

func GetStatus(ctx context.Context, statusCh chan<- provider.ProviderStatus, dataCh chan string, client *kubeutil.Client, podName string, namespace string) {
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	statusCh <- provider.ProviderStatus{
		Stage:   "Creating",
		Message: fmt.Sprintf("Waiting for ephemeral debug container to be created in %s", podName),
	}

	pod, err := client.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		statusCh <- provider.ProviderStatus{
			Stage:   "Creating",
			Message: fmt.Sprintf("Failed waiting for ephemeral debug container to be created in %s", podName),
			Done:    true,
			Error:   err,
		}
		return
	}

	debugContainerName := fmt.Sprintf("%s-%s", debugContainerNamePrefix, rand.String(5))

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
		statusCh <- provider.ProviderStatus{
			Stage:   "Creating",
			Message: fmt.Sprintf("Failed waiting for ephemeral debug container to be created in %s", podName),
			Done:    true,
			Error:   err,
		}
		return
	}

	statusCh <- provider.ProviderStatus{
		Stage:   "Waiting",
		Message: "Waiting for ephemeral debug container to complete",
	}

	for {
		pod, err := client.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			statusCh <- provider.ProviderStatus{
				Stage:   "Waiting",
				Message: "Error waiting for ephemeral debug container to complete",
			}
			return
		}

		containerTerminated := false
		for _, status := range pod.Status.EphemeralContainerStatuses {
			if status.Name == debugContainerName && status.State.Terminated != nil {
				containerTerminated = true
				break
			}
		}

		if containerTerminated {
			break
		}

		select {
		case <-waitCtx.Done():
			statusCh <- provider.ProviderStatus{
				Stage:   "Waiting",
				Message: "Error waiting for ephemeral debug container to complete",
			}
			return
		default:
			time.Sleep(time.Second)
			continue
		}
	}

	logs, err := client.Clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{
		Container: debugContainerName,
	}).Stream(ctx)
	if err != nil {
		statusCh <- provider.ProviderStatus{
			Stage:   "Waiting",
			Message: "Error waiting for ephemeral debug container logs",
		}
		return
	}
	defer logs.Close()

	// Read the logs
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, logs)
	if err != nil {
		statusCh <- provider.ProviderStatus{
			Stage:   "Waiting",
			Message: "Error waiting for ephemeral debug container logs",
		}
	}

	dataCh <- buf.String()

	statusCh <- provider.ProviderStatus{
		Stage:   "Complete",
		Message: fmt.Sprintf("Successfully executed ephemeral debug container in %s", podName),
		Done:    true,
	}
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
