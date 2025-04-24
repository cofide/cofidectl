// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	pluginspb "github.com/cofide/cofide-api-sdk/gen/go/proto/plugins/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConfig_YAMLMarshall(t *testing.T) {
	// Ensure that the YAML representation of Config is as expected.
	tests := []struct {
		name     string
		config   *Config
		wantFile string
	}{
		{
			name: "default",
			config: &Config{
				Plugins: &pluginspb.Plugins{},
			},
			wantFile: "default.yaml",
		},
		{
			name: "full",
			config: &Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
					fixtures.TrustZone("tz6"),
				},
				Clusters: []*clusterpb.Cluster{
					fixtures.Cluster("local1"),
					fixtures.Cluster("local2"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
					fixtures.AttestationPolicy("ap2"),
					fixtures.AttestationPolicy("ap3"),
					fixtures.AttestationPolicy("ap4"),
				},
				PluginConfig: map[string]*structpb.Struct{
					"plugin1": fixtures.PluginConfig("plugin1"),
					"plugin2": fixtures.PluginConfig("plugin2"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			},
			wantFile: "full.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.marshalYAML()
			if err != nil {
				t.Fatalf("error marshalling configuration to YAML: %v", err)
			}
			want := readTestConfig(t, tt.wantFile)
			assert.Equal(t, string(want), string(got))
		})
	}
}

func TestConfig_YAMLUnmarshall(t *testing.T) {
	// Ensure that the YAML representation of Config is as expected.
	tests := []struct {
		name string
		file string
		want *Config
	}{
		{
			name: "default",
			file: "default.yaml",
			want: &Config{
				TrustZones:          []*trust_zone_proto.TrustZone{},
				Clusters:            []*clusterpb.Cluster{},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
				PluginConfig:        map[string]*structpb.Struct{},
				Plugins:             &pluginspb.Plugins{},
			},
		},
		{
			name: "full",
			file: "full.yaml",
			want: &Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
					fixtures.TrustZone("tz6"),
				},
				Clusters: []*clusterpb.Cluster{
					fixtures.Cluster("local1"),
					fixtures.Cluster("local2"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
					fixtures.AttestationPolicy("ap2"),
					fixtures.AttestationPolicy("ap3"),
					fixtures.AttestationPolicy("ap4"),
				},
				PluginConfig: map[string]*structpb.Struct{
					"plugin1": fixtures.PluginConfig("plugin1"),
					"plugin2": fixtures.PluginConfig("plugin2"),
				},
				Plugins: fixtures.Plugins("plugins1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlConfig := readTestConfig(t, tt.file)
			got, err := unmarshalYAML(yamlConfig)
			require.NoError(t, err, err)
			assert.EqualExportedValues(t, tt.want, got)
		})
	}
}

func TestConfig_GetTrustZoneByName(t *testing.T) {
	tests := []struct {
		name       string
		trustZones []*trust_zone_proto.TrustZone
		trustZone  string
		wantTz     *trust_zone_proto.TrustZone
		wantOk     bool
	}{
		{
			name: "found",
			trustZones: []*trust_zone_proto.TrustZone{
				fixtures.TrustZone("tz1"),
				fixtures.TrustZone("tz2"),
			},
			trustZone: "tz2",
			wantTz:    fixtures.TrustZone("tz2"),
			wantOk:    true,
		},
		{
			name:       "not found",
			trustZones: []*trust_zone_proto.TrustZone{},
			trustZone:  "tz1",
			wantTz:     nil,
			wantOk:     false,
		},
		{
			name:       "nil list",
			trustZones: nil,
			trustZone:  "tz1",
			wantTz:     nil,
			wantOk:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				TrustZones: tt.trustZones,
			}
			gotTz, gotOk := c.GetTrustZoneByName(tt.trustZone)
			assert.EqualExportedValues(t, tt.wantTz, gotTz)
			assert.Equal(t, tt.wantOk, gotOk)
		})
	}
}

