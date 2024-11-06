package trustzone

import (
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/trustprovider"
)

type TrustZone struct {
	TrustZoneProto *trust_zone_proto.TrustZone
}

func NewTrustZone(trustZone *trust_zone_proto.TrustZone) *TrustZone {
	return &TrustZone{
		TrustZoneProto: trustZone,
	}
}

func (tz *TrustZone) GetTrustProvider() (*trustprovider.TrustProvider, error) {
	return trustprovider.NewTrustProvider(tz.TrustZoneProto.TrustProvider.Kind)
}
