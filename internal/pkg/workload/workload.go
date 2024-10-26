package workload

import (
	"context"
	"fmt"
	"time"

	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	"github.com/cofide/cofidectl/internal/pkg/spire"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Workload struct {
	Name      string
	Namespace string
	SPIFFEID  string
	Status    string
	Type      string
	Secrets   []*WorkloadSecretMetadata
}

type WorkloadSecretMetadata struct {
	Name string
	Type string
	Age  time.Duration
}

// GetRegisteredWorkloads will find all workloads that are registered with the WI platform
func GetRegisteredWorkloads(kubeConfig string, kubeContext string) ([]Workload, error) {
	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeConfig, kubeContext)
	if err != nil {
		return nil, err
	}

	registeredEntries, err := spire.GetRegistrationEntries(context.Background(), client)
	if err != nil {
		return nil, err
	}

	pods, err := client.Clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
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
				SPIFFEID:  registeredEntry.Id.String(),
				Status:    string(pod.Status.Phase),
				Type:      "Pod",
			}

			registeredWorkloads = append(registeredWorkloads, *registeredWorkload)
		}
	}

	return registeredWorkloads, nil
}

// GetUnregisteredWorkloads will discover workloads in a Kubernetes cluster that are not (yet) registered
func GetUnregisteredWorkloads(kubeCfgFile string, kubeContext string, secretDiscovery bool) ([]Workload, error) {
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

	registeredEntries, err := spire.GetRegistrationEntries(context.Background(), client)
	if err != nil {
		return nil, err
	}

	pods, err := client.Clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var secrets *v1.SecretList
	if secretDiscovery {
		secrets, err = client.Clientset.CoreV1().Secrets("").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
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

			// add related secrets (metadata) if secret discovery is enabled
			if secretDiscovery {
				findRelatedSecrets(&pod, unregisteredWorkload, secrets)

			}

			unregisteredWorkloads = append(unregisteredWorkloads, *unregisteredWorkload)
		}
	}

	return unregisteredWorkloads, nil
}

func findRelatedSecrets(pod *v1.Pod, workload *Workload, secrets *v1.SecretList) {
	for _, secret := range secrets.Items {
		age := time.Since(secret.CreationTimestamp.Time)
		// Consider secrets older than 30 days as long-lived
		if age > 30*24*time.Hour {
			key := fmt.Sprintf("%s/%s", secret.Namespace, secret.Name)
			workload.Secrets = append(workload.Secrets, &WorkloadSecretMetadata{
				Name: key,
				Type: "secret",
				Age:  age,
			})
		}
	}

	for _, volume := range pod.Spec.Volumes {
		if volume.Secret != nil {
			key := fmt.Sprintf("%s/%s", pod.Namespace, volume.Secret.SecretName)
			workload.Secrets = append(workload.Secrets, &WorkloadSecretMetadata{
				Name: key,
				Type: "volume",
			})
		}
	}

	// Check secrets used in environment variables
	for _, container := range pod.Spec.Containers {
		for _, env := range container.EnvFrom {
			if env.SecretRef != nil {
				key := fmt.Sprintf("%s/%s", pod.Namespace, env.SecretRef.Name)
				workload.Secrets = append(workload.Secrets, &WorkloadSecretMetadata{
					Name: key,
					Type: "env",
				})
			}
		}
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				key := fmt.Sprintf("%s/%s", pod.Namespace, env.ValueFrom.SecretKeyRef.Name)
				workload.Secrets = append(workload.Secrets, &WorkloadSecretMetadata{
					Name: key,
					Type: "env",
				})
			}
		}
	}
}
