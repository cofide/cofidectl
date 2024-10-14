package helm

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/cofide/cofidectl/internal/pkg/provider"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"

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

// Execute creates a provider status channel and
func (h *HelmSPIREProvider) Execute() (<-chan provider.ProviderStatus, error) {
	statusCh := make(chan provider.ProviderStatus)

	h.install(statusCh)

	return statusCh, nil
}

// install installs the Cofide-enabled SPIRE stack to the selected Kubernetes context
// and updates the status channel accordingly.
func (h *HelmSPIREProvider) install(statusCh chan provider.ProviderStatus) {
	go func() {
		defer close(statusCh)

		statusCh <- provider.ProviderStatus{Stage: "Preparing", Message: "Preparing chart for installation"}
		time.Sleep(time.Duration(1) * time.Second)

		statusCh <- provider.ProviderStatus{Stage: "Installing", Message: "Installing CRDs to cluster"}
		_, err := h.installSPIRECRDs()
		if err != nil {
			statusCh <- provider.ProviderStatus{Stage: "Installing", Message: "Failed to install CRDs", Done: true, Error: err}
			return
		}

		statusCh <- provider.ProviderStatus{Stage: "Installing", Message: "Installing SPIRE chart to cluster"}
		_, err = h.installSPIRE()
		if err != nil {
			statusCh <- provider.ProviderStatus{Stage: "Installing", Message: "Failed to install chart", Done: true, Error: err}
			return
		}

		statusCh <- provider.ProviderStatus{Stage: "Complete", Message: "Installation complete", Done: true}
		time.Sleep(time.Duration(1) * time.Second)
	}()
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
	alreadyInstalled, err := checkIfAlreadyInstalled(cfg, chartName)
	if err != nil {
		return nil, fmt.Errorf("cannot determine chart installation status: %s", err)
	}
	if alreadyInstalled {
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

func checkIfAlreadyInstalled(cfg *action.Configuration, chartName string) (bool, error) {
	history := action.NewHistory(cfg)
	history.Max = 1
	ledger, err := history.Run(chartName)
	if err != nil && err != driver.ErrReleaseNotFound {
		return false, err
	}
	return len(ledger) > 0, nil
}

type HelmValuesGenerator struct {
	source cofidectl_plugin.DataSource
}

func NewHelmValuesGenerator(source cofidectl_plugin.DataSource) *HelmValuesGenerator {
	return &HelmValuesGenerator{source: source}
}

func (g *HelmValuesGenerator) GenerateValues() (map[string]interface{}, error) {
	trustZones, err := g.source.ListTrustZones()
	if err != nil {
		return nil, err
	}

	if len(trustZones) < 1 {
		return nil, fmt.Errorf("no trust zones have been configured")
	}

	// TODO: This should gracefully handle the case where more than one trust zone has been defined.
	globalValues := map[string]interface{}{
		"global.spire.clusterName":              trustZones[0].KubernetesCluster,
		"global.spire.trustDomain":              trustZones[0].TrustDomain,
		"global.spire.recommendations.create":   true,
		"global.installAndUpgradeHooks.enabled": false,
		"global.deleteHooks.enabled":            false,
	}

	agentConfig := trustZones[0].TrustProvider.AgentConfig

	spireAgentValues := map[string]interface{}{
		`"spire-agent"."fullnameOverride"`: "spire-agent", // NOTE: https://github.com/cue-lang/cue/issues/358
		`"spire-agent"."logLevel"`:         "DEBUG",
		fmt.Sprintf(`"spire-agent"."nodeAttestor"."%s"."enabled"`, agentConfig.NodeAttestor):                              agentConfig.NodeAttestorEnabled,
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."disableContainerSelectors"`, agentConfig.WorkloadAttestor):   agentConfig.WorkloadAttestorConfig.DisableContainerSelectors,
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."enabled"`, agentConfig.WorkloadAttestor):                     agentConfig.WorkloadAttestorConfig.Enabled,
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."skipKubeletVerification"`, agentConfig.WorkloadAttestor):     agentConfig.WorkloadAttestorConfig.SkipKubeletVerification,
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."useNewContainerLocator"`, agentConfig.WorkloadAttestor):      agentConfig.WorkloadAttestorConfig.UseNewContainerLocator,
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."verboseContainerLocatorLogs"`, agentConfig.WorkloadAttestor): agentConfig.WorkloadAttestorConfig.VerboseContainerLocatorLogs,
		`"spire-agent"."server"."address"`: "spire-server.spire",
	}

	serverConfig := trustZones[0].TrustProvider.ServerConfig

	spireServerValues := map[string]interface{}{
		`"spire-server"."caKeyType"`:                                                                           "rsa-2048",
		`"spire-server"."controllerManager"."enabled"`:                                                         true,
		`"spire-server"."controllerManager"."identities"."clusterSPIFFEIDs"."default"."enabled"`:               false, // TODO: Rethink this flow.
		`"spire-server"."caTTL"`:                                                                               "12h",
		`"spire-server"."fullnameOverride"`:                                                                    "spire-server",
		`"spire-server"."logLevel"`:                                                                            "DEBUG",
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."audience"`, serverConfig.NodeAttestor):                serverConfig.NodeAttestorConfig.Audience,
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."allowedPodLabelKeys"`, serverConfig.NodeAttestor):     serverConfig.NodeAttestorConfig.AllowedPodLabelKeys,
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."allowedNodeLabelKeys"`, serverConfig.NodeAttestor):    serverConfig.NodeAttestorConfig.AllowedNodeLabelKeys,
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."enabled"`, serverConfig.NodeAttestor):                 serverConfig.NodeAttestorConfig.Enabled,
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."serviceAccountAllowList"`, serverConfig.NodeAttestor): serverConfig.NodeAttestorConfig.ServiceAccountAllowList,
	}

	spiffeOIDCDiscoveryProviderValues := map[string]interface{}{
		`"spiffe-oidc-discovery-provider"."enabled"`: false,
	}

	spiffeCSIDriverValues := map[string]interface{}{
		`"spiffe-csi-driver"."fullnameOverride"`: "spiffe-csi-driver",
	}

	valuesMaps := []map[string]interface{}{
		globalValues,
		spireAgentValues,
		spireServerValues,
		spiffeOIDCDiscoveryProviderValues,
		spiffeCSIDriverValues,
	}

	ctx := cuecontext.New()
	combinedValuesCUE := ctx.CompileBytes([]byte{})

	for _, valuesMap := range valuesMaps {
		valuesCUE := ctx.CompileBytes([]byte{})

		for path, value := range valuesMap {
			valuesCUE = valuesCUE.FillPath(cue.ParsePath(path), value)
		}

		combinedValuesCUE = combinedValuesCUE.Unify(valuesCUE)
	}

	combinedValuesJSON, err := combinedValuesCUE.MarshalJSON()
	if err != nil {
		// TODO: Improve error messaging.
		return nil, err
	}

	var values map[string]interface{}
	err = json.Unmarshal([]byte(combinedValuesJSON), &values)
	if err != nil {
		// TODO: Improve error messaging.
		return nil, err
	}

	return values, nil
}
