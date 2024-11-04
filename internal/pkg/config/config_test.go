package config

import (
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v3"
)

func TestConfig_YAMLMarshall(t *testing.T) {
	// Ensure that the YAML representation of Config is as expected.
	tests := []struct {
		name     string
		config   Config
		wantFile string
	}{
		{
			name:     "default",
			config:   Config{},
			wantFile: "default.yaml",
		},
		{
			name: "full",
			config: Config{
				Plugins: []string{"test-plugin"},
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
					fixtures.AttestationPolicy("ap2"),
				},
			},
			wantFile: "full.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(&tt.config)
			if err != nil {
				t.Fatalf("error marshalling configuration to YAML: %v", err)
			}
			want := readTestConfig(t, tt.wantFile)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("yaml.Marshall(config) mismatch (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestConfig_YAMLUnmarshall(t *testing.T) {
	// Ensure that the YAML representation of Config is as expected.
	tests := []struct {
		name string
		file string
		want Config
	}{
		{
			name: "default",
			file: "default.yaml",
			want: Config{
				TrustZones:          []*trust_zone_proto.TrustZone{},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
			},
		},
		{
			name: "full",
			file: "full.yaml",
			want: Config{
				Plugins: []string{"test-plugin"},
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
					fixtures.AttestationPolicy("ap2"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Config
			yamlConfig := readTestConfig(t, tt.file)
			err := yaml.Unmarshal([]byte(yamlConfig), &got)
			if err != nil {
				t.Fatalf("error unmarshalling configuration from YAML: %v", err)
			}
			if diff := cmp.Diff(&got, &tt.want, protocmp.Transform()); diff != "" {
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