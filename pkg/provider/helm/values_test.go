// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/cofide/cofidectl/internal/pkg/trustprovider"
	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Values = map[string]any
type ValueList = []any

func TestHelmValuesGenerator_GenerateValues_success(t *testing.T) {
	tests := []struct {
		name      string
		trustZone *trust_zone_proto.TrustZone
		want      Values
	}{
		{
			name: "tz1 no binding or federation",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.AttestationPolicies = nil
				tz.Bundle = nil
				tz.BundleEndpointUrl = nil
				tz.Federations = nil
				tz.JwtIssuer = nil
				tz.ExtraHelmValues = nil
				return tz
			}(),
			want: Values{
				"global": Values{
					"deleteHooks": Values{
						"enabled": false,
					},
					"installAndUpgradeHooks": Values{
						"enabled": false,
					},
					"spire": Values{
						"clusterName": "local1",
						"recommendations": Values{
							"create": true,
						},
						"trustDomain": "td1",
					},
				},
				"spiffe-csi-driver": Values{
					"fullnameOverride": "spiffe-csi-driver",
				},
				"spiffe-oidc-discovery-provider": Values{
					"enabled": false,
				},
				"spire-agent": Values{
					"fullnameOverride": "spire-agent",
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPsat": Values{
							"enabled": true,
						},
					},
					"server": Values{
						"address": "spire-server.spire",
					},
					"workloadAttestors": Values{
						"k8s": Values{
							"disableContainerSelectors":   false,
							"enabled":                     true,
							"skipKubeletVerification":     true,
							"useNewContainerLocator":      false,
							"verboseContainerLocatorLogs": false,
						},
					},
				},
				"spire-server": Values{
					"caKeyType": "rsa-2048",
					"caTTL":     "12h",
					"controllerManager": Values{
						"enabled": true,
						"identities": Values{
							"clusterSPIFFEIDs": Values{
								"default": Values{
									"enabled": true,
								},
							},
						},
					},
					"fullnameOverride": "spire-server",
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPsat": Values{
							"allowedNodeLabelKeys": []string{},
							"allowedPodLabelKeys":  []string{},
							"audience": []string{
								"spire-server",
							},
							"enabled": true,
							"serviceAccountAllowList": []string{
								"spire:spire-agent",
							},
						},
					},
					"service": Values{
						"type": "LoadBalancer",
					},
				},
			},
		},
		{
			name:      "tz1",
			trustZone: fixtures.TrustZone("tz1"),
			want: Values{
				"global": Values{
					"deleteHooks": Values{
						"enabled": false,
					},
					"installAndUpgradeHooks": Values{
						"enabled": false,
					},
					"spire": Values{
						"clusterName": "local1",
						"jwtIssuer":   "https://tz1.example.com",
						"namespaces": Values{
							"create": true,
						},
						"recommendations": Values{
							"create": true,
						},
						"trustDomain": "td1",
					},
				},
				"spiffe-csi-driver": Values{
					"fullnameOverride": "spiffe-csi-driver",
				},
				"spiffe-oidc-discovery-provider": Values{
					"enabled": false,
				},
				"spire-agent": Values{
					"fullnameOverride": "spire-agent",
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPsat": Values{
							"enabled": true,
						},
					},
					"server": Values{
						"address": "spire-server.spire",
					},
					"workloadAttestors": Values{
						"k8s": Values{
							"disableContainerSelectors":   false,
							"enabled":                     true,
							"skipKubeletVerification":     true,
							"useNewContainerLocator":      false,
							"verboseContainerLocatorLogs": false,
						},
					},
				},
				"spire-server": Values{
					"caKeyType": "rsa-2048",
					"caTTL":     "12h",
					"controllerManager": Values{
						"enabled": true,
						"identities": Values{
							"clusterFederatedTrustDomains": Values{
								"tz2": Values{
									"bundleEndpointProfile": Values{
										"endpointSPIFFEID": "spiffe://td2/spire/server",
										"type":             "https_spiffe",
									},
									"bundleEndpointURL": "127.0.0.2",
									"trustDomain":       "td2",
									"trustDomainBundle": "",
								},
							},
							"clusterSPIFFEIDs": Values{
								"ap1": Values{
									"federatesWith": []string{
										"td2",
									},
									"namespaceSelector": Values{
										"matchExpressions": []map[string]any{},
										"matchLabels": map[string]string{
											"kubernetes.io/metadata.name": "ns1",
										},
									},
								},
								"default": Values{
									"enabled": false,
								},
							},
						},
					},
					"federation": Values{
						"enabled": true,
					},
					"fullnameOverride": "spire-server",
					"logLevel":         "INFO",
					"nodeAttestor": Values{
						"k8sPsat": Values{
							"allowedNodeLabelKeys": []string{},
							"allowedPodLabelKeys":  []string{},
							"audience": []string{
								"spire-server",
							},
							"enabled": true,
							"serviceAccountAllowList": []string{
								"spire:spire-agent",
							},
						},
					},
					"service": Values{
						"type": "LoadBalancer",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			source := newFakeDataSource(t, cfg)
			g := &HelmValuesGenerator{
				source:    source,
				trustZone: tt.trustZone,
				values:    nil,
			}

			got, err := g.GenerateValues()
			require.Nil(t, err, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHelmValuesGenerator_GenerateValues_AdditionalValues(t *testing.T) {
	tests := []struct {
		name      string
		trustZone *trust_zone_proto.TrustZone
		values    Values
		want      Values
	}{
		{
			name: "tz1 no binding or federation",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.AttestationPolicies = nil
				tz.Bundle = nil
				tz.BundleEndpointUrl = nil
				tz.Federations = nil
				tz.JwtIssuer = nil
				tz.ExtraHelmValues = nil
				return tz
			}(),
			values: Values{
				"spire-server": Values{
					"controllerManager": Values{
						"identities": Values{
							"clusterFederatedTrustDomains": Values{
								"cofide": Values{
									"bundleEndpointProfile": Values{
										"type": "https_web",
									},
									"bundleEndpointURL": "https://td1/connect/bundle",
									"trustDomain":       "td1",
								},
							},
						},
					},
				},
			},
			want: Values{
				"global": Values{
					"deleteHooks": Values{
						"enabled": false,
					},
					"installAndUpgradeHooks": Values{
						"enabled": false,
					},
					"spire": Values{
						"clusterName": "local1",
						"recommendations": Values{
							"create": true,
						},
						"trustDomain": "td1",
					},
				},
				"spiffe-csi-driver": Values{
					"fullnameOverride": "spiffe-csi-driver",
				},
				"spiffe-oidc-discovery-provider": Values{
					"enabled": false,
				},
				"spire-agent": Values{
					"fullnameOverride": "spire-agent",
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPsat": Values{
							"enabled": true,
						},
					},
					"server": Values{
						"address": "spire-server.spire",
					},
					"workloadAttestors": Values{
						"k8s": Values{
							"disableContainerSelectors":   false,
							"enabled":                     true,
							"skipKubeletVerification":     true,
							"useNewContainerLocator":      false,
							"verboseContainerLocatorLogs": false,
						},
					},
				},
				"spire-server": Values{
					"caKeyType": "rsa-2048",
					"caTTL":     "12h",
					"controllerManager": Values{
						"enabled": true,
						"identities": Values{
							"clusterSPIFFEIDs": Values{
								"default": Values{
									"enabled": true,
								},
							},
							"clusterFederatedTrustDomains": Values{
								"cofide": Values{
									"bundleEndpointProfile": Values{
										"type": "https_web",
									},
									"bundleEndpointURL": "https://td1/connect/bundle",
									"trustDomain":       "td1",
								},
							},
						},
					},
					"fullnameOverride": "spire-server",
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPsat": Values{
							"allowedNodeLabelKeys": []string{},
							"allowedPodLabelKeys":  []string{},
							"audience": []string{
								"spire-server",
							},
							"enabled": true,
							"serviceAccountAllowList": []string{
								"spire:spire-agent",
							},
						},
					},
					"service": Values{
						"type": "LoadBalancer",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			source := newFakeDataSource(t, cfg)
			g := &HelmValuesGenerator{
				source:    source,
				trustZone: tt.trustZone,
				values:    tt.values,
			}

			got, err := g.GenerateValues()
			require.Nil(t, err, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHelmValuesGenerator_GenerateValues_failure(t *testing.T) {
	tests := []struct {
		name          string
		trustZone     *trust_zone_proto.TrustZone
		wantErrString string
	}{
		{
			name: "no trust provider",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.TrustProvider = nil
				return tz
			}(),
			wantErrString: "no trust provider for trust zone tz1",
		},
		{
			name: "invalid trust provider kind",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.TrustProvider.Kind = fixtures.StringPtr("invalid-tp")
				return tz
			}(),
			wantErrString: "an unknown trust provider profile was specified: invalid-tp",
		},
		{
			name: "unknown attestation policy",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.AttestationPolicies[0].Policy = "invalid-ap"
				return tz
			}(),
			wantErrString: "failed to find attestation policy invalid-ap in local config",
		},
		{
			name: "unknown federated trust zone",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.Federations[0].To = "invalid-tz"
				return tz
			}(),
			wantErrString: "failed to find trust zone invalid-tz in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			source := newFakeDataSource(t, cfg)
			g := &HelmValuesGenerator{
				source:    source,
				trustZone: tt.trustZone,
				values:    nil,
			}
			_, err := g.GenerateValues()
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErrString)
		})
	}
}

