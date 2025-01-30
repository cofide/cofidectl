// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cofide/cofidectl/internal/pkg/config"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/cofide/cofidectl/pkg/plugin/provision"
	"github.com/cofide/cofidectl/pkg/plugin/provision/spirehelm"
	"github.com/hashicorp/go-hclog"
	go_plugin "github.com/hashicorp/go-plugin"
)

const (
	cofideConfigFile = "cofide.yaml"
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
	cmdCtx := cmdcontext.NewCommandContext(cofideConfigFile)
	defer cmdCtx.Shutdown()
	go cmdCtx.HandleSignals()

	// If the CLI is invoked with arguments plugin serve, the gRPC plugins are served.
	if plugin.IsPluginServeCmd(os.Args[1:]) {
		return serveDataSource()
	}
	return fmt.Errorf("unexpected command arguments: %v", os.Args[1:])
}

func serveDataSource() error {
	// go-plugin client expects logs on stdout.
	log.SetOutput(os.Stdout)

	logger := hclog.New(&hclog.LoggerOptions{
		Output: os.Stderr,
		// Log at trace level in the plugin, it will be filtered in the host.
		Level:      hclog.Trace,
		JSONFormat: true,
	})

	lds, err := local.NewLocalDataSource(config.NewFileLoader(cofideConfigFile))
	if err != nil {
		return err
	}
	spireHelm := spirehelm.NewSpireHelm(nil, nil)

	go_plugin.Serve(&go_plugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig,
		Plugins: map[string]go_plugin.Plugin{
			datasource.DataSourcePluginName: &datasource.DataSourcePlugin{Impl: lds},
			provision.ProvisionPluginName:   &provision.ProvisionPlugin{Impl: spireHelm},
		},
		Logger:     logger,
		GRPCServer: go_plugin.DefaultGRPCServer,
	})
	return nil
}
