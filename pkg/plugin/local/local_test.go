// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"slices"
	"testing"

	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/google/go-cmp/cmp"
	spiretypes "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
)

func TestLocalDataSource_ImplementsDataSource(t *testing.T) {
	local := LocalDataSource{}
	var _ datasource.DataSource = &local
}

func TestNewLocalDataSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		config     *config.Config
		wantConfig *config.Config
		wantErr    bool
	}{
		{
			name:       "non-existent config",
			config:     nil,
			wantConfig: nil,
			wantErr:    true,
		},
		{
			name:       "default config",
			config:     config.NewConfig(),
			wantConfig: config.NewConfig(),
		},
		{
			name: "non-default config",
			config: &config.Config{
				Plugins: fixtures.Plugins("plugins1"),
			},
			wantConfig: &config.Config{
				TrustZones:          []*trust_zone_proto.TrustZone{},
				Clusters:            []*clusterpb.Cluster{},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
				ApBindings:          []*ap_binding_proto.APBinding{},
				PluginConfig:        map[string]*structpb.Struct{},
				Plugins:             fixtures.Plugins("plugins1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := config.NewMemoryLoader(tt.config)
			require.Nil(t, err)

			got, err := NewLocalDataSource(loader)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				assert.Same(t, loader, got.loader)
				assert.EqualExportedValues(t, tt.wantConfig, got.config)
			}
		})
	}
}

func TestLocalDataSource_Validate(t *testing.T) {
	lds, _ := buildLocalDataSource(t, config.NewConfig())

	err := lds.Validate(context.Background())
	require.Nil(t, err)
}

func TestLocalDataSource_AddTrustZone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		config        *config.Config
		trustZone     *trust_zone_proto.TrustZone
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "success",
			config:    config.NewConfig(),
			trustZone: fixtures.TrustZone("tz1"),
			wantErr:   false,
		},
		{
			name: "duplicate",
			config: &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			},
			trustZone:     fixtures.TrustZone("tz1"),
			wantErr:       true,
			wantErrString: "trust zone tz1 already exists in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lds, loader := buildLocalDataSource(t, tt.config)

			tt.trustZone.Id = nil

			got, err := lds.AddTrustZone(tt.trustZone)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				tt.trustZone.Id = got.Id
				assert.EqualExportedValues(t, tt.trustZone, got)
				assert.False(t, slices.Contains(lds.config.TrustZones, tt.trustZone), "Pointer to trust zone stored in config")
				assert.False(t, slices.Contains(lds.config.TrustZones, got), "Pointer to trust zone in config returned")
				// Check that the trust zone was persisted.
				gotConfig := readConfig(t, loader)
				gotTrustZone, ok := gotConfig.GetTrustZoneByName(tt.trustZone.Name)
				assert.True(t, ok)
				assert.EqualExportedValues(t, tt.trustZone, gotTrustZone)
				assert.NotNil(t, gotTrustZone.Id)
			}
		})
	}
}

func TestLocalDataSource_DestroyTrustZone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		trustZoneID   string
		wantErr       bool
		wantErrString string
	}{
		{
			name:        "success",
			trustZoneID: "tz1-id",
			wantErr:     false,
		},
		{
			name:          "invalid trust zone",
			trustZoneID:   "invalid-tz",
			wantErr:       true,
			wantErrString: "failed to find trust zone invalid-tz in local config",
		},
		{
			name:          "cluster exists in trust zone",
			trustZoneID:   "tz2-id",
			wantErr:       true,
			wantErrString: "one or more clusters exist in trust zone tz2-id in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
				},
				Clusters: []*clusterpb.Cluster{
					fixtures.Cluster("local2"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
					fixtures.AttestationPolicy("ap2"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, loader := buildLocalDataSource(t, cfg)
			err := lds.DestroyTrustZone(tt.trustZoneID)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrString)
				assert.Len(t, lds.config.TrustZones, 2)
			} else {
				require.Nil(t, err)
				assert.Len(t, lds.config.TrustZones, 1)
				// nolint:staticcheck
				assert.Len(t, lds.config.TrustZones[0].Federations, 0)
				// Check that the trust zone removal was persisted.
				gotConfig := readConfig(t, loader)
				assert.Len(t, gotConfig.TrustZones, 1)
				// nolint:staticcheck
				assert.Len(t, gotConfig.TrustZones[0].Federations, 0)
			}
		})
	}
}

