// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package apbinding

import (
	"errors"
	"fmt"
	"os"
	"strings"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
	trustzonepb "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
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
			return renderList(ds, bindings)
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
		trustZone, err := source.GetTrustZoneByName(opts.trustZone)
		if err != nil {
			return nil, err
		}
		filter.TrustZoneId = trustZone.Id
	}
	if opts.attestationPolicy != "" {
		policy, err := source.GetAttestationPolicyByName(opts.attestationPolicy)
		if err != nil {
			return nil, err
		}
		filter.PolicyId = policy.Id
	}
	return source.ListAPBindings(filter)
}

func renderFederations(bindings []*ap_binding_proto.APBindingFederation, tzMap map[string]string) string {
	federations := []string{}
	for _, binding := range bindings {
		federations = append(federations, tzMap[binding.GetTrustZoneId()])
	}

	return strings.Join(federations, ", ")
}

func renderList(source datasource.DataSource, bindings []*ap_binding_proto.APBinding) error {
	tzs, err := source.ListTrustZones()
	if err != nil {
		return err
	}
	tzMap := make(map[string]string)
	for _, tz := range tzs {
		tzMap[tz.GetId()] = tz.GetName()
	}

	policies, err := source.ListAttestationPolicies()
	if err != nil {
		return err
	}
	policyMap := make(map[string]string)
	for _, policy := range policies {
		policyMap[policy.GetId()] = policy.GetName()
	}

	data := make([][]string, len(bindings))
	for i, binding := range bindings {
		data[i] = []string{
			tzMap[binding.GetTrustZoneId()],
			policyMap[binding.GetPolicyId()],
			renderFederations(binding.GetFederations(), tzMap),
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Trust Zone", "Attestation Policy", "Federates With"})
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()
	return nil
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

			tz, err := ds.GetTrustZoneByName(opts.trustZone)
			if err != nil {
				return err
			}
			if tz == nil {
				return errors.New("trust zone not found")
			}
			trustZoneID := tz.GetId()

			policies, err := ds.ListAttestationPolicies()
			if err != nil {
				return err
			}
			var policyID string
			for _, policy := range policies {
				if policy.Name == opts.attestationPolicy {
					policyID = policy.GetId()
					break
				}
			}
			if policyID == "" {
				return errors.New("attestation policy not found")
			}

			federations := []*ap_binding_proto.APBindingFederation{}
			if len(opts.federatesWith) != 0 {
				tzs, err := ds.ListTrustZones()
				if err != nil {
					return err
				}

				for _, federation := range opts.federatesWith {
					var trustZone *trustzonepb.TrustZone
					for _, tz := range tzs {
						if tz.Name == federation {
							trustZone = tz
							break
						}
					}
					if trustZone == nil {
						return fmt.Errorf("federated trust zone not found: %s", federation)
					}
					federations = append(federations, &ap_binding_proto.APBindingFederation{
						TrustZoneId: trustZone.Id,
					})
				}
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

			trustZone, err := ds.GetTrustZoneByName(opts.trustZone)
			if err != nil {
				return err
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
	f.StringVar(&opts.trustZone, "trust-zone", "", "Trust zone name")
	f.StringVar(&opts.attestationPolicy, "attestation-policy", "", "Attestation policy name")

	cobra.CheckErr(cmd.MarkFlagRequired("trust-zone"))
	cobra.CheckErr(cmd.MarkFlagRequired("attestation-policy"))

	return cmd
}
