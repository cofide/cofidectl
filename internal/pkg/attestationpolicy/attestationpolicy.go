package attestationpolicy

import (
	"fmt"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	"gopkg.in/yaml.v3"
)

type AttestationPolicy struct {
	AttestationPolicyProto *attestation_policy_proto.AttestationPolicy `yaml:"attestationPolicy"`
}

type AttestationPolicyKind string

const (
	Annotated   = "annotated"
	Cluster     = "cluster"
	Namespace   = "namespace"
	Unspecified = "unspecified"
)

type AttestationPolicyOpts struct {
	// Annotated
	PodKey   string
	PodValue string

	// Namespace
	Namespace string
}

func NewAttestationPolicy(attestationPolicy *attestation_policy_proto.AttestationPolicy) *AttestationPolicy {
	return &AttestationPolicy{
		AttestationPolicyProto: attestationPolicy,
	}
}

func (ap *AttestationPolicy) MarshalYAML() (interface{}, error) {
	yamlMap := make(map[string]interface{})

	kind, err := GetAttestationPolicyKindString(ap.AttestationPolicyProto.Kind.String())
	if err != nil {
		return nil, err
	}

	yamlMap["name"] = ap.AttestationPolicyProto.Name
	yamlMap["kind"] = kind
	yamlMap["namespace"] = ap.AttestationPolicyProto.Namespace
	yamlMap["pod_key"] = ap.AttestationPolicyProto.PodKey
	yamlMap["pod_value"] = ap.AttestationPolicyProto.PodValue

	return yamlMap, nil
}

func (ap *AttestationPolicy) UnmarshalYAML(value *yaml.Node) error {
	tempMap := make(map[string]interface{})
	if err := value.Decode(&tempMap); err != nil {
		return err
	}

	if ap.AttestationPolicyProto == nil {
		ap.AttestationPolicyProto = &attestation_policy_proto.AttestationPolicy{}
	}

	kind, err := GetAttestationPolicyKind(tempMap["kind"].(string))
	if err != nil {
		return err
	}

	ap.AttestationPolicyProto.Name = tempMap["name"].(string)
	ap.AttestationPolicyProto.Kind = kind
	ap.AttestationPolicyProto.Namespace = tempMap["namespace"].(string)
	ap.AttestationPolicyProto.PodKey = tempMap["pod_key"].(string)
	ap.AttestationPolicyProto.PodValue = tempMap["pod_value"].(string)

	return nil
}

func GetAttestationPolicyKind(kind string) (attestation_policy_proto.AttestationPolicyKind, error) {
	switch kind {
	case "annotated", "ATTESTATION_POLICY_KIND_ANNOTATED":
		return attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_ANNOTATED, nil
	case "cluster", "ATTESTATION_POLICY_KIND_CLUSTER":
		return attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_CLUSTER, nil
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
	case attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_CLUSTER.String():
		return Cluster, nil
	case attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_NAMESPACE.String():
		return Namespace, nil
	}

	// TODO: Update error message.
	return Unspecified, fmt.Errorf(fmt.Sprintf("unknown attestation policy kind %v", kind))
}