func TestConfig_GetClusterByName(t *testing.T) {
	tests := []struct {
		name        string
		clusters    []*clusterpb.Cluster
		cluster     string
		trustZone   string
		wantCluster *clusterpb.Cluster
		wantOk      bool
	}{
		{
			name: "found",
			clusters: []*clusterpb.Cluster{
				fixtures.Cluster("local1"),
				fixtures.Cluster("local2"),
			},
			cluster:     "local2",
			trustZone:   "tz2-id",
			wantCluster: fixtures.Cluster("local2"),
			wantOk:      true,
		},
		{
			name:        "not found",
			clusters:    []*clusterpb.Cluster{},
			cluster:     "local1",
			trustZone:   "tz1-id",
			wantCluster: nil,
			wantOk:      false,
		},
		{
			name: "trust zone scoped",
			clusters: []*clusterpb.Cluster{
				fixtures.Cluster("local1"),
			},
			cluster:     "local1",
			trustZone:   "tz2-id",
			wantCluster: nil,
			wantOk:      false,
		},
		{
			name:        "nil list",
			clusters:    nil,
			cluster:     "local1",
			trustZone:   "tz1-id",
			wantCluster: nil,
			wantOk:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Clusters: tt.clusters,
			}
			gotCluster, gotOk := c.GetClusterByName(tt.cluster, tt.trustZone)
			assert.EqualExportedValues(t, tt.wantCluster, gotCluster)
			assert.Equal(t, tt.wantOk, gotOk)
		})
	}
}

func TestConfig_GetClustersByTrustZone(t *testing.T) {
	tests := []struct {
		name         string
		clusters     []*clusterpb.Cluster
		trustZone    string
		wantClusters []*clusterpb.Cluster
	}{
		{
			name: "found",
			clusters: []*clusterpb.Cluster{
				fixtures.Cluster("local1"),
				fixtures.Cluster("local2"),
			},
			trustZone:    "tz2-id",
			wantClusters: []*clusterpb.Cluster{fixtures.Cluster("local2")},
		},
		{
			name: "found multiple",
			clusters: []*clusterpb.Cluster{
				fixtures.Cluster("local1"),
				fixtures.Cluster("local1"),
			},
			trustZone: "tz1-id",
			wantClusters: []*clusterpb.Cluster{
				fixtures.Cluster("local1"),
				fixtures.Cluster("local1"),
			},
		},
		{
			name:         "not found",
			clusters:     []*clusterpb.Cluster{},
			trustZone:    "tz1-id",
			wantClusters: []*clusterpb.Cluster{},
		},
		{
			name: "trust zone scoped",
			clusters: []*clusterpb.Cluster{
				fixtures.Cluster("local1"),
			},
			trustZone:    "tz2-id",
			wantClusters: []*clusterpb.Cluster{},
		},
		{
			name:         "nil list",
			clusters:     nil,
			trustZone:    "tz1-id",
			wantClusters: []*clusterpb.Cluster{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Clusters: tt.clusters,
			}
			gotClusters := c.GetClustersByTrustZone(tt.trustZone)
			assert.EqualExportedValues(t, tt.wantClusters, gotClusters)
		})
	}
}

func TestConfig_GetAttestationPolicyByName(t *testing.T) {
	tests := []struct {
		name     string
		policies []*attestation_policy_proto.AttestationPolicy
		policy   string
		wantAp   *attestation_policy_proto.AttestationPolicy
		wantOk   bool
	}{
		{
			name: "found",
			policies: []*attestation_policy_proto.AttestationPolicy{
				fixtures.AttestationPolicy("ap1"),
				fixtures.AttestationPolicy("ap2"),
			},
			policy: "ap2",
			wantAp: fixtures.AttestationPolicy("ap2"),
			wantOk: true,
		},
		{
			name:     "not found",
			policies: []*attestation_policy_proto.AttestationPolicy{},
			policy:   "ap1",
			wantAp:   nil,
			wantOk:   false,
		},
		{
			name:     "nil list",
			policies: nil,
			policy:   "ap1",
			wantAp:   nil,
			wantOk:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				AttestationPolicies: tt.policies,
			}
			gotAp, gotOk := c.GetAttestationPolicyByName(tt.policy)
			assert.EqualExportedValues(t, tt.wantAp, gotAp)
			assert.Equal(t, tt.wantOk, gotOk)
		})
	}
}
