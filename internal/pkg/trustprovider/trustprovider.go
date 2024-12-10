// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustprovider

import (
	"fmt"

	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_provider/v1alpha1"
)

const (
	KubernetesTrustProvider string = "k8s"
	kubernetesPsat          string = "k8sPsat"
)

type TrustProviderAgentConfig struct {
	WorkloadAttestor        string         `yaml:"workloadAttestor"`
	WorkloadAttestorEnabled bool           `yaml:"workloadAttestorEnabled"`
	WorkloadAttestorConfig  map[string]any `yaml:"workloadAttestorConfig"`
	NodeAttestor            string         `yaml:"nodeAttestor"`
	NodeAttestorEnabled     bool           `yaml:"nodeAttestorEnabled"`
}

type TrustProviderServerConfig struct {
	NodeAttestor        string         `yaml:"nodeAttestor"`
	NodeAttestorEnabled bool           `yaml:"nodeAttestorEnabled"`
	NodeAttestorConfig  map[string]any `yaml:"nodeAttestorConfig"`
}

type SDSConfig struct {
	Enabled               bool   `yaml:"enabled"`
	DefaultSVIDName       string `yaml:"defaultSVIDName"`
	DefaultBundleName     string `yaml:"defaultBundleName"`
	DefaultAllBundlesName string `yaml:"defaultAllBundlesName"`
}

type TrustProvider struct {
	Name         string
	Kind         string
	AgentConfig  TrustProviderAgentConfig
	ServerConfig TrustProviderServerConfig
	SDSConfig    map[string]any
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
			WorkloadAttestorConfig: map[string]any{
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
			NodeAttestorConfig: map[string]any{
				"enabled":                 true,
				"serviceAccountAllowList": []string{"spire:spire-agent"},
				"audience":                []string{"spire-server"},
				"allowedNodeLabelKeys":    []string{},
				"allowedPodLabelKeys":     []string{},
			},
		}
		// Uses the Istio recommended values by default.
		// https://istio.io/latest/docs/ops/integrations/spire/#spiffe-federation
		tp.SDSConfig = map[string]any{
			"enabled":               true,
			"defaultSVIDName":       "default",
			"defaultBundleName":     "null",
			"defaultAllBundlesName": "ROOTCA",
		}
	default:
		return fmt.Errorf("an unknown trust provider profile was specified: %s", tp.Kind)
	}
	return nil
}
