// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"fmt"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/attestationpolicy"
	"github.com/cofide/cofidectl/internal/pkg/federation"
	"github.com/cofide/cofidectl/internal/pkg/trustprovider"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
)

const (
	k8sPSATSelectorType                = "k8s_psat"
	k8sPSATSPIREAgentNamespaceSelector = "agent_ns"
	k8sPSATSPIREAgentSASelector        = "agent_sa"
	k8sPSATClusterSelector             = "cluster"
	serverIdPath                       = "/spire/server"
	spireAgentNamespace                = "spire-system"
	spireAgentSA                       = "spire-agent"
)

type HelmValuesGenerator struct {
	source    datasource.DataSource
	trustZone *trust_zone_proto.TrustZone
	cluster   *clusterpb.Cluster
	values    map[string]any
}

type globalValues struct {
	deleteHooks                   bool
	installAndUpgradeHooksEnabled bool
	spireCASubject                caSubject
	spireClusterName              string
	spireJwtIssuer                string
	spireNamespacesCreate         bool
	spireRecommendationsEnabled   bool
	spireTrustDomain              string
}

type caSubject struct {
	commonName   string
	country      string
	organization string
}

type spireAgentValues struct {
	agentConfig      trustprovider.TrustProviderAgentConfig
	fullnameOverride string
	logLevel         string
	sdsConfig        map[string]any
}

