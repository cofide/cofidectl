// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	"fmt"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_provider/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/proto"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var trustZoneFixtures map[string]*trust_zone_proto.TrustZone = map[string]*trust_zone_proto.TrustZone{
	"tz1": {
		Name:              "tz1",
		TrustDomain:       "td1",
		KubernetesCluster: StringPtr("local1"),
		KubernetesContext: StringPtr("kind-local1"),
		TrustProvider: &trust_provider_proto.TrustProvider{
			Kind: StringPtr("kubernetes"),
		},
		BundleEndpointUrl: StringPtr("127.0.0.1"),
		Federations: []*federation_proto.Federation{
			{
				From: "tz1",
				To:   "tz2",
			},
		},
		AttestationPolicies: []*ap_binding_proto.APBinding{
			{
				TrustZone:     "tz1",
				Policy:        "ap1",
				FederatesWith: []string{"tz2"},
			},
		},
		JwtIssuer: StringPtr("https://tz1.example.com"),
		ExtraHelmValues: func() *structpb.Struct {
			ev := map[string]any{
				"global": map[string]any{
					"spire": map[string]any{
						"namespaces": map[string]any{
							"create": true,
						},
					},
				},
				"spire-server": map[string]any{
					"logLevel": "INFO",
				},
			}
			value, err := structpb.NewStruct(ev)
			if err != nil {
				panic(err)
			}
			return value
		}(),
	},
	"tz2": {
		Name:              "tz2",
		TrustDomain:       "td2",
		KubernetesCluster: StringPtr("local2"),
		KubernetesContext: StringPtr("kind-local2"),
		TrustProvider: &trust_provider_proto.TrustProvider{
			Kind: StringPtr("kubernetes"),
		},
		BundleEndpointUrl: StringPtr("127.0.0.2"),
		Federations: []*federation_proto.Federation{
			{
				From: "tz2",
				To:   "tz1",
			},
		},
		AttestationPolicies: []*ap_binding_proto.APBinding{
			{
				TrustZone:     "tz2",
				Policy:        "ap2",
				FederatesWith: []string{"tz1"},
			},
		},
		JwtIssuer: StringPtr("https://tz2.example.com"),
	},
	// tz3 has no federations or bound attestation policies.
	"tz3": {
		Name:              "tz3",
		TrustDomain:       "td3",
		KubernetesCluster: StringPtr("local3"),
		KubernetesContext: StringPtr("kind-local3"),
		TrustProvider: &trust_provider_proto.TrustProvider{
			Kind: StringPtr("kubernetes"),
		},
		BundleEndpointUrl:   StringPtr("127.0.0.3"),
		Federations:         []*federation_proto.Federation{},
		AttestationPolicies: []*ap_binding_proto.APBinding{},
	},
}

var attestationPolicyFixtures map[string]*attestation_policy_proto.AttestationPolicy = map[string]*attestation_policy_proto.AttestationPolicy{
	"ap1": {
		Name: "ap1",
		Policy: &attestation_policy_proto.AttestationPolicy_Kubernetes{
			Kubernetes: &attestation_policy_proto.APKubernetes{
				NamespaceSelector: &attestation_policy_proto.APLabelSelector{
					MatchLabels: map[string]string{"kubernetes.io/metadata.name": "ns1"},
				},
			},
		},
	},
	"ap2": {
		Name: "ap2",
		Policy: &attestation_policy_proto.AttestationPolicy_Kubernetes{
			Kubernetes: &attestation_policy_proto.APKubernetes{
				PodSelector: &attestation_policy_proto.APLabelSelector{
					MatchExpressions: []*attestation_policy_proto.APMatchExpression{
						{
							Key:      "foo",
							Operator: string(metav1.LabelSelectorOpIn),
							Values:   []string{"bar"},
						},
					},
				},
			},
		},
	},
	"ap3": {
		Name: "ap3",
		Policy: &attestation_policy_proto.AttestationPolicy_Kubernetes{
			Kubernetes: &attestation_policy_proto.APKubernetes{
				NamespaceSelector: &attestation_policy_proto.APLabelSelector{
					MatchLabels: map[string]string{"kubernetes.io/metadata.name": "ns3"},
				},
				PodSelector: &attestation_policy_proto.APLabelSelector{
					MatchLabels: map[string]string{"label1": "value1", "label2": "value2"},
					MatchExpressions: []*attestation_policy_proto.APMatchExpression{
						{
							Key:      "foo",
							Operator: string(metav1.LabelSelectorOpIn),
							Values:   []string{"bar", "baz"},
						},
						{
							Key:      "foo",
							Operator: string(metav1.LabelSelectorOpNotIn),
							Values:   []string{"qux", "quux"},
						},
						{
							Key:      "bar",
							Operator: string(metav1.LabelSelectorOpExists),
						},
						{
							Key:      "baz",
							Operator: string(metav1.LabelSelectorOpDoesNotExist),
						},
					},
				},
			},
		},
	},
}

var pluginConfigFixtures map[string]*structpb.Struct = map[string]*structpb.Struct{
	"plugin1": func() *structpb.Struct {
		s, err := structpb.NewStruct(map[string]any{
			"list-cfg": []any{
				456,
				"another-string",
			},
			"map-cfg": map[string]any{
				"key1": "yet-another",
				"key2": 789,
			},
		})
		if err != nil {
			panic(fmt.Sprintf("failed to create struct: %s", err))
		}
		return s
	}(),
	"plugin2": func() *structpb.Struct {
		s, err := structpb.NewStruct(map[string]any{
			"string-cfg": "fake-string",
			"number-cfg": 123,
		})
		if err != nil {
			panic(fmt.Sprintf("failed to create struct: %s", err))
		}
		return s
	}(),
}

func TrustZone(name string) *trust_zone_proto.TrustZone {
	tz, ok := trustZoneFixtures[name]
	if !ok {
		panic(fmt.Sprintf("invalid trust zone fixture %s", name))
	}
	tz, err := proto.CloneTrustZone(tz)
	if err != nil {
		panic(fmt.Sprintf("failed to clone trust zone: %s", err))
	}
	return tz
}

func AttestationPolicy(name string) *attestation_policy_proto.AttestationPolicy {
	ap, ok := attestationPolicyFixtures[name]
	if !ok {
		panic(fmt.Sprintf("invalid attestation policy fixture %s", name))
	}
	ap, err := proto.CloneAttestationPolicy(ap)
	if err != nil {
		panic(fmt.Sprintf("failed to clone attestation policy: %s", err))
	}
	return ap
}

func PluginConfig(name string) *structpb.Struct {
	pc, ok := pluginConfigFixtures[name]
	if !ok {
		panic(fmt.Sprintf("invalid plugin config fixture %s", name))
	}
	pc, err := proto.CloneStruct(pc)
	if err != nil {
		panic(fmt.Sprintf("failed to clone plugin config: %s", err))
	}
	return pc
}

func StringPtr(s string) *string {
	return &s
}