func TestLocalDataSource_GetTrustZone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		trustZoneID   string
		wantErr       bool
		wantErrString string
	}{
		{
			name:        "success",
			trustZoneID: "tz1-id",
			wantErr:     false,
		},
		{
			name:          "non-existent",
			trustZoneID:   "tz2-id",
			wantErr:       true,
			wantErrString: "failed to find trust zone tz2-id in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, _ := buildLocalDataSource(t, cfg)

			got, err := lds.GetTrustZone(tt.trustZoneID)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, cfg.TrustZones[0], got)
				assert.False(t, slices.Contains(lds.config.TrustZones, got), "Pointer to trust zone in config returned")
			}
		})
	}
}

func TestLocalDataSource_GetTrustZoneByName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		trustZone     string
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "success",
			trustZone: "tz1",
			wantErr:   false,
		},
		{
			name:          "non-existent",
			trustZone:     "tz2",
			wantErr:       true,
			wantErrString: "failed to find trust zone tz2 in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, _ := buildLocalDataSource(t, cfg)

			got, err := lds.GetTrustZoneByName(tt.trustZone)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, cfg.TrustZones[0], got)
				assert.False(t, slices.Contains(lds.config.TrustZones, got), "Pointer to trust zone in config returned")
			}
		})
	}
}

func TestLocalDataSource_UpdateTrustZone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		trustZone     *trust_zone_proto.TrustZone
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "no changes",
			trustZone: fixtures.TrustZone("tz1"),
			wantErr:   false,
		},
		{
			name: "allowed changes",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.Bundle = &spiretypes.Bundle{}
				tz.BundleEndpointUrl = fixtures.StringPtr("http://new.bundle")
				return tz
			}(),
			wantErr: false,
		},
		{
			name:          "non-existent",
			trustZone:     &trust_zone_proto.TrustZone{Id: fixtures.StringPtr("tz2-id"), Name: "tz2"},
			wantErr:       true,
			wantErrString: "failed to find trust zone tz2-id in local config",
		},
		{
			name: "disallowed trust domain",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.TrustDomain = "new.domain"
				return tz
			}(),
			wantErr:       true,
			wantErrString: "cannot update trust domain for existing trust zone tz1",
		},
		{
			name: "disallowed attestation policy",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				// nolint:staticcheck
				tz.Federations = []*federation_proto.Federation{
					{TrustZoneId: fixtures.StringPtr("tz1-id"), RemoteTrustZoneId: fixtures.StringPtr("tz3-id")},
				}
				return tz
			}(),
			wantErr:       true,
			wantErrString: "cannot update federations for existing trust zone tz1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
				},
				Clusters: []*clusterpb.Cluster{
					fixtures.Cluster("local1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, loader := buildLocalDataSource(t, cfg)

			trustZone, err := lds.UpdateTrustZone(tt.trustZone)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, tt.trustZone, trustZone)
				assert.EqualExportedValues(t, tt.trustZone, lds.config.TrustZones[0])
				assert.False(t, slices.Contains(lds.config.TrustZones, tt.trustZone), "Pointer to trust zone stored in config")
				assert.False(t, slices.Contains(lds.config.TrustZones, trustZone), "Pointer to trust zone in config returned")
				// Check that the trust zone was persisted.
				gotConfig := readConfig(t, loader)
				gotTrustZone, ok := gotConfig.GetTrustZoneByName(tt.trustZone.Name)
				assert.True(t, ok)
				assert.EqualExportedValues(t, tt.trustZone, gotTrustZone)
			}
		})
	}
}

func TestLocalDataSource_ListTrustZones(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name:    "none",
			config:  config.NewConfig(),
			wantErr: false,
		},
		{
			name: "two",
			config: &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lds, _ := buildLocalDataSource(t, tt.config)
			got, err := lds.ListTrustZones()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				want := tt.config.TrustZones
				if diff := cmp.Diff(got, want, protocmp.Transform()); diff != "" {
					t.Errorf("LocalDataSource.ListTrustZones() mismatch (-want,+got):\n%s", diff)
				}
				for _, gotTrustZone := range got {
					assert.False(t, slices.Contains(lds.config.TrustZones, gotTrustZone), "Pointer to trust zone in config returned")
				}
			}
		})
	}
}

