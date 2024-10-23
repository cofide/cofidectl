package cmd

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"

	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/config/local"
	"github.com/cofide/cofidectl/internal/pkg/provider/helm"
	"github.com/fatih/color"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"

	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/spf13/cobra"
)

type UpCommand struct {
	source cofidectl_plugin.DataSource
}

func NewUpCommand(source cofidectl_plugin.DataSource) *UpCommand {
	return &UpCommand{
		source: source,
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
			if err := u.source.Validate(); err != nil {
				return err
			}

			ds, _ := u.source.(*cofidectl_plugin.LocalDataSource)
			configProvider := local.YAMLConfigProvider{DataSource: ds}
			config, err := configProvider.GetConfig()

			if err != nil {
				return err
			}

			if len(config.TrustZones.TrustZones) == 0 {
				return fmt.Errorf("no trust zones have been configured")
			}

			if err := installSPIREStack(config); err != nil {
				return err
			}

			if err := watchAndConfigure(config); err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}

func installSPIREStack(config *config.Config) error {
	for _, trustZone := range config.TrustZones.TrustZones {
		generator := helm.NewHelmValuesGenerator(trustZone, config)
		spireValues, err := generator.GenerateValues()
		if err != nil {
			return err
		}

		spireCRDsValues := map[string]interface{}{}
		prov := helm.NewHelmSPIREProvider(trustZone, spireValues, spireCRDsValues)

		// Create a spinner to display whilst installation is underway
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Start()
		statusCh, err := prov.Execute()
		if err != nil {
			s.Stop()
			return fmt.Errorf("failed to start installation: %w", err)
		}

		for status := range statusCh {
			s.Suffix = fmt.Sprintf(" %s: %s\n", status.Stage, status.Message)

			if status.Done {
				s.Stop()
				if status.Error != nil {
					fmt.Printf("❌ %s: %s\n", status.Stage, status.Message)
					return fmt.Errorf("installation failed: %w", status.Error)
				}
				green := color.New(color.FgGreen).SprintFunc()
				fmt.Printf("%s %s: %s\n\n", green("✅"), status.Stage, status.Message)
			}
		}

		s.Stop()
	}
	return nil
}

func watchAndConfigure(config *config.Config) error {
	// wait for SPIRE servers to be available and update status before applying federation(s)
	for _, trustZone := range config.TrustZones.TrustZones {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Prefix = fmt.Sprintf("Waiting for pod and service in %s: ", trustZone.KubernetesCluster)
		s.Start()

		clusterIP, err := watchSPIREPodAndService(trustZone.KubernetesContext)
		if err != nil {
			s.Stop()
			return fmt.Errorf("error in context %s: %v", trustZone.KubernetesContext, err)
		}

		trustZone.BundleEndpointUrl = clusterIP

		// obtain the bundle
		bundle, err := getBundle(trustZone.KubernetesContext)
		if err != nil {
			s.Stop()
			return fmt.Errorf("error obtaining bundle in context %s: %v", trustZone.KubernetesContext, err)
		}

		trustZone.Bundle = bundle

		s.Stop()
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s All pods and services are ready.\n\n", green("✅"))

	err := applyPostInstallHelmConfig(config)
	if err != nil {
		return err
	}

	return nil
}

func watchSPIREPodAndService(kubeContext string) (string, error) {
	podWatcher, err := createPodWatcher(kubeContext)
	if err != nil {
		return "", err
	}
	defer podWatcher.Stop()

	serviceWatcher, err := createServiceWatcher(kubeContext)
	if err != nil {
		return "", err
	}
	defer serviceWatcher.Stop()

	podReady := false
	serviceReady := false
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
				if isPodReady(pod) {
					podReady = true
				}
			}
		case event, ok := <-serviceWatcher.ResultChan():
			if !ok {
				return "", fmt.Errorf("service watcher channel closed")
			}
			if event.Type == watch.Added || event.Type == watch.Modified {
				service := event.Object.(*v1.Service)
				if isServiceReady(service) {
					serviceReady = true
					serviceIP, _ = getServiceExternalIP(service)
				}
			}
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for pod and service to be ready")
		}

		if podReady && serviceReady {
			return serviceIP, nil
		}
	}
}

func getBundle(kubeContext string) (string, error) {
	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, kubeContext)
	if err != nil {
		return "", err
	}

	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err = kubeutil.RunCommand(
		context.TODO(),
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

func createPodWatcher(kubeContext string) (watch.Interface, error) {
	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, kubeContext)
	if err != nil {
		return nil, err
	}
	watchFunc := func(opts metav1.ListOptions) (watch.Interface, error) {
		timeout := int64(120)
		return client.Clientset.CoreV1().Pods("spire").Watch(context.Background(), metav1.ListOptions{
			TimeoutSeconds: &timeout,
		})
	}

	watcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher for context %s: %v", kubeContext, err)
	}

	return watcher, nil
}

func createServiceWatcher(kubeContext string) (watch.Interface, error) {
	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, kubeContext)
	if err != nil {
		return nil, err
	}
	watchFunc := func(opts metav1.ListOptions) (watch.Interface, error) {
		timeout := int64(120)
		return client.Clientset.CoreV1().Services("spire").Watch(context.Background(), metav1.ListOptions{
			TimeoutSeconds: &timeout,
		})
	}

	watcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})
	if err != nil {
		return nil, fmt.Errorf("failed to create service watcher for context %s: %v", kubeContext, err)
	}

	return watcher, nil
}

func applyPostInstallHelmConfig(config *config.Config) error {
	for _, trustZone := range config.TrustZones.TrustZones {
		generator := helm.NewHelmValuesGenerator(trustZone, config)

		spireValues, err := generator.GenerateValues()
		if err != nil {
			return err
		}

		spireCRDsValues := map[string]interface{}{}

		prov := helm.NewHelmSPIREProvider(trustZone, spireValues, spireCRDsValues)

		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Start()

		statusCh, err := prov.ExecuteUpgrade(true)
		if err != nil {
			s.Stop()
			return fmt.Errorf("failed to start post-installation configuration: %w", err)
		}

		for status := range statusCh {
			s.Suffix = fmt.Sprintf(" %s: %s\n", status.Stage, status.Message)

			if status.Done {
				s.Stop()
				if status.Error != nil {
					fmt.Printf("❌ %s: %s\n", status.Stage, status.Message)
					return fmt.Errorf("post-installation configuration failed: %w", status.Error)
				}
				green := color.New(color.FgGreen).SprintFunc()
				fmt.Printf("%s %s: %s\n\n", green("✅"), status.Stage, status.Message)
			}
		}

		s.Stop()
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

func isServiceReady(service *v1.Service) bool {
	return service.Spec.ClusterIP != ""
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
