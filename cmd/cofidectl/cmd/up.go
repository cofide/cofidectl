package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"

	"github.com/cofide/cofidectl/internal/pkg/attestationpolicy"
	"github.com/cofide/cofidectl/internal/pkg/federation"
	"github.com/cofide/cofidectl/internal/pkg/provider/helm"
	"github.com/cofide/cofidectl/internal/pkg/trustzone"
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
This command deploys a Cofide configuration
`

func (u *UpCommand) UpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [ARGS]",
		Short: "Deploy a Cofide configuration",
		Long:  upCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			//configProvider := local.YAMLConfigProvider{DataSource: u.source}
			//config, err := configProvider.GetConfig()

			trustZoneProtos, err := u.source.ListTrustZones()
			if err != nil {
				return err
			}

			if len(trustZoneProtos) == 0 {
				fmt.Println("no trust zones have been configured")
				return nil
			}

			// convert to structs
			trustZones := make(map[string]*trustzone.TrustZone, len(trustZoneProtos))
			for _, trustZoneProto := range trustZoneProtos {
				federationProtos, _ := u.source.ListFederationByTrustZone(trustZoneProto.Name)
				// convert to structs
				federations := make(map[string]*federation.Federation, len(federationProtos))
				for _, federationProto := range federationProtos {
					federations[federationProto.Right.Name] = federation.NewFederation(federationProto)
				}
				trustZoneStruct := trustzone.NewTrustZone(trustZoneProto)
				trustZoneStruct.Federations = federations
				trustZones[trustZoneProto.TrustDomain] = trustZoneStruct
			}

			err = installSPIREStack(trustZones)
			if err != nil {
				return err
			}

			// post-install additionally requires attestation policy config
			attestationPoliciesProtos, err := u.source.ListAttestationPolicies()
			if err != nil {
				return err
			}

			// convert to structs
			attestationPolicies := make([]*attestationpolicy.AttestationPolicy, 0, len(attestationPoliciesProtos))
			for _, attestationPolicyProto := range attestationPoliciesProtos {
				attestationPolicies = append(attestationPolicies, attestationpolicy.NewAttestationPolicy(attestationPolicyProto))
			}

			err = watchAndConfigure(trustZones, attestationPolicies)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func installSPIREStack(trustZones map[string]*trustzone.TrustZone) error {
	for _, trustZone := range trustZones {
		generator := helm.NewHelmValuesGenerator(trustZone)
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

func watchAndConfigure(trustZones map[string]*trustzone.TrustZone, attestationPolicies []*attestationpolicy.AttestationPolicy) error {
	// wait for SPIRE servers to be available and update status before applying federation(s)
	for _, trustZone := range trustZones {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Prefix = fmt.Sprintf("Waiting for pod and service in %s: ", trustZone.TrustZoneProto.KubernetesCluster)
		s.Start()

		clusterIP, err := watchSPIREPodAndService(trustZone.TrustZoneProto.KubernetesContext)
		if err != nil {
			s.Stop()
			return fmt.Errorf("error in context %s: %v", trustZone.TrustZoneProto.KubernetesContext, err)
		}

		trustZone.TrustZoneProto.BundleEndpointUrl = clusterIP

		s.Stop()
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s All pods and services are ready.\n\n", green("✅"))

	// now update the federations with the discovered endpoint URL
	for _, trustZone := range trustZones {
		for _, federation := range trustZone.Federations {
			federation.BundleEndpointURL = trustZones[federation.ToTrustDomain].TrustZoneProto.BundleEndpointUrl
		}
	}

	err := applyPostInstallHelmConfig(trustZones, attestationPolicies)
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
	var clusterIP string

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
					clusterIP = service.Spec.ClusterIP
				}
			}
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for pod and service to be ready")
		}

		if podReady && serviceReady {
			return clusterIP, nil
		}
	}
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

func applyPostInstallHelmConfig(trustZones map[string]*trustzone.TrustZone, attestationPolicies []*attestationpolicy.AttestationPolicy) error {
	for _, trustZone := range trustZones {
		generator := helm.NewHelmValuesGenerator(trustZone)

		if len(attestationPolicies) > 0 {
			generator = generator.WithAttestationPolicies(attestationPolicies)
		}

		if len(trustZone.Federations) > 0 {
			generator = generator.WithFederations(trustZone.Federations)
		}

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
