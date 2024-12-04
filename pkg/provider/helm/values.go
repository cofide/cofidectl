// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"fmt"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/attestationpolicy"
	"github.com/cofide/cofidectl/internal/pkg/federation"
	"github.com/cofide/cofidectl/internal/pkg/trustprovider"
	"github.com/cofide/cofidectl/internal/pkg/trustzone"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
)

type HelmValuesGenerator struct {
	source    cofidectl_plugin.DataSource
	trustZone *trust_zone_proto.TrustZone
	values    map[string]any
}

type globalValues struct {
	deleteHooks                   bool
	installAndUpgradeHooksEnabled bool
	spireClusterName              string
	spireCreateRecommendations    bool
	spireTrustDomain              string
}

type spireAgentValues struct {
	agentConfig        trustprovider.TrustProviderAgentConfig
	fullnameOverride   string
	logLevel           string
	spireServerAddress string
}

type spireServerValues struct {
	caKeyType                string
	caTTL                    string
	controllerManagerEnabled bool
	fullnameOverride         string
	logLevel                 string
	serverConfig             trustprovider.TrustProviderServerConfig
	serviceType              string
}

type spiffeOIDCDiscoveryProviderValues struct {
	enabled bool
}

type spiffeCSIDriverValues struct {
	fullnameOverride string
}

func NewHelmValuesGenerator(trustZone *trust_zone_proto.TrustZone, source cofidectl_plugin.DataSource, values map[string]any) *HelmValuesGenerator {
	return &HelmValuesGenerator{
		trustZone: trustZone,
		source:    source,
		values:    values,
	}
}

func (g *HelmValuesGenerator) GenerateValues() (map[string]any, error) {
	tz := trustzone.NewTrustZone(g.trustZone)
	tp, err := tz.GetTrustProvider()
	if err != nil {
		return nil, err
	}

	agentConfig := tp.AgentConfig
	serverConfig := tp.ServerConfig

	gv := globalValues{
		spireClusterName:              g.trustZone.GetKubernetesCluster(),
		spireCreateRecommendations:    true,
		spireTrustDomain:              g.trustZone.TrustDomain,
		installAndUpgradeHooksEnabled: false,
		deleteHooks:                   false,
	}
	globalValues, err := gv.generateValues()
	if err != nil {
		return nil, err
	}

	if issuer := g.trustZone.GetJwtIssuer(); issuer != "" {
		global, ok := getNestedMap(globalValues, "global")
		if !ok {
			return nil, fmt.Errorf("failed to get global map from globalValues")
		}

		spire, ok := getNestedMap(global, "spire")
		if !ok {
			return nil, fmt.Errorf("failed to get spire map from global map")
		}

		spire["jwtIssuer"] = issuer
	}

	sav := spireAgentValues{
		fullnameOverride:   "spire-agent",
		logLevel:           "DEBUG",
		agentConfig:        agentConfig,
		spireServerAddress: "spire-server.spire",
	}
	spireAgentValues, err := sav.generateValues()
	if err != nil {
		return nil, err
	}

	ssv := spireServerValues{
		caKeyType:                "rsa-2048",
		caTTL:                    "12h",
		controllerManagerEnabled: true,
		fullnameOverride:         "spire-server",
		logLevel:                 "DEBUG",
		serverConfig:             serverConfig,
		serviceType:              "LoadBalancer",
	}
	spireServerValues, err := ssv.generateValues()
	if err != nil {
		return nil, err
	}

	spireServer, ok := getNestedMap(spireServerValues, "spire-server")
	if !ok {
		return nil, fmt.Errorf("failed to get spire-server map from spireServerValues")
	}

	controllerManager, ok := getNestedMap(spireServer, "controllerManager")
	if !ok {
		return nil, fmt.Errorf("failed to get controllerManager map from spireServer")
	}

	// Enables the default ClusterSPIFFEID CR by default.
	controllerManager["identities"] = map[string]any{
		"clusterSPIFFEIDs": map[string]any{
			"default": map[string]any{
				"enabled": true,
			},
		},
	}

	identities, ok := getNestedMap(controllerManager, "identities")
	if !ok {
		return nil, fmt.Errorf("failed to get identities map from controllerManager")
	}

	if len(g.trustZone.AttestationPolicies) > 0 {
		csids, ok := getNestedMap(identities, "clusterSPIFFEIDs")
		if !ok {
			return nil, fmt.Errorf("failed to get clusterSPIFFEIDs map from identities")
		}

		// Disables the default ClusterSPIFFEID CR.
		csids["default"] = map[string]any{
			"enabled": false,
		}

		// Adds the attestation policies as ClusterSPIFFEID CRs to be reconciled by the spire-controller-manager.
		for _, binding := range g.trustZone.AttestationPolicies {
			policy, err := g.source.GetAttestationPolicy(binding.Policy)
			if err != nil {
				return nil, err
			}

			clusterSPIFFEIDs, err := attestationpolicy.NewAttestationPolicy(policy).GetHelmConfig(g.source, binding)
			if err != nil {
				return nil, err
			}

			csids[policy.Name] = clusterSPIFFEIDs
		}
	}

	// Adds the federations as ClusterFederatedTrustDomain CRs to be reconciled by the spire-controller-manager.
	if len(g.trustZone.Federations) > 0 {
		for _, fed := range g.trustZone.Federations {
			tz, err := g.source.GetTrustZone(fed.To)
			if err != nil {
				return nil, err
			}

			if tz.GetBundleEndpointUrl() != "" {
				spireServer["federation"] = map[string]any{
					"enabled": true,
				}

				cftd, ok := getNestedMap(identities, "clusterFederatedTrustDomains")
				if !ok {
					identities["clusterFederatedTrustDomains"] = map[string]any{
						fed.To: federation.NewFederation(tz).GetHelmConfig(),
					}
				} else {
					cftd[fed.To] = federation.NewFederation(tz).GetHelmConfig()
				}
			}
		}
	}

	soidcpv := spiffeOIDCDiscoveryProviderValues{enabled: false}
	spiffeOIDCDiscoveryProviderValues, err := soidcpv.generateValues()
	if err != nil {
		return nil, err
	}

	scsidv := spiffeCSIDriverValues{fullnameOverride: "spiffe-csi-driver"}
	spiffeCSIDriverValues, err := scsidv.generateValues()
	if err != nil {
		return nil, err
	}

	valuesMaps := []map[string]any{
		globalValues,
		spireAgentValues,
		spireServerValues,
		spiffeOIDCDiscoveryProviderValues,
		spiffeCSIDriverValues,
	}

	combinedValues := flattenMaps(valuesMaps)

	if g.values != nil {
		combinedValues = mergeMaps(g.values, combinedValues, true)
	}

	if g.trustZone.ExtraHelmValues != nil {
		// TODO: Potentially retrieve Helm values as map[string]any directly.
		extraHelmValues := g.trustZone.ExtraHelmValues.AsMap()
		combinedValues = mergeMaps(extraHelmValues, combinedValues, true)
	}

	return combinedValues, nil
}

