// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spire

import (
	"context"
	"reflect"
	"testing"
	"time"

	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		StatefulSets("spire-server").
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
		Pods("spire-server").
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
		DaemonSets("spire-system").
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
		Pods("spire-system").
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
		Pods("spire-system").
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

func TestSpire_parseServerCABundle_parseFederatedBundles(t *testing.T) {
	// Example output from /opt/spire/bin/spire-server bundle show -output json
	// trust_domain: td1
	testServerCAJSONOutput := `{"jwt_authorities":[{"expires_at":"1732178926","key_id":"1eEODyZCgwlYD7PfzP3fV5svASUUJMsz","public_key":"MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvv4ljaugt9hjVMYGIURByGVvVFQOstjtqXrtVAqo/uBiSBbm7Zq3B4a+7sMJL93zqe7DNuF3qmXWchKfBGZ+8gkB9/zSSszBrpOFSFQYgslCBwI/PNyKtPDMpYt2EdvesjJh9MZIoTqZ0nPyY/fM1yBx0mlIf7gTYJqzB0q/banoD2Ruxc/R3vru8yXPM84bIv3oyYzCNlc52k32EAGasRI580SJRnJ1ukb4GkuAkLxZXQjwwLiXhMGlZDfzxhy0foGVF64ANFyjCRpf5CTC65Cegc2UsCoI89ykVY5nLB/LuDwse1jXc4mtWiWkHZ+wwYlNHK4QuPWWZCb5B+cN4wIDAQAB","tainted":false}],"refresh_hint":"0","sequence_number":"1","trust_domain":"td1","x509_authorities":[{"asn1":"MIIDtTCCAp2gAwIBAgIQE4Hqzopq+emwuVOnc52dDDANBgkqhkiG9w0BAQsFADBoMQ0wCwYDVQQGEwRBUlBBMRAwDgYDVQQKEwdFeGFtcGxlMRQwEgYDVQQDEwtleGFtcGxlLm9yZzEvMC0GA1UEBRMmMjU5Mjk5MDA2NjIzNTEzODQ0NTA4OTAxOTEzOTEyMDQxNTQ2MzYwHhcNMjQxMTIwMjA0ODM2WhcNMjQxMTIxMDg0ODQ2WjBoMQ0wCwYDVQQGEwRBUlBBMRAwDgYDVQQKEwdFeGFtcGxlMRQwEgYDVQQDEwtleGFtcGxlLm9yZzEvMC0GA1UEBRMmMjU5Mjk5MDA2NjIzNTEzODQ0NTA4OTAxOTEzOTEyMDQxNTQ2MzYwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDV1RlLOGjoheNKC+gia2vPGiBhDu4uHEXLRHMQ7Vqi+GEPhnl4R9MvJKnF1Au2ltkRO3o8qX9/Zy78ht5OitMBk1XaaEWTMwvGXYmlr4WksAan21rVb0b20qb5BTDVqFNPiWtGqMRnH0hwoGXX39ioOzYS1zU2WrtxohWYl9rxBPToDooHGg2k7pkGn0tkeyPkYHroOe1XU61cEAOelcoGeCipqQd+eFCCf16V/HemQKfiWb8tJZjHLEnvx0DBVPA33FngOsisIkwpGVA2Ycq7vRG35vyTH6Pa7Ryoom3ZvjCVV0eyZJvL3JznVMCoyCb3z1P4pypFT6XK1YRfz5tlAgMBAAGjWzBZMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTJdjj10TsO2JLYv1bycXz4dfFbdjAXBgNVHREEEDAOhgxzcGlmZmU6Ly90ZDEwDQYJKoZIhvcNAQELBQADggEBAB8u7hiIfT4SPCHAwPXdMrqv2Z/ivyLMtwFYJAgg0/HdSWmm0IaaMPN/ZzoN+lHtomY9trGZqw5I6zRyY03EwcGR+etpzi6nPDqeuMR35rK39q2aBTVLWAwcJSV7NEUMJDQ4vgQlQZ3iO41H48zHtdJYMh9p00elIRPJdd7AdHZb9lFs4Y+cxAJSYBQxMVwYkD65fdddF850QgESx2z74zVgmPMpia63khH8L5mY9+9evPw9bXo/xr4qUo8Mj2PFg4+GUPJobqN2eGkFr886+HeE67cjd77k8cHoQ/ZLFrYAT26qd7diJFNuR9R0zrDV0kfgdFiH6sfIao22ISM2d2Y=","tainted":false}]}`

	// Example from /opt/spire/bin/spire-server bundle list -output json with td1 successfully federated
	testBundleListJSONOutput := `{"bundles":[{"jwt_authorities":[{"expires_at":"0","key_id":"1eEODyZCgwlYD7PfzP3fV5svASUUJMsz","public_key":"MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvv4ljaugt9hjVMYGIURByGVvVFQOstjtqXrtVAqo/uBiSBbm7Zq3B4a+7sMJL93zqe7DNuF3qmXWchKfBGZ+8gkB9/zSSszBrpOFSFQYgslCBwI/PNyKtPDMpYt2EdvesjJh9MZIoTqZ0nPyY/fM1yBx0mlIf7gTYJqzB0q/banoD2Ruxc/R3vru8yXPM84bIv3oyYzCNlc52k32EAGasRI580SJRnJ1ukb4GkuAkLxZXQjwwLiXhMGlZDfzxhy0foGVF64ANFyjCRpf5CTC65Cegc2UsCoI89ykVY5nLB/LuDwse1jXc4mtWiWkHZ+wwYlNHK4QuPWWZCb5B+cN4wIDAQAB","tainted":false}],"refresh_hint":"300","sequence_number":"0","trust_domain":"td1","x509_authorities":[{"asn1":"MIIDtTCCAp2gAwIBAgIQE4Hqzopq+emwuVOnc52dDDANBgkqhkiG9w0BAQsFADBoMQ0wCwYDVQQGEwRBUlBBMRAwDgYDVQQKEwdFeGFtcGxlMRQwEgYDVQQDEwtleGFtcGxlLm9yZzEvMC0GA1UEBRMmMjU5Mjk5MDA2NjIzNTEzODQ0NTA4OTAxOTEzOTEyMDQxNTQ2MzYwHhcNMjQxMTIwMjA0ODM2WhcNMjQxMTIxMDg0ODQ2WjBoMQ0wCwYDVQQGEwRBUlBBMRAwDgYDVQQKEwdFeGFtcGxlMRQwEgYDVQQDEwtleGFtcGxlLm9yZzEvMC0GA1UEBRMmMjU5Mjk5MDA2NjIzNTEzODQ0NTA4OTAxOTEzOTEyMDQxNTQ2MzYwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDV1RlLOGjoheNKC+gia2vPGiBhDu4uHEXLRHMQ7Vqi+GEPhnl4R9MvJKnF1Au2ltkRO3o8qX9/Zy78ht5OitMBk1XaaEWTMwvGXYmlr4WksAan21rVb0b20qb5BTDVqFNPiWtGqMRnH0hwoGXX39ioOzYS1zU2WrtxohWYl9rxBPToDooHGg2k7pkGn0tkeyPkYHroOe1XU61cEAOelcoGeCipqQd+eFCCf16V/HemQKfiWb8tJZjHLEnvx0DBVPA33FngOsisIkwpGVA2Ycq7vRG35vyTH6Pa7Ryoom3ZvjCVV0eyZJvL3JznVMCoyCb3z1P4pypFT6XK1YRfz5tlAgMBAAGjWzBZMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTJdjj10TsO2JLYv1bycXz4dfFbdjAXBgNVHREEEDAOhgxzcGlmZmU6Ly90ZDEwDQYJKoZIhvcNAQELBQADggEBAB8u7hiIfT4SPCHAwPXdMrqv2Z/ivyLMtwFYJAgg0/HdSWmm0IaaMPN/ZzoN+lHtomY9trGZqw5I6zRyY03EwcGR+etpzi6nPDqeuMR35rK39q2aBTVLWAwcJSV7NEUMJDQ4vgQlQZ3iO41H48zHtdJYMh9p00elIRPJdd7AdHZb9lFs4Y+cxAJSYBQxMVwYkD65fdddF850QgESx2z74zVgmPMpia63khH8L5mY9+9evPw9bXo/xr4qUo8Mj2PFg4+GUPJobqN2eGkFr886+HeE67cjd77k8cHoQ/ZLFrYAT26qd7diJFNuR9R0zrDV0kfgdFiH6sfIao22ISM2d2Y=","tainted":false}]}],"next_page_token":""}`

	gotServerCA, err := parseServerCABundle([]byte(testServerCAJSONOutput))
	require.Nil(t, err)
	gotFederated, err := parseFederatedBundles([]byte(testBundleListJSONOutput))
	require.Nil(t, err)

	assert.NotNil(t, gotFederated["td1"])
	assert.Equal(t, gotServerCA, gotFederated["td1"])
}
