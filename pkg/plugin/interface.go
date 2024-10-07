package plugin

import (
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
)

// DataSource is the interface plugins have to implement.
type DataSource interface {
	ListTrustZones() ([]*trust_zone_proto.TrustZone, error)
	//CreateTrustZone() (*trust_zone_proto.TrustZone, error)
	//CreateAttestationPolicy() (*attestation_policy_proto.AttestationPolicy, error)
}
