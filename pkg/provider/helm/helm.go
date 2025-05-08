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

	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/provision_plugin/v1alpha1"
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

	// Kubernetes namespace in which Helm charts and CRDs will be installed.
	SPIREManagementNamespace = "spire-mgmt"
)

// Type assertion that HelmSPIREProvider implements the Provider interface.
var _ Provider = &HelmSPIREProvider{}

// HelmSPIREProvider implements a Helm-based installer for the Cofide stack. It uses the SPIFFE/SPIRE project's own
// helm-charts-hardened Helm chart to install a SPIRE stack to a given Kubernetes context, making use of the Cofide
// API concepts and abstractions
type HelmSPIREProvider struct {
	ctx              context.Context
	settings         *cli.EnvSettings
	cfg              *action.Configuration
	SPIREVersion     string
	SPIRECRDsVersion string
	spireValues      map[string]any
	spireCRDsValues  map[string]any
	cluster          *clusterpb.Cluster
}

func NewHelmSPIREProvider(ctx context.Context, cluster *clusterpb.Cluster, spireValues, spireCRDsValues map[string]any) (*HelmSPIREProvider, error) {
	settings := cli.New()
	settings.KubeContext = cluster.GetKubernetesContext()
	settings.SetNamespace(SPIREManagementNamespace)

	prov := &HelmSPIREProvider{
		ctx:              ctx,
		settings:         settings,
		SPIREVersion:     SPIREChartVersion,
		SPIRECRDsVersion: SPIRECRDsChartVersion,
		spireValues:      spireValues,
		spireCRDsValues:  spireCRDsValues,
		cluster:          cluster,
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
func (h *HelmSPIREProvider) AddRepository(statusCh chan<- *provisionpb.Status) error {
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
	return err
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
func (h *HelmSPIREProvider) Execute(statusCh chan<- *provisionpb.Status) error {
	sb := provision.NewStatusBuilder(h.cluster.GetTrustZone(), h.cluster.GetName())
	statusCh <- sb.Ok("Installing", "Installing SPIRE CRDs")
	_, err := h.installSPIRECRDs()
	if err != nil {
		statusCh <- sb.Error("Installing", "Failed to install SPIRE CRDs", err)
		return err
	}

	statusCh <- sb.Ok("Installing", "Installing SPIRE chart")
	_, err = h.installSPIRE()
	if err != nil {
		statusCh <- sb.Error("Installing", "Failed to install SPIRE chart", err)
		return err
	}

	statusCh <- sb.Done("Installed", "Installation completed")
	return nil
}

// ExecutePostInstallUpgrade upgrades the SPIRE stack to the selected Kubernetes context.
// The action is performed synchronously and status is streamed through the provided status channel.
func (h *HelmSPIREProvider) ExecutePostInstallUpgrade(statusCh chan<- *provisionpb.Status) error {
	sb := provision.NewStatusBuilder(h.cluster.GetTrustZone(), h.cluster.GetName())
	statusCh <- sb.Ok("Configuring", "Applying post-installation configuration")
	_, err := h.upgradeSPIRE()
	if err != nil {
		statusCh <- sb.Error("Configuring", "Failed to apply post-installation configuration", err)
		return err
	}

	statusCh <- sb.Done("Configured", "Post-installation configuration completed")
	return nil
}

// ExecuteUpgrade upgrades the SPIRE stack to the selected Kubernetes context.
// The action is performed synchronously and status is streamed through the provided status channel.
func (h *HelmSPIREProvider) ExecuteUpgrade(statusCh chan<- *provisionpb.Status) error {
	sb := provision.NewStatusBuilder(h.cluster.GetTrustZone(), h.cluster.GetName())
	statusCh <- sb.Ok("Upgrading", "Upgrading SPIRE chart")
	_, err := h.upgradeSPIRE()
	if err != nil {
		statusCh <- sb.Error("Upgrading", "Failed to upgrade SPIRE chart", err)
		return err
	}

	statusCh <- sb.Done("Upgraded", "Upgrade completed")
	return nil
}

// ExecuteUninstall uninstalls the SPIRE stack from the selected Kubernetes context.
// The action is performed synchronously and status is streamed through the provided status channel.
func (h *HelmSPIREProvider) ExecuteUninstall(statusCh chan<- *provisionpb.Status) error {
	sb := provision.NewStatusBuilder(h.cluster.GetTrustZone(), h.cluster.GetName())
	statusCh <- sb.Ok("Uninstalling", "Uninstalling SPIRE CRDs")
	_, err := h.uninstallSPIRECRDs()
	if err != nil {
		statusCh <- sb.Error("Uninstalling", "Failed to uninstall SPIRE CRDs", err)
		return err
	}

	statusCh <- sb.Ok("Uninstalling", "Uninstalling SPIRE chart")
	_, err = h.uninstallSPIRE()
	if err != nil {
		statusCh <- sb.Error("Uninstalling", "Failed to uninstall SPIRE chart", err)
		return err
	}

	statusCh <- sb.Done("Uninstalled", "Uninstallation completed")
	return nil
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
	install.Namespace = SPIREManagementNamespace
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

func installChart(ctx context.Context, cfg *action.Configuration, client *action.Install, chartName string, settings *cli.EnvSettings, values map[string]any) (*release.Release, error) {
	alreadyInstalled, err := checkIfAlreadyInstalled(cfg, chartName)
	if err != nil {
		return nil, fmt.Errorf("cannot determine chart installation status: %s", err)
	}
	if alreadyInstalled {
		return nil, nil
	}

	chartRef, err := getChartRef(chartName)
	if err != nil {
		return nil, err
	}

	options, err := client.ChartPathOptions.LocateChart(
		chartRef,
		settings,
	)
	if err != nil {
		return nil, err
	}

	cr, err := loader.Load(options)
	if err != nil {
		return nil, err
	}

	return client.RunWithContext(ctx, cr, values)
}

func newUpgrade(cfg *action.Configuration, version string) *action.Upgrade {
	upgrade := action.NewUpgrade(cfg)
	upgrade.Namespace = SPIREManagementNamespace
	upgrade.Version = version
	upgrade.ReuseValues = true
	return upgrade
}

func (h *HelmSPIREProvider) upgradeSPIRE() (*release.Release, error) {
	client := newUpgrade(h.cfg, h.SPIREVersion)
	return upgradeChart(h.ctx, h.cfg, client, SPIREChartName, h.settings, h.spireValues)
}

func upgradeChart(ctx context.Context, cfg *action.Configuration, client *action.Upgrade, chartName string, settings *cli.EnvSettings, values map[string]any) (*release.Release, error) {
	alreadyInstalled, err := checkIfAlreadyInstalled(cfg, chartName)
	if err != nil {
		return nil, fmt.Errorf("cannot determine chart installation status: %s", err)
	}

	if !alreadyInstalled {
		return nil, fmt.Errorf("%v not installed", chartName)
	}

	chartRef, err := getChartRef(chartName)
	if err != nil {
		return nil, err
	}

	options, err := client.ChartPathOptions.LocateChart(
		chartRef,
		settings,
	)
	if err != nil {
		return nil, err
	}

	chart, err := loader.Load(options)
	if err != nil {
		return nil, err
	}

	return client.RunWithContext(ctx, chartName, chart, values)
}

// getChartRef returns the full chart reference using either a custom repository path
// from HELM_REPO_PATH environment variable or the default SPIRE repository.
func getChartRef(chartName string) (string, error) {
	if chartName == "" {
		return "", fmt.Errorf("chart name cannot be empty")
	}

	repoPath, exists := os.LookupEnv("HELM_REPO_PATH")
	if exists {
		if repoPath == "" {
			return "", fmt.Errorf("HELM_REPO_PATH environment variable is set but empty")
		}

		repoPath = strings.TrimRight(repoPath, "/")
		return fmt.Sprintf("%s/%s", repoPath, chartName), nil
	}

	return fmt.Sprintf("%s/%s", SPIRERepositoryName, chartName), nil
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

// IsClusterDeployed returns whether a cluster has been deployed, i.e. whether a SPIRE Helm release has been installed.
func IsClusterDeployed(ctx context.Context, cluster *clusterpb.Cluster) (bool, error) {
	prov, err := NewHelmSPIREProvider(ctx, cluster, nil, nil)
	if err != nil {
		return false, err
	}
	return prov.CheckIfAlreadyInstalled()
}
