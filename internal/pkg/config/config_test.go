// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestConfig_YAMLMarshall(t *testing.T) {
	// Ensure that the YAML representation of Config is as expected.
	tests := []struct {
		name     string
		config   *Config
		wantFile string
	}{
		{
			name:     "default",
			config:   &Config{},
			wantFile: "default.yaml",
		},
		{
			name: "full",
			config: &Config{
				Plugins: []string{"test-plugin"},
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
					fixtures.AttestationPolicy("ap2"),
					fixtures.AttestationPolicy("ap3"),
				},
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
				Plugins:             []string{},
				TrustZones:          []*trust_zone_proto.TrustZone{},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
			},
		},
		{
			name: "full",
			file: "full.yaml",
			want: &Config{
				Plugins: []string{"test-plugin"},
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
					fixtures.AttestationPolicy("ap2"),
					fixtures.AttestationPolicy("ap3"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlConfig := readTestConfig(t, tt.file)
			got, err := unmarshalYAML(yamlConfig)
			if err != nil {
				t.Fatalf("error unmarshalling configuration from YAML: %v", err)
			}
			if diff := cmp.Diff(got, tt.want, protocmp.Transform()); diff != "" {
				t.Errorf("yaml.Unmarshall() mismatch (-want,+got):\n%s", diff)
			}
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
			if diff := cmp.Diff(tt.wantTz, gotTz, protocmp.Transform()); diff != "" {
				t.Errorf("Config.GetTrustZoneByName() mismatch (-want,+got):\n%s", diff)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Config.GetTrustZoneByName() got1 = %v, want %v", gotOk, tt.wantOk)
			}
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
			if diff := cmp.Diff(tt.wantAp, gotAp, protocmp.Transform()); diff != "" {
				t.Errorf("Config.GetAttestationPolicyByName() mismatch (-want,+got):\n%s", diff)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Config.GetAttestationPolicyByName() got1 = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}