func TestLocalDataSource_AddCluster(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		config        *config.Config
		cluster       *clusterpb.Cluster
		wantErr       bool
		wantErrString string
	}{
		{
			name:    "success",
			config:  config.NewConfig(),
			cluster: fixtures.Cluster("local1"),
			wantErr: false,
		},
		{
			name: "one cluster per trust zone",
			config: &config.Config{
				Clusters: []*clusterpb.Cluster{
					fixtures.Cluster("local1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			},
			cluster: func() *clusterpb.Cluster {
				cluster := fixtures.Cluster("local1")
				name := "local2"
				cluster.Name = &name
				return cluster
			}(),
			wantErr:       true,
			wantErrString: "trust zone tz1-id already has a cluster",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lds, loader := buildLocalDataSource(t, tt.config)

			tt.cluster.Id = nil
			got, err := lds.AddCluster(tt.cluster)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				tt.cluster.Id = got.Id
				assert.EqualExportedValues(t, tt.cluster, got)
				assert.False(t, slices.Contains(lds.config.Clusters, tt.cluster), "Pointer to cluster stored in config")
				assert.False(t, slices.Contains(lds.config.Clusters, got), "Pointer to cluster in config returned")
				// Check that the trust zone was persisted.
				gotConfig := readConfig(t, loader)
				gotCluster, ok := gotConfig.GetClusterByID(tt.cluster.GetId())
				assert.True(t, ok)
				assert.EqualExportedValues(t, tt.cluster, gotCluster)
				assert.NotNil(t, gotCluster.Id)
			}
		})
	}
}

func TestLocalDataSource_DestroyCluster(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		clusterID     string
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "success",
			clusterID: "local1-id",
			wantErr:   false,
		},
		{
			name:          "invalid cluster",
			clusterID:     "invalid-cluster",
			wantErr:       true,
			wantErrString: "failed to find cluster invalid-cluster in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
				},
				Clusters: []*clusterpb.Cluster{
					fixtures.Cluster("local1"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, loader := buildLocalDataSource(t, cfg)
			err := lds.DestroyCluster(tt.clusterID)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrString)
				assert.Len(t, lds.config.Clusters, 1)
			} else {
				require.Nil(t, err)
				assert.Len(t, lds.config.Clusters, 0)
				// Check that the trust zone removal was persisted.
				gotConfig := readConfig(t, loader)
				assert.Len(t, gotConfig.Clusters, 0)
			}
		})
	}
}

func TestLocalDataSource_GetCluster(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		clusterID     string
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "success",
			clusterID: "local1-id",
			wantErr:   false,
		},
		{
			name:          "non-existent",
			clusterID:     "local2-id",
			wantErr:       true,
			wantErrString: "failed to find cluster local2-id in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Clusters: []*clusterpb.Cluster{
					fixtures.Cluster("local1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, _ := buildLocalDataSource(t, cfg)

			got, err := lds.GetCluster(tt.clusterID)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, cfg.Clusters[0], got)
				assert.False(t, slices.Contains(lds.config.Clusters, got), "Pointer to cluster in config returned")
			}
		})
	}
}

func TestLocalDataSource_GetClusterByName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		cluster       string
		trustZoneID   string
		wantErr       bool
		wantErrString string
	}{
		{
			name:        "success",
			cluster:     "local1",
			trustZoneID: "tz1-id",
			wantErr:     false,
		},
		{
			name:          "non-existent",
			cluster:       "local2",
			trustZoneID:   "tz2-id",
			wantErr:       true,
			wantErrString: "failed to find cluster local2 in trust zone tz2-id in local config",
		},
		{
			name:          "wrong trust zone",
			cluster:       "local1",
			trustZoneID:   "tz2-id",
			wantErr:       true,
			wantErrString: "failed to find cluster local1 in trust zone tz2-id in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Clusters: []*clusterpb.Cluster{
					fixtures.Cluster("local1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, _ := buildLocalDataSource(t, cfg)

			got, err := lds.GetClusterByName(tt.cluster, tt.trustZoneID)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, cfg.Clusters[0], got)
				assert.False(t, slices.Contains(lds.config.Clusters, got), "Pointer to cluster in config returned")
			}
		})
	}
}

