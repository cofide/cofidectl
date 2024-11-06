// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"github.com/cofide/cofidectl/pkg/plugin/manager"
)

type CommandContext struct {
	PluginManager *manager.PluginManager
}
