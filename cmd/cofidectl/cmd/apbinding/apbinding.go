// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package apbinding

import (
	"os"
	"strings"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1alpha1"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"

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
		Use:   "attestation-policy-binding add|del|list [ARGS]",
		Short: "Manage attestation policy bindings",
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

func (c *APBindingCommand) list(source datasource.DataSource, opts ListOpts) ([]*ap_binding_proto.APBinding, error) {
	filter := &datasourcepb.ListAPBindingsRequest_Filter{}
	if opts.trustZone != "" {
		filter.TrustZoneName = &opts.trustZone
	}
	if opts.attestationPolicy != "" {
		filter.PolicyName = &opts.attestationPolicy
	}
	return source.ListAPBindings(filter)
}

func renderList(bindings []*ap_binding_proto.APBinding) {
	data := make([][]string, len(bindings))
	for i, binding := range bindings {
		data[i] = []string{
			// nolint:staticcheck
			binding.TrustZone,
			// nolint:staticcheck
			binding.Policy,
			// nolint:staticcheck
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
