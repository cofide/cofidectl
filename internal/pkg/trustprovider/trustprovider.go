package trustprovider

import (
	"fmt"

	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_provider/v1alpha1"
)

const (
	KubernetesTrustProvider string = "k8s"
	kubernetesPsat          string = "k8sPsat"
)

type TrustProvider struct {
	Name         string
	Kind         string
	AgentConfig  TrustProviderAgentConfig
	ServerConfig TrustProviderServerConfig
	Proto        *trust_provider_proto.TrustProvider
}

func NewTrustProvider(kind string) (*TrustProvider, error) {
	tp := &TrustProvider{
		Kind: kind,
	}
	if err := tp.GetValues(); err != nil {
		return nil, err
	}
	return tp, nil
}

func (tp *TrustProvider) GetValues() error {
	switch tp.Kind {
	case "kubernetes":
		tp.AgentConfig = TrustProviderAgentConfig{
			WorkloadAttestor:        KubernetesTrustProvider,
			WorkloadAttestorEnabled: true,
			WorkloadAttestorConfig: map[string]interface{}{
				"enabled":                     true,
				"skipKubeletVerification":     true,
				"disableContainerSelectors":   false,
				"useNewContainerLocator":      false,
				"verboseContainerLocatorLogs": false,
			},
			NodeAttestor:        kubernetesPsat,
			NodeAttestorEnabled: true,
		}
		tp.ServerConfig = TrustProviderServerConfig{
			NodeAttestor:        kubernetesPsat,
			NodeAttestorEnabled: true,
			NodeAttestorConfig: map[string]interface{}{
				"enabled":                 true,
				"serviceAccountAllowList": []string{"spire:spire-agent"},
				"audience":                []string{"spire-server"},
				"allowedNodeLabelKeys":    []string{},
				"allowedPodLabelKeys":     []string{},
			},
		}
	default:
		return fmt.Errorf("an unknown profile was specified")
	}
	return nil
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
