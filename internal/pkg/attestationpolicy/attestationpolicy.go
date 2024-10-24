package attestationpolicy

import (
	"fmt"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
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

func (ap *AttestationPolicy) GetHelmConfig() map[string]interface{} {
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
		clusterSPIFFEID["enabled"] = "false"
	}

	return clusterSPIFFEID
}

func GetAttestationPolicyKind(kind string) (attestation_policy_proto.AttestationPolicyKind, error) {
	switch kind {
	case "annotated", "ATTESTATION_POLICY_KIND_ANNOTATED":
		return attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_ANNOTATED, nil
	case "namespace", "ATTESTATION_POLICY_KIND_NAMESPACE":
		return attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_NAMESPACE, nil
	}

	// TODO: Update error message.
	return attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_UNSPECIFIED, fmt.Errorf(fmt.Sprintf("unknown attestation policy kind %v", kind))
}

func GetAttestationPolicyKindString(kind string) (string, error) {
	switch kind {
	case attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_ANNOTATED.String():
		return Annotated, nil
	case attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_NAMESPACE.String():
		return Namespace, nil
	}

	// TODO: Update error message.
	return Unspecified, fmt.Errorf(fmt.Sprintf("unknown attestation policy kind %v", kind))
}