func TestGetNestedMap(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]any
		key    string
		want   map[string]any
		exists bool
	}{
		{
			name: "map exists, valid key",
			values: map[string]any{
				"spire-server": map[string]any{
					"fullnameOverride": "spire-server",
				},
			},
			key: "spire-server",
			want: map[string]any{
				"fullnameOverride": "spire-server",
			},
			exists: true,
		},
		{
			name: "map doesn't exist, valid key",
			values: map[string]any{
				"spire-server": map[string]any{
					"caKeyType": "rsa-2048",
					"caTTL":     "12h",
				},
			},
			key: "global",
			want: map[string]any{
				"spire": map[string]any{
					"clusterName": "local1",
				},
			},
			exists: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, exists := getNestedMap(tt.values, tt.key)

			assert.Equal(t, tt.exists, exists)
			if tt.exists {
				assert.Equal(t, tt.want, resp)
			}
		})
	}
}

func TestMergeMaps(t *testing.T) {
	tests := []struct {
		name                  string
		src                   map[string]any
		dest                  map[string]any
		overwriteExistingKeys bool
		want                  map[string]any
	}{
		{
			name: "valid src and valid dest, no overwrites",
			src: map[string]any{
				"foo": "bar",
			},
			dest: map[string]any{
				"fizz": "buzz",
			},
			overwriteExistingKeys: false,
			want: map[string]any{
				"foo":  "bar",
				"fizz": "buzz",
			},
		},
		{
			name: "valid src and empty dest, no overwrites",
			src: map[string]any{
				"foo": "bar",
			},
			dest:                  map[string]any{},
			overwriteExistingKeys: false,
			want: map[string]any{
				"foo": "bar",
			},
		},
		{
			name: "empty src and valid dest, no overwrites",
			src:  map[string]any{},
			dest: map[string]any{
				"fizz": "buzz",
			},
			overwriteExistingKeys: false,
			want: map[string]any{
				"fizz": "buzz",
			},
		},
		{
			name: "valid src and valid dest, nested, no overwrites",
			src: map[string]any{
				"global": map[string]any{
					"spire": map[string]any{
						"clusterName": "local1-new",
					},
				},
			},
			dest: map[string]any{
				"global": map[string]any{
					"spire": map[string]any{
						"clusterName": "local1",
					},
					"trustDomain": "td1",
				},
			},
			overwriteExistingKeys: false,
			want: map[string]any{
				"global": map[string]any{
					"spire": map[string]any{
						"clusterName": "local1",
					},
					"trustDomain": "td1",
				},
			},
		},
		{
			name: "valid src and valid dest, with overwrites",
			src: map[string]any{
				"foo":   "bar",
				"hello": "world",
			},
			dest: map[string]any{
				"foo": "baz",
			},
			overwriteExistingKeys: true,
			want: map[string]any{
				"foo":   "bar",
				"hello": "world",
			},
		},
		{
			name: "valid src and valid dest, nested, with overwrites",
			src: map[string]any{
				"global": map[string]any{
					"spire": map[string]any{
						"clusterName": "local1-new",
					},
				},
			},
			dest: map[string]any{
				"global": map[string]any{
					"spire": map[string]any{
						"clusterName": "local1-old",
					},
					"trustDomain": "td1",
				},
			},
			overwriteExistingKeys: true,
			want: map[string]any{
				"global": map[string]any{
					"spire": map[string]any{
						"clusterName": "local1-new",
					},
					"trustDomain": "td1",
				},
			},
		},
		{
			name: "valid src and valid dest, additional nesting, with overwrites",
			src: map[string]any{
				"spire-server": map[string]any{
					"caKeyType": "rsa-2048",
					"controllerManager": map[string]any{
						"enabled": true,
						"identities": map[string]any{
							"clusterSPIFFEIDs": map[string]any{
								"default": Values{
									"enabled": false,
								},
							},
							"clusterFederatedTrustDomains": map[string]any{
								"cofide": map[string]any{
									"bundleEndpointProfile": map[string]any{
										"type": "https_web",
									},
									"bundleEndpointURL": "https://td1/connect/bundle",
									"trustDomain":       "td1",
								},
							},
						},
					},
				},
			},
			dest: map[string]any{
				"spire-server": map[string]any{
					"caKeyType": "rsa-2048",
					"controllerManager": map[string]any{
						"enabled": true,
						"identities": map[string]any{
							"clusterSPIFFEIDs": map[string]any{
								"default": Values{
									"enabled": true,
								},
							},
						},
					},
				},
			},
			overwriteExistingKeys: true,
			want: map[string]any{
				"spire-server": map[string]any{
					"caKeyType": "rsa-2048",
					"controllerManager": map[string]any{
						"enabled": true,
						"identities": map[string]any{
							"clusterSPIFFEIDs": map[string]any{
								"default": Values{
									"enabled": false,
								},
							},
							"clusterFederatedTrustDomains": map[string]any{
								"cofide": map[string]any{
									"bundleEndpointProfile": map[string]any{
										"type": "https_web",
									},
									"bundleEndpointURL": "https://td1/connect/bundle",
									"trustDomain":       "td1",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := mergeMaps(tt.src, tt.dest, tt.overwriteExistingKeys)
			assert.Equal(t, tt.want, resp)
		})
	}
}

func TestFlattenMaps(t *testing.T) {
	tests := []struct {
		name string
		maps []map[string]any
		want map[string]any
	}{
		{
			name: "valid slice of maps",
			maps: []map[string]any{
				map[string]any{
					"foo": "bar",
				},
				map[string]any{
					"fizz": "buzz",
				},
			},
			want: map[string]any{
				"foo":  "bar",
				"fizz": "buzz",
			},
		},
		{
			name: "empty slice of maps",
			maps: []map[string]any{},
			want: map[string]any{},
		},
		{
			name: "slice of empty maps",
			maps: []map[string]any{
				map[string]any{},
				map[string]any{},
			},
			want: map[string]any{},
		},
		{
			name: "nil maps value",
			maps: nil,
			want: map[string]any{},
		},
		{
			name: "nil map value in maps",
			maps: []map[string]any{nil},
			want: map[string]any{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := flattenMaps(tt.maps)
			assert.Equal(t, tt.want, resp)
		})
	}
}

func TestGlobalValues_GenerateValues(t *testing.T) {
	tests := []struct {
		name      string
		input     globalValues
		want      map[string]any
		wantErr   bool
		errString string
	}{
		{
			name: "valid global values",
			input: globalValues{
				spireClusterName: "local1",
				spireTrustDomain: "td1",
			},
			want: map[string]any{
				"global": map[string]any{
					"spire": map[string]any{
						"clusterName": "local1",
						"recommendations": map[string]any{
							"create": false,
						},
						"trustDomain": "td1",
					},
					"installAndUpgradeHooks": map[string]any{
						"enabled": false,
					},
					"deleteHooks": map[string]any{
						"enabled": false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid global values, missing spireTrustDomain value",
			input: globalValues{
				spireClusterName: "local1",
			},
			want:      nil,
			wantErr:   true,
			errString: "spireTrustDomain value is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tt.input.generateValues()
			if tt.wantErr {
				assert.Equal(t, tt.errString, err.Error())
				return
			}

			assert.Equal(t, tt.want, resp)
		})
	}
}

func TestSpireAgentValues_GenerateValues(t *testing.T) {
	tests := []struct {
		name      string
		input     spireAgentValues
		want      map[string]any
		wantErr   bool
		errString string
	}{
		{
			name: "valid SPIRE agent values",
			input: spireAgentValues{
				fullnameOverride: "spire-agent",
				logLevel:         "DEBUG",
				agentConfig: trustprovider.TrustProviderAgentConfig{
					WorkloadAttestor:        "k8s",
					WorkloadAttestorEnabled: true,
					WorkloadAttestorConfig: map[string]any{
						"enabled":                     true,
						"skipKubeletVerification":     true,
						"disableContainerSelectors":   false,
						"useNewContainerLocator":      false,
						"verboseContainerLocatorLogs": false,
					},
					NodeAttestor:        "k8sPsat",
					NodeAttestorEnabled: true,
				},
				spireServerAddress: "spire-server.spire",
			},
			want: map[string]any{
				"spire-agent": map[string]any{
					"fullnameOverride": "spire-agent",
					"logLevel":         "DEBUG",
					"nodeAttestor": map[string]any{
						"k8sPsat": map[string]any{
							"enabled": true,
						},
					},
					"server": map[string]any{
						"address": "spire-server.spire",
					},
					"workloadAttestors": map[string]any{
						"k8s": map[string]any{
							"enabled":                     true,
							"skipKubeletVerification":     true,
							"disableContainerSelectors":   false,
							"useNewContainerLocator":      false,
							"verboseContainerLocatorLogs": false,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid SPIRE agent values, missing logLevel value",
			input: spireAgentValues{
				fullnameOverride: "spire-agent",
				agentConfig: trustprovider.TrustProviderAgentConfig{
					WorkloadAttestor:        "k8s",
					WorkloadAttestorEnabled: true,
					WorkloadAttestorConfig: map[string]any{
						"enabled":                     true,
						"skipKubeletVerification":     true,
						"disableContainerSelectors":   false,
						"useNewContainerLocator":      false,
						"verboseContainerLocatorLogs": false,
					},
					NodeAttestor:        "k8sPsat",
					NodeAttestorEnabled: true,
				},
				spireServerAddress: "spire-server.spire",
			},
			want:      nil,
			wantErr:   true,
			errString: "logLevel value is empty",
		},
		{
			name: "invalid SPIRE agent values, empty WorkloadAttestorConfig value",
			input: spireAgentValues{
				fullnameOverride: "spire-agent",
				logLevel:         "DEBUG",
				agentConfig: trustprovider.TrustProviderAgentConfig{
					WorkloadAttestor:        "k8s",
					WorkloadAttestorEnabled: true,
					WorkloadAttestorConfig:  map[string]any{},
					NodeAttestor:            "k8sPsat",
					NodeAttestorEnabled:     true,
				},
				spireServerAddress: "spire-server.spire",
			},
			want:      nil,
			wantErr:   true,
			errString: "agentConfig.WorkloadAttestorConfig value is empty",
		},
		{
			name: "invalid SPIRE agent values, empty WorkloadAttestor value",
			input: spireAgentValues{
				fullnameOverride: "spire-agent",
				logLevel:         "DEBUG",
				agentConfig: trustprovider.TrustProviderAgentConfig{
					WorkloadAttestor:        "",
					WorkloadAttestorEnabled: true,
					WorkloadAttestorConfig: map[string]any{
						"enabled":                     true,
						"skipKubeletVerification":     true,
						"disableContainerSelectors":   false,
						"useNewContainerLocator":      false,
						"verboseContainerLocatorLogs": false,
					},
					NodeAttestor:        "k8sPsat",
					NodeAttestorEnabled: true,
				},
				spireServerAddress: "spire-server.spire",
			},
			want:      nil,
			wantErr:   true,
			errString: "agentConfig.WorkloadAttestor value is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tt.input.generateValues()
			if tt.wantErr {
				assert.Equal(t, tt.errString, err.Error())
				return
			}

			assert.Equal(t, tt.want, resp)
		})
	}
}

func TestSpireServerValues_GenerateValues(t *testing.T) {
	tests := []struct {
		name      string
		input     spireServerValues
		want      map[string]any
		wantErr   bool
		errString string
	}{
		{
			name: "valid SPIRE server values",
			input: spireServerValues{
				caKeyType:                "rsa-2048",
				caTTL:                    "12h",
				controllerManagerEnabled: true,
				fullnameOverride:         "spire-server",
				logLevel:                 "DEBUG",
				serverConfig: trustprovider.TrustProviderServerConfig{
					NodeAttestor:        "k8sPsat",
					NodeAttestorEnabled: true,
					NodeAttestorConfig: map[string]any{
						"enabled":                 true,
						"serviceAccountAllowList": []string{"spire:spire-agent"},
						"audience":                []string{"spire-server"},
						"allowedNodeLabelKeys":    []string{},
						"allowedPodLabelKeys":     []string{},
					},
				},
				serviceType: "LoadBalancer",
			},
			want: map[string]any{
				"spire-server": map[string]any{
					"caKeyType": "rsa-2048",
					"caTTL":     "12h",
					"controllerManager": map[string]any{
						"enabled": true,
					},
					"fullnameOverride": "spire-server",
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPsat": Values{
							"allowedNodeLabelKeys": []string{},
							"allowedPodLabelKeys":  []string{},
							"audience": []string{
								"spire-server",
							},
							"enabled": true,
							"serviceAccountAllowList": []string{
								"spire:spire-agent",
							},
						},
					},
					"service": map[string]any{
						"type": "LoadBalancer",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid SPIRE server values, empty serviceType value",
			input: spireServerValues{
				caKeyType:                "rsa-2048",
				caTTL:                    "12h",
				controllerManagerEnabled: true,
				fullnameOverride:         "spire-server",
				logLevel:                 "DEBUG",
				serverConfig: trustprovider.TrustProviderServerConfig{
					NodeAttestor:        "k8sPsat",
					NodeAttestorEnabled: true,
					NodeAttestorConfig: map[string]any{
						"enabled":                 true,
						"serviceAccountAllowList": []string{"spire:spire-agent"},
						"audience":                []string{"spire-server"},
						"allowedNodeLabelKeys":    []string{},
						"allowedPodLabelKeys":     []string{},
					},
				},
				serviceType: "",
			},
			want:      nil,
			wantErr:   true,
			errString: "serviceType value is empty",
		},
		{
			name: "invalid SPIRE server values, empty NodeAttestorConfig value",
			input: spireServerValues{
				caKeyType:                "rsa-2048",
				caTTL:                    "12h",
				controllerManagerEnabled: true,
				fullnameOverride:         "spire-server",
				logLevel:                 "DEBUG",
				serverConfig: trustprovider.TrustProviderServerConfig{
					NodeAttestor:        "k8sPsat",
					NodeAttestorEnabled: true,
					NodeAttestorConfig:  map[string]any{},
				},
				serviceType: "",
			},
			want:      nil,
			wantErr:   true,
			errString: "serverConfig.NodeAttestorConfig value is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tt.input.generateValues()
			if tt.wantErr {
				assert.Equal(t, tt.errString, err.Error())
				return
			}

			assert.Equal(t, tt.want, resp)
		})
	}
}

func TestSpiffeOIDCDiscoveryProviderValues_GenerateValues(t *testing.T) {
	tests := []struct {
		name      string
		input     spiffeOIDCDiscoveryProviderValues
		want      map[string]any
		wantErr   bool
		errString string
	}{
		{
			name:  "valid SPIFFE OIDC discovery provider values",
			input: spiffeOIDCDiscoveryProviderValues{enabled: true},
			want: map[string]any{
				"spiffe-oidc-discovery-provider": map[string]any{
					"enabled": true,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tt.input.generateValues()
			if tt.wantErr {
				assert.Equal(t, tt.errString, err.Error())
				return
			}

			assert.Equal(t, tt.want, resp)
		})
	}
}

func TestSpiffeCSIDriverValues_GenerateValues(t *testing.T) {
	tests := []struct {
		name      string
		input     spiffeCSIDriverValues
		want      map[string]any
		wantErr   bool
		errString string
	}{
		{
			name:  "valid SPIFFE CSI driver values",
			input: spiffeCSIDriverValues{fullnameOverride: "spiffe-csi-driver"},
			want: map[string]any{
				"spiffe-csi-driver": map[string]any{
					"fullnameOverride": "spiffe-csi-driver",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tt.input.generateValues()
			if tt.wantErr {
				assert.Equal(t, tt.errString, err.Error())
				return
			}

			assert.Equal(t, tt.want, resp)
		})
	}
}

func newFakeDataSource(t *testing.T, cfg *config.Config) plugin.DataSource {
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
			fixtures.TrustZone("tz2"),
		},
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
			fixtures.AttestationPolicy("ap1"),
			fixtures.AttestationPolicy("ap2"),
		},
	}
}
