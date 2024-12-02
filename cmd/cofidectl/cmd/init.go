// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/manager"
	"github.com/spf13/cobra"
)

const (
	connectPluginName = "cofidectl-connect"
)

type InitCommand struct {
	cmdCtx *context.CommandContext
}

func NewInitCommand(cmdCtx *context.CommandContext) *InitCommand {
	return &InitCommand{
		cmdCtx: cmdCtx,
	}
}

var initRootCmdDesc = `
This command initialises a new Cofide config file in the current working
directory
`

type Opts struct {
	enableConnect bool
}

func (i *InitCommand) GetRootCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "init [ARGS]",
		Short: "Initialises the Cofide config file",
		Long:  initRootCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var pluginName string
			if opts.enableConnect {
				if ok, _ := plugin.PluginExists(connectPluginName); ok {
					pluginName = connectPluginName
				} else {
					fmt.Println("👀 get in touch with us at hello@cofide.io to find out more")
					os.Exit(1)
				}
			} else {
				// Default to the local file data source.
				pluginName = manager.LocalPluginName
			}

			_, err := i.cmdCtx.PluginManager.Init(pluginName, nil)
			return err
		},
	}

	f := cmd.Flags()
	f.BoolVar(&opts.enableConnect, "enable-connect", false, "Enables Cofide Connect")

	return cmd
}
