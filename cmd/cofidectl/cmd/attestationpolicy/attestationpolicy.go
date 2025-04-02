// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package attestationpolicy

import (
	"fmt"
	"os"
	"strings"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	types "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AttestationPolicyCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewAttestationPolicyCommand(cmdCtx *cmdcontext.CommandContext) *AttestationPolicyCommand {
	return &AttestationPolicyCommand{
		cmdCtx: cmdCtx,
	}
}

var attestationPolicyRootCmdDesc = `
This command consists of multiple sub-commands to administer Cofide attestation policies.
`

func (c *AttestationPolicyCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attestation-policy add|list [ARGS]",
		Short: "Add, list attestation policies",
		Long:  attestationPolicyRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(c.GetListCommand())
	cmd.AddCommand(c.GetAddCommand())

	return cmd
}

var attestationPolicyListCmdDesc = `
This command will list attestation policies in the Cofide configuration state.
`

func (c *AttestationPolicyCommand) GetListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "List attestation policies",
		Long:  attestationPolicyListCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			attestationPolicies, err := ds.ListAttestationPolicies()
			if err != nil {
				return err
			}

			return renderPolicies(attestationPolicies)
		},
	}

	return cmd
}

// renderPolicies writes a table showing information about a list of attestation policies.
func renderPolicies(policies []*attestation_policy_proto.AttestationPolicy) error {
	data := make([][]string, len(policies))
	for i, policy := range policies {
		switch p := policy.Policy.(type) {
		case *attestation_policy_proto.AttestationPolicy_Kubernetes:
			kubernetes := p.Kubernetes
			namespaceSelector := formatLabelSelector(kubernetes.NamespaceSelector)
			podSelector := formatLabelSelector(kubernetes.PodSelector)
			data[i] = []string{
				policy.Name,
				"kubernetes",
				namespaceSelector,
				podSelector,
				"",
				"",
			}
		case *attestation_policy_proto.AttestationPolicy_Static:
			static := p.Static

			spiffeID := static.GetSpiffeId()
			selectors, err := formatSelectors(static.GetSelectors())
			if err != nil {
				return err
			}

			data[i] = []string{
				policy.Name,
				"static",
				"",
				"",
				spiffeID,
				selectors,
			}
		default:
			return fmt.Errorf("unexpected attestation policy type %T", policy)
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Kind", "Namespace Labels", "Pod Labels", "SPIFFE ID", "Selectors"})
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()
	return nil
}

// formatLabelSelector formats a Kubernetes label selector as a string.
func formatLabelSelector(selector *attestation_policy_proto.APLabelSelector) string {
	k8sSelector := apLabelSelectorToK8sLS(selector)
	if k8sSelector == nil {
		return ""
	}
	return metav1.FormatLabelSelector(k8sSelector)
}

// apLabelSelectorToK8sLS converts an `APLabelSelector` to a Kubernetes `LabelSelector`.
func apLabelSelectorToK8sLS(selector *attestation_policy_proto.APLabelSelector) *metav1.LabelSelector {
	if selector == nil {
		return nil
	}

	k8sSelector := &metav1.LabelSelector{
		MatchLabels:      selector.MatchLabels,
		MatchExpressions: make([]metav1.LabelSelectorRequirement, 0, len(selector.MatchExpressions)),
	}
	for _, expression := range selector.MatchExpressions {
		expression := metav1.LabelSelectorRequirement{
			Key:      expression.Key,
			Operator: metav1.LabelSelectorOperator(expression.Operator),
			Values:   expression.Values,
		}
		k8sSelector.MatchExpressions = append(k8sSelector.MatchExpressions, expression)
	}
	return k8sSelector
}

// formatSelectors formats SPIRE selectors into a comma-separated string.
func formatSelectors(selectors []*types.Selector) (string, error) {
	if len(selectors) == 0 {
		return "", fmt.Errorf("no selectors provided")
	}

	selectorStrs := make([]string, len(selectors))
	for i, s := range selectors {
		if s.Type == "" || s.Value == "" {
			return "", fmt.Errorf("invalid selector type=%q, value=%q", s.Type, s.Value)
		}

		selectorStrs[i] = s.Type + ":" + s.Value
	}

	return strings.Join(selectorStrs, ","), nil
}

var attestationPolicyAddCmdDesc = `
This command consists of multiple sub-commands to add new attestation policies to the Cofide configuration state.
`

func (c *AttestationPolicyCommand) GetAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add kubernetes [ARGS]",
		Short: "Add attestation policies",
		Long:  attestationPolicyAddCmdDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		c.GetAddK8sCommand(),
		c.GetAddStaticCommand(),
	)
	return cmd
}

var attestationPolicyAddK8sCmdDesc = `
This command will add a new Kubernetes attestation policy to the Cofide configuration state.
`

type AddK8sOpts struct {
	name      string
	namespace string
	podLabel  string
}

