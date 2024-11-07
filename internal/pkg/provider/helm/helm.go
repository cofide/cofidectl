// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"fmt"
	"os"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/provider"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
)

const (
	SPIRERepositoryName = "spire"

	SPIREChartName        = "spire"
	SPIREChartVersion     = "0.21.0"
	SPIRECRDsChartName    = "spire-crds"
	SPIRECRDsChartVersion = "0.4.0"

	SPIRENamespace = "spire"
)

// HelmSPIREProvider implements a Helm-based installer for the Cofide stack. It uses the SPIFFE/SPIRE project's own
// helm-charts-hardened Helm chart to install a SPIRE stack to a given Kubernetes context, making use of the Cofide
// API concepts and abstractions
type HelmSPIREProvider struct {
	settings         *cli.EnvSettings
	cfg              *action.Configuration
	SPIREVersion     string
	SPIRECRDsVersion string
	spireValues      map[string]interface{}
	spireCRDsValues  map[string]interface{}
	trustZone        *trust_zone_proto.TrustZone
}

func NewHelmSPIREProvider(trustZone *trust_zone_proto.TrustZone, spireValues, spireCRDsValues map[string]interface{}) (*HelmSPIREProvider, error) {
	settings := cli.New()
	settings.KubeContext = trustZone.GetKubernetesContext()

	prov := &HelmSPIREProvider{
		settings:         settings,
		SPIREVersion:     SPIREChartVersion,
		SPIRECRDsVersion: SPIRECRDsChartVersion,
		spireValues:      spireValues,
		spireCRDsValues:  spireCRDsValues,
		trustZone:        trustZone,
	}

	var err error
	prov.cfg, err = prov.initActionConfig()
	if err != nil {
		return nil, err
	}

	return prov, nil
}

// Execute creates a provider status channel and performs the Helm chart installations.
func (h *HelmSPIREProvider) Execute() (<-chan provider.ProviderStatus, error) {
	statusCh := make(chan provider.ProviderStatus)

	h.installChart(statusCh)

	return statusCh, nil
}

