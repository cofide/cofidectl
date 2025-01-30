// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustzone

import (
	"fmt"

	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
)

// GetClusterFromTrustZone returns a cluster from a trust zone.
// For now there should be exactly one cluster per trust zone.
func GetClusterFromTrustZone(trustZone *trust_zone_proto.TrustZone, ds datasource.DataSource) (*clusterpb.Cluster, error) {
	clusters, err := ds.ListClusters(trustZone.Name)
	if err != nil {
		return nil, err
	}

	if clusters == nil || len(clusters) != 1 {
		return nil, fmt.Errorf("expected exactly one cluster per trust zone")
	}
	return clusters[0], nil
}
