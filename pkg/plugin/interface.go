package plugin

import (
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
)

// DataSource is the interface plugins have to implement.
type DataSource interface {
	GetTrustZone(string) (*trust_zone_proto.TrustZone, error)
	ListTrustZones() ([]*trust_zone_proto.TrustZone, error)
	AddTrustZone(*trust_zone_proto.TrustZone) error
	AddAttestationPolicy(*attestation_policy_proto.AttestationPolicy) error
	BindAttestationPolicy(*attestation_policy_proto.AttestationPolicy, *trust_zone_proto.TrustZone) error
	ListAttestationPolicies() ([]*attestation_policy_proto.AttestationPolicy, error)
	AddFederation(*federation_proto.Federation) error
	ListFederation() ([]*federation_proto.Federation, error)
}
