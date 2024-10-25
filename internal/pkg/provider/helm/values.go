package helm

import (
	"encoding/json"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/cofide/cofidectl/internal/pkg/attestationpolicy"
	"github.com/cofide/cofidectl/internal/pkg/federation"
	"github.com/cofide/cofidectl/internal/pkg/trustzone"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
)

type HelmValuesGenerator struct {
	source    cofidectl_plugin.DataSource
	trustZone *trust_zone_proto.TrustZone
}

func NewHelmValuesGenerator(trustZone *trust_zone_proto.TrustZone, source cofidectl_plugin.DataSource) *HelmValuesGenerator {
	return &HelmValuesGenerator{
		trustZone: trustZone,
		source:    source,
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
		"global.spire.clusterName":              g.trustZone.KubernetesCluster,
		"global.spire.trustDomain":              g.trustZone.TrustDomain,
		"global.spire.recommendations.create":   true,
		"global.installAndUpgradeHooks.enabled": false,
		"global.deleteHooks.enabled":            false,
	}

	spireAgentValues := map[string]interface{}{
		`"spire-agent"."fullnameOverride"`: "spire-agent", // NOTE: https://github.com/cue-lang/cue/issues/358
		`"spire-agent"."logLevel"`:         "DEBUG",
		fmt.Sprintf(`"spire-agent"."nodeAttestor"."%s"."enabled"`, agentConfig.NodeAttestor):                              agentConfig.NodeAttestorEnabled,
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."disableContainerSelectors"`, agentConfig.WorkloadAttestor):   agentConfig.WorkloadAttestorConfig["disableContainerSelectors"],
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."enabled"`, agentConfig.WorkloadAttestor):                     agentConfig.WorkloadAttestorConfig["enabled"],
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."skipKubeletVerification"`, agentConfig.WorkloadAttestor):     agentConfig.WorkloadAttestorConfig["skipKubeletVerification"],
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."useNewContainerLocator"`, agentConfig.WorkloadAttestor):      agentConfig.WorkloadAttestorConfig["useNewContainerLocator"],
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."verboseContainerLocatorLogs"`, agentConfig.WorkloadAttestor): agentConfig.WorkloadAttestorConfig["verboseContainerLocatorLogs"],
		`"spire-agent"."server"."address"`: "spire-server.spire",
	}

	spireServerValues := map[string]interface{}{
		`"spire-server"."federation"."enabled"`:        true,
		`"spire-server"."service"."type"`:              "LoadBalancer",
		`"spire-server"."caKeyType"`:                   "rsa-2048",
		`"spire-server"."controllerManager"."enabled"`: true,
		`"spire-server"."caTTL"`:                       "12h",
		`"spire-server"."fullnameOverride"`:            "spire-server",
		`"spire-server"."logLevel"`:                    "DEBUG",
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."audience"`, serverConfig.NodeAttestor):                serverConfig.NodeAttestorConfig["audience"],
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."allowedPodLabelKeys"`, serverConfig.NodeAttestor):     serverConfig.NodeAttestorConfig["allowedPodLabelKeys"],
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."allowedNodeLabelKeys"`, serverConfig.NodeAttestor):    serverConfig.NodeAttestorConfig["allowedNodeLabelKeys"],
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."enabled"`, serverConfig.NodeAttestor):                 serverConfig.NodeAttestorConfig["enabled"],
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."serviceAccountAllowList"`, serverConfig.NodeAttestor): serverConfig.NodeAttestorConfig["serviceAccountAllowList"],
	}

	// add attestation policies as ClusterSPIFFEIDs to be reconcilced by spire-controller-manager
	if len(g.trustZone.AttestationPolicies) > 0 {
		spireServerValues[`"spire-server"."controllerManager"."identities"."clusterSPIFFEIDs"."default"."enabled"`] = false
		for _, binding := range g.trustZone.AttestationPolicies {
			policy, err := g.source.GetAttestationPolicy(binding.Policy)
			if err != nil {
				return nil, err
			}
			clusterSPIFFEIDs, err := attestationpolicy.NewAttestationPolicy(policy).GetHelmConfig(g.source, binding)
			if err != nil {
				return nil, err
			}
			spireServerValues[fmt.Sprintf(`"spire-server"."controllerManager"."identities"."clusterSPIFFEIDs"."%s"`, policy.Name)] = clusterSPIFFEIDs
		}
	} else {
		// defaults to true
		spireServerValues[`"spire-server"."controllerManager"."identities"."clusterSPIFFEIDs"."default"."enabled"`] = true
	}

	// add federations as clusterFederatedTrustDomains to be reconcilced by spire-controller-manager
	if len(g.trustZone.Federations) > 0 {
		spireServerValues[`"spire-server"."federation"."enabled"`] = true
		for _, fed := range g.trustZone.Federations {
			tz, err := g.source.GetTrustZone(fed.Right)
			if err != nil {
				return nil, err
			}
			spireServerValues[fmt.Sprintf(`"spire-server"."controllerManager"."identities"."clusterFederatedTrustDomains"."%s"`, fed.Right)] = federation.NewFederation(tz).GetHelmConfig()
		}
	}

	spiffeOIDCDiscoveryProviderValues := map[string]interface{}{
		`"spiffe-oidc-discovery-provider"."enabled"`: false,
	}

	spiffeCSIDriverValues := map[string]interface{}{
		`"spiffe-csi-driver"."fullnameOverride"`: "spiffe-csi-driver",
	}

	valuesMaps := []map[string]interface{}{
		globalValues,
		spireAgentValues,
		spireServerValues,
		spiffeOIDCDiscoveryProviderValues,
		spiffeCSIDriverValues,
	}

	ctx := cuecontext.New()
	combinedValuesCUE := ctx.CompileBytes([]byte{})

	for _, valuesMap := range valuesMaps {
		valuesCUE := ctx.CompileBytes([]byte{})

		for path, value := range valuesMap {
			valuesCUE = valuesCUE.FillPath(cue.ParsePath(path), value)
		}

		combinedValuesCUE = combinedValuesCUE.Unify(valuesCUE)
	}

	combinedValuesJSON, err := combinedValuesCUE.MarshalJSON()
	if err != nil {
		// TODO: Improve error messaging.
		return nil, err
	}

	var values map[string]interface{}
	err = json.Unmarshal([]byte(combinedValuesJSON), &values)
	if err != nil {
		// TODO: Improve error messaging.
		return nil, err
	}

	return values, nil
}
