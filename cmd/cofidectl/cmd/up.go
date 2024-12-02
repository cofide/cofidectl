// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
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

func (u *UpCommand) UpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [ARGS]",
		Short: "Installs a Cofide configuration",
		Long:  upCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := u.cmdCtx.PluginManager.GetDataSource()
			if err != nil {
				return err
			}

			provision := u.cmdCtx.PluginManager.GetProvision()
			return provision.Deploy(cmd.Context(), ds, kubeCfgFile)
		},
	}
	return cmd
}
