package cmd

import (
	"context"
	"fmt"
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
This command deploys a Cofide configuration
`

func (u *UpCommand) UpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [ARGS]",
		Short: "Deploy a Cofide configuration",
		Long:  upCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			trustZones, err := u.source.ListTrustZones()
			if err != nil {
				return err
			}

			if len(trustZones) == 0 {
				fmt.Println("no trust zones have been configured")
				return nil
			}

			err = installSPIREStack(trustZones)
			if err != nil {
				return err
			}

			err = watchAndConfigure(trustZones)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func installSPIREStack(trustZones []*trust_zone_proto.TrustZone) error {
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

func watchAndConfigure(trustZones []*trust_zone_proto.TrustZone) error {
	// wait for SPIRE servers to be available before applying CRs
	for _, trustZone := range trustZones {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Prefix = fmt.Sprintf("Waiting for pod in %s: ", trustZone.KubernetesCluster)
		s.Start()

		err := watchSPIREPod(trustZone.KubernetesContext)
		if err != nil {
			s.Stop()
			return fmt.Errorf("error in context %s: %v", trustZone.KubernetesContext, err)
		}

		s.Stop()
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s All pods are ready.\n", green("✅"))

	err := applyPostInstallHelmConfig(trustZones)
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

func applyPostInstallHelmConfig(trustZones []*trust_zone_proto.TrustZone) error {
	for _, trustZone := range trustZones {
		spireValues := map[string]interface{}{}
		spireCRDsValues := map[string]interface{}{}

		prov := helm.NewHelmSPIREProvider(trustZone, spireValues, spireCRDsValues)

		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Prefix = "Configuring CRs"
		s.Start()

		statusCh, err := prov.ExecuteUpgrade()
		if err != nil {
			s.Stop()
			return fmt.Errorf("failed to start upgrade: %w", err)
		}

		for status := range statusCh {
			s.Suffix = fmt.Sprintf(" %s: %s\n", status.Stage, status.Message)

			if status.Done {
				s.Stop()
				if status.Error != nil {
					fmt.Printf("❌ %s: %s\n", status.Stage, status.Message)
					return fmt.Errorf("upgrade failed: %w", status.Error)
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
