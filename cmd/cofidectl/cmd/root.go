package cmd

import (
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/trustzone"
	"github.com/spf13/cobra"
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

	spiffeOdpServiceName = "spiffe-oidc-discovery-provider"
)

var longDesc = `cofidectl - Workload identity for hybrid and multi-cloud security`

func NewRootCmd(args []string) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:          "cofidectl",
		Short:        "cofidectl",
		Long:         longDesc,
		SilenceUsage: true,
	}

	cmd.AddCommand(
		trustzone.TrustZoneListCmd(),
	)

	return cmd, nil
}
