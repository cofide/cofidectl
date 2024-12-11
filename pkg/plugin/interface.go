// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
)

// Alias DataSource in the plugin package while transitioning to a new package.
type DataSource = datasource.DataSource
