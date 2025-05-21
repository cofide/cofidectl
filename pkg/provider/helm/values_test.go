// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/cofide/cofidectl/internal/pkg/trustprovider"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Values = map[string]any

func TestHelmValuesGenerator_GenerateValues_success(t *testing.T) {
	tests := []struct {
		name       string
		trustZone  *trust_zone_proto.TrustZone
		cluster    *clusterpb.Cluster
		configFunc func(*config.Config)
		want       Values
	}{
		{
			name: "tz1 no binding or federation",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.Bundle = nil
				tz.BundleEndpointUrl = nil
				tz.JwtIssuer = nil
				return tz
			}(),
			cluster: func() *clusterpb.Cluster {
				cluster := fixtures.Cluster("local1")
				cluster.ExtraHelmValues = nil
				return cluster
			}(),
			configFunc: func(cfg *config.Config) {
				trustZone, ok := cfg.GetTrustZoneByName("tz1")
				require.True(t, ok)
				// nolint:staticcheck
				trustZone.AttestationPolicies = nil
				// nolint:staticcheck
				trustZone.Federations = nil
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
						"caSubject": Values{
							"commonName":   "cofide.io",
							"country":      "UK",
							"organization": "Cofide",
						},
						"clusterName": "local1",
						"namespaces": Values{
							"create": true,
						},
						"recommendations": Values{
							"enabled": true,
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
						"k8sPSAT": Values{
							"enabled": true,
						},
					},
					"sds": map[string]any{
						"enabled":               true,
						"defaultSVIDName":       "default",
						"defaultBundleName":     "ROOTCA",
						"defaultAllBundlesName": "ALL",
					},
					"workloadAttestors": Values{
						"k8s": Values{
							"disableContainerSelectors": true,
							"enabled":                   true,
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
									"enabled": false,
								},
							},
							"clusterStaticEntries": Values{},
						},
					},
					"enabled":          true,
					"fullnameOverride": "spire-server",
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPSAT": Values{
							"audience": []string{"spire-server"},
							"enabled":  true,
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
			cluster:   fixtures.Cluster("local1"),
			want: Values{
				"global": Values{
					"deleteHooks": Values{
						"enabled": false,
					},
					"installAndUpgradeHooks": Values{
						"enabled": false,
					},
					"spire": Values{
						"caSubject": Values{
							"country":      "UK",
							"organization": "acme-org",
							"commonName":   "cn.example.com",
						},
						"clusterName": "local1",
						"jwtIssuer":   "https://tz1.example.com",
						"namespaces": Values{
							"create": true,
						},
						"recommendations": Values{
							"enabled": true,
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
						"k8sPSAT": Values{
							"enabled": true,
						},
					},
					"sds": map[string]any{
						"enabled":               true,
						"defaultSVIDName":       "default",
						"defaultBundleName":     "ROOTCA",
						"defaultAllBundlesName": "ALL",
					},
					"workloadAttestors": Values{
						"k8s": Values{
							"disableContainerSelectors": true,
							"enabled":                   true,
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
										"type": "https_web",
									},
									"bundleEndpointURL": "127.0.0.2",
									"trustDomain":       "td2",
								},
							},
							"clusterSPIFFEIDs": Values{
								"ap1": Values{
									"federatesWith": []string{
										"td2",
									},
									"namespaceSelector": Values{
										"matchExpressions": []map[string]any{},
										"matchLabels": map[string]any{
											"kubernetes.io/metadata.name": "ns1",
										},
									},
								},
								"default": Values{
									"enabled": false,
								},
							},
							"clusterStaticEntries": Values{},
						},
					},
					"enabled": true,
					"federation": Values{
						"enabled": true,
					},
					"fullnameOverride": "spire-server",
					"logLevel":         "INFO",
					"nameOverride":     "custom-server-name",
					"nodeAttestor": Values{
						"k8sPSAT": Values{
							"audience": []string{"spire-server"},
							"enabled":  true,
						},
					},
					"service": Values{
						"type": "LoadBalancer",
					},
				},
			},
		},
		{
			name:      "tz2",
			trustZone: fixtures.TrustZone("tz2"),
			cluster:   fixtures.Cluster("local2"),
			want: Values{
				"global": Values{
					"deleteHooks": Values{
						"enabled": false,
					},
					"installAndUpgradeHooks": Values{
						"enabled": false,
					},
					"spire": Values{
						"caSubject": Values{
							"country":      "UK",
							"organization": "Cofide",
							"commonName":   "cofide.io",
						},
						"clusterName": "local2",
						"jwtIssuer":   "https://tz2.example.com",
						"namespaces": Values{
							"create": true,
						},
						"recommendations": Values{
							"enabled": true,
						},
						"trustDomain": "td2",
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
						"k8sPSAT": Values{
							"enabled": true,
						},
					},
					"sds": map[string]any{
						"enabled":               true,
						"defaultSVIDName":       "default",
						"defaultBundleName":     "ROOTCA",
						"defaultAllBundlesName": "ALL",
					},
					"workloadAttestors": Values{
						"k8s": Values{
							"disableContainerSelectors": true,
							"enabled":                   true,
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
								"tz1": Values{
									"bundleEndpointProfile": Values{
										"endpointSPIFFEID": "spiffe://td1/spire/server",
										"type":             "https_spiffe",
									},
									"bundleEndpointURL": "127.0.0.1",
									"trustDomain":       "td1",
									"trustDomainBundle": "{\"keys\":[{\"use\":\"x509-svid\",\"kty\":\"RSA\",\"n\":\"zQiMbwAURHf67uePImpXIdacY2oRaQ5YZeZSYQDHzt-kmvhTlfymJdnA8c9Uh9ysfWMg2kEA64IGKGsBuHABfQea2Q-3csVYxSnjvMdLv6xLLZNF64QdMu2rGAQbuLAEbaxvLRhOi1mdverVeOe7J07NgG4tcKCE2IiRL41F7aSg6_lB5Rkk3sigO4bpsaDpHgLCYcWEt4DZkxj1A79fs-Ej09wvrIO6s7GC3jeqOMY-uRRVVvi_iO7mLPwI5oWnWgZyECr7dOR3Q-X1VyW_y-WUMEMzEyOReMo7U_S56iaV1oElDmc_86mZWOERnlL0Q4pNx8nuqpLfdMAmcGenTQ\",\"e\":\"AQAB\",\"x5c\":[\"MIIDrjCCApagAwIBAgIRAL6Ru792Wi5AhHhh387STRIwDQYJKoZIhvcNAQELBQAwZDELMAkGA1UEBhMCVUsxDzANBgNVBAoTBkNvZmlkZTESMBAGA1UEAxMJY29maWRlLmlvMTAwLgYDVQQFEycyNTMzMTAwMTAyMjM0MjQ3NDE4NDYzOTczNzY0MDQzMTM0OTI3NTQwHhcNMjUwMjA3MTU1ODU1WhcNMjUwMjA4MDM1OTA1WjBkMQswCQYDVQQGEwJVSzEPMA0GA1UEChMGQ29maWRlMRIwEAYDVQQDEwljb2ZpZGUuaW8xMDAuBgNVBAUTJzI1MzMxMDAxMDIyMzQyNDc0MTg0NjM5NzM3NjQwNDMxMzQ5Mjc1NDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAM0IjG8AFER3+u7njyJqVyHWnGNqEWkOWGXmUmEAx87fpJr4U5X8piXZwPHPVIfcrH1jINpBAOuCBihrAbhwAX0HmtkPt3LFWMUp47zHS7+sSy2TReuEHTLtqxgEG7iwBG2sby0YTotZnb3q1XjnuydOzYBuLXCghNiIkS+NRe2koOv5QeUZJN7IoDuG6bGg6R4CwmHFhLeA2ZMY9QO/X7PhI9PcL6yDurOxgt43qjjGPrkUVVb4v4ju5iz8COaFp1oGchAq+3Tkd0Pl9Vclv8vllDBDMxMjkXjKO1P0ueomldaBJQ5nP/OpmVjhEZ5S9EOKTcfJ7qqS33TAJnBnp00CAwEAAaNbMFkwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFGCz3aiUExK4+2cTKGFcJpxBcAexMBcGA1UdEQQQMA6GDHNwaWZmZTovL3RkMjANBgkqhkiG9w0BAQsFAAOCAQEAfhzGZqw3UC+uJGsOLFQ0v7EWS35UB8PvgWABDd+2cRABnSSsNciaszN0Fz9t1qJcP20eldna5b0eZNJLOH89BEqWGTiXD37B3qAqKsT/pAU0eglMtDCNW+KipDpAoo9dFlbF+cSk9dJlH0gNYsMwO1vMFdrRK/4O79sRkxKn2JMf082EXsFpDzPORDsZ1FidOkWT3kTKbH469zFz8a0El7Tq58/2aELkF9qUnP3ZfN6H9CGiES7OV7kNuzuTadVIiFQpeYxd+U/ro6jKeyUdY83FZ6Qfx/bRTRqXStrbutDcdetWWQvRGRCHRoa0uMNmz8fkqLDRkc+emcJGyGSLAQ==\"]},{\"use\":\"jwt-svid\",\"kty\":\"RSA\",\"kid\":\"sHYIGH99d7NhlAVufX9a9e0D9HMPGCQw\",\"n\":\"0mg3S_3z_NlFHhqvd49RibgQpgsWvVBs66pC27AsJIh9UFs5jW17QQJkaBRt_LtA4jhQIQErj3g1ZPyv2JCfLOA-rFHcGFdsnuf8xTgKQfmp4v_xpvUQVmA9rzoFLx5DTDxLe0tU0lgGhJxPJcoSGzAae_Tn_1jenWkIvyPX1W5TMFiIJkpPpqASOUCOnkdwwZ-XeLo-7XWGUAjNtHVsEIOjiIRFkeZCwKSXJvXy9T5OMjCtGsQFaF6-fg5wE0VJBXCDXMr_uPIbVmozGC75opOOPJXcV8daVbEpCKm2BFDcm0MNchNijGGCR0JhYEhb04YSAhN8tmyjxeHHJiblmw\",\"e\":\"AQAB\"}],\"spiffe_sequence\":3,\"spiffe_refresh_hint\":2}",
								},
							},
							"clusterSPIFFEIDs": Values{
								"ap2": Values{
									"federatesWith": []string{
										"td1",
									},
									"podSelector": Values{
										"matchExpressions": []map[string]any{
											{
												"key":      "foo",
												"operator": "In",
												"values":   []string{"bar"},
											},
										},
										"matchLabels": map[string]any{},
									},
								},
								"default": Values{
									"enabled": false,
								},
							},
							"clusterStaticEntries": Values{},
						},
					},
					"enabled": true,
					"federation": Values{
						"enabled": true,
					},
					"fullnameOverride": "spire-server",
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPSAT": Values{
							"audience": []string{"spire-server"},
							"enabled":  true,
						},
					},
					"service": Values{
						"type": "LoadBalancer",
					},
				},
			},
		},
		{
			name:      "tz4 using the istio profile",
			trustZone: fixtures.TrustZone("tz4"),
			cluster:   fixtures.Cluster("local4"),
			want: Values{
				"global": Values{
					"deleteHooks": Values{
						"enabled": false,
					},
					"installAndUpgradeHooks": Values{
						"enabled": false,
					},
					"spire": Values{
						"caSubject": Values{
							"commonName":   "cofide.io",
							"country":      "UK",
							"organization": "Cofide",
						},
						"clusterName": "local4",
						"namespaces": Values{
							"create": true,
						},
						"recommendations": Values{
							"enabled": true,
						},
						"trustDomain": "td4",
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
						"k8sPSAT": Values{
							"enabled": true,
						},
					},
					"sds": map[string]any{
						"enabled":               true,
						"defaultSVIDName":       "default",
						"defaultBundleName":     "null",
						"defaultAllBundlesName": "ROOTCA",
					},
					"workloadAttestors": Values{
						"k8s": Values{
							"disableContainerSelectors": true,
							"enabled":                   true,
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
									"enabled": false,
								},
							},
							"clusterStaticEntries": Values{},
						},
					},
					"enabled":          true,
					"fullnameOverride": "spire-server",
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPSAT": Values{
							"audience": []string{"spire-server"},
							"enabled":  true,
						},
					},
					"service": Values{
						"type": "LoadBalancer",
					},
				},
			},
		},
		{
			name:      "tz6",
			trustZone: fixtures.TrustZone("tz6"),
			cluster:   fixtures.Cluster("local6"),
			want: Values{
				"global": Values{
					"deleteHooks": Values{
						"enabled": false,
					},
					"installAndUpgradeHooks": Values{
						"enabled": false,
					},
					"spire": Values{
						"caSubject": Values{
							"commonName":   "cofide.io",
							"country":      "UK",
							"organization": "Cofide",
						},
						"clusterName": "local6",
						"jwtIssuer":   "https://tz6.example.com",
						"namespaces": Values{
							"create": true,
						},
						"recommendations": Values{
							"enabled": true,
						},
						"trustDomain": "td6",
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
						"k8sPSAT": Values{
							"enabled": true,
						},
					},
					"sds": map[string]any{
						"enabled":               true,
						"defaultSVIDName":       "default",
						"defaultBundleName":     "ROOTCA",
						"defaultAllBundlesName": "ALL",
					},
					"workloadAttestors": Values{
						"k8s": Values{
							"disableContainerSelectors": true,
							"enabled":                   true,
						},
					},
				},
				"spire-server": Values{
					"controllerManager": Values{
						"identities": Values{
							"clusterSPIFFEIDs": Values{
								"default": Values{
									"enabled": false,
								},
							},
							"clusterStaticEntries": Values{
								"ap4": Values{
									"parentID":  "spiffe://td6/cluster/local6/spire/agents",
									"spiffeID":  "spiffe://example.com/foo",
									"selectors": []string{"k8s:ns:foo"},
								},
								"spire-agents": Values{
									"parentID": "spiffe://td6/spire/server",
									"selectors": []string{
										"k8s_psat:agent_ns:spire-system",
										"k8s_psat:agent_sa:spire-agent",
										"k8s_psat:cluster:local6",
									},
									"spiffeID": "spiffe://td6/cluster/local6/spire/agents",
								},
							},
						},
					},
					"enabled": false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			if tt.configFunc != nil {
				tt.configFunc(cfg)
			}
			source := newFakeDataSource(t, cfg)
			g := NewHelmValuesGenerator(tt.trustZone, tt.cluster, source, nil)

			got, err := g.GenerateValues()
			require.Nil(t, err, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHelmValuesGenerator_GenerateValues_AdditionalValues(t *testing.T) {
	tests := []struct {
		name       string
		trustZone  *trust_zone_proto.TrustZone
		cluster    *clusterpb.Cluster
		configFunc func(*config.Config)
		values     Values
		want       Values
	}{
		{
			name: "tz1 no binding or federation",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.Bundle = nil
				tz.BundleEndpointUrl = nil
				tz.JwtIssuer = nil
				return tz
			}(),
			cluster: func() *clusterpb.Cluster {
				cluster := fixtures.Cluster("local1")
				cluster.ExtraHelmValues = nil
				return cluster
			}(),
			configFunc: func(cfg *config.Config) {
				trustZone, ok := cfg.GetTrustZoneByName("tz1")
				require.True(t, ok)
				// nolint:staticcheck
				trustZone.AttestationPolicies = nil
				// nolint:staticcheck
				trustZone.Federations = nil
			},
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
						"caSubject": Values{
							"commonName":   "cofide.io",
							"country":      "UK",
							"organization": "Cofide",
						},
						"clusterName": "local1",
						"namespaces": Values{
							"create": true,
						},
						"recommendations": Values{
							"enabled": true,
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
						"k8sPSAT": Values{
							"enabled": true,
						},
					},
					"sds": map[string]any{
						"enabled":               true,
						"defaultSVIDName":       "default",
						"defaultBundleName":     "ROOTCA",
						"defaultAllBundlesName": "ALL",
					},
					"workloadAttestors": Values{
						"k8s": Values{
							"disableContainerSelectors": true,
							"enabled":                   true,
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
									"enabled": false,
								},
							},
							"clusterStaticEntries": Values{},
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
					"enabled":          true,
					"fullnameOverride": "spire-server",
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPSAT": Values{
							"audience": []string{"spire-server"},
							"enabled":  true,
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
			if tt.configFunc != nil {
				tt.configFunc(cfg)
			}
			source := newFakeDataSource(t, cfg)
			g := NewHelmValuesGenerator(tt.trustZone, tt.cluster, source, tt.values)

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
		cluster       *clusterpb.Cluster
		configFunc    func(*config.Config)
		wantErrString string
	}{
		{
			name:      "no trust provider",
			trustZone: fixtures.TrustZone("tz1"),
			cluster: func() *clusterpb.Cluster {
				cluster := fixtures.Cluster("local1")
				cluster.TrustProvider = nil
				return cluster
			}(),
			wantErrString: "trust provider cannot be nil",
		},
		{
			name:      "invalid trust provider kind",
			trustZone: fixtures.TrustZone("tz1"),
			cluster: func() *clusterpb.Cluster {
				cluster := fixtures.Cluster("local1")
				cluster.TrustProvider.Kind = fixtures.StringPtr("invalid-tp")
				return cluster
			}(),
			wantErrString: "an unknown trust provider kind was specified: invalid-tp",
		},
		{
			name:      "unknown attestation policy",
			trustZone: fixtures.TrustZone("tz1"),
			cluster:   fixtures.Cluster("local1"),
			configFunc: func(cfg *config.Config) {
				trustZone, ok := cfg.GetTrustZoneByName("tz1")
				require.True(t, ok)
				// nolint:staticcheck
				trustZone.AttestationPolicies[0].Policy = "invalid-ap"
			},
			wantErrString: "failed to find attestation policy invalid-ap in local config",
		},
		{
			name:      "unknown federated trust zone",
			trustZone: fixtures.TrustZone("tz1"),
			cluster:   fixtures.Cluster("local1"),
			configFunc: func(cfg *config.Config) {
				trustZone, ok := cfg.GetTrustZoneByName("tz1")
				require.True(t, ok)
				// nolint:staticcheck
				trustZone.Federations[0].To = "invalid-tz"
			},
			wantErrString: "failed to find trust zone invalid-tz in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			if tt.configFunc != nil {
				tt.configFunc(cfg)
			}
			source := newFakeDataSource(t, cfg)

			g := NewHelmValuesGenerator(tt.trustZone, tt.cluster, source, nil)

			_, err := g.GenerateValues()
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErrString)
		})
	}
}

func TestHelmValuesGenerator_GenerateValues_federationFailure(t *testing.T) {
	tests := []struct {
		name          string
		destTrustZone *trust_zone_proto.TrustZone
		wantErrString string
	}{
		{
			name: "nil bundle endpoint profile",
			destTrustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz2")
				tz.BundleEndpointProfile = nil
				return tz
			}(),
			wantErrString: "unexpected bundle endpoint profile 0",
		},
		{
			name: "unspecified bundle endpoint profile",
			destTrustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz2")
				tz.BundleEndpointProfile = trust_zone_proto.BundleEndpointProfile_BUNDLE_ENDPOINT_PROFILE_UNSPECIFIED.Enum()
				return tz
			}(),
			wantErrString: "unexpected bundle endpoint profile 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			cfg.TrustZones[1] = tt.destTrustZone
			source := newFakeDataSource(t, cfg)
			g := NewHelmValuesGenerator(cfg.TrustZones[0], cfg.Clusters[0], source, nil)

			_, err := g.GenerateValues()
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErrString)
		})
	}
}

