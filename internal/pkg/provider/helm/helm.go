package helm

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"

	"github.com/cofide/cofidectl/internal/pkg/provider"
	"github.com/cofide/cofidectl/internal/pkg/trustprovider"

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
	spireValues      map[string]interface{}
	spireCRDsValues  map[string]interface{}
	trustZone        *trust_zone_proto.TrustZone
}

func NewHelmSPIREProvider(trustZone *trust_zone_proto.TrustZone, spireValues, spireCRDsValues map[string]interface{}) *HelmSPIREProvider {
	settings := cli.New()
	settings.KubeContext = trustZone.KubernetesContext

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
		log.Fatal(err)
	}

	return prov
}

// Execute creates a provider status channel and performs the Helm chart installations.
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

		statusCh <- provider.ProviderStatus{Stage: "Installing", Message: fmt.Sprintf("Installing CRDs to cluster %s", h.trustZone.KubernetesCluster)}
		_, err := h.installSPIRECRDs()
		if err != nil {
			statusCh <- provider.ProviderStatus{Stage: "Installing", Message: fmt.Sprintf("Failed to install CRDs on cluster %s", h.trustZone.KubernetesCluster), Done: true, Error: err}
			return
		}

		statusCh <- provider.ProviderStatus{Stage: "Installing", Message: fmt.Sprintf("Installing SPIRE chart to cluster %s", h.trustZone.KubernetesCluster)}
		_, err = h.installSPIRE()
		if err != nil {
			statusCh <- provider.ProviderStatus{Stage: "Installing", Message: fmt.Sprintf("Failed to install SPIRE chart on cluster %s", h.trustZone.KubernetesCluster), Done: true, Error: err}
			return
		}

		statusCh <- provider.ProviderStatus{Stage: "Installed", Message: fmt.Sprintf("Installation completed for %s on cluster %s", h.trustZone.TrustDomain, h.trustZone.KubernetesCluster), Done: true}
		time.Sleep(time.Duration(1) * time.Second)
	}()
}

func (h *HelmSPIREProvider) ExecuteUpgrade() (<-chan provider.ProviderStatus, error) {
	statusCh := make(chan provider.ProviderStatus)

	h.upgrade(statusCh)

	return statusCh, nil
}

func (h *HelmSPIREProvider) upgrade(statusCh chan provider.ProviderStatus) {
	go func() {
		defer close(statusCh)

		statusCh <- provider.ProviderStatus{Stage: "Preparing", Message: "Preparing chart for upgrade"}
		time.Sleep(time.Duration(1) * time.Second)

		statusCh <- provider.ProviderStatus{Stage: "Upgrading", Message: fmt.Sprintf("Upgrading SPIRE chart on cluster %s", h.trustZone.KubernetesCluster)}
		_, err := h.upgradeSPIRE()
		if err != nil {
			statusCh <- provider.ProviderStatus{Stage: "Upgrading", Message: fmt.Sprintf("Failed to upgrade SPIRE chart on cluster %s", h.trustZone.KubernetesCluster), Done: true, Error: err}
			return
		}

		statusCh <- provider.ProviderStatus{Stage: "Upgraded", Message: fmt.Sprintf("Upgrade completed for %s on cluster %s", h.trustZone.TrustDomain, h.trustZone.KubernetesCluster), Done: true}
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

func (h *HelmSPIREProvider) upgradeSPIRECRDs() (*release.Release, error) {
	client := &action.Upgrade{}
	return upgradeChart(h.cfg, client, SPIRECRDsChartName, h.settings, h.spireCRDsValues)
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
		fmt.Sprintf("spire/%s", chartName),
		settings,
	)
	if err != nil {
		log.Fatal(err)
	}

	chart, err := loader.Load(options)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Upgrading %v...", chart.Name())
	return client.Run(chartName, chart, values)
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
	trustZone *trust_zone_proto.TrustZone
}

func NewHelmValuesGenerator(trustZone *trust_zone_proto.TrustZone) *HelmValuesGenerator {
	return &HelmValuesGenerator{trustZone: trustZone}
}

func (g *HelmValuesGenerator) GenerateValues() (map[string]interface{}, error) {
	trustProvider := trustprovider.NewTrustProvider(g.trustZone.TrustProvider.Kind)
	agentConfig := trustProvider.AgentConfig
	serverConfig := trustProvider.ServerConfig

	globalValues := map[string]interface{}{
		"global.spire.clusterName":              g.trustZone.KubernetesCluster,
		"global.spire.trustDomain":              g.trustZone.TrustDomain,
		"global.spire.recommendations.create":   true,
		"global.installAndUpgradeHooks.enabled": false,
		"global.deleteHooks.enabled":            false,
	}

	spireAgentValues := map[string]interface{}{
		`"spire-agent"."fullnameOverride"`: "spire-agent", // NOTE: https://github.com/cue-lang/cue/issues/358
		`"spire-agent"."logLevel"`:         "DEBUG",
		fmt.Sprintf(`"spire-agent"."nodeAttestor"."%s"."enabled"`, agentConfig.NodeAttestor):                              agentConfig.NodeAttestorEnabled,
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."disableContainerSelectors"`, agentConfig.WorkloadAttestor):   agentConfig.WorkloadAttestorConfig["disableContainerSelectors"],
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."enabled"`, agentConfig.WorkloadAttestor):                     agentConfig.WorkloadAttestorConfig["enabled"],
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."skipKubeletVerification"`, agentConfig.WorkloadAttestor):     agentConfig.WorkloadAttestorConfig["skipKubeletVerification"],
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."useNewContainerLocator"`, agentConfig.WorkloadAttestor):      agentConfig.WorkloadAttestorConfig["useNewContainerLocator"],
		fmt.Sprintf(`"spire-agent"."workloadAttestors"."%s"."verboseContainerLocatorLogs"`, agentConfig.WorkloadAttestor): agentConfig.WorkloadAttestorConfig["verboseContainerLocatorLogs"],
		`"spire-agent"."server"."address"`: "spire-server.spire",
	}

	spireServerValues := map[string]interface{}{
		`"spire-server"."caKeyType"`:                                                                           "rsa-2048",
		`"spire-server"."controllerManager"."enabled"`:                                                         true,
		`"spire-server"."controllerManager"."identities"."clusterSPIFFEIDs"."default"."enabled"`:               false, // TODO: Rethink this flow.
		`"spire-server"."caTTL"`:                                                                               "12h",
		`"spire-server"."fullnameOverride"`:                                                                    "spire-server",
		`"spire-server"."logLevel"`:                                                                            "DEBUG",
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."audience"`, serverConfig.NodeAttestor):                serverConfig.NodeAttestorConfig["audience"],
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."allowedPodLabelKeys"`, serverConfig.NodeAttestor):     serverConfig.NodeAttestorConfig["allowedPodLabelKeys"],
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."allowedNodeLabelKeys"`, serverConfig.NodeAttestor):    serverConfig.NodeAttestorConfig["allowedNodeLabelKeys"],
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."enabled"`, serverConfig.NodeAttestor):                 serverConfig.NodeAttestorConfig["enabled"],
		fmt.Sprintf(`"spire-server"."nodeAttestor"."%s"."serviceAccountAllowList"`, serverConfig.NodeAttestor): serverConfig.NodeAttestorConfig["serviceAccountAllowList"],
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
