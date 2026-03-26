// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustzone

import (
	"errors"
	"fmt"

	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
)

var ErrNoClustersInTrustZone = errors.New("no clusters in trust zone")

// GetClustersByTrustZone returns all clusters for a trust zone.
// Returns ErrNoClustersInTrustZone if the trust zone has no clusters.
func GetClustersByTrustZone(trustZone *trust_zone_proto.TrustZone, ds datasource.DataSource) ([]*clusterpb.Cluster, error) {
	clusters, err := ds.ListClusters(&datasourcepb.ListClustersRequest_Filter{
		TrustZoneId: trustZone.Id,
	})
	if err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		return nil, ErrNoClustersInTrustZone
	}
	return clusters, nil
}

// GetClusterFromTrustZoneByName looks up a named cluster within a trust zone.
func GetClusterFromTrustZoneByName(trustZone *trust_zone_proto.TrustZone, clusterName string, ds datasource.DataSource) (*clusterpb.Cluster, error) {
	cluster, err := ds.GetClusterByName(clusterName, trustZone.GetId())
	if err != nil {
		return nil, fmt.Errorf("cluster %q not found in trust zone %q: %w", clusterName, trustZone.GetName(), err)
	}
	return cluster, nil
}

// ResolveCluster resolves a cluster for a trust zone given an optional cluster name.
// If clusterName is non-empty, it is looked up by name.
// If clusterName is empty and the trust zone has exactly one cluster, that cluster is returned.
// If clusterName is empty and the trust zone has multiple clusters, an error is returned instructing
// the caller to use --cluster to specify one.
func ResolveCluster(trustZone *trust_zone_proto.TrustZone, clusterName string, ds datasource.DataSource) (*clusterpb.Cluster, error) {
	if clusterName != "" {
		return GetClusterFromTrustZoneByName(trustZone, clusterName, ds)
	}
	clusters, err := GetClustersByTrustZone(trustZone, ds)
	if err != nil {
		return nil, err
	}
	if len(clusters) > 1 {
		return nil, fmt.Errorf("trust zone %q has multiple clusters; specify one with --cluster", trustZone.GetName())
	}
	return clusters[0], nil
}
