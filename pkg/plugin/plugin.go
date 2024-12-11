// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"slices"
)

// PluginServeArgs contains the arguments passed to plugins when executing them as a gRPC plugin.
var PluginServeArgs []string = []string{"plugin", "serve"}

// IsPluginServeCmd returns whether the provided command line arguments indicate that a plugin
// should serve gRPC plugins.
func IsPluginServeCmd(args []string) bool {
	return slices.Equal(args, PluginServeArgs)
}
