// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spirehelm

import (
	"context"
	"fmt"
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/provision_plugin/v1alpha2"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/cofide/cofidectl/pkg/plugin/provision"
	"github.com/cofide/cofidectl/pkg/provider/helm"
	spiretypes "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpireHelm_Deploy(t *testing.T) {
	providerFactory := newFakeHelmSPIREProviderFactory()
	spireAPIFactory := newFakeSPIREAPIFactory()
	spireHelm := NewSpireHelm(providerFactory, spireAPIFactory)
	ds := newFakeDataSource(t, defaultConfig())

	tests := []struct {
		name string
		opts provision.DeployOpts
		want []*provisionpb.Status
	}{
		{
			name: "basic",
			opts: provision.DeployOpts{KubeCfgFile: "fake-kube.cfg"},
			want: []*provisionpb.Status{
				provision.StatusOk("Preparing", "Adding SPIRE Helm repo"),
				provision.StatusDone("Prepared", "Added SPIRE Helm repo"),
				provision.StatusOk("Installing", "Installing SPIRE CRDs for local1 in tz1"),
				provision.StatusOk("Installing", "Installing SPIRE chart for local1 in tz1"),
				provision.StatusDone("Installed", "Installation completed for local1 in tz1"),
				provision.StatusOk("Installing", "Installing SPIRE CRDs for local2 in tz2"),
				provision.StatusOk("Installing", "Installing SPIRE chart for local2 in tz2"),
				provision.StatusDone("Installed", "Installation completed for local2 in tz2"),
				provision.StatusOk("Waiting", "Waiting for SPIRE server pod and service for local1 in tz1"),
				provision.StatusDone("Ready", "All SPIRE server pods and services are ready for local1 in tz1"),
				provision.StatusOk("Waiting", "Waiting for SPIRE server pod and service for local2 in tz2"),
				provision.StatusDone("Ready", "All SPIRE server pods and services are ready for local2 in tz2"),
				provision.StatusOk("Configuring", "Applying post-installation configuration for local1 in tz1"),
				provision.StatusDone("Configured", "Post-installation configuration completed for local1 in tz1"),
				provision.StatusOk("Configuring", "Applying post-installation configuration for local2 in tz2"),
				provision.StatusDone("Configured", "Post-installation configuration completed for local2 in tz2"),
				provision.StatusOk("Waiting", "Waiting for SPIRE server pod and service for local1 in tz1"),
				provision.StatusDone("Ready", "All SPIRE server pods and services are ready for local1 in tz1"),
				provision.StatusOk("Waiting", "Waiting for SPIRE server pod and service for local2 in tz2"),
				provision.StatusDone("Ready", "All SPIRE server pods and services are ready for local2 in tz2"),
			},
		},
		{
			name: "skip wait",
			opts: provision.DeployOpts{KubeCfgFile: "fake-kube.cfg", SkipWait: true},
			want: []*provisionpb.Status{
				provision.StatusOk("Preparing", "Adding SPIRE Helm repo"),
				provision.StatusDone("Prepared", "Added SPIRE Helm repo"),
				provision.StatusOk("Installing", "Installing SPIRE CRDs for local1 in tz1"),
				provision.StatusOk("Installing", "Installing SPIRE chart for local1 in tz1"),
				provision.StatusDone("Installed", "Installation completed for local1 in tz1"),
				provision.StatusOk("Installing", "Installing SPIRE CRDs for local2 in tz2"),
				provision.StatusOk("Installing", "Installing SPIRE chart for local2 in tz2"),
				provision.StatusDone("Installed", "Installation completed for local2 in tz2"),
			},
		},
	}

	for _, tt := range tests {
		statusCh, err := spireHelm.Deploy(context.Background(), ds, &tt.opts)
		require.NoError(t, err, err)
		statuses := collectStatuses(statusCh)
		assert.EqualExportedValues(t, tt.want, statuses)
	}
}

