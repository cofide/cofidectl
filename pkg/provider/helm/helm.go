// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"os"

	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/storage/driver"
)

const (
	SPIREChartName = "spire"

	// Kubernetes namespace in which Helm charts and CRDs will be installed.
	SPIREManagementNamespace = "spire-mgmt"
)

// HelmSPIREProvider provides Helm-based checks against a SPIRE installation.
type HelmSPIREProvider struct {
	settings *cli.EnvSettings
	cfg      *action.Configuration
}

// HelmSPIREProviderOption is a function that configures a HelmSPIREProvider.
type HelmSPIREProviderOption func(*HelmSPIREProvider)

// WithKubeConfig sets the kubeconfig path.
func WithKubeConfig(kubeConfig string) HelmSPIREProviderOption {
	return func(p *HelmSPIREProvider) {
		if kubeConfig != "" {
			p.settings.KubeConfig = kubeConfig
		}
	}
}

func NewHelmSPIREProvider(cluster *clusterpb.Cluster, opts ...HelmSPIREProviderOption) (*HelmSPIREProvider, error) {
	settings := cli.New()
	settings.KubeContext = cluster.GetKubernetesContext()
	settings.SetNamespace(SPIREManagementNamespace)

	prov := &HelmSPIREProvider{
		settings: settings,
	}

	for _, opt := range opts {
		opt(prov)
	}

	var err error
	prov.cfg, err = prov.initActionConfig()
	if err != nil {
		return nil, err
	}

	return prov, nil
}

// CheckIfReachable returns no error if a Kubernetes cluster is reachable.
func (h *HelmSPIREProvider) CheckIfReachable() error {
	return h.cfg.KubeClient.IsReachable()
}

// CheckIfAlreadyInstalled returns true if the SPIRE chart has previously been installed.
func (h *HelmSPIREProvider) CheckIfAlreadyInstalled() (bool, error) {
	return checkIfAlreadyInstalled(h.cfg, SPIREChartName)
}

func DiscardLogger(format string, v ...any) {}

func (h *HelmSPIREProvider) initActionConfig() (*action.Configuration, error) {
	cfg := new(action.Configuration)
	err := cfg.Init(
		h.settings.RESTClientGetter(),
		h.settings.Namespace(),
		os.Getenv("HELM_DRIVER"),
		DiscardLogger,
	)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func checkIfAlreadyInstalled(cfg *action.Configuration, chartName string) (bool, error) {
	history := action.NewHistory(cfg)
	history.Max = 1
	ledger, err := history.Run(chartName)
	if err != nil && err != driver.ErrReleaseNotFound {
		return false, err
	}
	return len(ledger) > 0, nil
}

// IsClusterReachable returns no error if a Kubernetes cluster is reachable.
func IsClusterReachable(ctx context.Context, cluster *clusterpb.Cluster, kubeConfig string) error {
	prov, err := NewHelmSPIREProvider(cluster, WithKubeConfig(kubeConfig))
	if err != nil {
		return err
	}
	return prov.CheckIfReachable()
}

// IsClusterDeployed returns whether a cluster has been deployed, i.e. whether a SPIRE Helm release has been installed.
func IsClusterDeployed(ctx context.Context, cluster *clusterpb.Cluster, kubeConfig string) (bool, error) {
	prov, err := NewHelmSPIREProvider(cluster, WithKubeConfig(kubeConfig))
	if err != nil {
		return false, err
	}
	return prov.CheckIfAlreadyInstalled()
}
