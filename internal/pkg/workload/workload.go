package workload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

type RegisteredWorkload struct {
	Name      string
	Namespace string
	SPIFFEID  string
	Status    string
	Type      string
}

type ParentID struct {
	Path        string `json:"path"`
	TrustDomain string `json:"trust_domain"`
}

type Selector struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type SPIFFEID struct {
	Path        string `json:"path"`
	TrustDomain string `json:"trust_domain"`
}

type RegistrationEntry struct {
	Selectors []Selector `json:"selectors"`
	SPIFFEID  SPIFFEID   `json:"spiffe_id"`
}

type RegistrationEntries struct {
	Entries []RegistrationEntry
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

	registrationEntriesMap := make(map[string]string)

	for _, registrationEntry := range registrationEntries {
		var podUID string

		selectors := registrationEntry.Selectors
		if len(selectors) == 0 {
			continue
		}

		for _, selector := range selectors {
			if selector.Type == "k8s" {
				podUID = strings.TrimPrefix(selector.Value, "pod-uid:")
			}
		}

		if podUID == "" {
			continue
		}

		spiffeID := fmt.Sprintf("spiffe://%s%s", registrationEntry.SPIFFEID.TrustDomain, registrationEntry.SPIFFEID.Path)
		registrationEntriesMap[podUID] = spiffeID
	}

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

func getRegistrationEntries(ctx context.Context, client *kubeutil.Client) ([]RegistrationEntry, error) {
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

	parsedRegistrationEntries := &RegistrationEntries{}
	err = json.Unmarshal(stdout.Bytes(), parsedRegistrationEntries)
	if err != nil {
		return nil, err
	}

	registrationEntries := parsedRegistrationEntries.Entries

	return registrationEntries, nil
}
