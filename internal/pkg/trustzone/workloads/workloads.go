package workloads

import (
	"bytes"
	"context"
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

type UnregisteredWorkload struct {
	Name      string
	Namespace string
	Status    string
	Type      string
}

func GetRegisteredWorkloads(kubeCfgFile string, kubeContext string) ([]RegisteredWorkload, error) {
	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, kubeContext)
	if err != nil {
		return nil, err
	}

	registrationEntries, err := getRegistrationEntries(context.Background(), client)
	if err != nil {
		return nil, err
	}

	pods, err := client.Clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	registeredWorkloads := []RegisteredWorkload{}

	for _, pod := range pods.Items {
		spiffeID, ok := registrationEntries[string(pod.UID)]
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

		_, ok = registrationEntries[string(pod.UID)]
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

func getRegistrationEntries(ctx context.Context, client *kubeutil.Client) (map[string]string, error) {
	podExecOpts := &v1.PodExecOptions{
		Command:   []string{"/opt/spire/bin/spire-server", "entry", "show"},
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

	registrationEntries := make(map[string]string)
	stdoutLines := strings.Split(stdout.String(), "\n")

	for i := 0; i < len(stdoutLines); i++ {
		var podUID, spiffeID string

		if strings.HasPrefix(stdoutLines[i], "SPIFFE ID") {
			spiffeID = strings.TrimPrefix(stdoutLines[i], "SPIFFE ID        : ")

			for !strings.HasPrefix(stdoutLines[i], "Selector") && i < len(stdoutLines)-1 {
				i += 1
			}

			if strings.HasPrefix(stdoutLines[i], "Selector") {
				podUID = strings.Split(stdoutLines[i], ":")[3]
				registrationEntries[podUID] = spiffeID
			}
		}
	}

	return registrationEntries, nil
}