func TestSpireHelm_Deploy_ExternalServer(t *testing.T) {
	providerFactory := newFakeHelmSPIREProviderFactory()
	spireAPIFactory := newFakeSPIREAPIFactory()
	spireHelm := NewSpireHelm(providerFactory, spireAPIFactory)

	config := &config.Config{
		TrustZones: []*trust_zone_proto.TrustZone{
			fixtures.TrustZone("tz5"),
		},
		Clusters: []*clusterpb.Cluster{
			fixtures.Cluster("local5"),
		},
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
		Plugins:             fixtures.Plugins("plugins1"),
	}

	ds := newFakeDataSource(t, config)

	opts := provision.DeployOpts{KubeCfgFile: "fake-kube.cfg"}
	statusCh, err := spireHelm.Deploy(context.Background(), ds, &opts)
	require.NoError(t, err, err)

	statuses := collectStatuses(statusCh)
	want := []*provisionpb.Status{
		provision.StatusOk("Preparing", "Adding SPIRE Helm repo"),
		provision.StatusDone("Prepared", "Added SPIRE Helm repo"),
		provision.StatusOk("Installing", "Installing SPIRE CRDs for local5 in tz5"),
		provision.StatusOk("Installing", "Installing SPIRE chart for local5 in tz5"),
		provision.StatusDone("Installed", "Installation completed for local5 in tz5"),
		provision.StatusDone("Ready", "Skipped waiting for external SPIRE server pod and service for local5 in tz5"),
		provision.StatusOk("Configuring", "Applying post-installation configuration for local5 in tz5"),
		provision.StatusDone("Configured", "Post-installation configuration completed for local5 in tz5"),
		provision.StatusDone("Ready", "Skipped waiting for external SPIRE server pod and service for local5 in tz5"),
	}
	assert.EqualExportedValues(t, want, statuses)
}

func TestSpireHelm_Deploy_specificTrustZone(t *testing.T) {
	providerFactory := newFakeHelmSPIREProviderFactory()
	spireAPIFactory := newFakeSPIREAPIFactory()
	spireHelm := NewSpireHelm(providerFactory, spireAPIFactory)
	ds := newFakeDataSource(t, defaultConfig())

	opts := provision.DeployOpts{KubeCfgFile: "fake-kube.cfg", TrustZoneIDs: []string{"tz2-id"}}
	statusCh, err := spireHelm.Deploy(context.Background(), ds, &opts)
	require.NoError(t, err, err)

	statuses := collectStatuses(statusCh)
	want := []*provisionpb.Status{
		provision.StatusOk("Preparing", "Adding SPIRE Helm repo"),
		provision.StatusDone("Prepared", "Added SPIRE Helm repo"),
		provision.StatusOk("Installing", "Installing SPIRE CRDs for local2 in tz2"),
		provision.StatusOk("Installing", "Installing SPIRE chart for local2 in tz2"),
		provision.StatusDone("Installed", "Installation completed for local2 in tz2"),
		provision.StatusOk("Waiting", "Waiting for SPIRE server pod and service for local2 in tz2"),
		provision.StatusDone("Ready", "All SPIRE server pods and services are ready for local2 in tz2"),
		provision.StatusOk("Configuring", "Applying post-installation configuration for local2 in tz2"),
		provision.StatusDone("Configured", "Post-installation configuration completed for local2 in tz2"),
		provision.StatusOk("Waiting", "Waiting for SPIRE server pod and service for local2 in tz2"),
		provision.StatusDone("Ready", "All SPIRE server pods and services are ready for local2 in tz2"),
	}
	assert.EqualExportedValues(t, want, statuses)
}