type spireServerValues struct {
	caKeyType                string
	caTTL                    string
	controllerManagerEnabled bool
	enabled                  bool
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

func NewHelmValuesGenerator(trustZone *trust_zone_proto.TrustZone, cluster *clusterpb.Cluster, source datasource.DataSource, values map[string]any) *HelmValuesGenerator {
	return &HelmValuesGenerator{
		trustZone: trustZone,
		cluster:   cluster,
		source:    source,
		values:    values,
	}
}

func (g *HelmValuesGenerator) GenerateValues() (map[string]any, error) {
	tp, err := trustprovider.NewTrustProvider(g.cluster.GetTrustProvider())
	if err != nil {
		return nil, err
	}

	gv := globalValues{
		spireCASubject: caSubject{
			commonName:   "cofide.io",
			country:      "UK",
			organization: "Cofide",
		},
		spireClusterName:              g.cluster.GetName(),
		spireJwtIssuer:                g.trustZone.GetJwtIssuer(),
		spireNamespacesCreate:         true,
		spireRecommendationsEnabled:   true,
		spireTrustDomain:              g.trustZone.TrustDomain,
		installAndUpgradeHooksEnabled: false,
		deleteHooks:                   false,
	}

	globalValues, err := gv.generateValues()
	if err != nil {
		return nil, err
	}

	sdsConfig, err := getSDSConfig(g.cluster.GetProfile())
	if err != nil {
		return nil, err
	}

	sav := spireAgentValues{
		fullnameOverride: "spire-agent",
		logLevel:         "DEBUG",
		agentConfig:      tp.AgentConfig,
		sdsConfig:        sdsConfig,
	}
	spireAgentValues, err := sav.generateValues()
	if err != nil {
		return nil, err
	}

	spireServerEnabled := !g.cluster.GetExternalServer()

	ssv := spireServerValues{
		caKeyType:                "rsa-2048",
		caTTL:                    "12h",
		controllerManagerEnabled: true,
		enabled:                  spireServerEnabled,
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

	identities, err := getOrCreateNestedMap(controllerManager, "identities")
	if err != nil {
		return nil, fmt.Errorf("failed to get identities map from controllerManager: %w", err)
	}

	csids, err := getOrCreateNestedMap(identities, "clusterSPIFFEIDs")
	if err != nil {
		return nil, fmt.Errorf("failed to get clusterSPIFFEIDs map from identities: %w", err)
	}

	// Disables the default ClusterSPIFFEID CR.
	csids["default"] = map[string]any{
		"enabled": false,
	}

	cses, err := getOrCreateNestedMap(identities, "clusterStaticEntries")
	if err != nil {
		return nil, fmt.Errorf("failed to get clusterStaticEntries map from identities: %w", err)
	}

	filter := &datasourcepb.ListAPBindingsRequest_Filter{TrustZoneName: &g.trustZone.Name}
	bindings, err := g.source.ListAPBindings(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list attestation policy bindings: %w", err)
	}

	needSPIREAgentsStaticEntry := false

	// Adds the attestation policies as either ClusterSPIFFEID or ClusterStaticEntry CRs to be reconciled by the spire-controller-manager.
	for _, binding := range bindings {
		// nolint:staticcheck
		policy, err := g.source.GetAttestationPolicy(binding.Policy)
		if err != nil {
			return nil, err
		}

		if _, ok := policy.Policy.(*attestation_policy_proto.AttestationPolicy_Kubernetes); ok {
			clusterSPIFFEID, err := attestationpolicy.NewAttestationPolicy(policy).GetHelmConfig(g.source, binding)
			if err != nil {
				return nil, err
			}

			csids[policy.Name] = clusterSPIFFEID
		} else if _, ok := policy.Policy.(*attestation_policy_proto.AttestationPolicy_Static); ok {
			clusterStaticEntry, err := attestationpolicy.NewAttestationPolicy(policy).GetHelmConfig(g.source, binding)
			if err != nil {
				return nil, err
			}

			needSPIREAgentsStaticEntry = true

			cses[policy.Name] = clusterStaticEntry
		}
	}

	// Adds a ClusterStaticEntry CR for the SPIRE agents, so that the parent ID is deterministic.
	if needSPIREAgentsStaticEntry {
		cses["spire-agents"] = map[string]any{
			"parentID": fmt.Sprintf("spiffe://%s%s", g.trustZone.GetTrustDomain(), serverIdPath),
			"spiffeID": fmt.Sprintf("spiffe://%s/cluster/%s/spire/agents", g.trustZone.GetTrustDomain(), g.cluster.GetName()),
			"selectors": []string{
				fmt.Sprintf("%s:%s:%s", k8sPSATSelectorType, k8sPSATSPIREAgentNamespaceSelector, spireAgentNamespace),
				fmt.Sprintf("%s:%s:%s", k8sPSATSelectorType, k8sPSATSPIREAgentSASelector, spireAgentSA),
				fmt.Sprintf("%s:%s:%s", k8sPSATSelectorType, k8sPSATClusterSelector, g.cluster.GetName()),
			},
		}
	}

	federations, err := g.source.ListFederationsByTrustZone(g.trustZone.Name)
	if err != nil {
		return nil, err
	}
	// Adds the federations as ClusterFederatedTrustDomain CRs to be reconciled by the spire-controller-manager.
	if len(federations) > 0 {
		for _, fed := range federations {
			// nolint:staticcheck
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

				// nolint:staticcheck
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
		combinedValues, err = MergeMaps(combinedValues, g.values)
		if err != nil {
			return nil, err
		}
	}

	if g.cluster.ExtraHelmValues != nil {
		// TODO: Potentially retrieve Helm values as map[string]any directly.
		extraHelmValues := g.cluster.ExtraHelmValues.AsMap()
		combinedValues, err = MergeMaps(combinedValues, extraHelmValues)
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
				"caSubject":   g.spireCASubject.generateValues(),
				"clusterName": g.spireClusterName,
				"namespaces": map[string]any{
					"create": g.spireNamespacesCreate,
				},
				"recommendations": map[string]any{
					"enabled": g.spireRecommendationsEnabled,
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

// generateValues generates the global.spire.caSubject Helm values map.
func (c *caSubject) generateValues() map[string]any {
	return map[string]any{
		"country":      c.country,
		"organization": c.organization,
		"commonName":   c.commonName,
	}
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

	if s.agentConfig.WorkloadAttestor == "" {
		return nil, fmt.Errorf("agentConfig.WorkloadAttestor value is empty")
	}

	if s.agentConfig.WorkloadAttestorConfig == nil {
		return nil, fmt.Errorf("agentConfig.WorkloadAttestorConfig value is nil")
	}

	if len(s.agentConfig.WorkloadAttestorConfig) == 0 {
		return nil, fmt.Errorf("agentConfig.WorkloadAttestorConfig value is empty")
	}

	return map[string]any{
		"spire-agent": map[string]any{
			"fullnameOverride": s.fullnameOverride,
			"logLevel":         s.logLevel,
			"nodeAttestor": map[string]any{
				s.agentConfig.NodeAttestor: map[string]any{
					"enabled": true,
				},
			},
			"sds": s.sdsConfig,
			"workloadAttestors": map[string]any{
				s.agentConfig.WorkloadAttestor: s.agentConfig.WorkloadAttestorConfig,
			},
		},
	}, nil
}

// generateValues generates the spire-server Helm values map.
func (s *spireServerValues) generateValues() (map[string]any, error) {
	if !s.enabled {
		return map[string]any{
			"spire-server": map[string]any{
				"enabled": s.enabled,
			},
		}, nil
	}

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
			"enabled":   s.enabled,
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

// MergeMaps merges the source map into the destination map, returning a new merged map.
func MergeMaps(dest, src map[string]any) (map[string]any, error) {
	if src == nil {
		return nil, fmt.Errorf("source map is nil")
	}

	if dest == nil {
		return nil, fmt.Errorf("destination map is nil")
	}

	for key, value := range src {
		if srcMap, isSrcMap := value.(map[string]any); isSrcMap {
			if destMap, isDestMap := dest[key].(map[string]any); isDestMap {
				merged, err := MergeMaps(destMap, srcMap)
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
		return nil, fmt.Errorf("an invalid profile was specified: %s", profile)
	}
}
