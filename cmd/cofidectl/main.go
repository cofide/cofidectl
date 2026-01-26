// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/cofide/cofidectl/pkg/plugin"
)

const (
	cofidectlPluginPrefix = "cofidectl-"
	cofideConfigFile      = "cofide.yaml"
	shutdownTimeoutSec    = 10
)

var (
	name    = "cofidectl"
	version = "unreleased"
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
	cmdCtx := cmdcontext.NewCommandContext(cofideConfigFile, nil)
	defer cmdCtx.Shutdown()
	go cmdCtx.HandleSignals()

	rootCmd, err := cmd.NewRootCommand(name, version, cmdCtx).GetRootCommand()
	if err != nil {
		log.Println(err)
		return err
	}

	// Check if there is a CLI plugin to execute.
	cliPlugin, ok, err := getCliPlugin(rootCmd, os.Args)
	if err != nil {
		log.Println(err)
		return err
	}
	if ok {
		if err := cliPlugin.Execute(); err != nil {
			log.Println(err)
			return err
		}
		return nil
	}

	// Cobra logs any errors returned by commands, so don't log again.
	return rootCmd.ExecuteContext(cmdCtx.Ctx)
}

// getCliPlugin returns a `plugin.CliPlugin` for a CLI plugin if:
// 1. the first CLI argument does not match a registered subcommand
// 2. a cofidectl plugin exists with a name of cofidectl- followed by the first CLI argument
func getCliPlugin(rootCmd *cobra.Command, args []string) (*plugin.CliPlugin, bool, error) {
	if len(args) > 1 {
		if _, _, err := rootCmd.Find(args[0:2]); err != nil {
			pluginName := cofidectlPluginPrefix + args[1]
			if exists, err := plugin.PluginExists(pluginName); err != nil {
				return nil, false, err
			} else if exists {
				cliPlugin := plugin.NewCliPlugin(pluginName, args[2:])
				return cliPlugin, true, nil
			}
		}
	}
	return nil, false, nil
}
