// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spirehelm

import (
	"context"
	"fmt"

	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/provision_plugin/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"

	"github.com/cofide/cofidectl/internal/pkg/spire"
	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/provision"
	"github.com/cofide/cofidectl/pkg/provider/helm"
)

// Control flow and error handling require some care in this package due to the asynchronous nature
// of some of the methods. In general, if a function or method has been passed a Status channel,
// then any errors raised (rather than propagated) there should also be sent to the Status channel.
// They should also return the error, to allow the caller to halt execution early.

// Type check that SpireHelm implements the Provision interface.
var _ provision.Provision = &SpireHelm{}

// SpireHelm implements the `Provision` interface by deploying a SPIRE cluster using the SPIRE Helm charts.
type SpireHelm struct {
	providerFactory ProviderFactory
}

func NewSpireHelm(providerFactory ProviderFactory) *SpireHelm {
	if providerFactory == nil {
		providerFactory = &HelmSPIREProviderFactory{}
	}
	return &SpireHelm{providerFactory: providerFactory}
}

func (h *SpireHelm) Deploy(ctx context.Context, ds plugin.DataSource, kubeCfgFile string) (<-chan *provisionpb.Status, error) {
	statusCh := make(chan *provisionpb.Status)

	go func() {
		defer close(statusCh)
		// Ignore returned errors - they should be sent via the Status channel.
		_ = h.deploy(ctx, ds, kubeCfgFile, statusCh)
	}()

	return statusCh, nil
}

func (h *SpireHelm) TearDown(ctx context.Context, ds plugin.DataSource) (<-chan *provisionpb.Status, error) {
	statusCh := make(chan *provisionpb.Status)

	go func() {
		defer close(statusCh)
		// Ignore returned errors - they should be sent via the Status channel.
		_ = h.tearDown(ctx, ds, statusCh)
	}()

	return statusCh, nil
}

func (h *SpireHelm) deploy(ctx context.Context, ds plugin.DataSource, kubeCfgFile string, statusCh chan<- *provisionpb.Status) error {
	trustZones, err := h.ListTrustZones(ds)
	if err != nil {
		statusCh <- provision.StatusError("Deploying", "Failed listing trust zones", err)
		return err
	}

	if err := h.AddSPIRERepository(ctx, statusCh); err != nil {
		return err
	}

	if err := h.InstallSPIREStack(ctx, ds, trustZones, statusCh); err != nil {
		return err
	}

	if err := h.WatchAndConfigure(ctx, ds, trustZones, kubeCfgFile, statusCh); err != nil {
		return err
	}

	if err := h.ApplyPostInstallHelmConfig(ctx, ds, trustZones, statusCh); err != nil {
		return err
	}

	// Wait for spire-server to be ready again.
	if err := h.WatchAndConfigure(ctx, ds, trustZones, kubeCfgFile, statusCh); err != nil {
		return err
	}

	return nil
}

func (h *SpireHelm) tearDown(ctx context.Context, ds plugin.DataSource, statusCh chan<- *provisionpb.Status) error {
	trustZones, err := h.ListTrustZones(ds)
	if err != nil {
		statusCh <- provision.StatusError("Uninstalling", "Failed listing trust zones", err)
		return err
	}

	if err := h.UninstallSPIREStack(ctx, trustZones, statusCh); err != nil {
		return err
	}
	return nil
}

// ListTrustZones returns a list of all trust zones. If no trust zones exist, it returns an error.
func (h *SpireHelm) ListTrustZones(ds plugin.DataSource) ([]*trust_zone_proto.TrustZone, error) {
	trustZones, err := ds.ListTrustZones()
	if err != nil {
		return nil, err
	}

	if len(trustZones) == 0 {
		return nil, fmt.Errorf("no trust zones have been configured")
	}
	return trustZones, nil
}

func (h *SpireHelm) AddSPIRERepository(ctx context.Context, statusCh chan<- *provisionpb.Status) error {
	prov, err := helm.NewHelmSPIREProvider(ctx, nil, nil, nil)
	if err != nil {
		statusCh <- provision.StatusError("Preparing", "Failed to create Helm SPIRE provider", err)
		return err
	}

	return prov.AddRepository(statusCh)
}

