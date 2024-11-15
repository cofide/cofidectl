// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/provider"
	"github.com/cofide/cofidectl/internal/pkg/provider/helm"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"

	cmdcontext "github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/statusspinner"
	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
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
		generator := helm.NewHelmValuesGenerator(trustZone, source)
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

	clusterIP, err := watchSPIREPodAndService(ctx, trustZone.GetKubernetesContext())
	if err != nil {
		statusCh <- provider.ProviderStatus{Stage: "Waiting", Message: fmt.Sprintf("Failed waiting for SPIRE server pod and service for %s in cluster %s", trustZone.Name, trustZone.GetKubernetesCluster()), Done: true, Error: err}
		return
	}

	bundleEndpointUrl := fmt.Sprintf("https://%s:8443", clusterIP)
	trustZone.BundleEndpointUrl = &bundleEndpointUrl

	// obtain the bundle
	bundle, err := getBundle(ctx, trustZone.GetKubernetesContext())
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

func watchSPIREPodAndService(ctx context.Context, kubeContext string) (string, error) {
	podWatcher, err := createPodWatcher(ctx, kubeContext)
	if err != nil {
		return "", err
	}
	defer podWatcher.Stop()

	serviceWatcher, err := createServiceWatcher(ctx, kubeContext)
	if err != nil {
		return "", err
	}
	defer serviceWatcher.Stop()

	podReady := false
	var serviceIP string

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case event, ok := <-podWatcher.ResultChan():
			if !ok {
				return "", fmt.Errorf("pod watcher channel closed")
			}
			if event.Type == watch.Added || event.Type == watch.Modified {
				pod := event.Object.(*v1.Pod)
				// FieldSelector should ensure this, but use belt & braces.
				if pod.Name != "spire-server-0" {
					slog.Warn("Event received for unexpected pod", slog.String("pod", pod.Name))
				} else if isPodReady(pod) {
					podReady = true
				}
			}
		case event, ok := <-serviceWatcher.ResultChan():
			if !ok {
				return "", fmt.Errorf("service watcher channel closed")
			}
			if event.Type == watch.Added || event.Type == watch.Modified {
				service := event.Object.(*v1.Service)
				// FieldSelector should ensure this, but use belt & braces.
				if service.Name != "spire-server" {
					slog.Warn("Event received for unexpected service", slog.String("service", service.Name))
				} else if ip, err := getServiceExternalIP(service); err == nil {
					serviceIP = ip
				}
			}
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for pod and service to be ready")
		}

		if podReady && serviceIP != "" {
			return serviceIP, nil
		}
	}
}

func getBundle(ctx context.Context, kubeContext string) (string, error) {
	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, kubeContext)
	if err != nil {
		return "", err
	}

	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err = kubeutil.RunCommand(
		ctx,
		client.Clientset,
		client.RestConfig,
		"spire-server-0",
		"spire", // TODO use a const
		"spire-server",
		[]string{"/opt/spire/bin/spire-server", "bundle", "show", "-format", "spiffe"},
		stdin,
		stdout,
		stderr,
	)

	if err != nil {
		return "", err
	}

	bundle := stdout.String()

	stdin.Reset()
	stdout.Reset()
	stderr.Reset()

	return bundle, nil
}

func createPodWatcher(ctx context.Context, kubeContext string) (watch.Interface, error) {
	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, kubeContext)
	if err != nil {
		return nil, err
	}
	watchFunc := func(opts metav1.ListOptions) (watch.Interface, error) {
		timeout := int64(120)
		return client.Clientset.CoreV1().Pods("spire").Watch(ctx, metav1.ListOptions{
			FieldSelector:  "metadata.name=spire-server-0",
			TimeoutSeconds: &timeout,
		})
	}

	watcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher for context %s: %v", kubeContext, err)
	}

	return watcher, nil
}

func createServiceWatcher(ctx context.Context, kubeContext string) (watch.Interface, error) {
	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, kubeContext)
	if err != nil {
		return nil, err
	}
	watchFunc := func(opts metav1.ListOptions) (watch.Interface, error) {
		timeout := int64(120)
		return client.Clientset.CoreV1().Services("spire").Watch(ctx, metav1.ListOptions{
			FieldSelector:  "metadata.name=spire-server",
			TimeoutSeconds: &timeout,
		})
	}

	watcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})
	if err != nil {
		return nil, fmt.Errorf("failed to create service watcher for context %s: %v", kubeContext, err)
	}

	return watcher, nil
}

func applyPostInstallHelmConfig(ctx context.Context, source cofidectl_plugin.DataSource, trustZones []*trust_zone_proto.TrustZone) error {
	for _, trustZone := range trustZones {
		generator := helm.NewHelmValuesGenerator(trustZone, source)

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

func isPodReady(pod *v1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == v1.PodReady && cond.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func getServiceExternalIP(service *v1.Service) (string, error) {
	serviceLoadBalancerIngress := service.Status.LoadBalancer.Ingress
	if len(serviceLoadBalancerIngress) != 1 {
		return "", fmt.Errorf("failed to retrieve the service ingress information")
	}

	// Usually set on AWS load balancers
	ingressHostName := serviceLoadBalancerIngress[0].Hostname
	if ingressHostName != "" {
		return ingressHostName, nil
	}

	// Usually set on GCE/OpenStack load balancers
	ingressIP := serviceLoadBalancerIngress[0].IP
	if ingressIP != "" {
		return ingressIP, nil
	}

	return "", fmt.Errorf("failed to retrieve the service ingress information")
}
