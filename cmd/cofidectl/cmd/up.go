package cmd

import (
	"github.com/cofide/cofidectl/internal/pkg/provider"
	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [ARGS]",
		Short: "Deploy a Cofide configuration",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			prov := provider.NewHelmSPIREProvider()
			prov.Execute()
		},
	}
	return cmd
}