func TestSpireHelm_Deploy_with_env_HELM_REPO_PATH(t *testing.T) {
	providerFactory := newFakeHelmSPIREProviderFactory()
	spireAPIFactory := newFakeSPIREAPIFactory()
	spireHelm := NewSpireHelm(providerFactory, spireAPIFactory)
	ds := newFakeDataSource(t, defaultConfig())

	opts := provision.DeployOpts{KubeCfgFile: "fake-kube.cfg", TrustZoneIDs: []string{"tz2-id"}}
	statusCh, err := spireHelm.Deploy(context.Background(), ds, &opts)
	require.NoError(t, err, err)

	dummyPath := "/some/non/zero/path"
	t.Setenv("HELM_REPO_PATH", dummyPath)

	statuses := collectStatuses(statusCh)
	want := []*provisionpb.Status{
		provision.StatusOk("Preparing", fmt.Sprintf("Found HELM_REPO_PATH value, using local chart: %s", dummyPath)),
		provision.StatusOk("Installing", "Installing SPIRE CRDs for local2 in tz2"),
		provision.StatusOk("Installing", "Installing SPIRE chart for local2 in tz2"),
		provision.StatusDone("Installed", "Installation completed for local2 in tz2"),
		provision.StatusOk("Waiting", "Waiting for SPIRE server pod and service for local2 in tz2"),
		provision.StatusDone("Ready", "All SPIRE server pods and services are ready for local2 in tz2"),
		provision.StatusOk("Configuring", "Applying post-installation configuration for local2 in tz2"),
		provision.StatusDone("Configured", "Post-installation configuration completed for local2 in tz2"),
		provision.StatusOk("Waiting", "Waiting for SPIRE server pod and service for local2 in tz2"),
		provision.StatusDone("Ready", "All SPIRE server pods and services are ready for local2 in tz2"),
	}
	assert.EqualExportedValues(t, want, statuses)
}

func TestSpireHelm_TearDown(t *testing.T) {
	providerFactory := newFakeHelmSPIREProviderFactory()
	spireAPIFactory := newFakeSPIREAPIFactory()
	spireHelm := NewSpireHelm(providerFactory, spireAPIFactory)
	ds := newFakeDataSource(t, defaultConfig())

	opts := provision.TearDownOpts{KubeCfgFile: "fake-kube.cfg"}
	statusCh, err := spireHelm.TearDown(context.Background(), ds, &opts)
	require.NoError(t, err, err)

	statuses := collectStatuses(statusCh)
	want := []*provisionpb.Status{
		provision.StatusOk("Uninstalling", "Uninstalling SPIRE chart for local1 in tz1"),
		provision.StatusDone("Uninstalled", "Uninstallation completed for local1 in tz1"),
		provision.StatusOk("Uninstalling", "Uninstalling SPIRE chart for local2 in tz2"),
		provision.StatusDone("Uninstalled", "Uninstallation completed for local2 in tz2"),
	}
	assert.EqualExportedValues(t, want, statuses)
}

func TestSpireHelm_TearDown_specificTrustZone(t *testing.T) {
	providerFactory := newFakeHelmSPIREProviderFactory()
	spireAPIFactory := newFakeSPIREAPIFactory()
	spireHelm := NewSpireHelm(providerFactory, spireAPIFactory)
	ds := newFakeDataSource(t, defaultConfig())

	opts := provision.TearDownOpts{KubeCfgFile: "fake-kube.cfg", TrustZoneIDs: []string{"tz1-id"}}
	statusCh, err := spireHelm.TearDown(context.Background(), ds, &opts)
	require.NoError(t, err, err)

	statuses := collectStatuses(statusCh)
	want := []*provisionpb.Status{
		provision.StatusOk("Uninstalling", "Uninstalling SPIRE chart for local1 in tz1"),
		provision.StatusDone("Uninstalled", "Uninstallation completed for local1 in tz1"),
	}
	assert.EqualExportedValues(t, want, statuses)
}

func TestSpireHelm_GetHelmValues(t *testing.T) {
	providerFactory := newFakeHelmSPIREProviderFactory()
	spireAPIFactory := newFakeSPIREAPIFactory()
	spireHelm := NewSpireHelm(providerFactory, spireAPIFactory)
	ds := newFakeDataSource(t, defaultConfig())

	opts := provision.GetHelmValuesOpts{ClusterID: "local1-id"}
	values, err := spireHelm.GetHelmValues(context.Background(), ds, &opts)
	require.NoError(t, err, err)
	want := map[string]any{"key1": "value1", "key2": "value2"}
	assert.EqualExportedValues(t, want, values)
}

func collectStatuses(statusCh <-chan *provisionpb.Status) []*provisionpb.Status {
	statuses := []*provisionpb.Status{}
	for status := range statusCh {
		statuses = append(statuses, status)
	}
	return statuses
}

type fakeHelmSPIREProviderFactory struct{}

