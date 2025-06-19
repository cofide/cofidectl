// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/statusspinner"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	provisionplugin "github.com/cofide/cofidectl/pkg/plugin/provision"
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

type DownOpts struct {
	quiet          bool
	trustZoneNames []string
	trustzoneIDs   []string
}

func (d *DownCommand) DownCmd() *cobra.Command {
	opts := &DownOpts{}
	cmd := &cobra.Command{
		Use:   "down [ARGS]",
		Short: "Uninstalls a Cofide configuration",
		Long:  downCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := d.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			provision, err := d.cmdCtx.PluginManager.GetProvision(cmd.Context())
			if err != nil {
				return err
			}

			trustZoneIDs := opts.trustzoneIDs
			if len(opts.trustZoneNames) > 0 {
				tzs, err := ds.ListTrustZones()
				if err != nil {
					return err
				}
				for _, tzName := range opts.trustZoneNames {
					var trustZoneID string
					for _, tz := range tzs {
						if tz.Name == tzName {
							trustZoneID = tz.GetId()
							break
						}
					}
					if trustZoneID == "" {
						return fmt.Errorf("trust zone '%s' not found", tzName)
					}
					trustZoneIDs = append(trustZoneIDs, trustZoneID)
				}
			}

			tearDownOpts := provisionplugin.TearDownOpts{
				KubeCfgFile:  kubeCfgFile,
				TrustZoneIDs: trustZoneIDs,
			}
			statusCh, err := provision.TearDown(cmd.Context(), ds, &tearDownOpts)
			if err != nil {
				return err
			}
			return statusspinner.WatchProvisionStatus(cmd.Context(), statusCh, opts.quiet)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&opts.quiet, "quiet", false, "Minimise logging from uninstallation")
	f.StringSliceVar(&opts.trustZoneNames, "trust-zone-name", []string{}, "Trust zones to uninstall, or all if none is specified")
	f.StringSliceVar(&opts.trustzoneIDs, "trustzone-id", []string{}, "Trust zone IDs to uninstall, or all if none is specified")

	cmd.MarkFlagsMutuallyExclusive("trust-zone-name", "trustzone-id")

	return cmd
}
