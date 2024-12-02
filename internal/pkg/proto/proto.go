// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
)

func CloneTrustZone(trustZone *trust_zone_proto.TrustZone) (*trust_zone_proto.TrustZone, error) {
	if clone, ok := proto.Clone(trustZone).(*trust_zone_proto.TrustZone); !ok {
		return nil, fmt.Errorf("bug: type assertion failed for trust zone %s", trustZone.Name)
	} else {
		if clone == trustZone {
			return nil, fmt.Errorf("bug: trust zone %s clones point to same address", trustZone.Name)
		}
		return clone, nil
	}
}

func CloneAttestationPolicy(policy *attestation_policy_proto.AttestationPolicy) (*attestation_policy_proto.AttestationPolicy, error) {
	if clone, ok := proto.Clone(policy).(*attestation_policy_proto.AttestationPolicy); !ok {
		return nil, fmt.Errorf("bug: type assertion failed for attestation policy %s", policy.Name)
	} else {
		if clone == policy {
			return nil, fmt.Errorf("bug: attestation policy %s clones are the same", policy.Name)
		}
		return clone, nil
	}
}

func CloneAPBinding(binding *ap_binding_proto.APBinding) (*ap_binding_proto.APBinding, error) {
	if clone, ok := proto.Clone(binding).(*ap_binding_proto.APBinding); !ok {
		return nil, fmt.Errorf("bug: type assertion failed for attestation policy binding %s/%s", binding.Policy, binding.TrustZone)
	} else {
		if clone == binding {
			return nil, fmt.Errorf("bug: attestation policy binding %s/%s clones are the same", binding.Policy, binding.TrustZone)
		}
		return clone, nil
	}
}

func CloneStruct(spb *structpb.Struct) (*structpb.Struct, error) {
	if clone, ok := proto.Clone(spb).(*structpb.Struct); !ok {
		return nil, fmt.Errorf("bug: type assertion failed for struct %s", spb)
	} else {
		if clone == spb {
			return nil, fmt.Errorf("bug: struct %s clones are the same", spb)
		}
		return clone, nil
	}
}

func APBindingsEqual(apb1, apb2 *ap_binding_proto.APBinding) bool {
	return proto.Equal(apb1, apb2)
}

func CloneFederation(federation *federation_proto.Federation) (*federation_proto.Federation, error) {
	if clone, ok := proto.Clone(federation).(*federation_proto.Federation); !ok {
		return nil, fmt.Errorf("bug: type assertion failed for federation %s-%s", federation.From, federation.To)
	} else {
		if clone == federation {
			return nil, fmt.Errorf("bug: federation %s-%s clones are the same", federation.To, federation.To)
		}
		return clone, nil
	}
}

func FederationsEqual(f1, f2 *federation_proto.Federation) bool {
	return proto.Equal(f1, f2)
}

func StructsEqual(s1, s2 *structpb.Struct) bool {
	return proto.Equal(s1, s2)
}
