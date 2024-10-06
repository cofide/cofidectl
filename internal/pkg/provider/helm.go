package provider

import (
	"fmt"
	"log"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
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
	SPIREVersion     string
	SPIRECRDsVersion string
	spireClient      *action.Install
	spireCRDsClient  *action.Install
}

func NewHelmSPIREProvider() *HelmSPIREProvider {
	prov := &HelmSPIREProvider{
		settings:         cli.New(),
		SPIREVersion:     SPIREChartVersion,
		SPIRECRDsVersion: SPIRECRDsChartVersion,
	}

	cfg, err := prov.initActionConfig()
	if err != nil {
		log.Fatal(err)
	}
	prov.spireCRDsClient = newInstall(cfg, SPIRECRDsChartName, prov.SPIRECRDsVersion)
	prov.spireClient = newInstall(cfg, SPIREChartName, prov.SPIREVersion)

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
	return installChart(h.spireClient, SPIREChartName, h.settings)
}

func (h *HelmSPIREProvider) installSPIRECRDs() (*release.Release, error) {
	return installChart(h.spireCRDsClient, SPIRECRDsChartName, h.settings)
}

func installChart(client *action.Install, chartName string, settings *cli.EnvSettings) (*release.Release, error) {
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
	return client.Run(cr, nil)
}
