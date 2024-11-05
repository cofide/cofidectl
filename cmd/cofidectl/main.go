// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"os"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/pkg/plugin/loader"
)

func main() {
	log.SetFlags(0)

	// Defaults to the local data source
	configLoader := config.NewFileLoader("cofide.yaml")
	pluginLoader := loader.NewLoader(configLoader)
	plugins, err := pluginLoader.GetPlugins()
	if err != nil {
		log.Fatal(err)
	}

	if len(plugins) > 1 {
		log.Fatal("only a single plugin is currently supported")
	}
	if len(plugins) == 0 {
		log.Fatal("no plugins available")
	}

	ds := plugins[0]
	rootCmd, err := cmd.NewRootCommand(ds, os.Args[1:]).GetRootCommand()
	if err != nil {
		log.Fatal(err)
	}

	if err = rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
