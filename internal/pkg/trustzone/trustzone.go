// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustzone

import (
	"errors"

	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
)

var (
	ErrNoClustersInTrustZone  = errors.New("no clusters in trust zone")
	ErrOneClusterPerTrustZone = errors.New("expected exactly one cluster per trust zone")
)

// GetClusterFromTrustZone returns a cluster from a trust zone.
// For now there should be exactly one cluster per trust zone.
func GetClusterFromTrustZone(trustZone *trust_zone_proto.TrustZone, ds datasource.DataSource) (*clusterpb.Cluster, error) {
	clusters, err := ds.ListClusters(&datasourcepb.ListClustersRequest_Filter{
		TrustZoneId: trustZone.Id,
	})
	if err != nil {
		return nil, err
	}

	if len(clusters) < 1 {
		return nil, ErrNoClustersInTrustZone
	}
	if len(clusters) > 1 {
		return nil, ErrOneClusterPerTrustZone
	}
	return clusters[0], nil
}
