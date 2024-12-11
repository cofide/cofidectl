// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spirehelm

import (
	"context"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"

	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/provider/helm"
)

// Type check that HelmSPIREProviderFactory implements the ProviderFactory interface.
var _ ProviderFactory = &HelmSPIREProviderFactory{}

// ProviderFactory is an interface that abstracts the construction of helm.Provider objects.
type ProviderFactory interface {
	// Build returns a helm.Provider configured with values for an install/upgrade.
	Build(ctx context.Context, ds datasource.DataSource, trustZone *trust_zone_proto.TrustZone, genValues bool) (helm.Provider, error)
}

// HelmSPIREProviderFactory implements the ProviderFactory interface, building a HelmSPIREProvider
// using the default values generator.
type HelmSPIREProviderFactory struct{}

func (f *HelmSPIREProviderFactory) Build(ctx context.Context, ds datasource.DataSource, trustZone *trust_zone_proto.TrustZone, genValues bool) (helm.Provider, error) {
	spireValues := map[string]any{}
	var err error
	if genValues {
		generator := helm.NewHelmValuesGenerator(trustZone, ds, nil)
		spireValues, err = generator.GenerateValues()
		if err != nil {
			return nil, err
		}
	}
	spireCRDsValues := map[string]any{}
	return helm.NewHelmSPIREProvider(ctx, trustZone, spireValues, spireCRDsValues)
}
