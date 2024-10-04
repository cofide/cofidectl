package provider

import (
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
)

// HelmSPIREProvider implements a Helm-based installer for the Cofide stack
type HelmSPIREProvider struct {
	chart   string
	version string
}

func NewHelmSPIREProvider(chart string, version string) *HelmSPIREProvider {
	return &HelmSPIREProvider{
		chart:   chart,
		version: version,
	}
}

func (h *HelmSPIREProvider) Execute() {
	h.newInstall()
}

func (h *HelmSPIREProvider) newInstall() (*release.Release, error) {
	actionConfig := new(action.Configuration)
	client := action.NewInstall(actionConfig)
	client.Version = h.version

	options, _ := client.ChartPathOptions.LocateChart("spire/spire", nil)
	cr, _ := loader.Load(options)
	return client.Run(cr, nil)
}
