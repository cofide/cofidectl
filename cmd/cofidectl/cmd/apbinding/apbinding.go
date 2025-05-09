// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package apbinding

import (
	"errors"
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
	trustZone           string
	trustZoneID         string
	attestationPolicy   string
	attestationPolicyID string
	federatesWith       []string
	federatesWithIDs    []string
}

func (c *APBindingCommand) GetAddCommand() *cobra.Command {
	opts := AddOpts{}
	cmd := &cobra.Command{
		Use:   "add [ARGS]",
		Short: "Bind an attestation policy to a trust zone",
		Long:  apBindingAddCmdDesc,
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.trustZone == "" && opts.trustZoneID == "" {
				return errors.New("either --trust-zone or --trust-zone-id must be specified")
			}
			if opts.attestationPolicy == "" && opts.attestationPolicyID == "" {
				return errors.New("either --attestation-policy or --attestation-policy-id must be specified")
			}
			if opts.trustZone != "" && opts.trustZoneID != "" {
				return errors.New("only one of --trust-zone or --trust-zone-id can be specified")
			}
			if opts.attestationPolicy != "" && opts.attestationPolicyID != "" {
				return errors.New("only one of --attestation-policy or --attestation-policy-id can be specified")
			}

			if len(opts.federatesWith) != 0 && len(opts.federatesWithIDs) != 0 {
				return errors.New("only one of --federates-with or --federates-with-id can be specified")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			trustZoneID := opts.trustZoneID
			if trustZoneID == "" {
				tzs, err := ds.ListTrustZones()
				if err != nil {
					return err
				}
				for _, tz := range tzs {
					if tz.Name == opts.trustZone {
						trustZoneID = tz.GetId()
						break
					}
				}
			}
			if trustZoneID == "" {
				return errors.New("trust zone not found")
			}

			policyID := opts.attestationPolicyID
			if policyID == "" {
				policies, err := ds.ListAttestationPolicies()
				if err != nil {
					return err
				}
				for _, policy := range policies {
					if policy.Name == opts.attestationPolicy {
						policyID = policy.GetId()
						break
					}
				}
			}
			if policyID == "" {
				return errors.New("attestation policy not found")
			}

			federatesWith := opts.federatesWithIDs
			if len(opts.federatesWith) > 0 {
				federatesWith = []string{}
				tzs, err := ds.ListTrustZones()
				if err != nil {
					return err
				}
				for _, tz := range tzs {
					for _, federate := range opts.federatesWith {
						if tz.Name == federate {
							federatesWith = append(federatesWith, tz.GetId())
							break
						}
					}
				}
			}
			federations := []*ap_binding_proto.APBindingFederation{}
			for _, federate := range federatesWith {
				federations = append(federations, &ap_binding_proto.APBindingFederation{
					TrustZoneId: &federate,
				})
			}

			binding := &ap_binding_proto.APBinding{
				TrustZoneId: &trustZoneID,
				PolicyId:    &policyID,
				Federations: federations,
			}
			_, err = ds.AddAPBinding(binding)
			return err
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "Trust zone name")
	f.StringVar(&opts.trustZone, "trust-zone-id", "", "Trust zone ID")
	f.StringVar(&opts.attestationPolicy, "attestation-policy", "", "Attestation policy name")
	f.StringVar(&opts.attestationPolicy, "attestation-policy-id", "", "Attestation policy ID")
	f.StringSliceVar(&opts.federatesWith, "federates-with", nil, "Defines a trust zone to federate identity with. May be specified multiple times")
	f.StringSliceVar(&opts.federatesWithIDs, "federates-with-id", nil, "Defines a trust zone to federate identity with. May be specified multiple times")
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
