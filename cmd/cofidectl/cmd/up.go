// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/statusspinner"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	provisionplugin "github.com/cofide/cofidectl/pkg/plugin/provision"
	"github.com/spf13/cobra"
)

type UpCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewUpCommand(cmdCtx *cmdcontext.CommandContext) *UpCommand {
	return &UpCommand{
		cmdCtx: cmdCtx,
	}
}

var upCmdDesc = `
This command installs a Cofide configuration
`

type UpOpts struct {
	quiet      bool
	trustZones []string
}

func (u *UpCommand) UpCmd() *cobra.Command {
	opts := UpOpts{}
	cmd := &cobra.Command{
		Use:   "up [ARGS]",
		Short: "Installs a Cofide configuration",
		Long:  upCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := u.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			provision, err := u.cmdCtx.PluginManager.GetProvision(cmd.Context())
			if err != nil {
				return err
			}

			deployOpts := provisionplugin.DeployOpts{
				KubeCfgFile: kubeCfgFile,
				TrustZones:  opts.trustZones,
			}
			statusCh, err := provision.Deploy(cmd.Context(), ds, &deployOpts)
			if err != nil {
				return err
			}

			return statusspinner.WatchProvisionStatus(
				cmd.Context(),
				statusCh,
				opts.quiet,
			)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&opts.quiet, "quiet", false, "Minimise logging from installation")
	f.StringSliceVar(&opts.trustZones, "trust-zone", []string{}, "Trust zones to install, or all if none is specified")

	return cmd
}
