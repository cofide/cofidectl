// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"context"
	"errors"

	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/pkg/plugin/manager"
)

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
