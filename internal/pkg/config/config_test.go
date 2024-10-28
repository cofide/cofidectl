package config

import (
	"reflect"
	"slices"
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/cofide/cofidectl/internal/pkg/proto"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"gopkg.in/yaml.v3"
)

var emptyYAMLConfig string = `trustzones: []
attestationpolicies: []
`

var fullYAMLConfig string = `plugins:
    - test-plugin
trustzones:
    - name: tz1
      trustdomain: td1
      kubernetescluster: local1
      kubernetescontext: kind-local1
      trustprovider:
        name: ""
        kind: kubernetes
      bundleendpointurl: 127.0.0.1
      bundle: ""
      federations:
        - left: tz1
          right: tz2
      attestationpolicies:
        - name: ap1
          kind: 2
          podkey: ""
          podvalue: ""
          namespace: ns1
    - name: tz2
      trustdomain: td2
      kubernetescluster: local2
      kubernetescontext: kind-local2
      trustprovider:
        name: ""
        kind: kubernetes
      bundleendpointurl: 127.0.0.2
      bundle: ""
      federations:
        - left: tz2
          right: tz1
      attestationpolicies:
        - name: ap2
          kind: 1
          podkey: foo
          podvalue: bar
          namespace: ""
attestationpolicies:
    - name: ap1
      kind: 2
      podkey: ""
      podvalue: ""
      namespace: ns1
    - name: ap2
      kind: 1
      podkey: foo
      podvalue: bar
      namespace: ""
`

func TestConfig_YAMLMarshall(t *testing.T) {
	// Ensure that the YAML representation of Config is as expected.
	tests := []struct {
		name   string
		config Config
		want   string
	}{
		{
			name:   "empty",
			config: Config{},
			want:   emptyYAMLConfig,
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
			want: fullYAMLConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(&tt.config)
			if err != nil {
				t.Fatalf("error marshalling configuration to YAML: %v", err)
			}
			gotStr := string(got)
			if !reflect.DeepEqual(gotStr, tt.want) {
				t.Errorf("yaml.Marshall(config) = %v, want %v", gotStr, tt.want)
			}
		})
	}
}

func TestConfig_YAMLUnmarshall(t *testing.T) {
	// Ensure that the YAML representation of Config is as expected.
	tests := []struct {
		name string
		yaml string
		want Config
	}{
		{
			name: "empty",
			yaml: emptyYAMLConfig,
			want: Config{
				TrustZones:          []*trust_zone_proto.TrustZone{},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
			},
		},
		{
			name: "full",
			yaml: fullYAMLConfig,
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
			err := yaml.Unmarshal([]byte(tt.yaml), &got)
			if err != nil {
				t.Fatalf("error unmarshalling configuration from YAML: %v", err)
			}
			if !configsEqual(&got, &tt.want) {
				t.Errorf("yaml.Unmarshall() = %v, want %v", got, tt.want)
			}
		})
	}
}

// configsEqual compares two `Config`s for equality.
// `reflect.DeepEqual` may see differences in the protobuf internals, so we need to use proto.Equal to compare messages.
func configsEqual(c1, c2 *Config) bool {
	if !slices.Equal(c1.Plugins, c2.Plugins) {
		return false
	}
	if !slices.EqualFunc(c1.TrustZones, c2.TrustZones, proto.TrustZonesEqual) {
		return false
	}
	if !slices.EqualFunc(c1.AttestationPolicies, c2.AttestationPolicies, proto.AttestationPoliciesEqual) {
		return false
	}
	return true
}
