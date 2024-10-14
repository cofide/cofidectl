package trustprovider

import (
	"fmt"

	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_provider/v1"
)

const (
	KubernetesTrustProvider string = "k8s"
	kubernetesPsat          string = "k8sPsat"
)

type TrustProvider struct {
	Name         string                    `yaml:"name"`
	Kind         string                    `yaml:"kind"`
	AgentConfig  TrustProviderAgentConfig  `yaml:"agentConfig"`
	ServerConfig TrustProviderServerConfig `yaml:"serverConfig"`
}

type TrustProviderAgentConfig struct {
	WorkloadAttestor        string                 `yaml:"workloadAttestor"`
	WorkloadAttestorEnabled bool                   `yaml:"workloadAttestorEnabled"`
	WorkloadAttestorConfig  map[string]interface{} `yaml:"workloadAttestorConfig"`
	NodeAttestor            string                 `yaml:"nodeAttestor"`
	NodeAttestorEnabled     bool                   `yaml:"nodeAttestorEnabled"`
}

type TrustProviderServerConfig struct {
	NodeAttestor        string                 `yaml:"nodeAttestor"`
	NodeAttestorEnabled bool                   `yaml:"nodeAttestorEnabled"`
	NodeAttestorConfig  map[string]interface{} `yaml:"nodeAttestorConfig"`
}

func GetTrustProvider(profile string) (*trust_provider_proto.TrustProvider, error) {
	switch profile {
	case "kubernetes":
		{
			tp := trust_provider_proto.TrustProvider{
				Name: "kubernetes",
				Kind: "k8s",
				AgentConfig: &trust_provider_proto.TrustProviderAgentConfig{
					WorkloadAttestor:        KubernetesTrustProvider,
					WorkloadAttestorEnabled: true,
					WorkloadAttestorConfig: &trust_provider_proto.WorkloadAttestorConfig{
						Enabled:                     true,
						SkipKubeletVerification:     true,
						DisableContainerSelectors:   false,
						UseNewContainerLocator:      false,
						VerboseContainerLocatorLogs: false,
					},
					NodeAttestor:        kubernetesPsat,
					NodeAttestorEnabled: true,
				},
				ServerConfig: &trust_provider_proto.TrustProviderServerConfig{
					NodeAttestor:        kubernetesPsat,
					NodeAttestorEnabled: true,
					NodeAttestorConfig: &trust_provider_proto.NodeAttestorConfig{
						Enabled:                 true,
						ServiceAccountAllowList: []string{"spire:spire-agent"},
						Audience:                []string{"spire-server"},
						AllowedNodeLabelKeys:    []string{},
						AllowedPodLabelKeys:     []string{},
					},
				},
			}
			return &tp, nil
		}
	default:
		return nil, fmt.Errorf("an unknown profile was specified")
	}
}
