// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"

	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/provision_plugin/v1alpha1"
	"github.com/cofide/cofidectl/pkg/plugin"
)

// Provision is the interface that provision plugins have to implement.
type Provision interface {
	// Deploy deploys the workload identity configuration to the clusters in the system.
	// The method is asynchronous, returning a channel over which Status messages are sent
	// describing the various stages of deployment and their outcomes.
	Deploy(ctx context.Context, ds plugin.DataSource, kubeCfgFile string) (<-chan *provisionpb.Status, error)

	// TearDown tears down the workload identity configuration from the clusters in the system.
	// The method is asynchronous, returning a channel over which Status messages are sent
	// describing the various stages of tear down and their outcomes.
	TearDown(ctx context.Context, ds plugin.DataSource) (<-chan *provisionpb.Status, error)
}
