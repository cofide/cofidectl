// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	pluginspb "github.com/cofide/cofide-api-sdk/gen/go/proto/plugins/v1alpha1"
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
	enableConnect    bool
	dataSourcePlugin string
	provisionPlugin  string
}

func (i *InitCommand) GetRootCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "init [ARGS]",
		Short: "Initialises the Cofide config file",
		Long:  initRootCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.enableConnect {
				if ok, _ := plugin.PluginExists(connectPluginName); ok {
					fmt.Println(`Please run "cofidectl connect init"`)
				} else {
					fmt.Println("👀 get in touch with us at hello@cofide.io to find out more")
				}
				os.Exit(1)
			}

			plugins := &pluginspb.Plugins{
				DataSource: &opts.dataSourcePlugin,
				Provision:  &opts.provisionPlugin,
			}
			return i.cmdCtx.PluginManager.Init(cmd.Context(), plugins, nil)
		},
	}

	defaultPlugins := manager.GetDefaultPlugins()
	f := cmd.Flags()
	f.BoolVar(&opts.enableConnect, "enable-connect", false, "Enables Cofide Connect")
	f.StringVar(&opts.dataSourcePlugin, "data-source-plugin", defaultPlugins.GetDataSource(), "Data source plugin")
	f.StringVar(&opts.provisionPlugin, "provision-plugin", defaultPlugins.GetProvision(), "Provision plugin")

	return cmd
}
