// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Values = map[string]interface{}
type ValueList = []interface{}

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
					"federation": Values{
						"enabled": true,
					},
					"fullnameOverride": "spire-server",
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPsat": Values{
							"allowedNodeLabelKeys": ValueList{},
							"allowedPodLabelKeys":  ValueList{},
							"audience": ValueList{
								"spire-server",
							},
							"enabled": true,
							"serviceAccountAllowList": ValueList{
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
									"federatesWith": ValueList{
										"td2",
									},
									"namespaceSelector": Values{
										"matchExpressions": ValueList{},
										"matchLabels": Values{
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
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPsat": Values{
							"allowedNodeLabelKeys": ValueList{},
							"allowedPodLabelKeys":  ValueList{},
							"audience": ValueList{
								"spire-server",
							},
							"enabled": true,
							"serviceAccountAllowList": ValueList{
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
			}
			got, err := g.GenerateValues()
			require.Nil(t, err)
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
			}
			_, err := g.GenerateValues()
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErrString)
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
