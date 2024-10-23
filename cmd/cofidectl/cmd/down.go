package cmd

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/cofide/cofidectl/internal/pkg/provider/helm"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type DownCommand struct {
	source cofidectl_plugin.DataSource
}

func NewDownCommand(source cofidectl_plugin.DataSource) *DownCommand {
	return &DownCommand{
		source: source,
	}
}

var downCmdDesc = `
This command uninstalls a Cofide configuration
`

func (d *DownCommand) DownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down [ARGS]",
		Short: "Uninstalls a Cofide configuration",
		Long:  downCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.source.Validate(); err != nil {
				return err
			}

			trustZones, err := d.source.ListTrustZones()
			if err != nil {
				return err
			}

			if len(trustZones) == 0 {
				fmt.Println("no trust zones have been configured")
				return nil
			}

			if err := uninstallSPIREStack(trustZones); err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}

func uninstallSPIREStack(trustZones []*trust_zone_proto.TrustZone) error {
	for _, trustZone := range trustZones {
		prov := helm.NewHelmSPIREProvider(trustZone, nil, nil)

		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Start()
		statusCh, err := prov.ExecuteUninstall()
		if err != nil {
			s.Stop()
			return fmt.Errorf("failed to start uninstallation: %w", err)
		}

		for status := range statusCh {
			s.Suffix = fmt.Sprintf(" %s: %s\n", status.Stage, status.Message)

			if status.Done {
				s.Stop()
				if status.Error != nil {
					fmt.Printf("❌ %s: %s\n", status.Stage, status.Message)
					return fmt.Errorf("uninstallation failed: %w", status.Error)
				}
				green := color.New(color.FgGreen).SprintFunc()
				fmt.Printf("%s %s: %s\n\n", green("✅"), status.Stage, status.Message)
			}
		}

		s.Stop()
	}
	return nil
}
