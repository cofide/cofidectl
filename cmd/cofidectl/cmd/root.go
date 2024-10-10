package cmd

import (
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/trustzone"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/spf13/cobra"
)

var longDesc = `cofidectl - Workload identity for hybrid and multi-cloud security`

func NewRootCmd(args []string, source cofidectl_plugin.DataSource) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:          "cofidectl",
		Short:        "cofidectl",
		Long:         longDesc,
		SilenceUsage: true,
	}

	tzCmd := trustzone.NewTrustZoneCommand(source)
	cmd.AddCommand(
		tzCmd.GetRootCommand(),
		newUpCmd(),
	)

	return cmd, nil
}
