// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spirehelm

import (
	"context"
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/cofide/cofidectl/pkg/plugin/provision"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpireHelm_GetHelmValues(t *testing.T) {
	providerFactory := newFakeHelmSPIREProviderFactory()
	spireHelm := NewSpireHelm(providerFactory)
	ds := newFakeDataSource(t, defaultConfig())

	opts := provision.GetHelmValuesOpts{ClusterID: "local1-id"}
	values, err := spireHelm.GetHelmValues(context.Background(), ds, &opts)
	require.NoError(t, err, err)
	want := map[string]any{"key1": "value1", "key2": "value2"}
	assert.EqualExportedValues(t, want, values)
}

type fakeHelmSPIREProviderFactory struct{}

func newFakeHelmSPIREProviderFactory() *fakeHelmSPIREProviderFactory {
	return &fakeHelmSPIREProviderFactory{}
}

func (f *fakeHelmSPIREProviderFactory) GetHelmValues(
	ctx context.Context,
	ds datasource.DataSource,
	trustZone *trust_zone_proto.TrustZone,
	cluster *clusterpb.Cluster,
) (map[string]any, error) {
	return map[string]any{"key1": "value1", "key2": "value2"}, nil
}

func newFakeDataSource(t *testing.T, cfg *config.Config) datasource.DataSource {
	configLoader, err := config.NewMemoryLoader(cfg)
	require.Nil(t, err)
	lds, err := local.NewLocalDataSource(configLoader)
	require.Nil(t, err)
	return lds
}

func defaultConfig() *config.Config {
	return &config.Config{
		TrustZones: []*trust_zone_proto.TrustZone{
			fixtures.TrustZone("tz1"),
			fixtures.TrustZone("tz2"),
		},
		Clusters: []*clusterpb.Cluster{
			fixtures.Cluster("local1"),
			fixtures.Cluster("local2"),
		},
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
			fixtures.AttestationPolicy("ap1"),
			fixtures.AttestationPolicy("ap2"),
		},
		Plugins: fixtures.Plugins("plugins1"),
	}
}
