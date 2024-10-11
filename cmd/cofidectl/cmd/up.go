package cmd

import (
	"github.com/cofide/cofidectl/internal/pkg/provider/helm"

	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/spf13/cobra"
)

type UpCommand struct {
	source cofidectl_plugin.DataSource
}

func NewUpCommand(source cofidectl_plugin.DataSource) *UpCommand {
	return &UpCommand{
		source: source,
	}
}

var upCmdDesc = `
This command deploys a Cofide configuration
`

func (u *UpCommand) UpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [ARGS]",
		Short: "Deploy a Cofide configuration",
		Long:  upCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			generator := helm.NewHelmValuesGenerator(u.source)
			spireValues, err := generator.GenerateValues()
			if err != nil {
				return err
			}
			spireCRDsValues := map[string]interface{}{}

			prov := helm.NewHelmSPIREProvider(spireValues, spireCRDsValues)
			prov.Execute()

			return nil
		},
	}
	return cmd
}