func TestGetOrCreateNestedMap(t *testing.T) {
	tests := []struct {
		name      string
		values    map[string]any
		key       string
		want      map[string]any
		wantErr   bool
		errString string
	}{
		{
			name:   "create new map for missing key",
			values: map[string]any{"foo": "bar"},
			key:    "newkey",
			want:   map[string]any{},
		},
		{
			name: "get existing map",
			values: map[string]any{
				"existing": map[string]any{"foo": "bar"},
			},
			key:  "existing",
			want: map[string]any{"foo": "bar"},
		},
		{
			name:      "nil input map",
			values:    nil,
			key:       "key",
			wantErr:   true,
			errString: "input map is nil",
		},
		{
			name:      "empty key",
			values:    map[string]any{},
			key:       "",
			wantErr:   true,
			errString: "key cannot be empty",
		},
		{
			name: "key exists but wrong type",
			values: map[string]any{
				"wrongtype": "not a map",
			},
			key:       "wrongtype",
			wantErr:   true,
			errString: "value for key \"wrongtype\" is of type string, expected map[string]any",
		},
		{
			name: "key exists but value is nil",
			values: map[string]any{
				"nilvalue": nil,
			},
			key:  "nilvalue",
			want: map[string]any{},
		},
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := getOrCreateNestedMap(tt.values, tt.key)
			if tt.wantErr {
				assert.Equal(t, tt.errString, err.Error())
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, tt.want, resp)
			assert.IsType(t, map[string]any{}, resp)

			// For new maps, check that they were added to the input map.
			_, exists := tt.values[tt.key]
			assert.True(t, exists)
		})
	}
}

