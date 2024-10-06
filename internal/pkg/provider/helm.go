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

// HelmSPIREProvider implements a Helm-based installer for the Cofide stack
type HelmSPIREProvider struct {
	settings         *cli.EnvSettings
	SPIREVersion     string
	SPIRECRDsVersion string
}

func NewHelmSPIREProvider() *HelmSPIREProvider {
	return &HelmSPIREProvider{
		settings:         cli.New(),
		SPIREVersion:     SPIREChartVersion,
		SPIRECRDsVersion: SPIRECRDsChartVersion,
	}
}

func (h *HelmSPIREProvider) Execute() {
	cfg, err := h.initActionConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := h.newHelmClient(cfg, SPIRECRDsChartName, h.SPIRECRDsVersion)
	h.installChart(client, SPIRECRDsChartName)
	log.Printf("Successfully installed %v %v", SPIRECRDsChartName, SPIREChartVersion)

	client = h.newHelmClient(cfg, SPIREChartName, SPIREChartVersion)
	h.installChart(client, SPIREChartName)
	log.Printf("Successfully installed %v %v", SPIREChartName, SPIREChartVersion)
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

func (h *HelmSPIREProvider) newHelmClient(cfg *action.Configuration, chart string, version string) *action.Install {
	client := action.NewInstall(cfg)
	client.Version = version
	client.ReleaseName = chart
	client.Namespace = SPIRENamespace
	client.CreateNamespace = true

	return client
}

func (h *HelmSPIREProvider) installChart(client *action.Install, chartName string) (*release.Release, error) {
	options, err := client.ChartPathOptions.LocateChart(
		fmt.Sprintf("spire/%s", chartName),
		h.settings,
	)
	if err != nil {
		log.Fatal(err)
	}

	cr, err := loader.Load(options)
	if err != nil {
		log.Fatal(err)
	}

	return client.Run(cr, nil)
}
