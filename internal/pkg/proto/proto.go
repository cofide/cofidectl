// Package proto provides utilities for cofide-api-sdk protobuf types.
package proto

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
)

func CloneTrustZone(trustZone *trust_zone_proto.TrustZone) *trust_zone_proto.TrustZone {
	if clone, ok := proto.Clone(trustZone).(*trust_zone_proto.TrustZone); !ok {
		panic(fmt.Sprintf("type assertion failed for trust zone %v", trustZone))
	} else {
		if clone == trustZone {
			panic("trust zone clones are the same")
		}
		return clone
	}
}

func CloneAttestationPolicy(policy *attestation_policy_proto.AttestationPolicy) *attestation_policy_proto.AttestationPolicy {
	if clone, ok := proto.Clone(policy).(*attestation_policy_proto.AttestationPolicy); !ok {
		panic(fmt.Sprintf("type assertion failed for attestation policy %v", policy))
	} else {
		if clone == policy {
			panic("attestation policy clones are the same")
		}
		return clone
	}
}

func AttestationPoliciesEqual(ap1, ap2 *attestation_policy_proto.AttestationPolicy) bool {
	return proto.Equal(ap1, ap2)
}

func CloneFederation(federation *federation_proto.Federation) *federation_proto.Federation {
	if clone, ok := proto.Clone(federation).(*federation_proto.Federation); !ok {
		panic(fmt.Sprintf("type assertion failed for federation %v", federation))
	} else {
		if clone == federation {
			panic("federation clones are the same")
		}
		return clone
	}
}

func FederationsEqual(f1, f2 *federation_proto.Federation) bool {
	return proto.Equal(f1, f2)
}
