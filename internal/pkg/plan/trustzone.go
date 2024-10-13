package plan

import (
	"log/slog"

	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_provider/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"

	"gopkg.in/yaml.v3"
)

type TrustZone struct {
	TrustZoneProto *trust_zone_proto.TrustZone `yaml:"trustZone"`
	TrustProvider  *TrustProvider              `yaml:"trustProvider"`

	AttestationPolicies []AttestationPolicy `yaml:"attestationPolicies"`
}

func NewTrustZone(trustZone *trust_zone_proto.TrustZone) *TrustZone {
	return &TrustZone{
		TrustZoneProto: trustZone,
		TrustProvider:  NewTrustProvider(trustZone.TrustProvider.Kind),
	}
}

func (tz *TrustZone) MarshalYAML() (interface{}, error) {
	yamlMap := make(map[string]interface{})

	yamlMap["name"] = tz.TrustZoneProto.Name
	yamlMap["trust_domain"] = tz.TrustZoneProto.TrustDomain
	yamlMap["cluster"] = tz.TrustZoneProto.KubernetesCluster
	yamlMap["context"] = tz.TrustZoneProto.KubernetesContext

	yamlMap["trust_providers"] = tz.TrustProvider.Kind

	return yamlMap, nil
}

func (tz *TrustZone) UnmarshalYAML(value *yaml.Node) error {
	tempMap := make(map[string]interface{})

	// Unmarshal the YAML into the temporary map
	if err := value.Decode(&tempMap); err != nil {
		return err
	}

	if tz.TrustZoneProto == nil {
		tz.TrustZoneProto = &trust_zone_proto.TrustZone{}
	}

	tz.TrustZoneProto.Name = tempMap["name"].(string)
	tz.TrustZoneProto.KubernetesCluster = tempMap["cluster"].(string)
	tz.TrustZoneProto.KubernetesContext = tempMap["context"].(string)
	tz.TrustZoneProto.TrustDomain = tempMap["trust_domain"].(string)
	tz.TrustProvider = NewTrustProvider(tempMap["trust_providers"].(string))
	tz.TrustZoneProto.TrustProvider = &trust_provider_proto.TrustProvider{Kind: tz.TrustProvider.Kind}
	tz.TrustZoneProto.KubernetesContext = tempMap["context"].(string)

	slog.Info("unmarshal", "proto", tz.TrustZoneProto)

	return nil
}
