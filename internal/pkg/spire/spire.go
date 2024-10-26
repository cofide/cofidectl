package spire

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	types "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	namespace             = "spire"
	serverStatefulsetName = "spire-server"
	serverPodName         = "spire-server-0"
	serverContainerName   = "spire-server"
	serverExecutable      = "/opt/spire/bin/spire-server"
	scmContainerName      = "spire-controller-manager"
	agentDaemonSetName    = "spire-agent"
)

// ServerStatus contains status information about a running SPIRE server cluster.
type ServerStatus struct {
	Replicas      int
	ReadyReplicas int
	Containers    []ServerContainer
	SCMs          []SCMContainer
}

// ServerContainer contains status information about a running SPIRE server container.
type ServerContainer struct {
	Name  string
	Ready bool
}

// SCMContainer contains status information about a running SPIRE controller manager container.
type SCMContainer struct {
	Name  string
	Ready bool
}

// GetServerStatus queries the status of a SPIRE server and returns a `*ServerStatus`.
func GetServerStatus(ctx context.Context, client *kubeutil.Client) (*ServerStatus, error) {
	statefulset, err := getServerStatefulSet(ctx, client)
	if err != nil {
		return nil, err
	}

	pods, err := getPodsForStatefulSet(ctx, client, statefulset)
	if err != nil {
		return nil, err
	}

	containers, err := getServerContainers(pods)
	if err != nil {
		return nil, err
	}

	scms, err := getSCMContainers(pods)
	if err != nil {
		return nil, err
	}

	status := &ServerStatus{
		Replicas:      int(*statefulset.Spec.Replicas),
		ReadyReplicas: int(statefulset.Status.ReadyReplicas),
		Containers:    containers,
		SCMs:          scms,
	}
	return status, nil
}

func getServerStatefulSet(ctx context.Context, client *kubeutil.Client) (*appsv1.StatefulSet, error) {
	return client.Clientset.AppsV1().
		StatefulSets(namespace).
		Get(ctx, serverStatefulsetName, metav1.GetOptions{})
}

func getPodsForStatefulSet(ctx context.Context, client *kubeutil.Client, statefulset *appsv1.StatefulSet) (*v1.PodList, error) {
	set := labels.Set(statefulset.Spec.Selector.MatchLabels)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	return client.Clientset.CoreV1().
		Pods(namespace).
		List(ctx, listOptions)
}

// getServerContainers queries the pods in a SPIRE server statefulset and returns a slice of `ServerContainers`.
func getServerContainers(pods *v1.PodList) ([]ServerContainer, error) {
	serverContainers := []ServerContainer{}
	for _, pod := range pods.Items {
		for _, container := range pod.Status.ContainerStatuses {
			if container.Name == serverContainerName {
				serverContainers = append(serverContainers, ServerContainer{
					Name:  pod.Name,
					Ready: container.Ready})
				break
			}
		}
	}
	return serverContainers, nil
}

// getSCMContainers queries the pods in a SPIRE server statefulset and returns a slice of `SCMContainers`.
func getSCMContainers(pods *v1.PodList) ([]SCMContainer, error) {
	controllerManagers := []SCMContainer{}
	for _, pod := range pods.Items {
		for _, container := range pod.Status.ContainerStatuses {
			if container.Name == scmContainerName {
				controllerManagers = append(controllerManagers, SCMContainer{
					Name:  pod.Name,
					Ready: container.Ready})
				break
			}
		}
	}
	return controllerManagers, nil
}

// AgentStatus contains status information about a running cluster of SPIRE agents.
type AgentStatus struct {
	Expected int
	Ready    int
	Agents   []Agent
}

// Agent contains status information about a running SPIRE agent.
type Agent struct {
	Name            string
	Status          string
	Id              string
	AttestationType string
	ExpirationTime  time.Time
	Serial          string
	CanReattest     bool
}

