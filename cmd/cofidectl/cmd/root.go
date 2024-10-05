package cmd

import (
	"github.com/spf13/cobra"
)

const (
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
		newUpCmd(),
	)

	return cmd, nil
}
