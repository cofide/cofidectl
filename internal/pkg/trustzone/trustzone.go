package trustzone

import (
	"buf.build/go/protoyaml"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
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

func (tz *TrustZone) marshalToYAML() ([]byte, error) {
	return protoyaml.Marshal(tz.TrustZoneProto)
}

func (tz *TrustZone) unmarshalFromYAML(data []byte) error {
	return protoyaml.Unmarshal(data, tz.TrustZoneProto)
}

func (tz *TrustZone) GetTrustProviderProto() (*trustprovider.TrustProvider, error) {
	return trustprovider.NewTrustProvider(tz.TrustZoneProto.TrustProvider.Kind), nil
}
