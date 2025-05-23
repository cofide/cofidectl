// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustprovider

import (
	"errors"
	"fmt"

	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_provider/v1alpha1"
)

const (
	KubernetesTrustProvider string = "k8s"
	kubernetesPSAT          string = "k8sPSAT"
)

type TrustProvider struct {
	Kind         string
	AgentConfig  TrustProviderAgentConfig
	ServerConfig TrustProviderServerConfig
}

func NewTrustProvider(tpp *trust_provider_proto.TrustProvider) (*TrustProvider, error) {
	if tpp == nil {
		return nil, errors.New("trust provider cannot be nil")
	}

	tp := &TrustProvider{
		Kind: tpp.GetKind(),
	}
	if err := tp.getValues(); err != nil {
		return nil, err
	}
	return tp, nil
}

func (tp *TrustProvider) getValues() error {
	switch tp.Kind {
	case "kubernetes":
		tp.AgentConfig = TrustProviderAgentConfig{
			WorkloadAttestor: KubernetesTrustProvider,
			WorkloadAttestorConfig: map[string]any{
				"enabled":                   true,
				"disableContainerSelectors": true,
			},
			NodeAttestor: kubernetesPSAT,
		}
		tp.ServerConfig = TrustProviderServerConfig{
			NodeAttestor: kubernetesPSAT,
			NodeAttestorConfig: map[string]any{
				"enabled":  true,
				"audience": []string{"spire-server"},
			},
		}
	default:
		return fmt.Errorf("an unknown trust provider kind was specified: %s", tp.Kind)
	}
	return nil
}

type TrustProviderAgentConfig struct {
	WorkloadAttestor       string
	WorkloadAttestorConfig map[string]any
	NodeAttestor           string
}

type TrustProviderServerConfig struct {
	NodeAttestor       string
	NodeAttestorConfig map[string]any
}

// GetTrustProviderKindFromProfile returns the valid kind of trust provider for the
// corresponding profile.
func GetTrustProviderKindFromProfile(profile string) (string, error) {
	switch profile {
	case "istio", "kubernetes":
		return "kubernetes", nil
	default:
		return "", fmt.Errorf("failed to get trust provider kind, an invalid profile was specified: %s", profile)
	}
}
