// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spirehelm

import (
	"context"
	"errors"
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/provision_plugin/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/cofide/cofidectl/pkg/plugin/provision"
	"github.com/cofide/cofidectl/pkg/provider/helm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpireHelm_Deploy(t *testing.T) {
	providerFactory := newFakeHelmSPIREProviderFactory()
	spireHelm := NewSpireHelm(providerFactory)
	ds := newFakeDataSource(t, defaultConfig())

	statusCh, err := spireHelm.Deploy(context.Background(), ds, "fake-kube.cfg")
	require.NoError(t, err, err)

	statuses := collectStatuses(statusCh)
	want := []*provisionpb.Status{
		provision.StatusOk("Preparing", "Adding SPIRE Helm repo"),
		provision.StatusDone("Prepared", "Added SPIRE Helm repo"),
		provision.StatusOk("Installing", "Installing SPIRE CRDs for local1 in tz1"),
		provision.StatusOk("Installing", "Installing SPIRE chart for local1 in tz1"),
		provision.StatusDone("Installed", "Installation completed for local1 in tz1"),
		provision.StatusOk("Installing", "Installing SPIRE CRDs for local2 in tz2"),
		provision.StatusOk("Installing", "Installing SPIRE chart for local2 in tz2"),
		provision.StatusDone("Installed", "Installation completed for local2 in tz2"),
		provision.StatusOk("Waiting", "Waiting for SPIRE server pod and service for local1 in tz1"),
		// FIXME: This attempts to create a real Kubernetes client and fails.
		provision.StatusError(
			"Waiting",
			"Failed waiting for SPIRE server pod and service for local1 in tz1",
			errors.New("load from file: open fake-kube.cfg: no such file or directory"),
		),
	}
	assert.EqualExportedValues(t, want, statuses)
}

func TestSpireHelm_TearDown(t *testing.T) {
	providerFactory := newFakeHelmSPIREProviderFactory()
	spireHelm := NewSpireHelm(providerFactory)
	ds := newFakeDataSource(t, defaultConfig())

	statusCh, err := spireHelm.TearDown(context.Background(), ds, "fake-kube.cfg")
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

func (f *fakeHelmSPIREProviderFactory) Build(ctx context.Context, ds datasource.DataSource, trustZone *trust_zone_proto.TrustZone, genValues bool) (helm.Provider, error) {
	return newFakeHelmSPIREProvider(trustZone), nil
}

// fakeHelmSPIREProvider implements a fake helm.Provider that can be used in testing.
type fakeHelmSPIREProvider struct {
	trustZone *trust_zone_proto.TrustZone
}

func newFakeHelmSPIREProvider(trustZone *trust_zone_proto.TrustZone) helm.Provider {
	return &fakeHelmSPIREProvider{trustZone: trustZone}
}

func (p *fakeHelmSPIREProvider) AddRepository(statusCh chan<- *provisionpb.Status) error {
	statusCh <- provision.StatusOk("Preparing", "Adding SPIRE Helm repo")
	statusCh <- provision.StatusDone("Prepared", "Added SPIRE Helm repo")
	return nil
}

func (p *fakeHelmSPIREProvider) Execute(statusCh chan<- *provisionpb.Status) error {
	sb := provision.NewStatusBuilder(p.trustZone.Name, p.trustZone.GetKubernetesCluster())
	statusCh <- sb.Ok("Installing", "Installing SPIRE CRDs")
	statusCh <- sb.Ok("Installing", "Installing SPIRE chart")
	statusCh <- sb.Done("Installed", "Installation completed")
	return nil
}

func (p *fakeHelmSPIREProvider) ExecutePostInstallUpgrade(statusCh chan<- *provisionpb.Status) error {
	return nil
}

func (p *fakeHelmSPIREProvider) ExecuteUpgrade(statusCh chan<- *provisionpb.Status) error {
	return nil
}

func (p *fakeHelmSPIREProvider) ExecuteUninstall(statusCh chan<- *provisionpb.Status) error {
	sb := provision.NewStatusBuilder(p.trustZone.Name, p.trustZone.GetKubernetesCluster())
	statusCh <- sb.Ok("Uninstalling", "Uninstalling SPIRE chart")
	statusCh <- sb.Done("Uninstalled", "Uninstallation completed")
	return nil
}

func (p *fakeHelmSPIREProvider) CheckIfAlreadyInstalled() (bool, error) {
	return false, nil
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
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
			fixtures.AttestationPolicy("ap1"),
			fixtures.AttestationPolicy("ap2"),
		},
		Plugins: fixtures.Plugins("plugins1"),
	}
}
