// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd"
	cmdcontext "github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/manager"
)

const (
	cofidectlPluginPrefix = "cofidectl-"
	cofideConfigFile      = "cofide.yaml"
)

func main() {
	log.SetFlags(0)

	rootCmd, err := getRootCommand()
	if err != nil {
		log.Fatal(err)
	}

	// Check if there is a plugin sub-command to execute.
	extCmd, ok := getPluginSubCommand(rootCmd, os.Args)
	if ok {
		if err := extCmd.Execute(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := rootCmd.Execute(); err != nil {
		// Cobra logs any errors returned by commands, so don't log again.
		os.Exit(1)
	}
}

// getRootCommand returns a root CLI command wired up with a config loader and plugin manager.
func getRootCommand() (*cobra.Command, error) {
	// Defaults to the local data source
	configLoader := config.NewFileLoader(cofideConfigFile)
	pluginManager := manager.NewManager(configLoader)

	cmdCtx := &cmdcontext.CommandContext{
		PluginManager: pluginManager,
	}

	return cmd.NewRootCommand(cmdCtx).GetRootCommand()
}

// getPluginSubCommand returns a `plugin.SubCommand` for a CLI plugin if:
// 1. the first CLI argument does not match a registered subcommand
// 2. a cofidectl plugin exists with a name of cofidectl- followed by hte first CLI argument
func getPluginSubCommand(rootCmd *cobra.Command, args []string) (*plugin.SubCommand, bool) {
	if len(args) > 1 {
		if _, _, err := rootCmd.Find(args[0:2]); err != nil {
			pluginName := cofidectlPluginPrefix + args[1]
			if exists, err := plugin.PluginExists(pluginName); err != nil {
				log.Fatal(err)
			} else if exists {
				subcommand := plugin.NewSubCommand(pluginName, args[2:])
				return subcommand, true
			}
		}
	}
	return nil, false
}
