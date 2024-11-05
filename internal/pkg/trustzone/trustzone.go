// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustzone

import (
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

func (tz *TrustZone) GetTrustProvider() (*trustprovider.TrustProvider, error) {
	return trustprovider.NewTrustProvider(tz.TrustZoneProto.TrustProvider.Kind)
}
