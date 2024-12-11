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
	spireJwtIssuer                string
	spireTrustDomain              string
}

type spireAgentValues struct {
	agentConfig        trustprovider.TrustProviderAgentConfig
	fullnameOverride   string
	logLevel           string
	sdsConfig          map[string]any
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

	gv := globalValues{
		spireClusterName:              g.trustZone.GetKubernetesCluster(),
		spireCreateRecommendations:    true,
		spireJwtIssuer:                g.trustZone.GetJwtIssuer(),
		spireTrustDomain:              g.trustZone.TrustDomain,
		installAndUpgradeHooksEnabled: false,
		deleteHooks:                   false,
	}

	globalValues, err := gv.generateValues()
	if err != nil {
		return nil, err
	}

	sdsConfig, err := getSDSConfig(tz.TrustZoneProto.GetProfile())
	if err != nil {
		return nil, err
	}

	sav := spireAgentValues{
		fullnameOverride:   "spire-agent",
		logLevel:           "DEBUG",
		agentConfig:        tp.AgentConfig,
		sdsConfig:          sdsConfig,
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
		serverConfig:             tp.ServerConfig,
		serviceType:              "LoadBalancer",
	}
	spireServerValues, err := ssv.generateValues()
	if err != nil {
		return nil, err
	}

	spireServer, err := getOrCreateNestedMap(spireServerValues, "spire-server")
	if err != nil {
		return nil, fmt.Errorf("failed to get spire-server map from spireServerValues: %w", err)
	}

	controllerManager, err := getOrCreateNestedMap(spireServer, "controllerManager")
	if err != nil {
		return nil, fmt.Errorf("failed to get controllerManager map from spireServer: %w", err)
	}

	// Enables the default ClusterSPIFFEID CR by default.
	controllerManager["identities"] = map[string]any{
		"clusterSPIFFEIDs": map[string]any{
			"default": map[string]any{
				"enabled": true,
			},
		},
	}

	identities, err := getOrCreateNestedMap(controllerManager, "identities")
	if err != nil {
		return nil, fmt.Errorf("failed to get identities map from controllerManager: %w", err)
	}

	if len(g.trustZone.AttestationPolicies) > 0 {
		csids, err := getOrCreateNestedMap(identities, "clusterSPIFFEIDs")
		if err != nil {
			return nil, fmt.Errorf("failed to get clusterSPIFFEIDs map from identities: %w", err)
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
				fedMap, err := getOrCreateNestedMap(spireServer, "federation")
				if err != nil {
					return nil, fmt.Errorf("failed to get federation map from spireServer: %w", err)
				}

				fedMap["enabled"] = true

				cftd, err := getOrCreateNestedMap(identities, "clusterFederatedTrustDomains")
				if err != nil {
					return nil, fmt.Errorf("failed to get clusterFederatedTrustDomains map from identities: %w", err)
				}

				cftd[fed.To], err = federation.NewFederation(tz).GetHelmConfig()
				if err != nil {
					return nil, err
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

	combinedValues := shallowMerge(valuesMaps)

	if g.values != nil {
		combinedValues, err = mergeMaps(combinedValues, g.values)
		if err != nil {
			return nil, err
		}
	}

	if g.trustZone.ExtraHelmValues != nil {
		// TODO: Potentially retrieve Helm values as map[string]any directly.
		extraHelmValues := g.trustZone.ExtraHelmValues.AsMap()
		combinedValues, err = mergeMaps(combinedValues, extraHelmValues)
		if err != nil {
			return nil, err
		}
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

	values := map[string]any{
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
	}

	if g.spireJwtIssuer != "" {
		global, err := getOrCreateNestedMap(values, "global")
		if err != nil {
			return nil, fmt.Errorf("failed to get global map: %w", err)
		}

		spire, err := getOrCreateNestedMap(global, "spire")
		if err != nil {
			return nil, fmt.Errorf("failed to get spire map from global map: %w", err)
		}

		spire["jwtIssuer"] = g.spireJwtIssuer
	}

	return values, nil
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

	if s.sdsConfig == nil {
		return nil, fmt.Errorf("sdsConfig value is nil")
	}

	if len(s.sdsConfig) == 0 {
		return nil, fmt.Errorf("sdsConfig value is empty")
	}

	if len(s.agentConfig.WorkloadAttestorConfig) == 0 {
		return nil, fmt.Errorf("agentConfig.WorkloadAttestorConfig value is empty")
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
			"sds": s.sdsConfig,
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

// getOrCreateNestedMap retrieves a nested map[string]any from a parent map or creates it
// if it doesn't exist.
func getOrCreateNestedMap(m map[string]any, key string) (map[string]any, error) {
	if m == nil {
		return nil, fmt.Errorf("input map is nil")
	}

	if key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}

	if value, exists := m[key]; exists {
		if value == nil {
			newMap := make(map[string]any)
			m[key] = newMap
			return newMap, nil
		}

		subMap, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("value for key %q is of type %T, expected map[string]any", key, value)
		}

		return subMap, nil
	}

	// When the key doesn't exist, create a new map.
	newMap := make(map[string]any)
	m[key] = newMap
	return newMap, nil
}

// mergeMaps merges the source map into the destination map, returning a new merged map.
func mergeMaps(dest, src map[string]any) (map[string]any, error) {
	if src == nil {
		return nil, fmt.Errorf("source map is nil")
	}

	if dest == nil {
		return nil, fmt.Errorf("destination map is nil")
	}

	for key, value := range src {
		if srcMap, isSrcMap := value.(map[string]any); isSrcMap {
			if destMap, isDestMap := dest[key].(map[string]any); isDestMap {
				merged, err := mergeMaps(destMap, srcMap)
				if err != nil {
					return nil, err
				}

				dest[key] = merged
			} else {
				dest[key] = srcMap
			}
		} else {
			// Always overwrite existing keys.
			dest[key] = value
		}
	}

	return dest, nil
}

// shallowMerge flattens a slice of maps into a single map.
func shallowMerge(maps []map[string]any) map[string]any {
	flattened := make(map[string]any)
	for _, m := range maps {
		for key, value := range m {
			flattened[key] = value
		}
	}

	return flattened
}

// getSDSConfig returns the appropriate SPIRE agent Envoy SDS configuration for the
// specified profile.
func getSDSConfig(profile string) (map[string]any, error) {
	switch profile {
	case "istio":
		// https://istio.io/latest/docs/ops/integrations/spire/#spiffe-federation
		return map[string]any{
			"enabled":               true,
			"defaultSVIDName":       "default",
			"defaultBundleName":     "null",
			"defaultAllBundlesName": "ROOTCA",
		}, nil
	case "kubernetes":
		// https://github.com/spiffe/spire/blob/main/doc/spire_agent.md#sds-configuration
		return map[string]any{
			"enabled":               true,
			"defaultSVIDName":       "default",
			"defaultBundleName":     "ROOTCA",
			"defaultAllBundlesName": "ALL",
		}, nil
	default:
		return nil, fmt.Errorf("an unknown profile was specified: %s", profile)
	}
}
