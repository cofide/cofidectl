package fixtures

import (
	"fmt"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/proto/ap_binding/v1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_provider/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/cofide/cofidectl/internal/pkg/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var trustZoneFixtures map[string]*trust_zone_proto.TrustZone = map[string]*trust_zone_proto.TrustZone{
	"tz1": {
		Name:              "tz1",
		TrustDomain:       "td1",
		KubernetesCluster: "local1",
		KubernetesContext: "kind-local1",
		TrustProvider: &trust_provider_proto.TrustProvider{
			Kind: "kubernetes",
		},
		BundleEndpointUrl: "127.0.0.1",
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
	},
	"tz2": {
		Name:              "tz2",
		TrustDomain:       "td2",
		KubernetesCluster: "local2",
		KubernetesContext: "kind-local2",
		TrustProvider: &trust_provider_proto.TrustProvider{
			Kind: "kubernetes",
		},
		BundleEndpointUrl: "127.0.0.2",
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