// GetAgentStatus queries a SPIRE server for the status of agents attested to it and returns an `*AgentStatus`.
func GetAgentStatus(ctx context.Context, client *kubeutil.Client) (*AgentStatus, error) {
	command := []string{"agent", "list", "-output", "json"}
	stdout, _, err := execInServerContainer(ctx, client, command)
	if err != nil {
		return nil, err
	}

	agents, err := parseAgentList(stdout)
	if err != nil {
		return nil, err
	}

	return addAgentK8sStatus(ctx, client, agents)
}

// addAgentK8sStatus queries the SPIRE agent daemonset and pods, then updates the provided `agents` slice with pod information.
// It returns an `*AgentStatus` including information from the daemonset and the updated agents list.
func addAgentK8sStatus(ctx context.Context, client *kubeutil.Client, agents []Agent) (*AgentStatus, error) {
	daemonset, err := getAgentDaemonSet(ctx, client)
	if err != nil {
		return nil, err
	}

	pods, err := getPodsforDaemonSet(ctx, client, daemonset)
	if err != nil {
		return nil, err
	}

	podMap := make(map[string]v1.Pod)
	for _, pod := range pods.Items {
		podMap[pod.Name] = pod
	}

	// Update info from SPIRE server with pod status
	for i, agent := range agents {
		pod, ok := podMap[agent.Name]
		if ok {
			agents[i].Status = string(pod.Status.Phase)
			delete(podMap, pod.Name)
		}
	}

	// Add any pods with no matching SPIRE agent
	for name, pod := range podMap {
		agent := Agent{
			Name:            name,
			Status:          string(pod.Status.Phase),
			Id:              "unknown",
			AttestationType: "unknown",
			ExpirationTime:  time.Unix(0, 0),
			Serial:          "unknown",
			CanReattest:     false,
		}
		agents = append(agents, agent)
	}

	status := &AgentStatus{
		Expected: int(daemonset.Status.DesiredNumberScheduled),
		Ready:    int(daemonset.Status.NumberReady),
		Agents:   agents,
	}

	return status, nil
}

func getAgentDaemonSet(ctx context.Context, client *kubeutil.Client) (*appsv1.DaemonSet, error) {
	return client.Clientset.AppsV1().
		DaemonSets(namespace).
		Get(ctx, agentDaemonSetName, metav1.GetOptions{})
}

func getPodsforDaemonSet(ctx context.Context, client *kubeutil.Client, daemonset *appsv1.DaemonSet) (*v1.PodList, error) {
	set := labels.Set(daemonset.Spec.Selector.MatchLabels)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	return client.Clientset.CoreV1().
		Pods(namespace).
		List(ctx, listOptions)
}

// Workload contains details of a workload registered with SPIRE
type RegisteredEntry struct {
	Id *types.SPIFFEID
}

func GetRegistrationEntries(ctx context.Context, client *kubeutil.Client) (map[string]*RegisteredEntry, error) {
	command := []string{"entry", "show", "-output", "json"}
	stdout, _, err := execInServerContainer(ctx, client, command)
	if err != nil {
		return nil, err
	}

	registrationEntries, err := parseEntryList(stdout)
	if err != nil {
		return nil, err
	}

	registrationEntriesMap := make(map[string]*RegisteredEntry)

	for _, registrationEntry := range registrationEntries.Entries {
		var podUID string

		selectors := registrationEntry.Selectors
		if len(selectors) == 0 {
			continue
		}

		for _, selector := range selectors {
			if selector.Type == k8sSelectorType {
				if !strings.HasPrefix(selector.Value, k8sPodUIDSelectorPrefix) {
					slog.Warn(fmt.Sprintf("failed to find the k8s:pod-uid selector value for workload with workload id: %s", registrationEntry.Id))
					continue
				}
				podUID = strings.TrimPrefix(selector.Value, k8sPodUIDSelectorPrefix)
			}
		}

		if podUID == "" {
			continue
		}

		registrationEntriesMap[podUID] = &RegisteredEntry{registrationEntry.Id}
	}

	return registrationEntriesMap, nil
}
