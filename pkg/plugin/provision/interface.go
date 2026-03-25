// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"

	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/validator"
)

// Provision is the interface that provision plugins have to implement.
type Provision interface {
	validator.Validator

	// GetHelmValues retrieves the Helm values for the specified trust zone and cluster.
	GetHelmValues(ctx context.Context, ds datasource.DataSource, opts *GetHelmValuesOpts) (map[string]any, error)
}

type GetHelmValuesOpts struct {
	ClusterID string
}