func (h *SpireHelm) InstallSPIREStack(ctx context.Context, ds plugin.DataSource, trustZones []*trust_zone_proto.TrustZone, statusCh chan<- *provisionpb.Status) error {
	for _, trustZone := range trustZones {
		prov, err := h.providerFactory.Build(ctx, ds, trustZone)
		if err != nil {
			sb := provision.NewStatusBuilder(trustZone.Name, trustZone.GetKubernetesCluster())
			statusCh <- sb.Error("Deploying", "Failed to create Helm SPIRE provider", err)
			return err
		}

		if err := prov.Execute(statusCh); err != nil {
			return err
		}
	}
	return nil
}

func (h *SpireHelm) WatchAndConfigure(ctx context.Context, ds plugin.DataSource, trustZones []*trust_zone_proto.TrustZone, kubeCfgFile string, statusCh chan<- *provisionpb.Status) error {
	// Wait for SPIRE servers to be available and update status before applying federation(s)
	for _, trustZone := range trustZones {
		if err := h.GetBundleAndEndpoint(ctx, statusCh, ds, trustZone, kubeCfgFile); err != nil {
			return err
		}
	}
	return nil
}

func (h *SpireHelm) GetBundleAndEndpoint(ctx context.Context, statusCh chan<- *provisionpb.Status, ds plugin.DataSource, trustZone *trust_zone_proto.TrustZone, kubeCfgFile string) error {
	sb := provision.NewStatusBuilder(trustZone.Name, trustZone.GetKubernetesCluster())
	statusCh <- sb.Ok("Waiting", "aiting for SPIRE server pod and service")

	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, trustZone.GetKubernetesContext())
	if err != nil {
		statusCh <- sb.Error("Waiting", "Failed waiting for SPIRE server pod and service", err)
		return err
	}

	clusterIP, err := spire.WaitForServerIP(ctx, client)
	if err != nil {
		statusCh <- sb.Error("Waiting", "Failed waiting for SPIRE server pod and service", err)
		return err
	}

	if trustZone.GetBundleEndpointProfile() == trust_zone_proto.BundleEndpointProfile_BUNDLE_ENDPOINT_PROFILE_HTTPS_SPIFFE {
		bundleEndpointUrl := fmt.Sprintf("https://%s:8443", clusterIP)
		trustZone.BundleEndpointUrl = &bundleEndpointUrl

		// Obtain the bundle
		bundle, err := spire.GetBundle(ctx, client)
		if err != nil {
			statusCh <- sb.Error("Waiting", "Failed obtaining bundle", err)
			return err
		}

		trustZone.Bundle = &bundle

		if err := ds.UpdateTrustZone(trustZone); err != nil {
			msg := fmt.Sprintf("Failed updating trust zone %s", trustZone.Name)
			statusCh <- provision.StatusError("Waiting", msg, err)
			return err
		}
	}

	statusCh <- sb.Done("Ready", "All SPIRE server pods and services are ready")
	return nil
}

func (h *SpireHelm) ApplyPostInstallHelmConfig(ctx context.Context, ds plugin.DataSource, trustZones []*trust_zone_proto.TrustZone, statusCh chan<- *provisionpb.Status) error {
	for _, trustZone := range trustZones {
		prov, err := h.providerFactory.Build(ctx, ds, trustZone)
		if err != nil {
			sb := provision.NewStatusBuilder(trustZone.Name, trustZone.GetKubernetesCluster())
			statusCh <- sb.Error("Configuring", "Failed to create Helm SPIRE provider", err)
			return err
		}

		if err := prov.ExecutePostInstallUpgrade(statusCh); err != nil {
			return err
		}
	}

	return nil
}

func (h *SpireHelm) UninstallSPIREStack(ctx context.Context, trustZones []*trust_zone_proto.TrustZone, statusCh chan<- *provisionpb.Status) error {
	for _, trustZone := range trustZones {
		prov, err := helm.NewHelmSPIREProvider(ctx, trustZone, nil, nil)
		if err != nil {
			sb := provision.NewStatusBuilder(trustZone.Name, trustZone.GetKubernetesCluster())
			statusCh <- sb.Error("Uninstalling", "Failed to create Helm SPIRE provider", err)
			return err
		}

		if err := prov.ExecuteUninstall(statusCh); err != nil {
			return err
		}
	}
	return nil
}
