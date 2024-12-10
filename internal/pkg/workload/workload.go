// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package workload

import (
	"context"
	"fmt"
	"time"

	"github.com/cofide/cofidectl/pkg/spire"
	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
