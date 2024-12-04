// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/attestationpolicy"
	"github.com/cofide/cofidectl/internal/pkg/federation"
	"github.com/cofide/cofidectl/internal/pkg/trustzone"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
)

type HelmValuesGenerator struct {
	source    cofidectl_plugin.DataSource
	trustZone *trust_zone_proto.TrustZone
	values    map[string]interface{}
}

func NewHelmValuesGenerator(trustZone *trust_zone_proto.TrustZone, source cofidectl_plugin.DataSource, values map[string]interface{}) *HelmValuesGenerator {
	return &HelmValuesGenerator{
		trustZone: trustZone,
		source:    source,
		values:    values,
	}
}

func (g *HelmValuesGenerator) GenerateValues() (map[string]interface{}, error) {
	tz := trustzone.NewTrustZone(g.trustZone)
	tp, err := tz.GetTrustProvider()
	if err != nil {
		return nil, err
	}

	agentConfig := tp.AgentConfig
	serverConfig := tp.ServerConfig

	globalValues := map[string]interface{}{
		"global": map[string]interface{}{
			"spire": map[string]interface{}{
				"clusterName": g.trustZone.GetKubernetesCluster(),
				"recommendations": map[string]interface{}{
					"create": true,
				},
				"trustDomain": g.trustZone.TrustDomain,
			},
			"installAndUpgradeHooks": map[string]interface{}{
				"enabled": false,
			},
			"deleteHooks": map[string]interface{}{
				"enabled": false,
			},
		},
	}

	if issuer := g.trustZone.GetJwtIssuer(); issuer != "" {
		if global, ok := getNestedMap(globalValues, "global"); ok {
			if spire, ok := getNestedMap(global, "spire"); ok {
				spire["jwtIssuer"] = issuer
			}
		}
	}

	spireAgentValues := map[string]interface{}{
		"spire-agent": map[string]interface{}{
			"fullnameOverride": "spire-agent",
			"logLevel":         "DEBUG",
			"nodeAttestor": map[string]interface{}{
				agentConfig.NodeAttestor: map[string]interface{}{
					"enabled": agentConfig.NodeAttestorEnabled,
				},
			},
			"server": map[string]interface{}{
				"address": "spire-server.spire",
			},
			"workloadAttestors": map[string]interface{}{
				agentConfig.WorkloadAttestor: agentConfig.WorkloadAttestorConfig,
			},
		},
	}

	spireServerValues := map[string]interface{}{
		"spire-server": map[string]interface{}{
			"caKeyType": "rsa-2048",
			"caTTL":     "12h",
			"controllerManager": map[string]interface{}{
				"enabled": true,
			},
			"fullnameOverride": "spire-server",
			"logLevel":         "DEBUG",
			"nodeAttestor": map[string]interface{}{
				serverConfig.NodeAttestor: serverConfig.NodeAttestorConfig,
			},
			"service": map[string]interface{}{
				"type": "LoadBalancer",
			},
		},
	}

	if len(g.trustZone.AttestationPolicies) > 0 {
		// Disables the default ClusterSPIFFEID CR.
		if spireServer, ok := getNestedMap(spireServerValues, "spire-server"); ok {
			if controllerManager, ok := getNestedMap(spireServer, "controllerManager"); ok {
				controllerManager["identities"] = map[string]interface{}{
					"clusterSPIFFEIDs": map[string]interface{}{
						"default": map[string]interface{}{
							"enabled": false,
						},
					},
				}
			}
		}

		// Adds the attestation policies as ClusterSPIFFEID CRs to be reconciled by spire-controller-manager.
		for _, binding := range g.trustZone.AttestationPolicies {
			policy, err := g.source.GetAttestationPolicy(binding.Policy)
			if err != nil {
				return nil, err
			}

			clusterSPIFFEIDs, err := attestationpolicy.NewAttestationPolicy(policy).GetHelmConfig(g.source, binding)
			if err != nil {
				return nil, err
			}

			if spireServer, ok := getNestedMap(spireServerValues, "spire-server"); ok {
				if controllerManager, ok := getNestedMap(spireServer, "controllerManager"); ok {
					if identities, ok := getNestedMap(controllerManager, "identities"); ok {
						if csid, ok := getNestedMap(identities, "clusterSPIFFEIDs"); ok {
							csid[policy.Name] = clusterSPIFFEIDs
						}
					}
				}
			}
		}
	} else {
		// Enables the default ClusterSPIFFEID CR.
		if spireServer, ok := getNestedMap(spireServerValues, "spire-server"); ok {
			if controllerManager, ok := getNestedMap(spireServer, "controllerManager"); ok {
				controllerManager["identities"] = map[string]interface{}{
					"clusterSPIFFEIDs": map[string]interface{}{
						"default": map[string]interface{}{
							"enabled": true,
						},
					},
				}
			}
		}
	}

	// Adds the federations as ClusterFederatedTrustDomain CRs to be reconciled by spire-controller-manager.
	if len(g.trustZone.Federations) > 0 {
		for _, fed := range g.trustZone.Federations {
			tz, err := g.source.GetTrustZone(fed.To)
			if err != nil {
				return nil, err
			}

			if tz.GetBundleEndpointUrl() != "" {
				if spireServer, ok := getNestedMap(spireServerValues, "spire-server"); ok {
					spireServer["federation"] = map[string]interface{}{
						"enabled": true,
					}

					if controllerManager, ok := getNestedMap(spireServer, "controllerManager"); ok {
						if identities, ok := getNestedMap(controllerManager, "identities"); ok {
							if cftd, ok := getNestedMap(identities, "clusterFederatedTrustDomains"); ok {
								cftd[fed.To] = federation.NewFederation(tz).GetHelmConfig()
							} else {
								identities["clusterFederatedTrustDomains"] = map[string]interface{}{
									fed.To: federation.NewFederation(tz).GetHelmConfig(),
								}
							}
						}
					}
				}
			}
		}
	}

	spiffeOIDCDiscoveryProviderValues := map[string]interface{}{
		"spiffe-oidc-discovery-provider": map[string]interface{}{
			"enabled": false,
		},
	}

	spiffeCSIDriverValues := map[string]interface{}{
		"spiffe-csi-driver": map[string]interface{}{
			"fullnameOverride": "spiffe-csi-driver",
		},
	}

	valuesMaps := []map[string]interface{}{
		globalValues,
		spireAgentValues,
		spireServerValues,
		spiffeOIDCDiscoveryProviderValues,
		spiffeCSIDriverValues,
	}

	if g.values != nil {
		mergeValues(valuesMaps, g.values, true)
	}

	if g.trustZone.ExtraHelmValues != nil {
		// TODO: Potentially retrieve Helm values as a map[string]interface directly.
		extraHelmValues := g.trustZone.ExtraHelmValues.AsMap()
		mergeValues(valuesMaps, extraHelmValues, true)
	}

	combinedValues := make(map[string]interface{})

	for _, valuesMap := range valuesMaps {
		for key, value := range valuesMap {
			combinedValues[key] = value
		}
	}

	return combinedValues, nil
}

