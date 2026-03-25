// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spirehelm

import (
	"context"

	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"

	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/provider/helm"
)

// Type check that HelmSPIREProviderFactory implements the ProviderFactory interface.
var _ ProviderFactory = &HelmSPIREProviderFactory{}

// ProviderFactory is an interface that abstracts the retrieval of Helm values.
type ProviderFactory interface {
	GetHelmValues(
		ctx context.Context,
		ds datasource.DataSource,
		trustZone *trust_zone_proto.TrustZone,
		cluster *clusterpb.Cluster,
	) (map[string]any, error)
}

// HelmSPIREProviderFactory implements the ProviderFactory interface.
type HelmSPIREProviderFactory struct{}

func (f *HelmSPIREProviderFactory) GetHelmValues(
	ctx context.Context,
	ds datasource.DataSource,
	trustZone *trust_zone_proto.TrustZone,
	cluster *clusterpb.Cluster,
) (map[string]any, error) {
	generator := helm.NewHelmValuesGenerator(trustZone, cluster, ds, nil)
	return generator.GenerateValues()
}
