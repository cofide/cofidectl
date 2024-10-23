package trustzone

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"

	"github.com/manifoldco/promptui"

	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_provider/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	"github.com/cofide/cofidectl/internal/pkg/provider/helm"
	"github.com/cofide/cofidectl/internal/pkg/spire"
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
		Short: "Add, list or interact with trust zones",
		Long:  trustZoneRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		c.GetListCommand(),
		c.GetAddCommand(),
		c.GetStatusCommand(),
	)

	return cmd
}

var trustZoneListCmdDesc = `
This command will list trust zones in the Cofide configuration state.
`

func (c *TrustZoneCommand) GetListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "List trust zones",
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
			err := c.getKubernetesContext(cmd)
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
	f.StringVar(&opts.context, "context", "", "Kubernetes context to use for this trust zone")
	f.StringVar(&opts.profile, "profile", "kubernetes", "Cofide profile used in the installation (e.g. kubernetes, istio)")

	cmd.MarkFlagRequired("trust-domain")
	cmd.MarkFlagRequired("kubernetes-cluster")

	return cmd
}

var trustZoneStatusCmdDesc = `
This command will display the status of trust zones in the Cofide configuration state.

NOTE: This command relies on privileged access to execute SPIRE server CLI commands within the SPIRE server container, which may be considered a security risk in production environments.
`

func (c *TrustZoneCommand) GetStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [NAME]",
		Short: "Display trust zone status",
		Long:  trustZoneStatusCmdDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeConfig, err := cmd.Flags().GetString("kube-config")
			if err != nil {
				return fmt.Errorf("failed to retrieve the kubeconfig file location")
			}
			return c.status(cmd.Context(), kubeConfig, args[0])
		},
	}

	return cmd
}

func (c *TrustZoneCommand) status(ctx context.Context, kubeConfig, tzName string) error {
	trustZone, err := c.source.GetTrustZone(tzName)
	if err != nil {
		return err
	}

	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeConfig, trustZone.KubernetesContext)
	if err != nil {
		return err
	}

	prov := helm.NewHelmSPIREProvider(trustZone, nil, nil)
	if installed, err := prov.CheckIfAlreadyInstalled(); err != nil {
		return err
	} else if !installed {
		return errors.New("Cofide configuration has not been installed. Have you run cofidectl up?")
	}

	server, err := spire.GetServerStatus(ctx, client)
	if err != nil {
		return err
	}

	agents, err := spire.GetAgentStatus(ctx, client)
	if err != nil {
		return err
	}

	return renderStatus(server, agents)
}

func renderStatus(server *spire.ServerStatus, agents *spire.AgentStatus) error {
	serverData := make([][]string, 0)
	for _, container := range server.Containers {
		serverData = append(serverData, []string{
			container.Name,
			strconv.FormatBool(container.Ready),
		})
	}

	scmData := make([][]string, 0)
	for _, scm := range server.SCMs {
		scmData = append(scmData, []string{
			scm.Name,
			strconv.FormatBool(scm.Ready),
		})
	}

	agentData := make([][]string, 0)
	for _, agent := range agents.Agents {
		agentData = append(agentData, []string{
			agent.Name,
			agent.Status,
			agent.AttestationType,
			agent.ExpirationTime.String(),
			strconv.FormatBool(agent.CanReattest),
		})
	}

	agentIdData := make([][]string, 0)
	for _, agent := range agents.Agents {
		agentIdData = append(agentIdData, []string{
			agent.Name,
			agent.Id,
		})
	}

	fmt.Printf("SPIRE Servers (%d/%d ready)\n", server.ReadyReplicas, server.Replicas)
	fmt.Println()
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Pod", "Ready"})
	table.SetBorder(false)
	table.AppendBulk(serverData)
	table.Render()

	fmt.Println()
	fmt.Printf("SPIRE Controller Managers (%d/%d ready)\n", server.ReadyReplicas, server.Replicas)
	fmt.Println()
	table = tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Pod", "Ready"})
	table.SetBorder(false)
	table.AppendBulk(scmData)
	table.Render()

	fmt.Println()
	fmt.Printf("SPIRE Agents (%d/%d ready)\n", agents.Ready, agents.Expected)
	fmt.Println()
	table = tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Pod", "Status", "Attestation type", "Expiration time", "Can re-attest"})
	table.SetBorder(false)
	table.AppendBulk(agentData)
	table.Render()

	fmt.Println()
	fmt.Println("SPIRE Agents SPIFFE IDs")
	fmt.Println()
	table = tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Pod", "SPIFFE ID"})
	table.SetBorder(false)
	table.AppendBulk(agentIdData)
	table.Render()

	return nil
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
	cobra.CheckErr(err)

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

func validateOpts(opts Opts) error {
	_, err := spiffeid.TrustDomainFromString(opts.trust_domain)
	return err
}
