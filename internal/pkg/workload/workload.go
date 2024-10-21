package workload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

const (
	k8sSelectorType         = "k8s"
	k8sPodUIDSelectorPrefix = "pod-uid:"
)

type RegisteredWorkload struct {
	Name      string
	Namespace string
	SPIFFEID  string
	Status    string
	Type      string
}

type selector struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type spiffeID struct {
	Path        string `json:"path"`
	TrustDomain string `json:"trust_domain"`
}

type registrationEntry struct {
	Selectors []selector `json:"selectors"`
	SPIFFEID  spiffeID   `json:"spiffe_id"`
}

type registrationEntries struct {
	Entries []registrationEntry
}

type UnregisteredWorkload struct {
	Name      string
	Namespace string
	Status    string
	Type      string
}

func GetRegisteredWorkloads(kubeConfig string, kubeContext string) ([]RegisteredWorkload, error) {
	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeConfig, kubeContext)
	if err != nil {
		return nil, err
	}

	registrationEntries, err := getRegistrationEntries(context.Background(), client)
	if err != nil {
		return nil, err
	}

	registrationEntriesMap := generateRegistrationEntriesMap(registrationEntries)

	pods, err := client.Clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	registeredWorkloads := []RegisteredWorkload{}

	for _, pod := range pods.Items {
		spiffeID, ok := registrationEntriesMap[string(pod.UID)]
		if ok {
			registeredWorkload := &RegisteredWorkload{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				SPIFFEID:  spiffeID,
				Status:    string(pod.Status.Phase),
				Type:      "Pod",
			}

			registeredWorkloads = append(registeredWorkloads, *registeredWorkload)
		}
	}

	return registeredWorkloads, nil
}

func GetUnregisteredWorkloads(kubeCfgFile string, kubeContext string) ([]UnregisteredWorkload, error) {
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

	registrationEntries, err := getRegistrationEntries(context.Background(), client)
	if err != nil {
		return nil, err
	}

	registrationEntriesMap := generateRegistrationEntriesMap(registrationEntries)

	pods, err := client.Clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	unregisteredWorkloads := []UnregisteredWorkload{}

	for _, pod := range pods.Items {
		_, ok := ignoredNamespaces[pod.Namespace]
		if ok {
			continue
		}

		_, ok = registrationEntriesMap[string(pod.UID)]
		if !ok {
			unregisteredWorkload := &UnregisteredWorkload{
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

func generateRegistrationEntriesMap(registrationEntries []registrationEntry) map[string]string {
	registrationEntriesMap := make(map[string]string)

	for _, registrationEntry := range registrationEntries {
		var podUID string

		spiffeID := fmt.Sprintf("spiffe://%s%s", registrationEntry.SPIFFEID.TrustDomain, registrationEntry.SPIFFEID.Path)

		selectors := registrationEntry.Selectors
		if len(selectors) == 0 {
			continue
		}

		for _, selector := range selectors {
			if selector.Type == k8sSelectorType {
				if !strings.HasPrefix(selector.Value, k8sPodUIDSelectorPrefix) {
					slog.Warn(fmt.Sprintf("failed to find the k8s:pod-uid selector value for workload with workload id: %s", spiffeID))
					continue
				}
				podUID = strings.TrimPrefix(selector.Value, k8sPodUIDSelectorPrefix)
			}
		}

		if podUID == "" {
			continue
		}

		registrationEntriesMap[podUID] = spiffeID
	}

	return registrationEntriesMap
}

func getRegistrationEntries(ctx context.Context, client *kubeutil.Client) ([]registrationEntry, error) {
	podExecOpts := &v1.PodExecOptions{
		Command:   []string{"/opt/spire/bin/spire-server", "entry", "show", "-output", "json"},
		Container: "spire-server",
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
	}

	request := client.Clientset.CoreV1().
		RESTClient().
		Post().
		Namespace("spire").
		Resource("pods").
		Name("spire-server-0").
		SubResource("exec").
		VersionedParams(podExecOpts, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(client.RestConfig, "POST", request.URL())
	if err != nil {
		return nil, err
	}

	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return nil, err
	}

	parsedRegistrationEntries := &registrationEntries{}
	err = json.Unmarshal(stdout.Bytes(), parsedRegistrationEntries)
	if err != nil {
		return nil, err
	}

	registrationEntries := parsedRegistrationEntries.Entries

	return registrationEntries, nil
}
