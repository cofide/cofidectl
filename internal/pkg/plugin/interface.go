package plugin

import trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1"

type DataSource interface {
	GetTrustZones() ([]*trust_zone_proto.TrustZone, error)
}
