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
	if err := run(); err != nil {
		// This should be the only place that calls os.Exit, to ensure proper clean up.
		// This includes functions that call os.Exit, e.g. cobra.CheckErr, log.Fatal
		os.Exit(1)
	}
}

func run() error {
	cmdCtx := getCommandContext()
	defer cmdCtx.Shutdown()

	rootCmd, err := cmd.NewRootCommand(cmdCtx).GetRootCommand()
	if err != nil {
		log.Println(err)
		return err
	}

	// Check if there is a plugin sub-command to execute.
	extCmd, ok, err := getPluginSubCommand(rootCmd, os.Args)
	if err != nil {
		log.Println(err)
		return err
	}
	if ok {
		if err := extCmd.Execute(); err != nil {
			log.Println(err)
			return err
		}
		return nil
	}

	// Cobra logs any errors returned by commands, so don't log again.
	return rootCmd.Execute()
}

// getCommandContext returns a command context wired up with a config loader and plugin manager.
func getCommandContext() *cmdcontext.CommandContext {
	configLoader := config.NewFileLoader(cofideConfigFile)
	pluginManager := manager.NewManager(configLoader)

	return &cmdcontext.CommandContext{
		PluginManager: pluginManager,
	}
}

// getPluginSubCommand returns a `plugin.SubCommand` for a CLI plugin if:
// 1. the first CLI argument does not match a registered subcommand
// 2. a cofidectl plugin exists with a name of cofidectl- followed by hte first CLI argument
func getPluginSubCommand(rootCmd *cobra.Command, args []string) (*plugin.SubCommand, bool, error) {
	if len(args) > 1 {
		if _, _, err := rootCmd.Find(args[0:2]); err != nil {
			pluginName := cofidectlPluginPrefix + args[1]
			if exists, err := plugin.PluginExists(pluginName); err != nil {
				return nil, false, err
			} else if exists {
				subcommand := plugin.NewSubCommand(pluginName, args[2:])
				return subcommand, true, nil
			}
		}
	}
	return nil, false, nil
}