// getNestedMap retrieves a nested map[string]interface{} from a parent map.
func getNestedMap(m map[string]interface{}, key string) (map[string]interface{}, bool) {
	val, exists := m[key]
	if !exists {
		return nil, false
	}

	nestedMap := val.(map[string]interface{})
	return nestedMap, true
}

// mergeValues iterates over a slice of maps and merges each map with the provided values map.
func mergeValues(valuesMaps []map[string]interface{}, values map[string]interface{}, overwriteExistingKeys bool) {
	for i, valuesMap := range valuesMaps {
		for key := range valuesMap {
			if inputValue, exists := values[key]; exists {
				if inputMap, ok := inputValue.(map[string]interface{}); ok {
					if existingMap, exists := valuesMap[key].(map[string]interface{}); exists {
						valuesMaps[i][key] = mergeMaps(inputMap, existingMap, overwriteExistingKeys)
					}
				}
			}
		}
	}
}

// mergeMaps merges the source map into the destination map, returning a new merged map.
func mergeMaps(src, dest map[string]interface{}, overwriteExistingKeys bool) map[string]interface{} {
	merged := make(map[string]interface{})

	for key, value := range dest {
		merged[key] = value
	}

	for key, value := range src {
		if srcMap, isSrcMap := value.(map[string]interface{}); isSrcMap {
			if destMap, isDestMap := dest[key].(map[string]interface{}); isDestMap {
				merged[key] = mergeMaps(srcMap, destMap, overwriteExistingKeys)
			} else {
				merged[key] = srcMap
			}
		} else {
			if overwriteExistingKeys || merged[key] == nil {
				merged[key] = value
			}
		}
	}

	return merged
}
