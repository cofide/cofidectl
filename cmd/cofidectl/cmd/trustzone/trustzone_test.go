// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustzone

import (
	"context"
	"errors"
	"testing"

	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofide-api-sdk/pkg/connect/client/test"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOpts(t *testing.T) {
	// https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE-ID.md#21-trust-domain
	tt := []struct {
		name        string
		domain      string
		errExpected bool
	}{
		{domain: "example.com", errExpected: false},
		{domain: "example-domain.com", errExpected: false},
		{domain: "example_domain.com", errExpected: false},
		{domain: "spiffe://example.com", errExpected: false},
		{domain: "EXAMPLE.COM", errExpected: true},
		{domain: "example.com:1234", errExpected: true},
		{domain: "user:password@example.com", errExpected: true},
		{domain: "example?.com", errExpected: true},
		{domain: "exam%3Aple.com", errExpected: true},
	}

	for _, tc := range tt {
		t.Run(tc.domain, func(t *testing.T) {
			err := validateOpts(addOpts{trustDomain: tc.domain})
			assert.Equal(t, tc.errExpected, err != nil)
		})
	}
}

func TestTrustZoneCommand_addTrustZone(t *testing.T) {
	tests := []struct {
		name           string
		trustZoneName  string
		injectFailure  bool
		wantErr        bool
		wantErrMessage string
	}{
		{
			name:          "success",
			trustZoneName: "tz3",
		},
		{
			name:           "already exists",
			trustZoneName:  "tz1",
			wantErr:        true,
			wantErrMessage: "trust zone tz1 already exists in local config",
		},
		{
			name:           "trust zone add rollback",
			trustZoneName:  "tz3",
			injectFailure:  true,
			wantErr:        true,
			wantErrMessage: "fake add failure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := newFakeDataSource(t, defaultConfig())
			if tt.injectFailure {
				ds = &failingDS{LocalDataSource: ds.(*local.LocalDataSource)}
			}
			opts := addOpts{
				name:              tt.trustZoneName,
				trustDomain:       "td3",
				kubernetesCluster: "local3",
				context:           "kind-local3",
				profile:           "kubernetes",
				noCluster:         false,
			}
			c := TrustZoneCommand{}
			err := c.addTrustZone(context.Background(), opts, ds)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrMessage)

				// Check that trust zone was not added.
				trustZones, err := ds.ListTrustZones()
				require.NoError(t, err)
				for _, trustZone := range trustZones {
					require.NotEqual(t, "tz3", trustZone.Name)
				}
			} else {
				require.NoError(t, err)

				// Check that trust zone and cluster were added.
				trustZone, err := ds.GetTrustZoneByName(tt.trustZoneName)
				require.NoError(t, err)
				require.NotNil(t, trustZone)

				clusters, err := ds.ListClusters(&datasourcepb.ListClustersRequest_Filter{
					TrustZoneId: trustZone.Id,
				})
				require.NoError(t, err)
				require.Len(t, clusters, 1)
				assert.Equal(t, "local3", clusters[0].GetName())
			}
		})
	}
}

func TestTrustZoneCommand_deleteTrustZone(t *testing.T) {
	tests := []struct {
		name           string
		trustZoneName  string
		injectFailure  bool
		wantErr        bool
		wantErrMessage string
	}{
		{
			name:          "exists",
			trustZoneName: "tz1",
		},
		{
			name:           "doesn't exist",
			trustZoneName:  "invalid tz",
			wantErr:        true,
			wantErrMessage: "failed to find trust zone invalid tz in local config",
		},
		{
			name:           "cluster delete rollback",
			trustZoneName:  "tz1",
			injectFailure:  true,
			wantErr:        true,
			wantErrMessage: "fake destroy failure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := newFakeDataSource(t, defaultConfig())
			if tt.injectFailure {
				ds = &failingDS{LocalDataSource: ds.(*local.LocalDataSource)}
			}
			err := deleteTrustZone(context.Background(), tt.trustZoneName, ds, "", true)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrMessage)

				// Check that trust zone and clusters were not deleted.
				_, err := ds.GetTrustZone("tz1-id")
				require.NoError(t, err)
				if tt.injectFailure {
					// Currently the local datasource limits us to one cluster per trust zone, so rolling back deletion fails.
					assert.Equal(t, 1, ds.(*failingDS).clustersAdded)
				} else {
					for _, cluster := range defaultConfig().Clusters {
						_, err := ds.GetCluster(cluster.GetId())
						require.NoError(t, err)
					}
				}
			} else {
				require.NoError(t, err)

				// Check that trust zone and clusters were deleted.
				_, err := ds.GetTrustZoneByName(tt.trustZoneName)
				require.Error(t, err)
				for _, cluster := range defaultConfig().Clusters {
					_, err := ds.GetCluster(cluster.GetId())
					require.Error(t, err)
				}
			}
		})
	}
}

type failingDS struct {
	*local.LocalDataSource
	clustersDestroyed int
	clustersAdded     int
}

// AddTrustZone fails unconditionally.
func (f *failingDS) AddTrustZone(trustZone *trust_zone_proto.TrustZone) (*trust_zone_proto.TrustZone, error) {
	return nil, errors.New("fake add failure")
}

// AddCluster keeps track of the number of clusters added.
func (f *failingDS) AddCluster(cluster *clusterpb.Cluster) (*clusterpb.Cluster, error) {
	f.clustersAdded++
	return f.LocalDataSource.AddCluster(cluster)
}

// DestroyCluster fails when a second cluster is destroyed, allowing testing of rollback.
func (f *failingDS) DestroyCluster(id string) error {
	f.clustersDestroyed++
	if f.clustersDestroyed == 2 {
		return errors.New("fake destroy failure")
	}
	return f.LocalDataSource.DestroyCluster(id)
}

func newFakeDataSource(t *testing.T, cfg *config.Config) datasource.DataSource {
	configLoader, err := config.NewMemoryLoader(cfg)
	require.Nil(t, err)
	lds, err := local.NewLocalDataSource(configLoader)
	require.Nil(t, err)
	return lds
}

func defaultConfig() *config.Config {
	return &config.Config{
		TrustZones: []*trust_zone_proto.TrustZone{
			fixtures.TrustZone("tz1"),
		},
		Clusters: []*clusterpb.Cluster{
			fixtures.Cluster("local1"),
			func() *clusterpb.Cluster {
				cluster := fixtures.Cluster("local1")
				cluster.Name = test.PtrOf("local2")
				return cluster
			}(),
		},
		Plugins: fixtures.Plugins("plugins1"),
	}
}
