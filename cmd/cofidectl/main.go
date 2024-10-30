package main

import (
	"log"
	"os"
	"strings"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/manager"
)

const (
	localPlugin   = "local"
	connectPlugin = "cofidectl-connect"
	pluginPrefix  = "cofidectl-"
)

func Run() error {
	// Defaults to the local data source
	configLoader := config.NewFileLoader("cofide.yaml")
	pluginManager := manager.NewManager(configLoader)
	ds, err := pluginManager.GetPlugin()
	if err != nil {
		return err
	}

	/*
		if len(plugins) > 1 {
			log.Fatal("only a single plugin is currently supported")
		}
		if len(plugins) == 0 {
			log.Fatal("no plugins available")
		}
	*/

	// Check if there is a plugin subcommand to execute
	if len(os.Args) > 1 && strings.HasSuffix(connectPlugin, os.Args[1]) {
		return plugin.ExecuteSubCommand(connectPlugin, os.Args[2:])
	}

	rootCmd, err := cmd.NewRootCommand(ds, os.Args[1:]).GetRootCommand()
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
