// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"

	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/provision_plugin/v1alpha2"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/validator"
)

// Provision is the interface that provision plugins have to implement.
type Provision interface {
	validator.Validator

	// Deploy deploys the workload identity configuration to the clusters in the system.
	// The method is asynchronous, returning a channel over which Status messages are sent
	// describing the various stages of deployment and their outcomes.
	Deploy(ctx context.Context, ds datasource.DataSource, opts *DeployOpts) (<-chan *provisionpb.Status, error)

	// TearDown tears down the workload identity configuration from the clusters in the system.
	// The method is asynchronous, returning a channel over which Status messages are sent
	// describing the various stages of tear down and their outcomes.
	TearDown(ctx context.Context, ds datasource.DataSource, opts *TearDownOpts) (<-chan *provisionpb.Status, error)

	// GetHelmValues retrieves the Helm values for the specified trust zone and cluster.
	GetHelmValues(ctx context.Context, ds datasource.DataSource, opts *GetHelmValuesOpts) (map[string]any, error)
}

type DeployOpts struct {
	KubeCfgFile  string
	TrustZoneIDs []string
	SkipWait     bool
}

type TearDownOpts struct {
	KubeCfgFile  string
	TrustZoneIDs []string
}

type GetHelmValuesOpts struct {
	ClusterID string
}
