// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"context"

	"github.com/cofide/cofidectl/pkg/plugin/manager"
)

type CommandContext struct {
	Ctx           context.Context
	cancel        func()
	PluginManager *manager.PluginManager
}

func NewCommandContext(pluginManager *manager.PluginManager) *CommandContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &CommandContext{Ctx: ctx, cancel: cancel, PluginManager: pluginManager}
}

func (cc *CommandContext) Shutdown() {
	if cc.cancel != nil {
		cc.cancel()
		cc.cancel = nil
	}
	cc.PluginManager.Shutdown()
}
