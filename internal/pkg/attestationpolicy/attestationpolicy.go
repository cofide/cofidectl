package attestationpolicy

import (
	"fmt"

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

func (ap *AttestationPolicy) GetHelmConfig(source cofidectl_plugin.DataSource) (map[string]interface{}, error) {
	var clusterSPIFFEID = make(map[string]interface{})
	switch ap.AttestationPolicyProto.Kind {
	case attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_ANNOTATED:
		clusterSPIFFEID["podSelector"] = map[string]interface{}{
			"matchLabels": map[string]interface{}{
				ap.AttestationPolicyProto.PodKey: ap.AttestationPolicyProto.PodValue,
			},
		}
	case attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_NAMESPACE:
		clusterSPIFFEID["namespaceSelector"] = map[string]interface{}{
			"matchExpressions": []map[string]interface{}{
				{
					"key":      "kubernetes.io/metadata.name",
					"operator": "In",
					"values":   []string{ap.AttestationPolicyProto.Namespace},
				},
			},
		}
	default:
		return nil, fmt.Errorf("unexpected attestation policy kind %s", attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_NAMESPACE)
	}

	if len(ap.AttestationPolicyProto.FederatesWith) > 0 {
		// Convert from trust zones to trust domains.
		federatesWith := []string{}
		for _, tzName := range ap.AttestationPolicyProto.FederatesWith {
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

func GetAttestationPolicyKind(kind string) (attestation_policy_proto.AttestationPolicyKind, error) {
	switch kind {
	case "annotated", "ATTESTATION_POLICY_KIND_ANNOTATED":
		return attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_ANNOTATED, nil
	case "namespace", "ATTESTATION_POLICY_KIND_NAMESPACE":
		return attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_NAMESPACE, nil
	}

	return attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_UNSPECIFIED, fmt.Errorf("unknown attestation policy kind %s", kind)
}

func GetAttestationPolicyKindString(kind attestation_policy_proto.AttestationPolicyKind) (string, error) {
	switch kind {
	case attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_ANNOTATED:
		return Annotated, nil
	case attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_NAMESPACE:
		return Namespace, nil
	}

	// TODO: Update error message.
	return Unspecified, fmt.Errorf("unknown attestation policy kind %s", kind)
}