// generateValues generates the global Helm values map.
func (g *globalValues) generateValues() (map[string]any, error) {
	if g.spireClusterName == "" {
		return nil, fmt.Errorf("spireClusterName value is empty")
	}

	if g.spireTrustDomain == "" {
		return nil, fmt.Errorf("spireTrustDomain value is empty")
	}

	return map[string]any{
		"global": map[string]any{
			"spire": map[string]any{
				"clusterName": g.spireClusterName,
				"recommendations": map[string]any{
					"create": g.spireCreateRecommendations,
				},
				"trustDomain": g.spireTrustDomain,
			},
			"installAndUpgradeHooks": map[string]any{
				"enabled": g.installAndUpgradeHooksEnabled,
			},
			"deleteHooks": map[string]any{
				"enabled": g.deleteHooks,
			},
		},
	}, nil
}

// generateValues generates the spire-agent Helm values map.
func (s *spireAgentValues) generateValues() (map[string]any, error) {
	if s.fullnameOverride == "" {
		return nil, fmt.Errorf("fullnameOverride value is empty")
	}

	if s.logLevel == "" {
		return nil, fmt.Errorf("logLevel value is empty")
	}

	if s.agentConfig.NodeAttestor == "" {
		return nil, fmt.Errorf("agentConfig.NodeAttestor value is empty")
	}

	if s.agentConfig.WorkloadAttestor == "" {
		return nil, fmt.Errorf("agentConfig.WorkloadAttestor value is empty")
	}

	if s.agentConfig.WorkloadAttestorConfig == nil {
		return nil, fmt.Errorf("agentConfig.WorkloadAttestorConfig value is nil")
	}

	if len(s.agentConfig.WorkloadAttestorConfig) == 0 {
		return nil, fmt.Errorf("agentConfig.WorkloadAttestorConfig value is empty")
	}

	if s.spireServerAddress == "" {
		return nil, fmt.Errorf("spireServerAddress value is empty")
	}

	return map[string]any{
		"spire-agent": map[string]any{
			"fullnameOverride": s.fullnameOverride,
			"logLevel":         s.logLevel,
			"nodeAttestor": map[string]any{
				s.agentConfig.NodeAttestor: map[string]any{
					"enabled": s.agentConfig.NodeAttestorEnabled,
				},
			},
			"server": map[string]any{
				"address": s.spireServerAddress,
			},
			"workloadAttestors": map[string]any{
				s.agentConfig.WorkloadAttestor: s.agentConfig.WorkloadAttestorConfig,
			},
		},
	}, nil
}