func TestLocalDataSource_ListClusters(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		config       *config.Config
		filter       *datasourcepb.ListClustersRequest_Filter
		wantClusters []*clusterpb.Cluster
		wantErr      bool
	}{
		{
			name:         "none",
			config:       config.NewConfig(),
			wantClusters: []*clusterpb.Cluster{},
			wantErr:      false,
		},
		{
			name: "two",
			config: &config.Config{
				Clusters: []*clusterpb.Cluster{
					fixtures.Cluster("local1"),
					fixtures.Cluster("local2"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			},
			wantClusters: []*clusterpb.Cluster{
				fixtures.Cluster("local1"),
				fixtures.Cluster("local2"),
			},
			wantErr: false,
		},
		{
			name: "scoped to trust zone",
			config: &config.Config{
				Clusters: []*clusterpb.Cluster{
					fixtures.Cluster("local1"),
					fixtures.Cluster("local2"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			},
			filter: &datasourcepb.ListClustersRequest_Filter{
				TrustZoneId: fixtures.StringPtr("tz1-id"),
			},
			wantClusters: []*clusterpb.Cluster{
				fixtures.Cluster("local1"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lds, _ := buildLocalDataSource(t, tt.config)
			got, err := lds.ListClusters(tt.filter)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				if diff := cmp.Diff(got, tt.wantClusters, protocmp.Transform()); diff != "" {
					t.Errorf("LocalDataSource.ListClusters() mismatch (-want,+got):\n%s", diff)
				}
				for _, gotCluster := range got {
					assert.False(t, slices.Contains(lds.config.Clusters, gotCluster), "Pointer to cluster in config returned")
				}
			}
		})
	}
}

func TestLocalDataSource_UpdateCluster(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		cluster       *clusterpb.Cluster
		wantErr       bool
		wantErrString string
	}{
		{
			name:    "no changes",
			cluster: fixtures.Cluster("local1"),
			wantErr: false,
		},
		{
			name: "allowed changes",
			cluster: func() *clusterpb.Cluster {
				cluster := fixtures.Cluster("local1")
				cluster.KubernetesContext = fixtures.StringPtr("new-context")
				cluster.ExtraHelmValues = nil
				return cluster
			}(),
			wantErr: false,
		},
		{
			name:          "non-existent",
			cluster:       fixtures.Cluster("local2"),
			wantErr:       true,
			wantErrString: "failed to find cluster local2-id in trust zone tz2-id in local config",
		},
		{
			name: "disallowed trust zone",
			cluster: func() *clusterpb.Cluster {
				cluster := fixtures.Cluster("local1")
				cluster.TrustZoneId = fixtures.StringPtr("tz2-id")
				return cluster
			}(),
			wantErr:       true,
			wantErrString: "cannot update trust zone for existing cluster local1-id in trust zone tz1-id",
		},
		{
			name: "disallowed nil trust provider",
			cluster: func() *clusterpb.Cluster {
				cluster := fixtures.Cluster("local1")
				cluster.TrustProvider = nil
				return cluster
			}(),
			wantErr:       true,
			wantErrString: "cannot remove trust provider for cluster local1-id in trust zone tz1-id",
		},
		{
			name: "disallowed trust provider kind",
			cluster: func() *clusterpb.Cluster {
				cluster := fixtures.Cluster("local1")
				cluster.TrustProvider.Kind = fixtures.StringPtr("invalid")
				return cluster
			}(),
			wantErr:       true,
			wantErrString: "cannot update trust provider kind for existing cluster local1-id in trust zone tz1-id",
		},
		{
			name: "disallowed profile",
			cluster: func() *clusterpb.Cluster {
				cluster := fixtures.Cluster("local1")
				cluster.Profile = fixtures.StringPtr("istio")
				return cluster
			}(),
			wantErr:       true,
			wantErrString: "cannot update profile for existing cluster local1-id in trust zone tz1-id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
				},
				Clusters: []*clusterpb.Cluster{
					fixtures.Cluster("local1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, loader := buildLocalDataSource(t, cfg)

			cluster, err := lds.UpdateCluster(tt.cluster)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, tt.cluster, cluster)
				assert.EqualExportedValues(t, tt.cluster, lds.config.Clusters[0])
				assert.False(t, slices.Contains(lds.config.Clusters, tt.cluster), "Pointer to cluster stored in config")
				assert.False(t, slices.Contains(lds.config.Clusters, cluster), "Pointer to cluster in config returned")
				// Check that the cluster was persisted.
				gotConfig := readConfig(t, loader)
				gotCluster, ok := gotConfig.GetClusterByID(tt.cluster.GetId())
				assert.True(t, ok)
				assert.EqualExportedValues(t, tt.cluster, gotCluster)
			}
		})
	}
}

