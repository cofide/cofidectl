package attestationpolicy

import (
	"encoding/json"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	"gopkg.in/yaml.v3"
)

type AttestationPolicy struct {
	AttestationPolicyProto *attestation_policy_proto.AttestationPolicy `yaml:"attestationPolicy"`
}

type AttestationPolicyKind string

const (
	Annotated = "annotated"
	Namespace = "namespace"
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

	yamlMap["kind"] = ap.AttestationPolicyProto.Kind.String()
	yamlMap["options"] = map[string]interface{}{
		"namespace": ap.AttestationPolicyProto.Options.Namespace,
		"pod_key":   ap.AttestationPolicyProto.Options.PodKey,
		"pod_value": ap.AttestationPolicyProto.Options.PodValue,
	}

	return yamlMap, nil
}

func (ap *AttestationPolicy) UnmarshalYAML(value *yaml.Node) error {
	tempMap := make(map[string]interface{})
	if err := value.Decode(&tempMap); err != nil {
		return err
	}

	optionsJSON, err := json.Marshal(tempMap["options"])
	if err != nil {
		return err
	}

	attestationPolicyOpts := AttestationPolicyOpts{}
	if err := json.Unmarshal(optionsJSON, &attestationPolicyOpts); err != nil {
		return err
	}

	if ap.AttestationPolicyProto == nil {
		ap.AttestationPolicyProto = &attestation_policy_proto.AttestationPolicy{}
	}

	ap.AttestationPolicyProto.Options = &attestation_policy_proto.AttestationPolicyOptions{
		Namespace: attestationPolicyOpts.Namespace,
		PodKey:    attestationPolicyOpts.PodKey,
		PodValue:  attestationPolicyOpts.PodValue,
	}

	return nil
}
