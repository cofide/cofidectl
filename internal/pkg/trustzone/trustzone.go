// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustzone

import (
	"fmt"

	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
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
	cluster, err := GetClusterFromTrustZone(tz.TrustZoneProto)
	if err != nil {
		return nil, err
	}

	trustProviderProto := cluster.GetTrustProvider()
	if trustProviderProto == nil {
		return nil, fmt.Errorf("no trust provider for trust zone %s", tz.TrustZoneProto.Name)
	}
	return trustprovider.NewTrustProvider(trustProviderProto.GetKind())
}

// GetClusterFromTrustZone returns a cluster from a trust zone.
// For now there should be exactly one cluster per trust zone.
func GetClusterFromTrustZone(trustZone *trust_zone_proto.TrustZone) (*clusterpb.Cluster, error) {
	clusters := trustZone.GetClusters()
	if clusters == nil || len(clusters) != 1 {
		return nil, fmt.Errorf("expected exactly one cluster per trust zone")
	}
	return clusters[0], nil
}
