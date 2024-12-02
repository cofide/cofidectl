// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustzone

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"

	cmdcontext "github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/trustzone/helm"
	"github.com/manifoldco/promptui"

	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_provider/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/spire"
	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	helmprovider "github.com/cofide/cofidectl/pkg/provider/helm"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

type TrustZoneCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewTrustZoneCommand(cmdCtx *cmdcontext.CommandContext) *TrustZoneCommand {
	return &TrustZoneCommand{
		cmdCtx: cmdCtx,
	}
}

var trustZoneRootCmdDesc = `
This command consists of multiple sub-commands to administer Cofide trust zones.
`

func (c *TrustZoneCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust-zone add|list|status [ARGS]",
		Short: "Add, list or interact with trust zones",
		Long:  trustZoneRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	helmCmd := helm.NewHelmCommand(c.cmdCtx)

	cmd.AddCommand(
		c.GetListCommand(),
		c.GetAddCommand(),
		c.GetStatusCommand(),
		helmCmd.GetRootCommand(),
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
			ds, err := c.cmdCtx.PluginManager.GetDataSource()
			if err != nil {
				return err
			}

			trustZones, err := ds.ListTrustZones()
			if err != nil {
				return err
			}

			data := make([][]string, len(trustZones))
			for i, trustZone := range trustZones {
				data[i] = []string{
					trustZone.Name,
					trustZone.TrustDomain,
					trustZone.GetKubernetesCluster(),
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

type addOpts struct {
	name              string
	trustDomain       string
	kubernetesCluster string
	context           string
	profile           string
	jwtIssuer         string
}

func (c *TrustZoneCommand) GetAddCommand() *cobra.Command {
	opts := addOpts{}
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
			ds, err := c.cmdCtx.PluginManager.GetDataSource()
			if err != nil {
				return err
			}

			err = c.getKubernetesContext(cmd, &opts)
			if err != nil {
				return err
			}

			newTrustZone := &trust_zone_proto.TrustZone{
				Name:              opts.name,
				TrustDomain:       opts.trustDomain,
				KubernetesCluster: &opts.kubernetesCluster,
				KubernetesContext: &opts.context,
				TrustProvider:     &trust_provider_proto.TrustProvider{Kind: &opts.profile},
				JwtIssuer:         &opts.jwtIssuer,
			}

			_, err = ds.AddTrustZone(newTrustZone)
			if err != nil {
				return fmt.Errorf("failed to create trust zone %s: %s", newTrustZone.Name, err)
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustDomain, "trust-domain", "", "Trust domain to use for this trust zone")
	f.StringVar(&opts.kubernetesCluster, "kubernetes-cluster", "", "Kubernetes cluster associated with this trust zone")
	f.StringVar(&opts.context, "kubernetes-context", "", "Kubernetes context to use for this trust zone")
	f.StringVar(&opts.profile, "profile", "kubernetes", "Cofide profile used in the installation (e.g. kubernetes, istio)")
	f.StringVar(&opts.jwtIssuer, "jwt-issuer", "", "JWT issuer to use for this trust zone")

	cobra.CheckErr(cmd.MarkFlagRequired("trust-domain"))
	cobra.CheckErr(cmd.MarkFlagRequired("kubernetes-cluster"))

	return cmd
}

var trustZoneStatusCmdDesc = `
This command will display the status of trust zones in the Cofide configuration state.

NOTE: This command relies on privileged access to execute SPIRE server CLI commands within the SPIRE server container, which may not be suitable for production environments.
`

func (c *TrustZoneCommand) GetStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [NAME]",
		Short: "Display trust zone status",
		Long:  trustZoneStatusCmdDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource()
			if err != nil {
				return err
			}

			kubeConfig, err := cmd.Flags().GetString("kube-config")
			if err != nil {
				return fmt.Errorf("failed to retrieve the kubeconfig file location")
			}
			return c.status(cmd.Context(), ds, kubeConfig, args[0])
		},
	}

	return cmd
}

func (c *TrustZoneCommand) status(ctx context.Context, source cofidectl_plugin.DataSource, kubeConfig, tzName string) error {
	trustZone, err := source.GetTrustZone(tzName)
	if err != nil {
		return err
	}

	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeConfig, trustZone.GetKubernetesContext())
	if err != nil {
		return err
	}

	prov, err := helmprovider.NewHelmSPIREProvider(ctx, trustZone, nil, nil)
	if err != nil {
		return err
	}
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

	return renderStatus(trustZone, server, agents)
}

func renderStatus(trustZone *trust_zone_proto.TrustZone, server *spire.ServerStatus, agents *spire.AgentStatus) error {
	trustZoneData := [][]string{
		{
			"Trust Zone",
			trustZone.Name,
		},
		{
			"SPIRE Servers ready",
			fmt.Sprintf("%d/%d", server.ReadyReplicas, server.Replicas),
		},
		{
			"SPIRE Agents ready",
			fmt.Sprintf("%d/%d", agents.Ready, agents.Expected),
		},
		{
			"Bundle Endpoint",
			trustZone.GetBundleEndpointUrl(),
		},
	}

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

	fmt.Printf("Trust Zone\n\n")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Item", "Value"})
	table.SetBorder(false)
	table.AppendBulk(trustZoneData)
	table.Render()

	fmt.Printf("\nSPIRE Servers\n\n")
	table = tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Pod", "Ready"})
	table.SetBorder(false)
	table.AppendBulk(serverData)
	table.Render()

	fmt.Printf("\nSPIRE Controller Managers\n\n")
	table = tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Pod", "Ready"})
	table.SetBorder(false)
	table.AppendBulk(scmData)
	table.Render()

	fmt.Printf("\nSPIRE Agents\n\n")
	table = tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Pod", "Status", "Attestation type", "Expiration time", "Can re-attest"})
	table.SetBorder(false)
	table.AppendBulk(agentData)
	table.Render()

	fmt.Printf("\nSPIRE Agents SPIFFE IDs\n\n")
	table = tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Pod", "SPIFFE ID"})
	table.SetBorder(false)
	table.AppendBulk(agentIdData)
	table.Render()

	return nil
}

func (c *TrustZoneCommand) getKubernetesContext(cmd *cobra.Command, opts *addOpts) error {
	kubeConfig, err := cmd.Flags().GetString("kube-config")
	if err != nil {
		return err
	}
	client, err := kubeutil.NewKubeClient(kubeConfig)
	if err != nil {
		return err
	}

	kubeRepo := kubeutil.NewKubeRepository(client)
	contexts, err := kubeRepo.GetContexts()
	if err != nil {
		return err
	}

	if opts.context != "" {
		if checkContext(contexts, opts.context) {
			return nil
		}
		return fmt.Errorf("could not find kubectl context '%s'", opts.context)
	}

	opts.context, err = promptContext(contexts, client.CmdConfig.CurrentContext)
	return err
}

func promptContext(contexts []string, currentContext string) (string, error) {
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
	if err != nil {
		return "", err
	}

	return result, nil
}

func checkContext(contexts []string, context string) bool {
	return slices.Contains(contexts, context)
}

func validateOpts(opts addOpts) error {
	_, err := spiffeid.TrustDomainFromString(opts.trustDomain)
	return err
}
