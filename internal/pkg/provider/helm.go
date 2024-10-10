package provider

import (
	"fmt"
	"log"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
)

const (
	SPIRERepositoryURL      = "https://spiffe.github.io/helm-charts-hardened/"
	SPIRERepositoryName     = "spire"
	SPIRECRDsRepositoryName = "spire-crds"

	SPIREChartName        = "spire"
	SPIREChartVersion     = "0.21.0"
	SPIRECRDsChartName    = "spire-crds"
	SPIRECRDsChartVersion = "0.4.0"

	SPIREReleaseName     = "spire"
	SPIRECRDsReleaseName = "spire-crds"
	SPIRENamespace       = "spire"
)

// HelmSPIREProvider implements a Helm-based installer for the Cofide stack. It uses the SPIFFE/SPIRE project's own
// helm-charts-hardened Helm chart to install a SPIRE stack to a given Kubernetes context, making use of the Cofide
// API concepts and abstractions
type HelmSPIREProvider struct {
	settings         *cli.EnvSettings
	cfg              *action.Configuration
	SPIREVersion     string
	SPIRECRDsVersion string
	spireClient      *action.Install
	spireCRDsClient  *action.Install
	spireValues      map[string]interface{}
	spireCRDsValues  map[string]interface{}
}

func NewHelmSPIREProvider(spireValues, spireCRDsValues map[string]interface{}) *HelmSPIREProvider {
	prov := &HelmSPIREProvider{
		settings:         cli.New(),
		SPIREVersion:     SPIREChartVersion,
		SPIRECRDsVersion: SPIRECRDsChartVersion,
		spireValues:      spireValues,
		spireCRDsValues:  spireCRDsValues,
	}

	var err error
	prov.cfg, err = prov.initActionConfig()
	if err != nil {
		log.Fatal(err)
	}
	prov.spireCRDsClient = newInstall(prov.cfg, SPIRECRDsChartName, prov.SPIRECRDsVersion)
	prov.spireClient = newInstall(prov.cfg, SPIREChartName, prov.SPIREVersion)

	return prov
}

// Execute installs the Cofide-enabled SPIRE stack to the selected Kubernetes context
func (h *HelmSPIREProvider) Execute() {
	h.installSPIRECRDs()
	h.installSPIRE()

	log.Print("✅ cofidectl up complete")
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
	return installChart(h.cfg, h.spireClient, SPIREChartName, h.settings, h.spireValues)
}

func (h *HelmSPIREProvider) installSPIRECRDs() (*release.Release, error) {
	return installChart(h.cfg, h.spireCRDsClient, SPIRECRDsChartName, h.settings, h.spireCRDsValues)
}

func installChart(cfg *action.Configuration, client *action.Install, chartName string, settings *cli.EnvSettings, values map[string]interface{}) (*release.Release, error) {
	if checkIfAlreadyInstalled(cfg, chartName) {
		log.Printf("%v already installed", chartName)
		return nil, nil
	}

	options, err := client.ChartPathOptions.LocateChart(
		fmt.Sprintf("spire/%s", chartName),
		settings,
	)
	if err != nil {
		log.Fatal(err)
	}

	cr, err := loader.Load(options)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Installing %v...", cr.Name())
	return client.Run(cr, values)
}

func checkIfAlreadyInstalled(cfg *action.Configuration, chartName string) bool {
	history := action.NewHistory(cfg)
	history.Max = 1
	ledger, err := history.Run(chartName)

	if err != nil && err != driver.ErrReleaseNotFound {
		log.Fatal(err)
	}

	return len(ledger) > 0
}
