package plugin

import trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1"

// DataSource is the interface plugins have to implement.
type DataSource interface {
	GetTrustZones() ([]*trust_zone_proto.TrustZone, error)
}
