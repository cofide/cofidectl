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

// Type check that SpireHelm implements the Provision interface.
var _ provision.Provision = &SpireHelm{}

// SpireHelm implements the `Provision` interface by deploying a SPIRE cluster using the SPIRE Helm charts.
type SpireHelm struct{}

func NewSpireHelm() *SpireHelm {
	return &SpireHelm{}
}

func (h *SpireHelm) Deploy(ctx context.Context, ds plugin.DataSource, kubeCfgFile string) (<-chan *provisionpb.Status, error) {
	statusCh := make(chan *provisionpb.Status)

	go func() {
		defer close(statusCh)
		if err := deploy(ctx, ds, kubeCfgFile, statusCh); err != nil {
			statusCh <- provision.StatusError("Deploying", "Error", err)
		}
	}()

	return statusCh, nil
}

func (h *SpireHelm) TearDown(ctx context.Context, ds plugin.DataSource) (<-chan *provisionpb.Status, error) {
	statusCh := make(chan *provisionpb.Status)

	go func() {
		defer close(statusCh)
		if err := tearDown(ctx, ds, statusCh); err != nil {
			statusCh <- provision.StatusError("Uninstalling", "Error", err)
		}
	}()

	return statusCh, nil
}

func deploy(ctx context.Context, ds plugin.DataSource, kubeCfgFile string, statusCh chan<- *provisionpb.Status) error {
	trustZones, err := ds.ListTrustZones()
	if err != nil {
		return err
	}
	if len(trustZones) == 0 {
		return fmt.Errorf("no trust zones have been configured")
	}

	if err := addSPIRERepository(ctx, statusCh); err != nil {
		return err
	}

	if err := installSPIREStack(ctx, ds, trustZones, statusCh); err != nil {
		return err
	}

	if err := watchAndConfigure(ctx, ds, trustZones, kubeCfgFile, statusCh); err != nil {
		return err
	}

	if err := applyPostInstallHelmConfig(ctx, ds, trustZones, statusCh); err != nil {
		return err
	}

	// Wait for spire-server to be ready again.
	if err := watchAndConfigure(ctx, ds, trustZones, kubeCfgFile, statusCh); err != nil {
		return err
	}

	return nil
}

func tearDown(ctx context.Context, ds plugin.DataSource, statusCh chan<- *provisionpb.Status) error {
	trustZones, err := ds.ListTrustZones()
	if err != nil {
		return err
	}

	if len(trustZones) == 0 {
		return fmt.Errorf("no trust zones have been configured")
	}

	if err := uninstallSPIREStack(ctx, trustZones, statusCh); err != nil {
		return err
	}
	return nil
}

func addSPIRERepository(ctx context.Context, statusCh chan<- *provisionpb.Status) error {
	emptyValues := map[string]interface{}{}
	prov, err := helm.NewHelmSPIREProvider(ctx, nil, emptyValues, emptyValues)
	if err != nil {
		return err
	}

	return prov.AddRepository(statusCh)
}

func installSPIREStack(ctx context.Context, source plugin.DataSource, trustZones []*trust_zone_proto.TrustZone, statusCh chan<- *provisionpb.Status) error {
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

		if err := prov.Execute(statusCh); err != nil {
			return err
		}
	}
	return nil
}

func watchAndConfigure(ctx context.Context, source plugin.DataSource, trustZones []*trust_zone_proto.TrustZone, kubeCfgFile string, statusCh chan<- *provisionpb.Status) error {
	// wait for SPIRE servers to be available and update status before applying federation(s)
	for _, trustZone := range trustZones {
		if err := getBundleAndEndpoint(ctx, statusCh, source, trustZone, kubeCfgFile); err != nil {
			return err
		}
	}
	return nil
}

func getBundleAndEndpoint(ctx context.Context, statusCh chan<- *provisionpb.Status, source plugin.DataSource, trustZone *trust_zone_proto.TrustZone, kubeCfgFile string) error {
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

	bundleEndpointUrl := fmt.Sprintf("https://%s:8443", clusterIP)
	trustZone.BundleEndpointUrl = &bundleEndpointUrl

	// obtain the bundle
	bundle, err := spire.GetBundle(ctx, client)
	if err != nil {
		statusCh <- sb.Error("Waiting", "Failed obtaining bundle", err)
		return err
	}

	trustZone.Bundle = &bundle

	if err := source.UpdateTrustZone(trustZone); err != nil {
		msg := fmt.Sprintf("Failed updating trust zone %s", trustZone.Name)
		statusCh <- provision.StatusError("Waiting", msg, err)
		return err
	}

	statusCh <- sb.Done("Ready", "All SPIRE server pods and services are ready")
	return nil
}

func applyPostInstallHelmConfig(ctx context.Context, source plugin.DataSource, trustZones []*trust_zone_proto.TrustZone, statusCh chan<- *provisionpb.Status) error {
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

		if err := prov.ExecutePostInstallUpgrade(statusCh); err != nil {
			return err
		}
	}

	return nil
}

func uninstallSPIREStack(ctx context.Context, trustZones []*trust_zone_proto.TrustZone, statusCh chan<- *provisionpb.Status) error {
	for _, trustZone := range trustZones {
		prov, err := helm.NewHelmSPIREProvider(ctx, trustZone, nil, nil)
		if err != nil {
			return err
		}

		if err := prov.ExecuteUninstall(statusCh); err != nil {
			return err
		}
	}
	return nil
}