func TestLocalDataSource_AddAttestationPolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		config        *config.Config
		policy        *attestation_policy_proto.AttestationPolicy
		wantErr       bool
		wantErrString string
	}{
		{
			name:    "success",
			config:  config.NewConfig(),
			policy:  fixtures.AttestationPolicy("ap1"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lds, loader := buildLocalDataSource(t, tt.config)

			tt.policy.Id = nil
			got, err := lds.AddAttestationPolicy(tt.policy)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				tt.policy.Id = got.Id
				assert.EqualExportedValues(t, tt.policy, got)
				assert.False(t, slices.Contains(lds.config.AttestationPolicies, tt.policy), "Pointer to attestation policy stored in config")
				assert.False(t, slices.Contains(lds.config.AttestationPolicies, got), "Pointer to attestation policy in config returned")
				// Check that the policy was persisted.
				gotConfig := readConfig(t, loader)
				gotPolicy, ok := gotConfig.GetAttestationPolicyByName(tt.policy.Name)
				assert.True(t, ok)
				assert.EqualExportedValues(t, tt.policy, gotPolicy)
				assert.NotNil(t, gotPolicy.Id)
			}
		})
	}
}

func TestLocalDataSource_DestroyAttestationPolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		policyID      string
		wantErr       bool
		wantErrString string
	}{
		{
			name:     "success",
			policyID: "ap1-id",
			wantErr:  false,
		},
		{
			name:          "invalid policy",
			policyID:      "invalid-ap",
			wantErr:       true,
			wantErrString: "failed to find attestation policy invalid-ap in local config",
		},
		{
			name:          "bound to trust zone",
			policyID:      "ap2-id",
			wantErr:       true,
			wantErrString: "attestation policy ap2-id is bound to trust zone tz2-id in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz2"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
					fixtures.AttestationPolicy("ap2"),
				},
				ApBindings: []*ap_binding_proto.APBinding{
					fixtures.APBinding("apb2"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, loader := buildLocalDataSource(t, cfg)
			err := lds.DestroyAttestationPolicy(tt.policyID)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrString)
				assert.Len(t, lds.config.AttestationPolicies, 2)
			} else {
				require.Nil(t, err)
				assert.Len(t, lds.config.AttestationPolicies, 1)
				// Check that the attestation policy removal was persisted.
				gotConfig := readConfig(t, loader)
				assert.Len(t, gotConfig.AttestationPolicies, 1)
			}
		})
	}
}

func TestLocalDataSource_GetAttestationPolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		policyID      string
		wantErr       bool
		wantErrString string
	}{
		{
			name:     "success",
			policyID: "ap1-id",
			wantErr:  false,
		},
		{
			name:          "non-existent",
			policyID:      "ap2-id",
			wantErr:       true,
			wantErrString: "failed to find attestation policy ap2-id in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, _ := buildLocalDataSource(t, cfg)

			got, err := lds.GetAttestationPolicy(tt.policyID)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, cfg.AttestationPolicies[0], got)
				assert.False(t, slices.Contains(lds.config.AttestationPolicies, got), "Pointer to attestation policy in config returned")
			}
		})
	}
}

func TestLocalDataSource_GetAttestationPolicyByName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		policy        string
		wantErr       bool
		wantErrString string
	}{
		{
			name:    "success",
			policy:  "ap1",
			wantErr: false,
		},
		{
			name:          "non-existent",
			policy:        "ap2",
			wantErr:       true,
			wantErrString: "failed to find attestation policy ap2 in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, _ := buildLocalDataSource(t, cfg)

			got, err := lds.GetAttestationPolicyByName(tt.policy)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, cfg.AttestationPolicies[0], got)
				assert.False(t, slices.Contains(lds.config.AttestationPolicies, got), "Pointer to attestation policy in config returned")
			}
		})
	}
}

func TestLocalDataSource_ListAttestationPolicies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name:    "none",
			config:  config.NewConfig(),
			wantErr: false,
		},
		{
			name: "two",
			config: &config.Config{
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
					fixtures.AttestationPolicy("ap2"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lds, _ := buildLocalDataSource(t, tt.config)
			got, err := lds.ListAttestationPolicies()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				want := tt.config.AttestationPolicies
				if diff := cmp.Diff(got, want, protocmp.Transform()); diff != "" {
					t.Errorf("LocalDataSource.ListAttestationPolicies() mismatch (-want,+got):\n%s", diff)
				}
				for _, gotPolicy := range got {
					assert.False(t, slices.Contains(lds.config.AttestationPolicies, gotPolicy), "Pointer to attestation policy in config returned")
				}
			}
		})
	}
}