func TestMergeMaps(t *testing.T) {
	tests := []struct {
		name      string
		src       map[string]any
		dest      map[string]any
		want      map[string]any
		wantErr   bool
		errString string
	}{
		{
			name: "valid src and valid dest",
			src: map[string]any{
				"foo": "bar",
			},
			dest: map[string]any{
				"fizz": "buzz",
			},
			want: map[string]any{
				"foo":  "bar",
				"fizz": "buzz",
			},
		},
		{
			name: "valid src and empty dest",
			src: map[string]any{
				"foo": "bar",
			},
			dest: map[string]any{},
			want: map[string]any{
				"foo": "bar",
			},
		},
		{
			name: "empty src and valid dest",
			src:  map[string]any{},
			dest: map[string]any{
				"fizz": "buzz",
			},
			want: map[string]any{
				"fizz": "buzz",
			},
		},
		{
			name: "valid src and valid dest, src and dest types differ",
			src: map[string]any{
				"fizz": "buzz",
			},
			dest: map[string]any{
				"fizz": map[string]any{
					"fizz nested": "buzz",
				},
			},
			want: map[string]any{
				"fizz": "buzz",
			},
		},
		{
			name: "valid src and valid dest, dest and src types differ",
			src: map[string]any{
				"fizz": map[string]any{
					"fizz nested": "buzz",
				},
			},
			dest: map[string]any{
				"fizz": "buzz",
			},
			want: map[string]any{
				"fizz": map[string]any{
					"fizz nested": "buzz",
				},
			},
		},
		{
			name: "valid src and valid dest, nested",
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
			name: "valid src and valid dest, with existing key",
			src: map[string]any{
				"foo":   "bar",
				"hello": "world",
			},
			dest: map[string]any{
				"foo": "baz",
			},
			want: map[string]any{
				"foo":   "bar",
				"hello": "world",
			},
		},
		{
			name: "valid src and valid dest, nested, with existing key",
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
			name: "valid src and valid dest, additional nesting, existing key",
			src: map[string]any{
				"spire-server": map[string]any{
					"controllerManager": map[string]any{
						"identities": map[string]any{
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
							"clusterStaticEntries": Values{},
						},
					},
					"enabled": true,
				},
			},
			want: map[string]any{
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
							"clusterStaticEntries": Values{},
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
					"enabled": true,
				},
			},
		},
		{
			name:      "nil src and valid dest",
			src:       nil,
			dest:      map[string]any{"foo": "bar"},
			wantErr:   true,
			errString: "source map is nil",
		},
		{
			name:      "valid src and nil dest",
			src:       map[string]any{"foo": "bar"},
			dest:      nil,
			wantErr:   true,
			errString: "destination map is nil",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := MergeMaps(tt.dest, tt.src)
			if tt.wantErr {
				assert.Equal(t, tt.errString, err.Error())
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, tt.want, resp)
		})
	}
}

