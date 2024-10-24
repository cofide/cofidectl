package spire

import (
	"context"
	"reflect"
	"testing"
	"time"

	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	v1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetServerStatus(t *testing.T) {
	ctx := context.Background()
	clientSet := fake.NewClientset()
	client := &kubeutil.Client{Clientset: clientSet}

	// Create spire-server statefulset
	ssConfig := (&applyv1.StatefulSetApplyConfiguration{}).
		WithName("spire-server").
		WithKind("StatefulSet").
		WithAPIVersion("apps/v1").
		WithSpec((&applyv1.StatefulSetSpecApplyConfiguration{}).
			WithReplicas(1).
			WithSelector((&applymetav1.LabelSelectorApplyConfiguration{}).
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "server"}),
			),
		)
	_, err := clientSet.AppsV1().
		StatefulSets("spire").
		Apply(ctx, ssConfig, metav1.ApplyOptions{})
	if err != nil {
		t.Fatalf("failed to create statefulset: %v", err)
	}

	// Create spire-server-0 pod.
	podConfig := (&v1.PodApplyConfiguration{}).
		WithName("spire-server-0").
		WithKind("Pod").
		WithAPIVersion("v1").
		WithLabels(map[string]string{"app.kubernetes.io/name": "server"}).
		WithStatus((&v1.PodStatusApplyConfiguration{}).
			WithContainerStatuses(
				(&v1.ContainerStatusApplyConfiguration{}).
					WithName("spire-server").
					WithReady(true),
				(&v1.ContainerStatusApplyConfiguration{}).
					WithName("spire-controller-manager").
					WithReady(false),
			),
		)
	_, err = clientSet.CoreV1().
		Pods("spire").
		Apply(ctx, podConfig, metav1.ApplyOptions{})
	if err != nil {
		t.Fatalf("failed to create pod: %v", err)
	}

	got, err := GetServerStatus(ctx, client)
	if err != nil {
		t.Fatalf("unexpected error %c", err)
	}
	want := &ServerStatus{
		Replicas:      1,
		ReadyReplicas: 0,
		Containers: []ServerContainer{
			{Name: "spire-server-0", Ready: true},
		},
		SCMs: []SCMContainer{
			{Name: "spire-server-0", Ready: false},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected status got = %v, want = %v", got, want)
	}
}

func Test_addAgentK8sStatus(t *testing.T) {
	ctx := context.Background()
	clientSet := fake.NewClientset()
	client := &kubeutil.Client{Clientset: clientSet}

	// Create spire-agent daemonset
	dsConfig := (&applyv1.DaemonSetApplyConfiguration{}).
		WithName("spire-agent").
		WithKind("DaemonSet").
		WithAPIVersion("apps/v1").
		WithSpec((&applyv1.DaemonSetSpecApplyConfiguration{}).
			WithSelector((&applymetav1.LabelSelectorApplyConfiguration{}).
				WithMatchLabels(map[string]string{"app.kubernetes.io/name": "agent"}),
			),
		).
		WithStatus((&applyv1.DaemonSetStatusApplyConfiguration{}).
			WithDesiredNumberScheduled(2).
			WithNumberReady(1),
		)
	_, err := clientSet.AppsV1().
		DaemonSets("spire").
		Apply(ctx, dsConfig, metav1.ApplyOptions{})
	if err != nil {
		t.Fatalf("failed to create daemonset: %v", err)
	}

	// Create spire-agent-xyz pod.
	podConfig := (&v1.PodApplyConfiguration{}).
		WithName("spire-agent-xyz").
		WithKind("Pod").
		WithAPIVersion("v1").
		WithLabels(map[string]string{"app.kubernetes.io/name": "agent"}).
		WithStatus((&v1.PodStatusApplyConfiguration{}).
			WithPhase("Running").
			WithContainerStatuses(
				(&v1.ContainerStatusApplyConfiguration{}).
					WithName("spire-agent").
					WithReady(true),
			),
		)
	_, err = clientSet.CoreV1().
		Pods("spire").
		Apply(ctx, podConfig, metav1.ApplyOptions{})
	if err != nil {
		t.Fatalf("failed to create pod: %v", err)
	}

	// Create spire-agent-not-in-list pod (not included in agents list from SPIRE server).
	podConfig = (&v1.PodApplyConfiguration{}).
		WithName("spire-agent-not-in-list").
		WithKind("Pod").
		WithAPIVersion("v1").
		WithLabels(map[string]string{"app.kubernetes.io/name": "agent"}).
		WithStatus((&v1.PodStatusApplyConfiguration{}).
			WithPhase("Running").
			WithContainerStatuses(
				(&v1.ContainerStatusApplyConfiguration{}).
					WithName("spire-agent").
					WithReady(false),
			),
		)
	_, err = clientSet.CoreV1().
		Pods("spire").
		Apply(ctx, podConfig, metav1.ApplyOptions{})
	if err != nil {
		t.Fatalf("failed to create pod: %v", err)
	}

	now := time.Now()

	// This is the list of agents from spire-server agent list.
	agents := []Agent{
		{
			Name:            "spire-agent-xyz",
			Status:          "unknown",
			Id:              "spiffe://foo.bar/baz",
			AttestationType: "k8s_psat",
			ExpirationTime:  now,
			Serial:          "1234",
			CanReattest:     true,
		},
		{
			Name:            "spire-agent-without-pod",
			Status:          "unknown",
			Id:              "spiffe://foo.bar/qux",
			AttestationType: "k8s_psat",
			ExpirationTime:  now,
			Serial:          "5678",
			CanReattest:     false,
		},
	}

	got, err := addAgentK8sStatus(ctx, client, agents)
	if err != nil {
		t.Fatalf("unexpected error %c", err)
	}

	want := &AgentStatus{
		Expected: 2,
		Ready:    1,
		Agents: []Agent{
			{
				Name:            "spire-agent-xyz",
				Status:          "Running",
				Id:              "spiffe://foo.bar/baz",
				AttestationType: "k8s_psat",
				ExpirationTime:  now,
				Serial:          "1234",
				CanReattest:     true,
			},
			{
				Name:            "spire-agent-without-pod",
				Status:          "unknown",
				Id:              "spiffe://foo.bar/qux",
				AttestationType: "k8s_psat",
				ExpirationTime:  now,
				Serial:          "5678",
				CanReattest:     false,
			},
			{
				Name:            "spire-agent-not-in-list",
				Status:          "Running",
				Id:              "unknown",
				AttestationType: "unknown",
				ExpirationTime:  time.Unix(0, 0),
				Serial:          "unknown",
				CanReattest:     false,
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected status got = %v, want = %v", got, want)
	}
}
