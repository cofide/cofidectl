// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spire

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/cofide/cofidectl/internal/pkg/kube"
	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	types "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"
)

const (
	namespace             = "spire"
	serverStatefulsetName = "spire-server"
	serverPodName         = "spire-server-0"
	serverContainerName   = "spire-server"
	serverServiceName     = "spire-server"
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

	containers := getServerContainers(pods)
	scms := getSCMContainers(pods)

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
func getServerContainers(pods *v1.PodList) []ServerContainer {
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
	return serverContainers
}

// getSCMContainers queries the pods in a SPIRE server statefulset and returns a slice of `SCMContainers`.
func getSCMContainers(pods *v1.PodList) []SCMContainer {
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
	return controllerManagers
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

// RegisteredEntry contains details of a workload registered with SPIRE
type RegisteredEntry struct {
	Id string
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

		id, err := formatIdUrl(registrationEntry.Id)
		if err != nil {
			return nil, err
		}
		registrationEntriesMap[podUID] = &RegisteredEntry{Id: id}
	}

	return registrationEntriesMap, nil
}

// WaitForServerIP waits for a SPIRE server pod and service to become ready, then returns the external IP of the service.
func WaitForServerIP(ctx context.Context, client *kubeutil.Client) (string, error) {
	podWatcher, err := createPodWatcher(ctx, client)
	if err != nil {
		return "", err
	}
	defer podWatcher.Stop()

	serviceWatcher, err := createServiceWatcher(ctx, client)
	if err != nil {
		return "", err
	}
	defer serviceWatcher.Stop()

	podReady := false
	var serviceIP string

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case event, ok := <-podWatcher.ResultChan():
			if !ok {
				return "", fmt.Errorf("pod watcher channel closed")
			}
			if event.Type == watch.Added || event.Type == watch.Modified {
				pod := event.Object.(*v1.Pod)
				// FieldSelector should ensure this, but use belt & braces.
				if pod.Name != serverPodName {
					slog.Warn("Event received for unexpected pod", slog.String("pod", pod.Name))
				} else if isPodReady(pod) {
					podReady = true
				}
			}
		case event, ok := <-serviceWatcher.ResultChan():
			if !ok {
				return "", fmt.Errorf("service watcher channel closed")
			}
			if event.Type == watch.Added || event.Type == watch.Modified {
				service := event.Object.(*v1.Service)
				// FieldSelector should ensure this, but use belt & braces.
				if service.Name != serverServiceName {
					slog.Warn("Event received for unexpected service", slog.String("service", service.Name))
				} else if ip, err := getServiceExternalIP(service); err == nil {
					serviceIP = ip
				}
			}
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for pod and service to be ready")
		}

		if podReady && serviceIP != "" {
			return serviceIP, nil
		}
	}
}

// GetBundle retrieves a SPIFFE bundle for the local trust zone by exec'ing into a SPIRE Server.
func GetBundle(ctx context.Context, client *kubeutil.Client) (string, error) {
	command := []string{"bundle", "show", "-format", "spiffe"}
	stdout, _, err := execInServerContainer(ctx, client, command)
	if err != nil {
		return "", err
	}
	return string(stdout), nil
}

func createPodWatcher(ctx context.Context, client *kubeutil.Client) (watch.Interface, error) {
	watchFunc := func(opts metav1.ListOptions) (watch.Interface, error) {
		timeout := int64(120)
		return client.Clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
			FieldSelector:  fmt.Sprintf("metadata.name=%s", serverPodName),
			TimeoutSeconds: &timeout,
		})
	}

	watcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher for context %s: %v", client.CmdConfig.CurrentContext, err)
	}

	return watcher, nil
}

func createServiceWatcher(ctx context.Context, client *kubeutil.Client) (watch.Interface, error) {
	watchFunc := func(opts metav1.ListOptions) (watch.Interface, error) {
		timeout := int64(120)
		return client.Clientset.CoreV1().Services(namespace).Watch(ctx, metav1.ListOptions{
			FieldSelector:  fmt.Sprintf("metadata.name=%s", serverServiceName),
			TimeoutSeconds: &timeout,
		})
	}

	watcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})
	if err != nil {
		return nil, fmt.Errorf("failed to create service watcher for context %s: %v", client.CmdConfig.CurrentContext, err)
	}

	return watcher, nil
}

func isPodReady(pod *v1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == v1.PodReady && cond.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func getServiceExternalIP(service *v1.Service) (string, error) {
	serviceLoadBalancerIngress := service.Status.LoadBalancer.Ingress
	if len(serviceLoadBalancerIngress) != 1 {
		return "", fmt.Errorf("failed to retrieve the service ingress information")
	}

	// Usually set on AWS load balancers
	ingressHostName := serviceLoadBalancerIngress[0].Hostname
	if ingressHostName != "" {
		return ingressHostName, nil
	}

	// Usually set on GCE/OpenStack load balancers
	ingressIP := serviceLoadBalancerIngress[0].IP
	if ingressIP != "" {
		return ingressIP, nil
	}

	return "", fmt.Errorf("failed to retrieve the service ingress information")
}

// formatIdUrl formats a SPIFFE ID as a URL string.
func formatIdUrl(id *types.SPIFFEID) (string, error) {
	trustDomain, err := spiffeid.TrustDomainFromString(id.TrustDomain)
	if err != nil {
		return "", err
	}
	if id, err := spiffeid.FromPath(trustDomain, id.Path); err != nil {
		return "", err
	} else {
		return id.String(), nil
	}
}

// GetServerCABundleAndFederatedBundles retrieves the server CA bundle (i.e. bundle of the host) and any available
// federated bundles from the SPIRE server, in order to do a federation health check
func GetServerCABundleAndFederatedBundles(ctx context.Context, client *kube.Client) (string, map[string]string, error) {
	serverCABundle, err := getServerCABundle(ctx, client)
	if err != nil {
		return "", nil, err
	}
	federatedBundles, err := getFederatedBundles(ctx, client)
	if err != nil {
		return "", nil, err
	}
	return serverCABundle, federatedBundles, err
}

// getServerCABundle retrives the x509_authorities component of the server CA trust bundle
func getServerCABundle(ctx context.Context, client *kube.Client) (string, error) {
	command := []string{"bundle", "show", "-output", "json"}
	stdout, _, err := execInServerContainer(ctx, client, command)
	if err != nil {
		return "", err
	}
	return parseServerCABundle(stdout)
}

func parseServerCABundle(stdout []byte) (string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(stdout, &data); err != nil {
		return "", err
	}
	return fmt.Sprint(data["x509_authorities"]), nil
}

type federatedBundles struct {
	Bundles []map[string]interface{} `json:"bundles"`
}

func getFederatedBundles(ctx context.Context, client *kube.Client) (map[string]string, error) {
	command := []string{"bundle", "list", "-output", "json"}
	stdout, _, err := execInServerContainer(ctx, client, command)
	if err != nil {
		return nil, err
	}
	return parseFederatedBundles(stdout)
}

func parseFederatedBundles(stdout []byte) (map[string]string, error) {
	result := make(map[string]string)
	var data federatedBundles
	if err := json.Unmarshal(stdout, &data); err != nil {
		return nil, err
	}
	for _, bundle := range data.Bundles {
		// Store x509_authorities for comparison, keyed by trust domain
		result[bundle["trust_domain"].(string)] = fmt.Sprint(bundle["x509_authorities"])
	}
	return result, nil
}