func TestShallowMerge(t *testing.T) {
	tests := []struct {
		name string
		maps []map[string]any
		want map[string]any
	}{
		{
			name: "valid slice of maps",
			maps: []map[string]any{
				{
					"foo": "bar",
				},
				{
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
				{},
				{},
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
			resp := shallowMerge(tt.maps)
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
						"caSubject": Values{
							"commonName":   "",
							"country":      "",
							"organization": "",
						},
						"clusterName": "local1",
						"namespaces": Values{
							"create": false,
						},
						"recommendations": map[string]any{
							"enabled": false,
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
			name: "valid global values, empty jwtIssuer value",
			input: globalValues{
				spireClusterName: "local1",
				spireTrustDomain: "td1",
				spireJwtIssuer:   "",
			},
			want: map[string]any{
				"global": map[string]any{
					"spire": map[string]any{
						"caSubject": Values{
							"commonName":   "",
							"country":      "",
							"organization": "",
						},
						"clusterName": "local1",
						"namespaces": Values{
							"create": false,
						},
						"recommendations": map[string]any{
							"enabled": false,
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
			name: "valid global values, populated jwtIssuer value",
			input: globalValues{
				spireClusterName: "local1",
				spireTrustDomain: "td1",
				spireJwtIssuer:   "https://tz1.example.com",
			},
			want: map[string]any{
				"global": map[string]any{
					"spire": map[string]any{
						"caSubject": Values{
							"commonName":   "",
							"country":      "",
							"organization": "",
						},
						"clusterName": "local1",
						"jwtIssuer":   "https://tz1.example.com",
						"namespaces": Values{
							"create": false,
						},
						"recommendations": map[string]any{
							"enabled": false,
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

			assert.Nil(t, err)
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
					WorkloadAttestor: "k8s",
					WorkloadAttestorConfig: map[string]any{
						"enabled":                   true,
						"disableContainerSelectors": true,
					},
					NodeAttestor: "k8sPSAT",
				},
				sdsConfig: map[string]any{
					"enabled":               true,
					"defaultSVIDName":       "default",
					"defaultBundleName":     "ROOTCA",
					"defaultAllBundlesName": "ALL",
				},
			},
			want: map[string]any{
				"spire-agent": map[string]any{
					"fullnameOverride": "spire-agent",
					"logLevel":         "DEBUG",
					"nodeAttestor": map[string]any{
						"k8sPSAT": map[string]any{
							"enabled": true,
						},
					},
					"sds": map[string]any{
						"enabled":               true,
						"defaultSVIDName":       "default",
						"defaultBundleName":     "ROOTCA",
						"defaultAllBundlesName": "ALL",
					},
					"workloadAttestors": map[string]any{
						"k8s": map[string]any{
							"enabled":                   true,
							"disableContainerSelectors": true,
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
					WorkloadAttestor: "k8s",
					WorkloadAttestorConfig: map[string]any{
						"enabled":                   true,
						"disableContainerSelectors": true,
					},
					NodeAttestor: "k8sPSAT",
				},
				sdsConfig: map[string]any{
					"enabled":               true,
					"defaultSVIDName":       "default",
					"defaultBundleName":     "ROOTCA",
					"defaultAllBundlesName": "ALL",
				},
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
					WorkloadAttestor:       "k8s",
					WorkloadAttestorConfig: map[string]any{},
					NodeAttestor:           "k8sPSAT",
				},
				sdsConfig: map[string]any{
					"enabled":               true,
					"defaultSVIDName":       "default",
					"defaultBundleName":     "ROOTCA",
					"defaultAllBundlesName": "ALL",
				},
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
					WorkloadAttestor: "",
					WorkloadAttestorConfig: map[string]any{
						"enabled":                   true,
						"disableContainerSelectors": true,
					},
					NodeAttestor: "k8sPSAT",
				},
				sdsConfig: map[string]any{
					"enabled":               true,
					"defaultSVIDName":       "default",
					"defaultBundleName":     "ROOTCA",
					"defaultAllBundlesName": "ALL",
				},
			},
			want:      nil,
			wantErr:   true,
			errString: "agentConfig.WorkloadAttestor value is empty",
		},
		{
			name: "invalid SPIRE agent values, empty sdsConfig value",
			input: spireAgentValues{
				fullnameOverride: "spire-agent",
				logLevel:         "DEBUG",
				agentConfig: trustprovider.TrustProviderAgentConfig{
					WorkloadAttestor: "",
					WorkloadAttestorConfig: map[string]any{
						"enabled":                   true,
						"disableContainerSelectors": true,
					},
					NodeAttestor: "k8sPSAT",
				},
				sdsConfig: map[string]any{},
			},
			want:      nil,
			wantErr:   true,
			errString: "sdsConfig value is empty",
		},
		{
			name: "invalid SPIRE agent values, nil sdsConfig value",
			input: spireAgentValues{
				fullnameOverride: "spire-agent",
				logLevel:         "DEBUG",
				agentConfig: trustprovider.TrustProviderAgentConfig{
					WorkloadAttestor: "",
					WorkloadAttestorConfig: map[string]any{
						"enabled":                   true,
						"disableContainerSelectors": true,
					},
					NodeAttestor: "k8sPSAT",
				},
				sdsConfig: nil,
			},
			want:      nil,
			wantErr:   true,
			errString: "sdsConfig value is nil",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tt.input.generateValues()
			if tt.wantErr {
				assert.Equal(t, tt.errString, err.Error())
				return
			}

			assert.Nil(t, err)
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
				enabled:                  true,
				fullnameOverride:         "spire-server",
				logLevel:                 "DEBUG",
				serverConfig: trustprovider.TrustProviderServerConfig{
					NodeAttestor: "k8sPSAT",
					NodeAttestorConfig: map[string]any{
						"enabled":  true,
						"audience": []string{"spire-server"},
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
					"enabled":          true,
					"fullnameOverride": "spire-server",
					"logLevel":         "DEBUG",
					"nodeAttestor": Values{
						"k8sPSAT": Values{
							"audience": []string{"spire-server"},
							"enabled":  true,
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
			name: "valid SPIRE server values, enabled set to false",
			input: spireServerValues{
				enabled: false,
			},
			want: map[string]any{
				"spire-server": map[string]any{
					"enabled": false,
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
				enabled:                  true,
				fullnameOverride:         "spire-server",
				logLevel:                 "DEBUG",
				serverConfig: trustprovider.TrustProviderServerConfig{
					NodeAttestor: "k8sPSAT",
					//NodeAttestorEnabled: true,
					NodeAttestorConfig: map[string]any{
						"enabled":  true,
						"audience": []string{"spire-server"},
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
				enabled:                  true,
				fullnameOverride:         "spire-server",
				logLevel:                 "DEBUG",
				serverConfig: trustprovider.TrustProviderServerConfig{
					NodeAttestor: "k8sPSAT",
					//NodeAttestorEnabled: true,
					NodeAttestorConfig: map[string]any{},
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

			assert.Nil(t, err)
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

			assert.Nil(t, err)
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

			assert.Nil(t, err)
			assert.Equal(t, tt.want, resp)
		})
	}
}

func TestGetSDSConfig(t *testing.T) {
	tests := []struct {
		name      string
		profile   string
		want      map[string]any
		wantErr   bool
		errString string
	}{
		{
			name:    "valid kubernetes profile",
			profile: "kubernetes",
			want: map[string]any{
				"enabled":               true,
				"defaultSVIDName":       "default",
				"defaultBundleName":     "ROOTCA",
				"defaultAllBundlesName": "ALL",
			},
			wantErr: false,
		},
		{
			name:    "valid istio profile",
			profile: "istio",
			want: map[string]any{
				"enabled":               true,
				"defaultSVIDName":       "default",
				"defaultBundleName":     "null",
				"defaultAllBundlesName": "ROOTCA",
			},
			wantErr: false,
		},
		{
			name:      "invalid profile",
			profile:   "invalid",
			wantErr:   true,
			errString: "an invalid profile was specified: invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := getSDSConfig(tt.profile)
			if tt.wantErr {
				assert.Equal(t, tt.errString, err.Error())
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, tt.want, resp)
		})
	}
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
			fixtures.TrustZone("tz2"),
			fixtures.TrustZone("tz4"),
			fixtures.TrustZone("tz6"),
		},
		Clusters: []*clusterpb.Cluster{
			fixtures.Cluster("local1"),
			fixtures.Cluster("local2"),
			fixtures.Cluster("local4"),
			fixtures.Cluster("local6"),
		},
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
			fixtures.AttestationPolicy("ap1"),
			fixtures.AttestationPolicy("ap2"),
			fixtures.AttestationPolicy("ap4"),
		},
		Plugins: fixtures.Plugins("plugins1"),
	}
}