func (c *AttestationPolicyCommand) GetAddK8sCommand() *cobra.Command {
	opts := AddK8sOpts{}
	cmd := &cobra.Command{
		Use:   "kubernetes [ARGS]",
		Short: "Add a new kubernetes attestation policy",
		Long:  attestationPolicyAddK8sCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			kubernetes := &attestation_policy_proto.APKubernetes{}
			if opts.namespace != "" {
				kubernetes.NamespaceSelector = &attestation_policy_proto.APLabelSelector{
					MatchLabels: map[string]string{"kubernetes.io/metadata.name": opts.namespace},
				}
			}
			if opts.podLabel != "" {
				selector, err := parseLabelSelector(opts.podLabel)
				if err != nil {
					return err
				}
				kubernetes.PodSelector = selector
			}
			newAttestationPolicy := &attestation_policy_proto.AttestationPolicy{
				Name: opts.name,
				Policy: &attestation_policy_proto.AttestationPolicy_Kubernetes{
					Kubernetes: kubernetes,
				},
			}
			_, err = ds.AddAttestationPolicy(newAttestationPolicy)
			if err != nil {
				return err
			}
			if opts.namespace == "" && opts.podLabel == "" {
				fmt.Println("This attestation policy will provide identity to all workloads in this trust domain")
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.name, "name", "", "Name to use for the attestation policy")
	f.StringVar(&opts.namespace, "namespace", "", "Namespace name selector")
	f.StringVar(&opts.podLabel, "pod-label", "", "Pod label selector in Kubernetes label selector format")

	cobra.CheckErr(cmd.MarkFlagRequired("name"))

	return cmd
}

var attestationPolicyAddStaticCmdDesc = `
This command will add a new static attestation policy to the Cofide configuration state.
`

type AddStaticOpts struct {
	name      string
	spiffeID  string
	selectors []string
}

func (c *AttestationPolicyCommand) GetAddStaticCommand() *cobra.Command {
	opts := AddStaticOpts{}
	cmd := &cobra.Command{
		Use:   "static [ARGS]",
		Short: "Add a new static attestation policy",
		Long:  attestationPolicyAddStaticCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource(cmd.Context())
			if err != nil {
				return err
			}

			selectors, err := parseSelectors(opts.selectors)
			if err != nil {
				return err
			}

			newAttestationPolicy := &attestation_policy_proto.AttestationPolicy{
				Name: opts.name,
				Policy: &attestation_policy_proto.AttestationPolicy_Static{
					Static: &attestation_policy_proto.APStatic{
						SpiffeId:  &opts.spiffeID,
						Selectors: selectors,
					},
				},
			}
			_, err = ds.AddAttestationPolicy(newAttestationPolicy)
			if err != nil {
				return err
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.name, "name", "", "Name to use for the attestation policy")
	f.StringVar(&opts.spiffeID, "spiffeid", "", "SPIFFE ID to use for the attestation policy")
	f.StringSliceVar(&opts.selectors, "selectors", []string{}, "Workload selectors to use for the attestation policy")

	cobra.CheckErr(cmd.MarkFlagRequired("name"))
	cobra.CheckErr(cmd.MarkFlagRequired("spiffeid"))
	cobra.CheckErr(cmd.MarkFlagRequired("selectors"))

	return cmd
}

// parseLabelSelector parses a Kubernetes label selector from a string.
// See https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors.
func parseLabelSelector(selector string) (*attestation_policy_proto.APLabelSelector, error) {
	k8sSelector, err := metav1.ParseToLabelSelector(selector)
	if err != nil {
		return nil, fmt.Errorf("--pod-label argument \"%s\" invalid: %w", selector, err)
	}
	return apLabelSelectorFromK8sLS(k8sSelector), nil
}

// apLabelSelectorFromK8sLS converts a Kubernetes `LabelSelector` to an `APLabelSelector`.
func apLabelSelectorFromK8sLS(k8sSelector *metav1.LabelSelector) *attestation_policy_proto.APLabelSelector {
	selector := &attestation_policy_proto.APLabelSelector{
		MatchLabels:      k8sSelector.MatchLabels,
		MatchExpressions: make([]*attestation_policy_proto.APMatchExpression, 0, len(k8sSelector.MatchExpressions)),
	}

	for _, expression := range k8sSelector.MatchExpressions {
		expression := &attestation_policy_proto.APMatchExpression{
			Key:      expression.Key,
			Operator: string(expression.Operator),
			Values:   expression.Values,
		}
		selector.MatchExpressions = append(selector.MatchExpressions, expression)
	}
	return selector
}

// parseSelectors parses a list of selectors from a string.
func parseSelectors(selectorStrings []string) ([]*types.Selector, error) {
	selectors := make([]*types.Selector, len(selectorStrings))

	for i, s := range selectorStrings {
		if strings.Count(s, ":") > 2 {
			return nil, fmt.Errorf("invalid selector format %q, too many ':' characters, expected 'type:key:value'", s)
		}

		selectorParts := strings.SplitN(s, ":", 3)
		if len(selectorParts) != 3 {
			return nil, fmt.Errorf("invalid selector format %q, expected 'type:key:value'", s)
		}

		selectorType, selectorKey, selectorVal := selectorParts[0], selectorParts[1], selectorParts[2]
		switch {
		case selectorType == "":
			return nil, fmt.Errorf("invalid selector format, type is empty: %q", s)
		case selectorKey == "":
			return nil, fmt.Errorf("invalid selector format, key is empty: %q", s)
		case selectorVal == "":
			return nil, fmt.Errorf("invalid selector format, value is empty: %q", s)
		}

		selectors[i] = &types.Selector{
			Type:  selectorType,
			Value: fmt.Sprintf("%s:%s", selectorKey, selectorVal),
		}
	}

	return selectors, nil
}