func newFakeHelmSPIREProviderFactory() *fakeHelmSPIREProviderFactory {
	return &fakeHelmSPIREProviderFactory{}
}

func (f *fakeHelmSPIREProviderFactory) Build(
	ctx context.Context,
	ds datasource.DataSource,
	trustZone *trust_zone_proto.TrustZone,
	cluster *clusterpb.Cluster,
	genValues bool,
	kubeConfig string,
) (helm.Provider, error) {
	return newFakeHelmSPIREProvider(trustZone, cluster), nil
}

func (f *fakeHelmSPIREProviderFactory) GetHelmValues(
	ctx context.Context,
	ds datasource.DataSource,
	trustZone *trust_zone_proto.TrustZone,
	cluster *clusterpb.Cluster,
) (map[string]any, error) {
	return map[string]any{"key1": "value1", "key2": "value2"}, nil
}

// fakeHelmSPIREProvider implements a fake helm.Provider that can be used in testing.
type fakeHelmSPIREProvider struct {
	trustZone *trust_zone_proto.TrustZone
	cluster   *clusterpb.Cluster
}

func newFakeHelmSPIREProvider(trustZone *trust_zone_proto.TrustZone, cluster *clusterpb.Cluster) helm.Provider {
	return &fakeHelmSPIREProvider{trustZone: trustZone, cluster: cluster}
}

func (p *fakeHelmSPIREProvider) AddRepository(statusCh chan<- *provisionpb.Status) error {
	statusCh <- provision.StatusOk("Preparing", "Adding SPIRE Helm repo")
	statusCh <- provision.StatusDone("Prepared", "Added SPIRE Helm repo")
	return nil
}

func (p *fakeHelmSPIREProvider) Execute(statusCh chan<- *provisionpb.Status) error {
	sb := provision.NewStatusBuilder(p.trustZone.Name, p.cluster.GetName())
	statusCh <- sb.Ok("Installing", "Installing SPIRE CRDs")
	statusCh <- sb.Ok("Installing", "Installing SPIRE chart")
	statusCh <- sb.Done("Installed", "Installation completed")
	return nil
}

func (p *fakeHelmSPIREProvider) ExecutePostInstallUpgrade(statusCh chan<- *provisionpb.Status) error {
	sb := provision.NewStatusBuilder(p.trustZone.Name, p.cluster.GetName())
	statusCh <- sb.Ok("Configuring", "Applying post-installation configuration")
	statusCh <- sb.Done("Configured", "Post-installation configuration completed")
	return nil
}

func (p *fakeHelmSPIREProvider) ExecuteUpgrade(statusCh chan<- *provisionpb.Status) error {
	sb := provision.NewStatusBuilder(p.trustZone.Name, p.cluster.GetName())
	statusCh <- sb.Ok("Upgrading", "Upgrading SPIRE chart")
	statusCh <- sb.Done("Upgraded", "Upgrade completed")
	return nil
}

func (p *fakeHelmSPIREProvider) ExecuteUninstall(statusCh chan<- *provisionpb.Status) error {
	sb := provision.NewStatusBuilder(p.trustZone.Name, p.cluster.GetName())
	statusCh <- sb.Ok("Uninstalling", "Uninstalling SPIRE chart")
	statusCh <- sb.Done("Uninstalled", "Uninstallation completed")
	return nil
}

func (p *fakeHelmSPIREProvider) CheckIfAlreadyInstalled() (bool, error) {
	return false, nil
}

type fakeSPIREAPIFactory struct{}

func newFakeSPIREAPIFactory() SPIREAPIFactory {
	return &fakeSPIREAPIFactory{}
}

func (f *fakeSPIREAPIFactory) Build(kubeCfgFile, kubeContext string) (SPIREAPI, error) {
	return &fakeSPIREAPI{}, nil
}

type fakeSPIREAPI struct {
	ip        string
	ipErr     error
	bundle    *spiretypes.Bundle
	bundleErr error
}

func (s *fakeSPIREAPI) WaitForServerIP(ctx context.Context) (string, error) {
	return s.ip, s.ipErr
}

func (s *fakeSPIREAPI) GetBundle(ctx context.Context) (*spiretypes.Bundle, error) {
	return s.bundle, s.bundleErr
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
