// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spirehelm

import (
	"context"

	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/provision"
)

// Type check that SpireHelm implements the Provision interface.
var _ provision.Provision = &SpireHelm{}

// SpireHelm implements the `Provision` interface.
type SpireHelm struct {
	providerFactory ProviderFactory
}

func NewSpireHelm(providerFactory ProviderFactory) *SpireHelm {
	if providerFactory == nil {
		providerFactory = &HelmSPIREProviderFactory{}
	}
	return &SpireHelm{providerFactory: providerFactory}
}

func (h *SpireHelm) Validate(_ context.Context) error {
	return nil
}

func (h *SpireHelm) GetHelmValues(ctx context.Context, ds datasource.DataSource, opts *provision.GetHelmValuesOpts) (map[string]any, error) {
	cluster, err := ds.GetCluster(opts.ClusterID)
	if err != nil {
		return nil, err
	}

	trustZone, err := ds.GetTrustZone(cluster.GetTrustZoneId())
	if err != nil {
		return nil, err
	}

	return h.providerFactory.GetHelmValues(ctx, ds, trustZone, cluster)
}