// generateValues generates the spire-server Helm values map.
func (s *spireServerValues) generateValues() (map[string]any, error) {
	if s.caKeyType == "" {
		return nil, fmt.Errorf("caKeyType value is empty")
	}

	if s.caTTL == "" {
		return nil, fmt.Errorf("caTTL value is empty")
	}

	if s.fullnameOverride == "" {
		return nil, fmt.Errorf("fullnameOverride value is empty")
	}

	if s.logLevel == "" {
		return nil, fmt.Errorf("logLevel value is empty")
	}

	if s.serverConfig.NodeAttestor == "" {
		return nil, fmt.Errorf("serverConfig.NodeAttestor value is empty")
	}

	if s.serverConfig.NodeAttestorConfig == nil {
		return nil, fmt.Errorf("serverConfig.NodeAttestorConfig value is nil")
	}

	if len(s.serverConfig.NodeAttestorConfig) == 0 {
		return nil, fmt.Errorf("serverConfig.NodeAttestorConfig value is empty")
	}

	if s.serviceType == "" {
		return nil, fmt.Errorf("serviceType value is empty")
	}

	return map[string]any{
		"spire-server": map[string]any{
			"caKeyType": s.caKeyType,
			"caTTL":     s.caTTL,
			"controllerManager": map[string]any{
				"enabled": s.controllerManagerEnabled,
			},
			"fullnameOverride": s.fullnameOverride,
			"logLevel":         s.logLevel,
			"nodeAttestor": map[string]any{
				s.serverConfig.NodeAttestor: s.serverConfig.NodeAttestorConfig,
			},
			"service": map[string]any{
				"type": s.serviceType,
			},
		},
	}, nil
}

// generateValues generates the spiffe-oidc-discovery-provider Helm values map.
func (s *spiffeOIDCDiscoveryProviderValues) generateValues() (map[string]any, error) {
	return map[string]any{
		"spiffe-oidc-discovery-provider": map[string]any{
			"enabled": s.enabled,
		},
	}, nil
}

// generateValues generates the spiffe-csi-driver Helm values map.
func (s *spiffeCSIDriverValues) generateValues() (map[string]any, error) {
	if s.fullnameOverride == "" {
		return nil, fmt.Errorf("fullnameOverride value is empty")
	}

	return map[string]any{
		"spiffe-csi-driver": map[string]any{
			"fullnameOverride": s.fullnameOverride,
		},
	}, nil
}

// getNestedMap retrieves a nested map[string]any from a parent map.
func getNestedMap(m map[string]any, key string) (map[string]any, bool) {
	value, exists := m[key]
	if !exists {
		return nil, false
	}

	nestedMap := value.(map[string]any)
	return nestedMap, true
}

// mergeMaps merges the source map into the destination map, returning a new merged map.
func mergeMaps(src, dest map[string]any, overwriteExistingKeys bool) map[string]any {
	merged := make(map[string]any)

	for key, value := range dest {
		merged[key] = value
	}

	for key, value := range src {
		if srcMap, isSrcMap := value.(map[string]any); isSrcMap {
			if destMap, isDestMap := merged[key].(map[string]any); isDestMap {
				merged[key] = mergeMaps(srcMap, destMap, overwriteExistingKeys)
			} else {
				merged[key] = srcMap
			}
		} else if overwriteExistingKeys || merged[key] == nil {
			merged[key] = value
		}
	}

	return merged
}

// flattenMaps flattens a slice of maps into a single map.
func flattenMaps(maps []map[string]any) map[string]any {
	flattened := make(map[string]any)
	for _, m := range maps {
		for key, value := range m {
			flattened[key] = value
		}
	}

	return flattened
}
