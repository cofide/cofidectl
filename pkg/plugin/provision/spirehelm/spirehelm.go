// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spirehelm

import (
	"context"
	"fmt"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/statusspinner"
	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/provision"
	"github.com/cofide/cofidectl/pkg/provider"
	"github.com/cofide/cofidectl/pkg/provider/helm"
	"github.com/cofide/cofidectl/pkg/spire"
)

// Type check that SpireHelm implements the Provision interface.
var _ provision.Provision = &SpireHelm{}

// SpireHelm implements the `Provision` interface by deploying a SPIRE cluster using the SPIRE Helm charts.
type SpireHelm struct{}

func NewSpireHelm() *SpireHelm {
	return &SpireHelm{}
}

func (h *SpireHelm) Deploy(ctx context.Context, ds plugin.DataSource, kubeCfgFile string) error {
	trustZones, err := ds.ListTrustZones()
	if err != nil {
		return err
	}
	if len(trustZones) == 0 {
		return fmt.Errorf("no trust zones have been configured")
	}

	if err := addSPIRERepository(ctx); err != nil {
		return err
	}

	if err := installSPIREStack(ctx, ds, trustZones); err != nil {
		return err
	}

	if err := watchAndConfigure(ctx, ds, trustZones, kubeCfgFile); err != nil {
		return err
	}

	if err := applyPostInstallHelmConfig(ctx, ds, trustZones); err != nil {
		return err
	}

	// Wait for spire-server to be ready again.
	if err := watchAndConfigure(ctx, ds, trustZones, kubeCfgFile); err != nil {
		return err
	}

	return nil
}

func (h *SpireHelm) TearDown(ctx context.Context, ds plugin.DataSource) error {
	trustZones, err := ds.ListTrustZones()
	if err != nil {
		return err
	}

	if len(trustZones) == 0 {
		fmt.Println("no trust zones have been configured")
		return nil
	}

	if err := uninstallSPIREStack(ctx, trustZones); err != nil {
		return err
	}
	return nil
}

func addSPIRERepository(ctx context.Context) error {
	emptyValues := map[string]interface{}{}
	prov, err := helm.NewHelmSPIREProvider(ctx, nil, emptyValues, emptyValues)
	if err != nil {
		return err
	}

	statusCh := prov.AddRepository()
	s := statusspinner.New()
	if err := s.Watch(statusCh); err != nil {
		return fmt.Errorf("adding SPIRE Helm repository failed: %w", err)
	}
	return nil
}

func installSPIREStack(ctx context.Context, source plugin.DataSource, trustZones []*trust_zone_proto.TrustZone) error {
	for _, trustZone := range trustZones {
		generator := helm.NewHelmValuesGenerator(trustZone, source, nil)
		spireValues, err := generator.GenerateValues()
		if err != nil {
			return err
		}

		spireCRDsValues := map[string]interface{}{}
		prov, err := helm.NewHelmSPIREProvider(ctx, trustZone, spireValues, spireCRDsValues)
		if err != nil {
			return err
		}

		statusCh := prov.Execute()

		// Create a spinner to display whilst installation is underway
		s := statusspinner.New()
		if err := s.Watch(statusCh); err != nil {
			return fmt.Errorf("installation failed: %w", err)
		}
	}
	return nil
}

func watchAndConfigure(ctx context.Context, source plugin.DataSource, trustZones []*trust_zone_proto.TrustZone, kubeCfgFile string) error {
	// wait for SPIRE servers to be available and update status before applying federation(s)
	for _, trustZone := range trustZones {
		statusCh := make(chan provider.ProviderStatus)

		go getBundleAndEndpoint(ctx, statusCh, source, trustZone, kubeCfgFile)

		s := statusspinner.New()
		if err := s.Watch(statusCh); err != nil {
			return fmt.Errorf("configuration failed: %w", err)
		}
	}
	return nil
}

func getBundleAndEndpoint(ctx context.Context, statusCh chan<- provider.ProviderStatus, source plugin.DataSource, trustZone *trust_zone_proto.TrustZone, kubeCfgFile string) {
	defer close(statusCh)
	statusCh <- provider.ProviderStatus{Stage: "Waiting", Message: fmt.Sprintf("Waiting for SPIRE server pod and service for %s in cluster %s", trustZone.Name, trustZone.GetKubernetesCluster())}

	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, trustZone.GetKubernetesContext())
	if err != nil {
		statusCh <- provider.ProviderStatus{Stage: "Waiting", Message: fmt.Sprintf("Failed waiting for SPIRE server pod and service for %s in cluster %s", trustZone.Name, trustZone.GetKubernetesCluster()), Done: true, Error: err}
		return
	}

	clusterIP, err := spire.WaitForServerIP(ctx, client)
	if err != nil {
		statusCh <- provider.ProviderStatus{Stage: "Waiting", Message: fmt.Sprintf("Failed waiting for SPIRE server pod and service for %s in cluster %s", trustZone.Name, trustZone.GetKubernetesCluster()), Done: true, Error: err}
		return
	}

	bundleEndpointUrl := fmt.Sprintf("https://%s:8443", clusterIP)
	trustZone.BundleEndpointUrl = &bundleEndpointUrl

	// obtain the bundle
	bundle, err := spire.GetBundle(ctx, client)
	if err != nil {
		statusCh <- provider.ProviderStatus{Stage: "Waiting", Message: fmt.Sprintf("Failed obtaining bundle for %s in cluster %s", trustZone.Name, trustZone.GetKubernetesCluster()), Done: true, Error: err}
		return
	}

	trustZone.Bundle = &bundle

	if err := source.UpdateTrustZone(trustZone); err != nil {
		statusCh <- provider.ProviderStatus{Stage: "Waiting", Message: fmt.Sprintf("Failed updating trust zone %s", trustZone.Name), Done: true, Error: err}
		return
	}

	statusCh <- provider.ProviderStatus{Stage: "Ready", Message: fmt.Sprintf("All SPIRE server pods and services are ready for %s in cluster %s", trustZone.Name, trustZone.GetKubernetesCluster()), Done: true}
}

func applyPostInstallHelmConfig(ctx context.Context, source plugin.DataSource, trustZones []*trust_zone_proto.TrustZone) error {
	for _, trustZone := range trustZones {
		generator := helm.NewHelmValuesGenerator(trustZone, source, nil)

		spireValues, err := generator.GenerateValues()
		if err != nil {
			return err
		}

		spireCRDsValues := map[string]interface{}{}

		prov, err := helm.NewHelmSPIREProvider(ctx, trustZone, spireValues, spireCRDsValues)
		if err != nil {
			return err
		}

		statusCh := prov.ExecuteUpgrade(true)

		s := statusspinner.New()
		if err := s.Watch(statusCh); err != nil {
			return fmt.Errorf("post-installation configuration failed: %w", err)
		}
	}

	return nil
}

func uninstallSPIREStack(ctx context.Context, trustZones []*trust_zone_proto.TrustZone) error {
	for _, trustZone := range trustZones {
		prov, err := helm.NewHelmSPIREProvider(ctx, trustZone, nil, nil)
		if err != nil {
			return err
		}

		s := statusspinner.New()
		statusCh := prov.ExecuteUninstall()
		if err := s.Watch(statusCh); err != nil {
			return fmt.Errorf("uninstallation failed: %w", err)
		}
	}
	return nil
}
