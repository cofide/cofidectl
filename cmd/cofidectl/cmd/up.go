package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"

	"github.com/cofide/cofidectl/internal/pkg/attestationpolicy"
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
			trustZoneProtos, err := u.source.ListTrustZones()
			if err != nil {
				return err
			}

			if len(trustZoneProtos) == 0 {
				fmt.Println("no trust zones have been configured")
				return nil
			}

			// convert to structs
			trustZones := make([]*trustzone.TrustZone, 0, len(trustZoneProtos))
			for _, trustZoneProto := range trustZoneProtos {
				trustZones = append(trustZones, trustzone.NewTrustZone(trustZoneProto))

			}

			err = installSPIREStack(trustZones)
			if err != nil {
				return err
			}

			// post-install additionally requires federations and attestation policies config
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

func installSPIREStack(trustZones []*trustzone.TrustZone) error {
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

func watchAndConfigure(trustZones []*trustzone.TrustZone, attestationPolicies []*attestationpolicy.AttestationPolicy) error {
	// wait for SPIRE servers to be available before applying CRs
	for _, trustZone := range trustZones {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Prefix = fmt.Sprintf("Waiting for pod in %s: ", trustZone.TrustZoneProto.KubernetesCluster)
		s.Start()

		err := watchSPIREPod(trustZone.TrustZoneProto.KubernetesContext)
		if err != nil {
			s.Stop()
			return fmt.Errorf("error in context %s: %v", trustZone.TrustZoneProto.KubernetesContext, err)
		}

		s.Stop()
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s All pods are ready.\n\n", green("✅"))

	err := applyPostInstallHelmConfig(trustZones, attestationPolicies)
	if err != nil {
		return err
	}

	return nil
}

func watchSPIREPod(kubeContext string) error {
	watcher, err := createPodWatcher(kubeContext)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		if pod, ok := event.Object.(*v1.Pod); ok {
			if isPodReady(pod) {
				return nil
			}
		}
	}

	return fmt.Errorf("watcher closed unexpectedly for cluster: %s", kubeContext)
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

func applyPostInstallHelmConfig(trustZones []*trustzone.TrustZone, attestationPolicies []*attestationpolicy.AttestationPolicy) error {
	for _, trustZone := range trustZones {
		generator := helm.NewHelmValuesGenerator(trustZone)

		if len(attestationPolicies) > 0 {
			generator = generator.WithAttestationPolicies(attestationPolicies)
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
