// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"testing"

	"github.com/cofide/cofidectl/pkg/plugin"
)

func TestLocalDataSource_ImplementsDataSource(t *testing.T) {
	local := LocalDataSource{}
	var _ plugin.DataSource = &local
}
