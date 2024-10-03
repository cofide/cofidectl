package plugin

import (
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1"
)

// DataSource is the interface plugins have to implement.
type DataSource interface {
	GetTrustZones() ([]*trust_zone_proto.TrustZone, error)
	CreateTrustZone() (*trust_zone_proto.TrustZone, error)
	CreateAttestationPolicy() (*attestation_policy_proto.AttestationPolicy, error)
}
