// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package apbinding

import (
	"os"
	"strings"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type APBindingCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewAPBindingCommand(cmdCtx *cmdcontext.CommandContext) *APBindingCommand {
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
		c.GetDelCommand(),
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
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			bindings, err := c.list(ds, opts)
			if err != nil {
				return err
			}
			renderList(bindings)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "list the attestation policies bound to a specific trust zone")
	f.StringVar(&opts.attestationPolicy, "attestation-policy", "", "list the bindings for a specific attestation policy")

	return cmd
}

func (c *APBindingCommand) list(source cofidectl_plugin.DataSource, opts ListOpts) ([]*ap_binding_proto.APBinding, error) {
	var err error
	var trustZones []*trust_zone_proto.TrustZone

	if opts.trustZone != "" {
		trustZone, err := source.GetTrustZone(opts.trustZone)
		if err != nil {
			return nil, err
		}

		trustZones = append(trustZones, trustZone)
	} else {
		trustZones, err = source.ListTrustZones()
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
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			binding := &ap_binding_proto.APBinding{
				TrustZone:     opts.trustZone,
				Policy:        opts.attestationPolicy,
				FederatesWith: opts.federatesWith,
			}
			_, err = ds.AddAPBinding(binding)
			return err
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

var apBindingDelCmdDesc = `
This command will unbind an attestation policy from a trust zone.`

type DelOpts struct {
	trustZone         string
	attestationPolicy string
}

func (c *APBindingCommand) GetDelCommand() *cobra.Command {
	opts := DelOpts{}
	cmd := &cobra.Command{
		Use:   "del [ARGS]",
		Short: "Unbind an attestation policy from a trust zone",
		Long:  apBindingDelCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			binding := &ap_binding_proto.APBinding{
				TrustZone: opts.trustZone,
				Policy:    opts.attestationPolicy,
			}
			return ds.DestroyAPBinding(binding)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "Trust zone name")
	f.StringVar(&opts.attestationPolicy, "attestation-policy", "", "Attestation policy name")

	cobra.CheckErr(cmd.MarkFlagRequired("trust-zone"))
	cobra.CheckErr(cmd.MarkFlagRequired("attestation-policy"))

	return cmd
}
