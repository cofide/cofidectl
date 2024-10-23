package trustzone

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/manifoldco/promptui"

	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_provider/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"

	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

type TrustZoneCommand struct {
	source cofidectl_plugin.DataSource
}

func NewTrustZoneCommand(source cofidectl_plugin.DataSource) *TrustZoneCommand {
	return &TrustZoneCommand{
		source: source,
	}
}

var trustZoneRootCmdDesc = `
This command consists of multiple sub-commands to administer Cofide trust zones.
`

func (c *TrustZoneCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust-zone add|list [ARGS]",
		Short: "add, list trust zones",
		Long:  trustZoneRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(c.GetListCommand())
	cmd.AddCommand(c.GetAddCommand())

	return cmd
}

var trustZoneListCmdDesc = `
This command will list trust zones in the Cofide configuration state.
`

func (c *TrustZoneCommand) GetListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "List trust-zones",
		Long:  trustZoneListCmdDesc,
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

var trustZoneAddCmdDesc = `
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
		Long:  trustZoneAddCmdDesc,
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			if err := validateOpts(opts); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := c.getKubernetesContext(cmd, &opts)
			if err != nil {
				return err
			}

			newTrustZone := &trust_zone_proto.TrustZone{
				Name:              opts.name,
				TrustDomain:       opts.trust_domain,
				KubernetesCluster: opts.kubernetes_cluster,
				KubernetesContext: opts.context,
				TrustProvider:     &trust_provider_proto.TrustProvider{Kind: opts.profile},
			}
			return c.source.AddTrustZone(newTrustZone)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trust_domain, "trust-domain", "", "Trust domain to use for this trust zone")
	f.StringVar(&opts.kubernetes_cluster, "kubernetes-cluster", "", "Kubernetes cluster associated with this trust zone")
	f.StringVar(&opts.context, "kubernetes-context", "", "Kubernetes context to use for this trust zone")
	f.StringVar(&opts.profile, "profile", "kubernetes", "Cofide profile used in the installation (e.g. kubernetes, istio)")

	cmd.MarkFlagRequired("trust-domain")
	cmd.MarkFlagRequired("kubernetes-cluster")

	return cmd
}

func (c *TrustZoneCommand) getKubernetesContext(cmd *cobra.Command, opts *Opts) error {
	kubeConfig, err := cmd.Flags().GetString("kube-config")
	if err != nil {
		return err
	}
	client, err := kubeutil.NewKubeClient(kubeConfig)
	cobra.CheckErr(err)

	kubeRepo := kubeutil.NewKubeRepository(client)
	contexts, err := kubeRepo.GetContexts()
	cobra.CheckErr(err)

	if opts.context != "" {
		if checkContext(contexts, opts.context) {
			return nil
		}
		return errors.New(fmt.Sprintf("could not find kubectl context '%s'", opts.context))
	}

	opts.context = promptContext(contexts, client.CmdConfig.CurrentContext)
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

func validateOpts(opts Opts) error {
	_, err := spiffeid.TrustDomainFromString(opts.trust_domain)
	return err
}
