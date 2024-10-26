package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/briandowns/spinner"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
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

			trustZones, err := u.source.ListTrustZones()
			if err != nil {
				return err
			}
			if len(trustZones) == 0 {
				return fmt.Errorf("no trust zones have been configured")
			}

			if err := u.installSPIREStack(trustZones); err != nil {
				return err
			}

			if err := u.watchAndConfigure(trustZones); err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}

func (u *UpCommand) installSPIREStack(trustZones []*trust_zone_proto.TrustZone) error {
	for _, trustZone := range trustZones {
		generator := helm.NewHelmValuesGenerator(trustZone, u.source)
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

func (u *UpCommand) watchAndConfigure(trustZones []*trust_zone_proto.TrustZone) error {
	// wait for SPIRE servers to be available and update status before applying federation(s)
	for _, trustZone := range trustZones {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Suffix = fmt.Sprintf(" Waiting for SPIRE server pod and service for %s in cluster %s", trustZone.Name, trustZone.KubernetesCluster)
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
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s All SPIRE server pods and services are ready for %s in cluster %s\n\n", green("✅"), trustZone.Name, trustZone.KubernetesCluster)
	}

	if err := u.applyPostInstallHelmConfig(trustZones); err != nil {
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

func createServiceWatcher(kubeContext string) (watch.Interface, error) {
	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, kubeContext)
	if err != nil {
		return nil, err
	}
	watchFunc := func(opts metav1.ListOptions) (watch.Interface, error) {
		timeout := int64(120)
		return client.Clientset.CoreV1().Services("spire").Watch(context.Background(), metav1.ListOptions{
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

func (u *UpCommand) applyPostInstallHelmConfig(trustZones []*trust_zone_proto.TrustZone) error {
	for _, trustZone := range trustZones {
		generator := helm.NewHelmValuesGenerator(trustZone, u.source)

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