func TestLocalDataSource_AddAPBinding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		binding       *ap_binding_proto.APBinding
		wantErr       bool
		wantErrString string
	}{
		{
			name: "success",
			binding: &ap_binding_proto.APBinding{
				TrustZoneId: fixtures.StringPtr("tz1-id"),
				PolicyId:    fixtures.StringPtr("ap2-id"),
			},
			wantErr: false,
		},
		{
			name: "success with federations",
			binding: &ap_binding_proto.APBinding{
				TrustZoneId: fixtures.StringPtr("tz1-id"),
				PolicyId:    fixtures.StringPtr("ap2-id"),
				Federations: []*ap_binding_proto.APBindingFederation{{TrustZoneId: fixtures.StringPtr("tz2-id")}},
			},
			wantErr: false,
		},
		{
			name: "invalid trust zone",
			binding: &ap_binding_proto.APBinding{
				TrustZoneId: fixtures.StringPtr("invalid"),
				PolicyId:    fixtures.StringPtr("ap2"),
			},
			wantErr:       true,
			wantErrString: "failed to find trust zone invalid in local config",
		},
		{
			name: "invalid policy",
			binding: &ap_binding_proto.APBinding{
				TrustZoneId: fixtures.StringPtr("tz1-id"),
				PolicyId:    fixtures.StringPtr("invalid"),
			},
			wantErr:       true,
			wantErrString: "failed to find attestation policy invalid in local config",
		},
		{
			name: "federates with self",
			binding: &ap_binding_proto.APBinding{
				TrustZoneId: fixtures.StringPtr("tz1-id"),
				PolicyId:    fixtures.StringPtr("ap2-id"),
				Federations: []*ap_binding_proto.APBindingFederation{{TrustZoneId: fixtures.StringPtr("tz1-id")}},
			},
			wantErr:       true,
			wantErrString: "attestation policy ap2-id federates with its own trust zone tz1-id",
		},
		{
			name: "federates with invalid tz",
			binding: &ap_binding_proto.APBinding{
				TrustZoneId: fixtures.StringPtr("tz1-id"),
				PolicyId:    fixtures.StringPtr("ap2-id"),
				Federations: []*ap_binding_proto.APBindingFederation{{TrustZoneId: fixtures.StringPtr("invalid")}},
			},
			wantErr:       true,
			wantErrString: "attestation policy ap2-id federates with unknown trust zone invalid",
		},
		{
			name: "federates with unfederated tz",
			binding: &ap_binding_proto.APBinding{
				TrustZoneId: fixtures.StringPtr("tz1-id"),
				PolicyId:    fixtures.StringPtr("ap2-id"),
				Federations: []*ap_binding_proto.APBindingFederation{{TrustZoneId: fixtures.StringPtr("tz3-id")}},
			},
			wantErr:       true,
			wantErrString: "attestation policy ap2-id federates with tz3-id but trust zone tz1-id does not",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
					fixtures.TrustZone("tz3"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
					fixtures.AttestationPolicy("ap2"),
				},
				ApBindings: []*ap_binding_proto.APBinding{
					fixtures.APBinding("apb1"),
					fixtures.APBinding("apb2"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, loader := buildLocalDataSource(t, cfg)

			tt.binding.Id = nil
			got, err := lds.AddAPBinding(tt.binding)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				tt.binding.Id = got.Id
				assert.EqualExportedValues(t, tt.binding, got)
				assert.False(t, slices.Contains(lds.config.ApBindings, tt.binding), "Pointer to attestation policy binding stored in config")
				assert.False(t, slices.Contains(lds.config.ApBindings, got), "Pointer to attestation policy binding in config returned")
				// Check that the binding was persisted.
				gotConfig := readConfig(t, loader)
				gotBinding := gotConfig.ApBindings[2]
				assert.EqualExportedValues(t, tt.binding, gotBinding)
				assert.NotNil(t, gotBinding.Id)
			}
		})
	}
}

