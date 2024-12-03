// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/spire"
	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	"github.com/cofide/cofidectl/pkg/provider"
	"github.com/cofide/cofidectl/pkg/provider/helm"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/statusspinner"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/spf13/cobra"
)

type UpCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewUpCommand(cmdCtx *cmdcontext.CommandContext) *UpCommand {
	return &UpCommand{
		cmdCtx: cmdCtx,
	}
}

var upCmdDesc = `
This command installs a Cofide configuration
`

func (u *UpCommand) UpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [ARGS]",
		Short: "Installs a Cofide configuration",
		Long:  upCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := u.cmdCtx.PluginManager.GetDataSource()
			if err != nil {
				return err
			}

			trustZones, err := ds.ListTrustZones()
			if err != nil {
				return err
			}
			if len(trustZones) == 0 {
				return fmt.Errorf("no trust zones have been configured")
			}

			if err := addSPIRERepository(cmd.Context()); err != nil {
				return err
			}

			if err := installSPIREStack(cmd.Context(), ds, trustZones); err != nil {
				return err
			}

			if err := watchAndConfigure(cmd.Context(), ds, trustZones); err != nil {
				return err
			}

			if err := applyPostInstallHelmConfig(cmd.Context(), ds, trustZones); err != nil {
				return err
			}

			// Wait for spire-server to be ready again.
			if err := watchAndConfigure(cmd.Context(), ds, trustZones); err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
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

func installSPIREStack(ctx context.Context, source cofidectl_plugin.DataSource, trustZones []*trust_zone_proto.TrustZone) error {
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

func watchAndConfigure(ctx context.Context, source cofidectl_plugin.DataSource, trustZones []*trust_zone_proto.TrustZone) error {
	// wait for SPIRE servers to be available and update status before applying federation(s)
	for _, trustZone := range trustZones {
		statusCh := make(chan provider.ProviderStatus)

		go getBundleAndEndpoint(ctx, statusCh, source, trustZone)

		s := statusspinner.New()
		if err := s.Watch(statusCh); err != nil {
			return fmt.Errorf("configuration failed: %w", err)
		}
	}
	return nil
}

func getBundleAndEndpoint(ctx context.Context, statusCh chan<- provider.ProviderStatus, source cofidectl_plugin.DataSource, trustZone *trust_zone_proto.TrustZone) {
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

func applyPostInstallHelmConfig(ctx context.Context, source cofidectl_plugin.DataSource, trustZones []*trust_zone_proto.TrustZone) error {
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
