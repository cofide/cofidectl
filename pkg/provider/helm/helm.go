// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/provision_plugin/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/pkg/plugin/provision"

	"github.com/gofrs/flock"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
)

const (
	SPIRERepositoryName = "spire"
	SPIRERepositoryUrl  = "https://spiffe.github.io/helm-charts-hardened/"

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
	ctx              context.Context
	settings         *cli.EnvSettings
	cfg              *action.Configuration
	SPIREVersion     string
	SPIRECRDsVersion string
	spireValues      map[string]interface{}
	spireCRDsValues  map[string]interface{}
	trustZone        *trust_zone_proto.TrustZone
}

func NewHelmSPIREProvider(ctx context.Context, trustZone *trust_zone_proto.TrustZone, spireValues, spireCRDsValues map[string]interface{}) (*HelmSPIREProvider, error) {
	settings := cli.New()
	settings.KubeContext = trustZone.GetKubernetesContext()

	prov := &HelmSPIREProvider{
		ctx:              ctx,
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

// AddRepository adds the SPIRE Helm repository to the local repositories.yaml.
// The action is performed synchronously and status is streamed through the provided status channel.
// This function should be called once, not per-trust zone.
// The SPIRE Helm repository is added to the local repositories.yaml, locking the repositories.lock
// file while making changes.
func (h *HelmSPIREProvider) AddRepository(statusCh chan<- *provisionpb.Status) {
	statusCh <- provision.StatusOk("Preparing", "Adding SPIRE Helm repo")
	lockCtx, cancel := context.WithTimeout(h.ctx, 30*time.Second)
	defer cancel()
	err := runWithFileLock(lockCtx, h.settings.RepositoryConfig, func() error {
		f, err := repo.LoadFile(h.settings.RepositoryConfig)
		if err != nil {
			if err := repo.NewFile().WriteFile(h.settings.RepositoryConfig, 0600); err != nil {
				return fmt.Errorf("failed to create repositories file: %w", err)
			}

			f, err = repo.LoadFile(h.settings.RepositoryConfig)
			if err != nil {
				return fmt.Errorf("failed to load repositories file: %w", err)
			}
		}

		entry := &repo.Entry{
			Name: SPIRERepositoryName,
			URL:  SPIRERepositoryUrl,
		}

		chartRepo, err := repo.NewChartRepository(entry, getter.All(h.settings))
		if err != nil {
			return fmt.Errorf("failed to create chart repo: %w", err)
		}

		chartRepo.CachePath = h.settings.RepositoryCache
		if _, err = chartRepo.DownloadIndexFile(); err != nil {
			return fmt.Errorf("failed to download index file: %w", err)
		}

		f.Update(entry)
		if err = f.WriteFile(h.settings.RepositoryConfig, 0600); err != nil {
			return fmt.Errorf("failed to write repositories file: %w", err)
		}

		return nil
	})

	if err != nil {
		statusCh <- provision.StatusError("Preparing", "Failed to add SPIRE Helm repo", err)
	} else {
		statusCh <- provision.StatusDone("Prepared", "Added SPIRE Helm repo")
	}
}

// runWithFileLock attempts to lock a file, and if successful calls `f` with the lock held.
func runWithFileLock(ctx context.Context, filePath string, f func() error) error {
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("mkdirall: %w", err)
	}

	fileLock := flock.New(lockPath(filePath))

	locked, err := fileLock.TryLockContext(ctx, time.Second)
	if err == nil && locked {
		defer func() {
			_ = fileLock.Unlock()
		}()
	}
	if err != nil {
		return fmt.Errorf("try lock: %w", err)
	}

	return f()
}

func lockPath(filePath string) string {
	repoFileExt := filepath.Ext(filePath)
	if len(repoFileExt) > 0 && len(repoFileExt) < len(filePath) {
		return strings.TrimSuffix(filePath, repoFileExt) + ".lock"
	} else {
		return filePath + ".lock"
	}
}

// Execute installs the SPIRE Helm stack to the selected Kubernetes context.
// The action is performed synchronously and status is streamed through the provided status channel.
func (h *HelmSPIREProvider) Execute(statusCh chan<- *provisionpb.Status) {
	msg := fmt.Sprintf("Installing SPIRE CRDs for %s to cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
	statusCh <- provision.StatusOk("Installing", msg)
	_, err := h.installSPIRECRDs()
	if err != nil {
		msg := fmt.Sprintf("Failed to install SPIRE CRDs for %s to cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
		statusCh <- provision.StatusError("Installing", msg, err)
		return
	}

	msg = fmt.Sprintf("Installing SPIRE chart for %s to cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
	statusCh <- provision.StatusOk("Installing", msg)
	_, err = h.installSPIRE()
	if err != nil {
		msg := fmt.Sprintf("Failed to install SPIRE chart for %s to cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
		statusCh <- provision.StatusError("Installing", msg, err)
		return
	}

	msg = fmt.Sprintf("Installation completed for %s on cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
	statusCh <- provision.StatusDone("Installed", msg)
}

// ExecutePostInstallUpgrade upgrades the SPIRE stack to the selected Kubernetes context.
// The action is performed synchronously and status is streamed through the provided status channel.
func (h *HelmSPIREProvider) ExecutePostInstallUpgrade(statusCh chan<- *provisionpb.Status) {
	msg := fmt.Sprintf("Applying post-installation configuration for %s to cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
	statusCh <- provision.StatusOk("Configuring", msg)
	_, err := h.upgradeSPIRE()
	if err != nil {
		msg := fmt.Sprintf("Failed to apply post-installation configuration for %s to cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
		statusCh <- provision.StatusError("Configuring", msg, err)
		return
	}

	msg = fmt.Sprintf("Post-installation configuration completed for %s on cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
	statusCh <- provision.StatusDone("Configured", msg)
}

// ExecuteUpgrade upgrades the SPIRE stack to the selected Kubernetes context.
// The action is performed synchronously and status is streamed through the provided status channel.
func (h *HelmSPIREProvider) ExecuteUpgrade(statusCh chan<- *provisionpb.Status) {
	msg := fmt.Sprintf("Upgrading SPIRE chart for %s on cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
	statusCh <- provision.StatusOk("Upgrading", msg)
	_, err := h.upgradeSPIRE()
	if err != nil {
		msg := fmt.Sprintf("Failed to upgrade SPIRE chart for %s on cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
		statusCh <- provision.StatusError("Upgrading", msg, err)
		return
	}

	msg = fmt.Sprintf("Upgrade completed for %s on cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
	statusCh <- provision.StatusDone("Upgraded", msg)
}

// ExecuteUninstall uninstalls the SPIRE stack from the selected Kubernetes context.
// The action is performed synchronously and status is streamed through the provided status channel.
func (h *HelmSPIREProvider) ExecuteUninstall(statusCh chan<- *provisionpb.Status) {
	msg := fmt.Sprintf("Uninstalling SPIRE CRDs for %s from cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
	statusCh <- provision.StatusOk("Uninstalling", msg)
	_, err := h.uninstallSPIRECRDs()
	if err != nil {
		msg := fmt.Sprintf("Failed to uninstall SPIRE CRDs for %s from cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
		statusCh <- provision.StatusError("Uninstalling", msg, err)
		return
	}

	msg = fmt.Sprintf("Uninstalling SPIRE chart for %s from cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
	statusCh <- provision.StatusOk("Uninstalling", msg)
	_, err = h.uninstallSPIRE()
	if err != nil {
		msg := fmt.Sprintf("Failed to uninstall SPIRE chart for %s from cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
		statusCh <- provision.StatusError("Uninstalling", msg, err)
		return
	}

	msg = fmt.Sprintf("Uninstallation completed for %s on cluster %s", h.trustZone.Name, h.trustZone.GetKubernetesCluster())
	statusCh <- provision.StatusDone("Uninstalled", msg)
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
	return installChart(h.ctx, h.cfg, client, SPIREChartName, h.settings, h.spireValues)
}

func (h *HelmSPIREProvider) installSPIRECRDs() (*release.Release, error) {
	client := newInstall(h.cfg, SPIRECRDsChartName, h.SPIRECRDsVersion)
	return installChart(h.ctx, h.cfg, client, SPIRECRDsChartName, h.settings, h.spireCRDsValues)
}

func installChart(ctx context.Context, cfg *action.Configuration, client *action.Install, chartName string, settings *cli.EnvSettings, values map[string]interface{}) (*release.Release, error) {
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
	return client.RunWithContext(ctx, cr, values)
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
	return upgradeChart(h.ctx, h.cfg, client, SPIREChartName, h.settings, h.spireValues)
}

func upgradeChart(ctx context.Context, cfg *action.Configuration, client *action.Upgrade, chartName string, settings *cli.EnvSettings, values map[string]interface{}) (*release.Release, error) {
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
	return client.RunWithContext(ctx, chartName, chart, values)
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
