// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPluginServeCmd(t *testing.T) {
	assert.True(t, IsPluginServeCmd([]string{"plugin", "serve"}))
	assert.False(t, IsPluginServeCmd([]string{"trust-zone", "list"}))
	assert.False(t, IsPluginServeCmd([]string{}))
	assert.False(t, IsPluginServeCmd([]string{"plugin"}))
	assert.False(t, IsPluginServeCmd([]string{"PLUGIN", "serve"}))
	assert.False(t, IsPluginServeCmd([]string{"plugin", "serve", "extra"}))
	assert.False(t, IsPluginServeCmd(nil))
}
