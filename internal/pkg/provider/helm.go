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
	// general
	spireNamespace = "spire"
	repoName       = "spire"
	repoUrl        = "https://spiffe.github.io/helm-charts-hardened/"

	// spire stack (server, agent, csi-driver, oidc-discovery-provider, controller-manager)
	stackRepo         = "spire"
	stackReleaseName  = "spire"
	stackChartName    = "spire"
	stackChartVersion = "0.21.0"

	// spire crds
	crdsRepo         = "spire-crds"
	crdsReleaseName  = "spire-crds"
	crdsChartName    = "spire-crds"
	crdsChartVersion = "0.4.0"
)

// HelmSPIREProvider implements a Helm-based installer for the Cofide stack
type HelmSPIREProvider struct {
	settings *cli.EnvSettings
	chart    string
	version  string
}

func NewHelmSPIREProvider() *HelmSPIREProvider {
	return &HelmSPIREProvider{
		settings: cli.New(),
		chart:    stackChartName,
		version:  stackChartVersion,
	}
}

func (h *HelmSPIREProvider) Execute() {
	cfg, err := h.initActionConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := h.newHelmClient(cfg, crdsChartName, crdsChartVersion)
	h.installChart(client, crdsChartName)
	log.Printf("Successfully installed %v %v", crdsChartName, crdsChartVersion)

	client = h.newHelmClient(cfg, stackChartName, stackChartVersion)
	h.installChart(client, stackChartName)
	log.Printf("Successfully installed %v %v", stackChartName, stackChartVersion)
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
	client.Namespace = spireNamespace
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
