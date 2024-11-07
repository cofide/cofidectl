// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	cmdcontext "github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"
	"github.com/cofide/cofidectl/internal/pkg/provider/helm"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type DownCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewDownCommand(cmdCtx *cmdcontext.CommandContext) *DownCommand {
	return &DownCommand{
		cmdCtx: cmdCtx}
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
			ds, err := d.cmdCtx.PluginManager.GetDataSource()
			if err != nil {
				return err
			}

			trustZones, err := ds.ListTrustZones()
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