func TestLocalDataSource_DestroyAPBinding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		bindingID     string
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "success",
			bindingID: "apb1-id",
			wantErr:   false,
		},
		{
			name:          "invalid binding",
			bindingID:     "invalid-binding",
			wantErr:       true,
			wantErrString: "failed to find attestation policy binding invalid-binding in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
				},
				ApBindings: []*ap_binding_proto.APBinding{
					fixtures.APBinding("apb1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, loader := buildLocalDataSource(t, cfg)
			err := lds.DestroyAPBinding(tt.bindingID)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				for _, binding := range lds.config.ApBindings {
					assert.NotEqual(t, tt.bindingID, binding.Id)
				}

				// Check that the binding removal was persisted.
				gotConfig := readConfig(t, loader)
				for _, binding := range gotConfig.ApBindings {
					assert.NotEqual(t, tt.bindingID, binding.Id)
				}
			}
		})
	}
}

func TestLocalDataSource_ListAPBindings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		filter        *datasourcepb.ListAPBindingsRequest_Filter
		want          []*ap_binding_proto.APBinding
		wantErr       bool
		wantErrString string
	}{
		{
			name:    "no filter",
			filter:  &datasourcepb.ListAPBindingsRequest_Filter{},
			want:    []*ap_binding_proto.APBinding{fixtures.APBinding("apb1")},
			wantErr: false,
		},
		{
			name: "filter by trust zone tz1",
			filter: &datasourcepb.ListAPBindingsRequest_Filter{
				TrustZoneId: fixtures.StringPtr("tz1-id"),
			},
			want:    []*ap_binding_proto.APBinding{fixtures.APBinding("apb1")},
			wantErr: false,
		},
		{
			name: "filter by trust zone tz3",
			filter: &datasourcepb.ListAPBindingsRequest_Filter{
				TrustZoneId: fixtures.StringPtr("tz3-id"),
			},
			want:    []*ap_binding_proto.APBinding{},
			wantErr: false,
		},
		{
			name: "filter by policy ap1",
			filter: &datasourcepb.ListAPBindingsRequest_Filter{
				PolicyId: fixtures.StringPtr("ap1-id"),
			},
			want:    []*ap_binding_proto.APBinding{fixtures.APBinding("apb1")},
			wantErr: false,
		},
		{
			name: "filter by trust zone and policy",
			filter: &datasourcepb.ListAPBindingsRequest_Filter{
				TrustZoneId: fixtures.StringPtr("tz1-id"),
				PolicyId:    fixtures.StringPtr("ap1-id"),
			},
			want:    []*ap_binding_proto.APBinding{fixtures.APBinding("apb1")},
			wantErr: false,
		},
		{
			name: "invalid trust zone",
			filter: &datasourcepb.ListAPBindingsRequest_Filter{
				TrustZoneId: fixtures.StringPtr("invalid"),
			},
			want:          []*ap_binding_proto.APBinding{},
			wantErr:       true,
			wantErrString: "failed to find trust zone invalid in local config",
		},
		{
			name: "invalid policy",
			filter: &datasourcepb.ListAPBindingsRequest_Filter{
				PolicyId: fixtures.StringPtr("invalid"),
			},
			want:    []*ap_binding_proto.APBinding{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz3"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
				},
				ApBindings: []*ap_binding_proto.APBinding{
					fixtures.APBinding("apb1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, _ := buildLocalDataSource(t, cfg)
			got, err := lds.ListAPBindings(tt.filter)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.NoError(t, err)
				assert.EqualExportedValues(t, tt.want, got)
				for _, gotBinding := range got {
					assert.False(t, slices.Contains(lds.config.ApBindings, gotBinding), "Pointer to attestation policy binding in config returned")
				}
			}
		})
	}
}

