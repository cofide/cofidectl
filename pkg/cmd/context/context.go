// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/pkg/plugin/manager"
)

const shutdownTimeoutSec = 10

type CommandContext struct {
	Ctx           context.Context
	cancel        context.CancelCauseFunc
	PluginManager *manager.PluginManager
}

// NewCommandContext returns a command context wired up with a config loader and plugin manager.
func NewCommandContext(cofideConfigFile string) *CommandContext {
	ctx, cancel := context.WithCancelCause(context.Background())
	configLoader := config.NewFileLoader(cofideConfigFile)
	pluginManager := manager.NewManager(configLoader)
	return &CommandContext{Ctx: ctx, cancel: cancel, PluginManager: pluginManager}
}

func (cc *CommandContext) Shutdown() {
	if cc.cancel != nil {
		cc.cancel(errors.New("shutting down"))
		cc.cancel = nil
	}
	cc.PluginManager.Shutdown()
}

// HandleSignals waits for SIGINT or SIGTERM, then triggers a clean shutdown using the command context.
// It should be called from a non-main goroutine.
func (cc *CommandContext) HandleSignals() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	s := <-shutdown
	fmt.Printf("Caught %s signal, exiting\n", s.String())
	cc.Shutdown()

	// Wait for a while to allow for graceful completion of the main goroutine.
	<-time.After(shutdownTimeoutSec * time.Second)
	fmt.Println("Timed out waiting for shutdown")
	os.Exit(1)
}
