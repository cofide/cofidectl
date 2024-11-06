// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"os"
	"strings"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd"
	cmdcontext "github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/manager"
)

const (
	connectPlugin = "cofidectl-connect"
)

func Run() error {
	// Defaults to the local data source
	configLoader := config.NewFileLoader("cofide.yaml")
	pluginManager := manager.NewManager(configLoader)

	// Check if there is a plugin sub-command to execute
	if len(os.Args) > 1 && strings.HasSuffix(connectPlugin, os.Args[1]) {
		return plugin.ExecuteSubCommand(connectPlugin, os.Args[2:])
	}

	cmdCtx := &cmdcontext.CommandContext{
		PluginManager: pluginManager,
	}

	rootCmd, err := cmd.NewRootCommand(cmdCtx).GetRootCommand()
	if err != nil {
		return err
	}

	if err = rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}

func main() {
	log.SetFlags(0)

	if err := Run(); err != nil {
		log.Fatal(err)
	}
}
