// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"

	"github.com/cofide/cofidectl/pkg/plugin"
)

// Provision is the interface that provision plugins have to implement.
type Provision interface {
	// Deploy deploys the workload identity configuration to the clusters in the system.
	Deploy(ctx context.Context, ds plugin.DataSource, kubeCfgFile string) error

	// TearDown tears down the workload identity configuration from the clusters in the system.
	TearDown(ctx context.Context, ds plugin.DataSource) error
}
