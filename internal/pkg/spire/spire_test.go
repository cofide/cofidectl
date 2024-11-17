// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

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

func Test_parseFederatedBundles(t *testing.T) {
	testServerCABundle := `
-----BEGIN CERTIFICATE-----
MIIDvzCCAqegAwIBAgIRAIfBA+bI9ZPvf2rz5cCyMvIwDQYJKoZIhvcNAQELBQAw
aTENMAsGA1UEBhMEQVJQQTEQMA4GA1UEChMHRXhhbXBsZTEUMBIGA1UEAxMLZXhh
bXBsZS5vcmcxMDAuBgNVBAUTJzE4MDQ0Nzk3MTg1NjU2MDk0MTIwNjAzMDA4OTU5
MTg5MTA0NzE1NDAeFw0yNDExMTcyMTU5MzFaFw0yNDExMTgwOTU5NDFaMGkxDTAL
BgNVBAYTBEFSUEExEDAOBgNVBAoTB0V4YW1wbGUxFDASBgNVBAMTC2V4YW1wbGUu
b3JnMTAwLgYDVQQFEycxODA0NDc5NzE4NTY1NjA5NDEyMDYwMzAwODk1OTE4OTEw
NDcxNTQwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDAnshruIk80EFv
oBgD7YglCLqeSdrTvlYKp9XU6X6AXtrnwG45W9vbnjubGspcKh4x0YhQA2DFmzuz
S2cbXDUWuxRCWvqiMl+WD9O2E6vgwz/zqIQteCyhSU4y1eROsJYABrHv6n99loAZ
bqzVY4e/Hqwr54hPNZb5G3m7tBBuJu3EJA5dAGn4yApFgzvBmAfrxMtyHZmLacN2
qXni43Xnt83UfWdtyvHJOpIrGnt7G4rLCmVtPEhcUvc9s5TpHbjZ1q69lB3oRgtc
pAW6+FQRLoMxRbhiCWO8EI/jBdxVSPl0peEVMPda9rocAo5rvIBUUyBL+mWQYyWF
G6l52n0BAgMBAAGjYjBgMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/
MB0GA1UdDgQWBBRfnPwDeWjIRC/65G2LLfOssIJOmjAeBgNVHREEFzAVhhNzcGlm
ZmU6Ly9hbHBoYS50ZXN0MA0GCSqGSIb3DQEBCwUAA4IBAQCVBlpy1BLr80ZxvKI7
MZLzMH2XVQknWzTnFBqsU3posHUpc8AKzWN7HwkoRBgOZuNV1sGZXSTFCq7BAYC3
R8bC2IRMcfsJMV3NgrguwN3Ij+Rb/h66F1x7ePYHuDeXblZysUXh7Rg88EiiXtnX
6aDisbGRDvoFy4ojbLMaHlvdm+f/MPpS4flSrxDOKYWMPdiKS+QefnxzQcGUiEZ9
E0byKNnT/G6fvk71z7qE3sw3dwWjYoWuzZKuPgOQKKQbclLeSMWlOX7qMUUv1bf/
PvsK5QZwKmo49e2NQezUI4V+pOnSw+9mklI+0IiMpkC8CJ4ZEWf973ufa2RREgsS
D6Eq
-----END CERTIFICATE-----
`
	testFederatedBundles := `
****************************************
* alpha.test
****************************************
-----BEGIN CERTIFICATE-----
MIIDvzCCAqegAwIBAgIRAIfBA+bI9ZPvf2rz5cCyMvIwDQYJKoZIhvcNAQELBQAw
aTENMAsGA1UEBhMEQVJQQTEQMA4GA1UEChMHRXhhbXBsZTEUMBIGA1UEAxMLZXhh
bXBsZS5vcmcxMDAuBgNVBAUTJzE4MDQ0Nzk3MTg1NjU2MDk0MTIwNjAzMDA4OTU5
MTg5MTA0NzE1NDAeFw0yNDExMTcyMTU5MzFaFw0yNDExMTgwOTU5NDFaMGkxDTAL
BgNVBAYTBEFSUEExEDAOBgNVBAoTB0V4YW1wbGUxFDASBgNVBAMTC2V4YW1wbGUu
b3JnMTAwLgYDVQQFEycxODA0NDc5NzE4NTY1NjA5NDEyMDYwMzAwODk1OTE4OTEw
NDcxNTQwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDAnshruIk80EFv
oBgD7YglCLqeSdrTvlYKp9XU6X6AXtrnwG45W9vbnjubGspcKh4x0YhQA2DFmzuz
S2cbXDUWuxRCWvqiMl+WD9O2E6vgwz/zqIQteCyhSU4y1eROsJYABrHv6n99loAZ
bqzVY4e/Hqwr54hPNZb5G3m7tBBuJu3EJA5dAGn4yApFgzvBmAfrxMtyHZmLacN2
qXni43Xnt83UfWdtyvHJOpIrGnt7G4rLCmVtPEhcUvc9s5TpHbjZ1q69lB3oRgtc
pAW6+FQRLoMxRbhiCWO8EI/jBdxVSPl0peEVMPda9rocAo5rvIBUUyBL+mWQYyWF
G6l52n0BAgMBAAGjYjBgMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/
MB0GA1UdDgQWBBRfnPwDeWjIRC/65G2LLfOssIJOmjAeBgNVHREEFzAVhhNzcGlm
ZmU6Ly9hbHBoYS50ZXN0MA0GCSqGSIb3DQEBCwUAA4IBAQCVBlpy1BLr80ZxvKI7
MZLzMH2XVQknWzTnFBqsU3posHUpc8AKzWN7HwkoRBgOZuNV1sGZXSTFCq7BAYC3
R8bC2IRMcfsJMV3NgrguwN3Ij+Rb/h66F1x7ePYHuDeXblZysUXh7Rg88EiiXtnX
6aDisbGRDvoFy4ojbLMaHlvdm+f/MPpS4flSrxDOKYWMPdiKS+QefnxzQcGUiEZ9
E0byKNnT/G6fvk71z7qE3sw3dwWjYoWuzZKuPgOQKKQbclLeSMWlOX7qMUUv1bf/
PvsK5QZwKmo49e2NQezUI4V+pOnSw+9mklI+0IiMpkC8CJ4ZEWf973ufa2RREgsS
D6Eq
-----END CERTIFICATE-----
`
	got := parseFederatedBundles([]byte(testFederatedBundles))
	if !reflect.DeepEqual(got["alpha.test"], testServerCABundle) {
		t.Errorf("bundles do not match got = %v, want = %v", got["alpha.test"], testServerCABundle)
	}
}
