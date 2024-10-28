package fixtures

import (
	"fmt"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_provider/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/cofide/cofidectl/internal/pkg/proto"
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
				Left:  "tz1",
				Right: "tz2",
			},
		},
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
			{
				Name:      "ap1",
				Kind:      2,
				Namespace: "ns1",
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
				Left:  "tz2",
				Right: "tz1",
			},
		},
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
			{
				Name:     "ap2",
				Kind:     1,
				PodKey:   "foo",
				PodValue: "bar",
			},
		},
	},
}

var attestationPolicyFixtures map[string]*attestation_policy_proto.AttestationPolicy = map[string]*attestation_policy_proto.AttestationPolicy{
	"ap1": {
		Name:      "ap1",
		Kind:      2,
		Namespace: "ns1",
	},
	"ap2": {
		Name:     "ap2",
		Kind:     1,
		PodKey:   "foo",
		PodValue: "bar",
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
