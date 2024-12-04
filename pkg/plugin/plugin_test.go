// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDataSourceServeCmd(t *testing.T) {
	assert.True(t, IsDataSourceServeCmd([]string{"data-source", "serve"}))
	assert.False(t, IsDataSourceServeCmd([]string{"trust-zone", "list"}))
	assert.False(t, IsDataSourceServeCmd([]string{}))
	assert.False(t, IsDataSourceServeCmd([]string{"data-source"}))
	assert.False(t, IsDataSourceServeCmd([]string{"DATA-SOURCE", "serve"}))
	assert.False(t, IsDataSourceServeCmd([]string{"data-source", "serve", "extra"}))
	assert.False(t, IsDataSourceServeCmd(nil))
}
