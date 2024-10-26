package workload

import (
	"context"

	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	"github.com/cofide/cofidectl/internal/pkg/spire"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Workload struct {
	Name      string
	Namespace string
	SPIFFEID  string
	Status    string
	Type      string
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

			unregisteredWorkloads = append(unregisteredWorkloads, *unregisteredWorkload)
		}
	}

	return unregisteredWorkloads, nil
}
