package trustzone

import (
	"fmt"

	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_provider/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/cofide/cofidectl/internal/pkg/trustprovider"

	"gopkg.in/yaml.v3"
)

type TrustZone struct {
	TrustZoneProto      *trust_zone_proto.TrustZone
	TrustProvider       *trustprovider.TrustProvider
	AttestationPolicies []string // keys for the policies which are defined in the config file
	Federations         []string // keys for federated trust zones
}

func NewTrustZone(trustZone *trust_zone_proto.TrustZone) *TrustZone {
	return &TrustZone{
		TrustZoneProto: trustZone,
		TrustProvider:  trustprovider.NewTrustProvider(trustZone.TrustProvider.Kind),
	}
}

func (tz *TrustZone) MarshalYAML() (interface{}, error) {
	yamlMap := make(map[string]interface{})

	yamlMap["name"] = tz.TrustZoneProto.Name
	yamlMap["trust_domain"] = tz.TrustZoneProto.TrustDomain
	yamlMap["cluster"] = tz.TrustZoneProto.KubernetesCluster
	yamlMap["context"] = tz.TrustZoneProto.KubernetesContext
	yamlMap["trust_providers"] = tz.TrustProvider.Kind
	yamlMap["attestation_policies"] = tz.AttestationPolicies
	yamlMap["federations"] = tz.Federations

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
	tz.TrustZoneProto.KubernetesContext = tempMap["context"].(string)

	tz.TrustProvider = trustprovider.NewTrustProvider(tempMap["trust_providers"].(string))
	tz.TrustZoneProto.TrustProvider = &trust_provider_proto.TrustProvider{Kind: tz.TrustProvider.Kind}

	ap := tempMap["attestation_policies"].([]interface{})
	tz.AttestationPolicies = make([]string, len(ap))
	for i, v := range ap {
		tz.AttestationPolicies[i] = fmt.Sprint(v)
	}

	fed := tempMap["federations"].([]interface{})
	tz.Federations = make([]string, len(fed))
	for i, v := range fed {
		tz.Federations[i] = fmt.Sprint(v)
	}

	return nil
}
