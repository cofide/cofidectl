package attestationpolicy

import (
	"fmt"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/proto/ap_binding/v1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
)

type AttestationPolicy struct {
	AttestationPolicyProto *attestation_policy_proto.AttestationPolicy
}

const (
	Annotated   = "annotated"
	Cluster     = "cluster"
	Namespace   = "namespace"
	Unspecified = "unspecified"
)

func NewAttestationPolicy(attestationPolicy *attestation_policy_proto.AttestationPolicy) *AttestationPolicy {
	return &AttestationPolicy{
		AttestationPolicyProto: attestationPolicy,
	}
}

func (ap *AttestationPolicy) GetHelmConfig(source cofidectl_plugin.DataSource, binding *ap_binding_proto.APBinding) (map[string]interface{}, error) {
	var clusterSPIFFEID = make(map[string]interface{})
	switch policy := ap.AttestationPolicyProto.Policy.(type) {
	case *attestation_policy_proto.AttestationPolicy_Kubernetes:
		kubernetes := policy.Kubernetes
		if kubernetes.NamespaceSelector != nil {
			selector := getAPLabelSelectorHelmConfig(kubernetes.NamespaceSelector)
			if selector != nil {
				clusterSPIFFEID["namespaceSelector"] = selector
			}
		}
		if kubernetes.PodSelector != nil {
			selector := getAPLabelSelectorHelmConfig(kubernetes.PodSelector)
			if selector != nil {
				clusterSPIFFEID["podSelector"] = selector
			}
		}
	default:
		return nil, fmt.Errorf("unexpected attestation policy kind: %T", policy)
	}

	if len(binding.FederatesWith) > 0 {
		// Convert from trust zones to trust domains.
		federatesWith := []string{}
		for _, tzName := range binding.FederatesWith {
			if trustZone, err := source.GetTrustZone(tzName); err != nil {
				return nil, err
			} else {
				federatesWith = append(federatesWith, trustZone.TrustDomain)
			}
		}
		clusterSPIFFEID["federatesWith"] = federatesWith
	}

	return clusterSPIFFEID, nil
}

func getAPLabelSelectorHelmConfig(selector *attestation_policy_proto.APLabelSelector) map[string]interface{} {
	if len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0 {
		return nil
	}

	return map[string]interface{}{
		"matchLabels":      selector.MatchLabels,
		"matchExpressions": selector.MatchExpressions,
	}
}
