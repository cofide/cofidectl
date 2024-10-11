package cmd

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/cofide/cofidectl/internal/pkg/provider"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [ARGS]",
		Short: "Deploy a Cofide configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			prov := provider.NewHelmSPIREProvider()

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
