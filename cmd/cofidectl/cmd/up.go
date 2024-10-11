package cmd

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/cofide/cofidectl/internal/pkg/provider/helm"
	"github.com/fatih/color"

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

			// create a spinner to display whilst installation is underway
			s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
			s.Start()
			statusCh, err := prov.Execute()
			if err != nil {
				s.Stop()
				return fmt.Errorf("failed to start installation: %w", err)
			}

			for status := range statusCh {
				s.Suffix = fmt.Sprintf(" %s: %s", status.Stage, status.Message)

				if status.Done {
					s.Stop()
					if status.Error != nil {
						fmt.Printf("❌ %s: %s\n", status.Stage, status.Message)
						return fmt.Errorf("installation failed: %w", status.Error)
					}
					green := color.New(color.FgGreen).SprintFunc()
					fmt.Printf("%s %s: %s\n", green("✅"), status.Stage, status.Message)
					return nil
				}
			}

			s.Stop()
			return fmt.Errorf("unexpected end of status channel")
		},
	}
	return cmd
}