// install installs the Cofide-enabled SPIRE stack to the selected Kubernetes context
// and updates the status channel accordingly.
func (h *HelmSPIREProvider) installChart(statusCh chan provider.ProviderStatus) {
	go func() {
		defer close(statusCh)

		statusCh <- provider.ProviderStatus{Stage: "Installing", Message: fmt.Sprintf("Installing SPIRE CRDs for %s to cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())}
		_, err := h.installSPIRECRDs()
		if err != nil {
			statusCh <- provider.ProviderStatus{Stage: "Installing", Message: fmt.Sprintf("Failed to install SPIRE CRDs for %s to cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster()), Done: true, Error: err}
			return
		}

		statusCh <- provider.ProviderStatus{Stage: "Installing", Message: fmt.Sprintf("Installing SPIRE chart for %s to cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())}
		_, err = h.installSPIRE()
		if err != nil {
			statusCh <- provider.ProviderStatus{Stage: "Installing", Message: fmt.Sprintf("Failed to install SPIRE chart for %s to cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster()), Done: true, Error: err}
			return
		}

		statusCh <- provider.ProviderStatus{Stage: "Installed", Message: fmt.Sprintf("Installation completed for %s on cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster()), Done: true}
	}()
}

func (h *HelmSPIREProvider) ExecuteUpgrade(postInstall bool) (<-chan provider.ProviderStatus, error) {
	statusCh := make(chan provider.ProviderStatus)

	// differentiate between a post-installation upgrade (ie configuration) and a full upgrade
	if postInstall {
		h.postInstallUpgrade(statusCh)
	} else {
		h.upgradeChart(statusCh)
	}

	return statusCh, nil
}

func (h *HelmSPIREProvider) postInstallUpgrade(statusCh chan provider.ProviderStatus) {
	go func() {
		defer close(statusCh)

		statusCh <- provider.ProviderStatus{Stage: "Configuring", Message: fmt.Sprintf("Applying post-installation configuration for %s to cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())}
		_, err := h.upgradeSPIRE()
		if err != nil {
			statusCh <- provider.ProviderStatus{Stage: "Configuring", Message: fmt.Sprintf("Failed to apply post-installation configuration for %s to cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster()), Done: true, Error: err}
			return
		}

		statusCh <- provider.ProviderStatus{Stage: "Configured", Message: fmt.Sprintf("Post-installation configuration completed for %s on cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster()), Done: true}
	}()
}

func (h *HelmSPIREProvider) upgradeChart(statusCh chan provider.ProviderStatus) {
	go func() {
		defer close(statusCh)

		statusCh <- provider.ProviderStatus{Stage: "Upgrading", Message: fmt.Sprintf("Upgrading SPIRE chart for %s on cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())}
		_, err := h.upgradeSPIRE()
		if err != nil {
			statusCh <- provider.ProviderStatus{Stage: "Upgrading", Message: fmt.Sprintf("Failed to upgrade SPIRE chart for %s on cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster()), Done: true, Error: err}
			return
		}

		statusCh <- provider.ProviderStatus{Stage: "Upgraded", Message: fmt.Sprintf("Upgrade completed for %s on cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster()), Done: true}
	}()
}

func (h *HelmSPIREProvider) ExecuteUninstall() (<-chan provider.ProviderStatus, error) {
	statusCh := make(chan provider.ProviderStatus)

	h.uninstall(statusCh)

	return statusCh, nil
}

// uninstall uninstalls the Cofide-enabled SPIRE stack from the selected Kubernetes context
// and updates the status channel accordingly.
func (h *HelmSPIREProvider) uninstall(statusCh chan provider.ProviderStatus) {
	go func() {
		defer close(statusCh)

		statusCh <- provider.ProviderStatus{Stage: "Uninstalling", Message: fmt.Sprintf("Uninstalling SPIRE CRDs for %s from cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())}
		_, err := h.uninstallSPIRECRDs()
		if err != nil {
			statusCh <- provider.ProviderStatus{Stage: "Uninstalling", Message: fmt.Sprintf("Failed to uninstall SPIRE CRDs for %s from cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster()), Done: true, Error: err}
			return
		}

		statusCh <- provider.ProviderStatus{Stage: "Uninstalling", Message: fmt.Sprintf("Uninstalling SPIRE chart for %s from cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())}
		_, err = h.uninstallSPIRE()
		if err != nil {
			statusCh <- provider.ProviderStatus{Stage: "Uninstalling", Message: fmt.Sprintf("Failed to uninstall SPIRE chart for %s from cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster()), Done: true, Error: err}
			return
		}

		statusCh <- provider.ProviderStatus{Stage: "Uninstalled", Message: fmt.Sprintf("Uninstallation completed for %s on cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster()), Done: true}
	}()
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

func newInstall(cfg *action.Configuration, chart string, version string) *action.Install {
	install := action.NewInstall(cfg)
	install.Version = version
	install.ReleaseName = chart
	install.Namespace = SPIRENamespace
	install.CreateNamespace = true
	return install
}

func (h *HelmSPIREProvider) installSPIRE() (*release.Release, error) {
	client := newInstall(h.cfg, SPIREChartName, h.SPIREVersion)
	return installChart(h.cfg, client, SPIREChartName, h.settings, h.spireValues)
}

func (h *HelmSPIREProvider) installSPIRECRDs() (*release.Release, error) {
	client := newInstall(h.cfg, SPIRECRDsChartName, h.SPIRECRDsVersion)
	return installChart(h.cfg, client, SPIRECRDsChartName, h.settings, h.spireCRDsValues)
}

func installChart(cfg *action.Configuration, client *action.Install, chartName string, settings *cli.EnvSettings, values map[string]interface{}) (*release.Release, error) {
	alreadyInstalled, err := checkIfAlreadyInstalled(cfg, chartName)
	if err != nil {
		return nil, fmt.Errorf("cannot determine chart installation status: %s", err)
	}
	if alreadyInstalled {
		fmt.Printf("%v already installed", chartName)
		return nil, nil
	}

	options, err := client.ChartPathOptions.LocateChart(
		fmt.Sprintf("%s/%s", SPIRERepositoryName, chartName),
		settings,
	)
	if err != nil {
		return nil, err
	}

	cr, err := loader.Load(options)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Installing %v...", cr.Name())
	return client.Run(cr, values)
}

func newUpgrade(cfg *action.Configuration, version string) *action.Upgrade {
	upgrade := action.NewUpgrade(cfg)
	upgrade.Namespace = SPIRENamespace
	upgrade.Version = version
	upgrade.ReuseValues = true
	return upgrade
}

func (h *HelmSPIREProvider) upgradeSPIRE() (*release.Release, error) {
	client := newUpgrade(h.cfg, h.SPIREVersion)
	return upgradeChart(h.cfg, client, SPIREChartName, h.settings, h.spireValues)
}

func upgradeChart(cfg *action.Configuration, client *action.Upgrade, chartName string, settings *cli.EnvSettings, values map[string]interface{}) (*release.Release, error) {
	alreadyInstalled, err := checkIfAlreadyInstalled(cfg, chartName)
	if err != nil {
		return nil, fmt.Errorf("cannot determine chart installation status: %s", err)
	}

	if !alreadyInstalled {
		return nil, fmt.Errorf("%v not installed", chartName)
	}

	options, err := client.ChartPathOptions.LocateChart(
		fmt.Sprintf("%s/%s", SPIRERepositoryName, chartName),
		settings,
	)
	if err != nil {
		return nil, err
	}

	chart, err := loader.Load(options)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Upgrading %v...", chart.Name())
	return client.Run(chartName, chart, values)
}

func newUninstall(cfg *action.Configuration) *action.Uninstall {
	uninstall := action.NewUninstall(cfg)
	return uninstall
}

func (h *HelmSPIREProvider) uninstallSPIRE() (*release.UninstallReleaseResponse, error) {
	client := newUninstall(h.cfg)
	return uninstallChart(h.cfg, client, SPIREChartName)
}

func (h *HelmSPIREProvider) uninstallSPIRECRDs() (*release.UninstallReleaseResponse, error) {
	client := newUninstall(h.cfg)
	return uninstallChart(h.cfg, client, SPIRECRDsChartName)
}

func uninstallChart(cfg *action.Configuration, client *action.Uninstall, chartName string) (*release.UninstallReleaseResponse, error) {
	alreadyInstalled, err := checkIfAlreadyInstalled(cfg, chartName)
	if err != nil {
		return nil, fmt.Errorf("cannot determine chart installation status: %s", err)
	}

	if !alreadyInstalled {
		return nil, fmt.Errorf("%v not installed", chartName)
	}

	fmt.Printf("Uninstalling %v...", chartName)
	return client.Run(chartName)
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
