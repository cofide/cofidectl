package apbinding

import (
	"os"
	"strings"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/proto/ap_binding/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	cmd_context "github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type APBindingCommand struct {
	cmdCtx *cmd_context.CommandContext
	source cofidectl_plugin.DataSource
}

func NewAPBindingCommand(cmdCtx *cmd_context.CommandContext) *APBindingCommand {
	return &APBindingCommand{
		cmdCtx: cmdCtx,
	}
}

var apBindingRootCmdDesc = `
This command consists of multiple sub-commands to administer Cofide attestation policy bindings.
`

func (c *APBindingCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attestation-policy-binding add|list [ARGS]",
		Short: "Add or list attestation policy bindings",
		Long:  apBindingRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		c.GetListCommand(),
		c.GetAddCommand(),
	)

	return cmd
}

var apBindingListCmdDesc = `
This command will list attestation policy bindings in the Cofide configuration state.
`

type ListOpts struct {
	trustZone         string
	attestationPolicy string
}

func (c *APBindingCommand) GetListCommand() *cobra.Command {
	opts := ListOpts{}
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "List attestation policy bindings",
		Long:  apBindingListCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetPlugin()
			if err != nil {
				return err
			}
			if err := ds.Validate(); err != nil {
				return err
			}
			bindings, err := c.list(opts)
			cobra.CheckErr(err)
			renderList(bindings)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "list the attestation policies bound to a specific trust zone")
	f.StringVar(&opts.attestationPolicy, "attestation-policy", "", "list the bindings for a specific attestation policy")

	return cmd
}

func (c *APBindingCommand) list(opts ListOpts) ([]*ap_binding_proto.APBinding, error) {
	var err error
	var trustZones []*trust_zone_proto.TrustZone

	if opts.trustZone != "" {
		trustZone, err := c.source.GetTrustZone(opts.trustZone)
		if err != nil {
			return nil, err
		}

		trustZones = append(trustZones, trustZone)
	} else {
		trustZones, err = c.source.ListTrustZones()
		if err != nil {
			return nil, err
		}
	}

	var bindings []*ap_binding_proto.APBinding
	for _, trustZone := range trustZones {
		for _, binding := range trustZone.AttestationPolicies {
			if opts.attestationPolicy == "" || binding.Policy == opts.attestationPolicy {
				bindings = append(bindings, binding)
			}
		}
	}

	return bindings, nil
}

func renderList(bindings []*ap_binding_proto.APBinding) {
	data := make([][]string, len(bindings))
	for i, binding := range bindings {
		data[i] = []string{
			binding.TrustZone,
			binding.Policy,
			strings.Join(binding.FederatesWith, ", "),
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Trust Zone", "Attestation Policy", "Federates With"})
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()
}

var apBindingAddCmdDesc = `
This command will bind an attestation policy to a trust zone.`

type AddOpts struct {
	trustZone         string
	attestationPolicy string
	federatesWith     []string
}

func (c *APBindingCommand) GetAddCommand() *cobra.Command {
	opts := AddOpts{}
	cmd := &cobra.Command{
		Use:   "add [ARGS]",
		Short: "Bind an attestation policy to a trust zone",
		Long:  apBindingAddCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.source.Validate(); err != nil {
				return err
			}

			binding := &ap_binding_proto.APBinding{
				TrustZone:     opts.trustZone,
				Policy:        opts.attestationPolicy,
				FederatesWith: opts.federatesWith,
			}
			return c.source.AddAPBinding(binding)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "Trust zone name")
	f.StringVar(&opts.attestationPolicy, "attestation-policy", "", "Attestation policy name")
	f.StringSliceVar(&opts.federatesWith, "federates-with", nil, "Defines a trust zone to federate identity with. May be specified multiple times")

	cobra.CheckErr(cmd.MarkFlagRequired("trust-zone"))
	cobra.CheckErr(cmd.MarkFlagRequired("attestation-policy"))

	return cmd
}
