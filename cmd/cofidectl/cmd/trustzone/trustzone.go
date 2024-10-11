package trustzone

import (
	"fmt"
	"os"
	"slices"

	"github.com/manifoldco/promptui"

	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_provider/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"

	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/gobeam/stringy"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type TrustZoneCommand struct {
	source cofidectl_plugin.DataSource
}

func NewTrustZoneCommand(source cofidectl_plugin.DataSource) *TrustZoneCommand {
	return &TrustZoneCommand{
		source: source,
	}
}

var trustZoneDesc = `
This command consists of multiple sub-commands to administer Cofide trust zones.
`

func (c *TrustZoneCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust-zone add|list [ARGS]",
		Short: "add, list trust zones",
		Long:  trustZoneDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(c.GetListCommand())
	cmd.AddCommand(c.GetAddCommand())

	return cmd
}

var trustZoneListDesc = `
This command will list trust zones in the Cofide configuration state.
`

func (c *TrustZoneCommand) GetListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "List trust-zones",
		Long:  trustZoneListDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			trustZones, err := c.source.ListTrustZones()
			if err != nil {
				return err
			}

			data := make([][]string, len(trustZones))
			for i, trustZone := range trustZones {
				data[i] = []string{
					trustZone.Name,
					trustZone.TrustDomain,
					trustZone.KubernetesCluster,
				}
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Name", "Trust Domain", "Cluster"})
			table.SetBorder(false)
			table.AppendBulk(data)
			table.Render()
			return nil
		},
	}

	return cmd
}

var trustZoneAddDesc = `
This command will add a new trust zone to the Cofide configuration state.
`

type Opts struct {
	name               string
	trust_domain       string
	kubernetes_cluster string
	context            string
	profile            string
}

func (c *TrustZoneCommand) GetAddCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "add [NAME]",
		Short: "Add a new trust zone",
		Long:  trustZoneAddDesc,
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			str := stringy.New(args[0])
			opts.name = str.KebabCase().ToLower()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := c.getKubernetesContext(cmd)
			if err != nil {
				return err
			}

			newTrustZone := &trust_zone_proto.TrustZone{
				Name:              opts.name,
				TrustDomain:       opts.trust_domain,
				KubernetesCluster: opts.kubernetes_cluster,
				KubernetesContext: opts.context,
				TrustProvider:     GetTrustProvider(opts.profile),
			}
			return c.source.AddTrustZone(newTrustZone)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trust_domain, "trust-domain", "", "Trust domain to use for this trust zone")
	f.StringVar(&opts.kubernetes_cluster, "k8s-cluster", "", "Kubernetes cluster associated with this trust zone")
	f.StringVar(&opts.context, "context", "", "Kubernetes context to use for this trust zone")
	f.StringVar(&opts.profile, "profile", "kubernetes", "Cofide profile used in the installation (e.g. k8s, istio)")

	cmd.MarkFlagRequired("trust-domain")
	cmd.MarkFlagRequired("k8s-cluster")
	cmd.MarkFlagRequired("profile")

	return cmd
}

func (c *TrustZoneCommand) getKubernetesContext(cmd *cobra.Command) error {
	kubeConfig, err := cmd.Flags().GetString("kube-config")
	if err != nil {
		return err
	}
	client, err := kubeutil.NewKubeClient(kubeConfig)
	cobra.CheckErr(err)

	kubeRepo := kubeutil.NewKubeRepository(client)
	contexts, err := kubeRepo.GetContexts()
	kubeContext, _ := cmd.Flags().GetString("context")

	if kubeContext != "" {
		if checkContext(contexts, kubeContext) {
			return nil
		}
		fmt.Printf("could not find kubectl context '%s'", kubeContext)
	}
	kubeContext = promptContext(contexts, client.CmdConfig.CurrentContext)
	cmd.Flags().Set("context", kubeContext)
	return nil
}

func promptContext(contexts []string, currentContext string) string {
	curPos := 0
	if currentContext != "" {
		curPos = slices.Index(contexts, currentContext)
	}

	prompt := promptui.Select{
		Label:     "Select kubectl context to use",
		Items:     contexts,
		CursorPos: curPos,
	}

	_, result, err := prompt.Run()
	cobra.CheckErr(err)

	return result
}

func checkContext(contexts []string, context string) bool {
	return slices.Contains(contexts, context)
}

// TODO: Rethink the location of this method.
func GetTrustProvider(profile string) *trust_provider_proto.TrustProvider {
	switch profile {
	case "kubernetes":
		{
			tp := trust_provider_proto.TrustProvider{
				Name: "kubernetes",
				Kind: "k8s",
				AgentConfig: &trust_provider_proto.TrustProviderAgentConfig{
					WorkloadAttestor:        "k8s",
					WorkloadAttestorEnabled: true,
					WorkloadAttestorConfig: &trust_provider_proto.WorkloadAttestorConfig{
						Enabled:                     true,
						SkipKubeletVerification:     true,
						DisableContainerSelectors:   false,
						UseNewContainerLocator:      false,
						VerboseContainerLocatorLogs: false,
					},
					NodeAttestor:        "k8sPsat",
					NodeAttestorEnabled: true,
				},
				ServerConfig: &trust_provider_proto.TrustProviderServerConfig{
					NodeAttestor:        "k8sPsat",
					NodeAttestorEnabled: true,
					NodeAttestorConfig: &trust_provider_proto.NodeAttestorConfig{
						Enabled:                 true,
						ServiceAccountAllowList: []string{"spire:spire-agent"},
						Audience:                []string{"spire-server"},
						AllowedNodeLabelKeys:    []string{},
						AllowedPodLabelKeys:     []string{},
					},
				},
			}
			return &tp
		}
	default:
		return nil
	}
}
