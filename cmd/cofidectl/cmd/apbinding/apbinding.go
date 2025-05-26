// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package apbinding

import (
	"errors"
	"os"
	"strings"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"

	"slices"

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
	trustZoneID         string
	attestationPolicyID string
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
	f.StringVar(&opts.trustZoneID, "trust-zone-id", "", "list the attestation policies bound to a specific trust zone")
	f.StringVar(&opts.attestationPolicyID, "attestation-policy-id", "", "list the bindings for a specific attestation policy")

	return cmd
}

func (c *APBindingCommand) list(source datasource.DataSource, opts ListOpts) ([]*ap_binding_proto.APBinding, error) {
	filter := &datasourcepb.ListAPBindingsRequest_Filter{}
	if opts.trustZoneID != "" {
		filter.TrustZoneId = &opts.trustZoneID
	}
	if opts.attestationPolicyID != "" {
		filter.PolicyId = &opts.attestationPolicyID
	}
	return source.ListAPBindings(filter)
}

func renderFederations(bindings []*ap_binding_proto.APBindingFederation) string {
	federations := []string{}
	for _, binding := range bindings {
		federations = append(federations, binding.GetTrustZoneId())
	}

	return strings.Join(federations, ", ")
}

func renderList(bindings []*ap_binding_proto.APBinding) {
	data := make([][]string, len(bindings))
	for i, binding := range bindings {
		data[i] = []string{
			binding.GetTrustZoneId(),
			binding.GetPolicyId(),
			renderFederations(binding.GetFederations()),
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
	trustZonename       string
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
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			trustZoneID := opts.trustZoneID
			if trustZoneID == "" {
				tz, err := ds.GetTrustZoneByName(opts.trustZonename)
				if err != nil {
					return err
				}
				if tz == nil {
					return errors.New("trust zone not found")
				}
				trustZoneID = tz.GetId()

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
					if slices.Contains(opts.federatesWith, tz.Name) {
						federatesWith = append(federatesWith, tz.GetId())
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
	f.StringVar(&opts.trustZonename, "trust-zone-name", "", "Trust zone name")
	f.StringVar(&opts.trustZoneID, "trust-zone-id", "", "Trust zone ID")
	f.StringVar(&opts.attestationPolicy, "attestation-policy", "", "Attestation policy name")
	f.StringVar(&opts.attestationPolicy, "attestation-policy-id", "", "Attestation policy ID")
	f.StringSliceVar(&opts.federatesWith, "federates-with", nil, "Defines a trust zone to federate identity with. May be specified multiple times")
	f.StringSliceVar(&opts.federatesWithIDs, "federates-with-id", nil, "Defines a trust zone to federate identity with. May be specified multiple times")

	cmd.MarkFlagsOneRequired("trust-zone-name", "trust-zone-id")
	cmd.MarkFlagsOneRequired("attestation-policy", "attestation-policy-id")
	cmd.MarkFlagsMutuallyExclusive("trust-zone-name", "trust-zone-id")
	cmd.MarkFlagsMutuallyExclusive("attestation-policy", "attestation-policy-id")
	cmd.MarkFlagsMutuallyExclusive("federates-with", "federates-with-id")
	return cmd
}

var apBindingDelCmdDesc = `
This command will unbind an attestation policy from a trust zone.`

type DelOpts struct {
	id                string
	trustZoneName     string
	trustZoneID       string
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

			if opts.id != "" {
				return ds.DestroyAPBinding(opts.id)
			}

			var trustZone *trust_zone_proto.TrustZone
			if opts.trustZoneName != "" {
				trustZone, err = ds.GetTrustZoneByName(opts.trustZoneName)
				if err != nil {
					return err
				}
			}
			if opts.trustZoneID != "" {
				trustZone, err = ds.GetTrustZone(opts.trustZoneID)
				if err != nil {
					return err
				}
			}
			policy, err := ds.GetAttestationPolicyByName(opts.attestationPolicy)
			if err != nil {
				return err
			}

			bindings, err := ds.ListAPBindings(&datasourcepb.ListAPBindingsRequest_Filter{
				TrustZoneId: trustZone.Id,
				PolicyId:    policy.Id,
			})
			if err != nil {
				return err
			}
			if len(bindings) == 0 {
				return errors.New("no binding found")
			}
			if len(bindings) > 1 {
				return errors.New("multiple bindings found")
			}
			binding := bindings[0]
			return ds.DestroyAPBinding(binding.GetId())
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustZoneName, "trust-zone-name", "", "Trust zone name")
	f.StringVar(&opts.trustZoneID, "trust-zone-id", "", "Trust zone ID")
	f.StringVar(&opts.attestationPolicy, "attestation-policy", "", "Attestation policy name")
	f.StringVar(&opts.id, "id", "", "Binding ID")

	cmd.MarkFlagsOneRequired("trust-zone-id", "trust-zone-name", "id")
	cmd.MarkFlagsOneRequired("attestation-policy", "id")
	cmd.MarkFlagsMutuallyExclusive("trust-zone-name", "id")
	cmd.MarkFlagsMutuallyExclusive("trust-zone-id", "id")
	cmd.MarkFlagsMutuallyExclusive("attestation-policy", "id")

	return cmd
}