func TestLocalDataSource_AddFederation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		federation    *federation_proto.Federation
		wantErr       bool
		wantErrString string
	}{
		{
			name: "success",
			federation: &federation_proto.Federation{
				TrustZoneId:       fixtures.StringPtr("tz1-id"),
				RemoteTrustZoneId: fixtures.StringPtr("tz3-id"),
			},
			wantErr: false,
		},
		{
			name: "invalid from trust zone",
			federation: &federation_proto.Federation{
				TrustZoneId:       fixtures.StringPtr("invalid"),
				RemoteTrustZoneId: fixtures.StringPtr("tz2-id"),
			},
			wantErr:       true,
			wantErrString: "failed to find trust zone invalid in local config",
		},
		{
			name: "invalid to trust zone",
			federation: &federation_proto.Federation{
				TrustZoneId:       fixtures.StringPtr("tz1-id"),
				RemoteTrustZoneId: fixtures.StringPtr("invalid"),
			},
			wantErr:       true,
			wantErrString: "failed to find trust zone invalid in local config",
		},
		{
			name: "federate with self",
			federation: &federation_proto.Federation{
				TrustZoneId:       fixtures.StringPtr("tz1-id"),
				RemoteTrustZoneId: fixtures.StringPtr("tz1-id"),
			},
			wantErr:       true,
			wantErrString: "cannot federate trust zone tz1-id with itself",
		},
		{
			name: "duplicate",
			federation: &federation_proto.Federation{
				TrustZoneId:       fixtures.StringPtr("tz1-id"),
				RemoteTrustZoneId: fixtures.StringPtr("tz2-id"),
			},
			wantErr:       true,
			wantErrString: "federation already exists between tz1-id and tz2-id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
					fixtures.TrustZone("tz3"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, loader := buildLocalDataSource(t, cfg)
			got, err := lds.AddFederation(tt.federation)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				tt.federation.Id = got.Id
				assert.EqualExportedValues(t, tt.federation, got)
				// nolint:staticcheck
				assert.False(t, slices.Contains(lds.config.TrustZones[0].Federations, tt.federation), "Pointer to federation stored in config")
				// nolint:staticcheck
				assert.False(t, slices.Contains(lds.config.TrustZones[0].Federations, got), "Pointer to federation in config returned")
				// Check that the federation was persisted.
				gotConfig := readConfig(t, loader)
				// nolint:staticcheck
				gotFederation := gotConfig.TrustZones[0].Federations[1]
				assert.EqualExportedValues(t, tt.federation, gotFederation)
				assert.NotNil(t, gotFederation.Id)
			}
		})
	}
}

func TestLocalDataSource_DestroyFederation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		federationID  string
		wantErr       bool
		wantErrString string
	}{
		{
			name:         "success",
			federationID: "fed1-id",
			wantErr:      false,
		},
		{
			name:          "invalid federation",
			federationID:  "invalid-federation",
			wantErr:       true,
			wantErrString: "failed to find federation invalid-federation in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			}
			lds, loader := buildLocalDataSource(t, cfg)
			err := lds.DestroyFederation(tt.federationID)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrString)
				// nolint:staticcheck
				assert.Len(t, lds.config.TrustZones[0].Federations, 1)
			} else {
				require.Nil(t, err)
				// nolint:staticcheck
				assert.Len(t, lds.config.TrustZones[0].Federations, 0)
				// Check that the trust zone removal was persisted.
				gotConfig := readConfig(t, loader)
				// nolint:staticcheck
				assert.Len(t, gotConfig.TrustZones[0].Federations, 0)
			}
		})
	}
}

func TestLocalDataSource_ListFederations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  *config.Config
		filter  *datasourcepb.ListFederationsRequest_Filter
		wantErr bool
		want    []*federation_proto.Federation
	}{
		{
			name:    "none",
			config:  config.NewConfig(),
			wantErr: false,
			want:    []*federation_proto.Federation{},
		},
		{
			name: "two",
			config: &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			},
			wantErr: false,
			// nolint:staticcheck
			want: append(fixtures.TrustZone("tz1").Federations, fixtures.TrustZone("tz2").Federations...),
		},
		{
			name: "with trust zone filter",
			config: &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			},
			filter: &datasourcepb.ListFederationsRequest_Filter{
				TrustZoneId: fixtures.StringPtr("tz1-id"),
			},
			wantErr: false,
			// nolint:staticcheck
			want: fixtures.TrustZone("tz1").Federations,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lds, _ := buildLocalDataSource(t, tt.config)
			got, err := lds.ListFederations(tt.filter)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				if diff := cmp.Diff(got, tt.want, protocmp.Transform()); diff != "" {
					t.Errorf("LocalDataSource.ListFederations() mismatch (-want,+got):\n%s", diff)
				}
				for _, gotFederation := range got {
					for _, tz := range tt.config.TrustZones {
						// nolint:staticcheck
						assert.False(t, slices.Contains(tz.Federations, gotFederation), "Pointer to federation in config returned")
					}
				}
			}
		})
	}
}

func buildLocalDataSource(t *testing.T, cfg *config.Config) (*LocalDataSource, *config.MemoryLoader) {
	loader, err := config.NewMemoryLoader(cfg)
	require.Nil(t, err)

	lds, err := NewLocalDataSource(loader)
	require.Nil(t, err)
	return lds, loader
}

func readConfig(t *testing.T, loader config.Loader) *config.Config {
	config, err := loader.Read()
	require.Nil(t, err)
	return config
}
