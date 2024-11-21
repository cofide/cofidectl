// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustzone

import (
	"fmt"

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
	trustProviderProto := tz.TrustZoneProto.GetTrustProvider()
	if trustProviderProto == nil {
		return nil, fmt.Errorf("no trust provider for trust zone %s", tz.TrustZoneProto.Name)
	}
	return trustprovider.NewTrustProvider(trustProviderProto.GetKind())
}
