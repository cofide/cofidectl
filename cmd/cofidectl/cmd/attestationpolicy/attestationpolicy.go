package attestationpolicy

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/gobeam/stringy"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
)

type AttestationPolicyCommand struct {
	source cofidectl_plugin.DataSource
}

func NewAttestationPolicyCommand(source cofidectl_plugin.DataSource) *AttestationPolicyCommand {
	return &AttestationPolicyCommand{
		source: source,
	}
}

var attestationPolicyRootCmdDesc = `
This command consists of multiple sub-commands to administer Cofide attestation policy.
`

func (c *AttestationPolicyCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attestation-policy add|list [ARGS]",
		Short: "add, list attestation policy",
		Long:  attestationPolicyRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(c.GetListCommand())
	cmd.AddCommand(c.GetAddCommand())

	return cmd
}

var attestationPolicyListCmdDesc = `
This command will list trust zones in the Cofide configuration state.
`

func (c *AttestationPolicyCommand) GetListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [ARGS]",
		Short: "List attestation-policy",
		Long:  attestationPolicyListCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			attestationPolicies, err := c.source.ListAttestationPolicy()
			if err != nil {
				return err
			}

			data := make([][]string, len(attestationPolicies))
			for i, policy := range attestationPolicies {
				data[i] = []string{
					policy.Kind.String(),
					policy.Options.Namespace,
					policy.Options.PodKey,
					policy.Options.PodValue,
				}
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Kind", "Namespace", "Pod Key", "Pod Value"})
			table.SetBorder(false)
			table.AppendBulk(data)
			table.Render()
			return nil
		},
	}

	return cmd
}

var attestationPolicyAddCmdDesc = `
This command will add a new attestation policy to the Cofide configuration state.
`

type Opts struct {
	kind                  string
	trustZoneName         string
	attestationPolicyOpts AttestationPolicyOpts
}

type AttestationPolicyOpts struct {
	FederatesWith string `yaml:"federatesWith,omitempty"`

	// annotated
	PodKey   string `yaml:"podKey,omitempty"`
	PodValue string `yaml:"podValue,omitempty"`

	// namespace
	Namespace string `yaml:"namespace,omitempty"`
}

func (c *AttestationPolicyCommand) GetAddCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "add [KIND]",
		Short: "Add a new attestation policy",
		Long:  attestationPolicyAddCmdDesc,
		Args:  require.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			str := stringy.New(args[0])
			opts.kind = str.KebabCase().ToLower()
			opts.trustZoneName = stringy.New(opts.trustZoneName).ToLower()

			if !validateOpts(opts) {
				return errors.New("unset flags for annotation policy")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			kind, err := GetAttestationPolicyKind(opts.kind)
			if err != nil {
				return err
			}
			newAttestationPolicy := &attestation_policy_proto.AttestationPolicy{
				Kind: kind,
				Options: &attestation_policy_proto.AttestionPolicyOptions{
					Namespace: opts.attestationPolicyOpts.Namespace,
					PodKey:    opts.attestationPolicyOpts.PodKey,
					PodValue:  opts.attestationPolicyOpts.PodValue,
				},
			}
			return c.source.AddAttestationPolicy(newAttestationPolicy)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustZoneName, "trust-zone", "", "Name of the trust zone to attach this attestation policy to")
	f.StringVar(&opts.attestationPolicyOpts.Namespace, "namespace", "", "Namespace to use in Namespace attestation policy")
	f.StringVar(&opts.attestationPolicyOpts.PodKey, "annotation-key", "", "Key of Pod annotation to use in Annotation attestation policy")
	f.StringVar(&opts.attestationPolicyOpts.PodValue, "annotation-value", "", "Value of Pod annotation to use in Annotation attestation policy")
	f.StringVar(&opts.attestationPolicyOpts.FederatesWith, "federates-with", "", "Defines a trust domain to federate identity with")

	cmd.MarkFlagRequired("trust-zone")

	return cmd
}

func validateOpts(opts Opts) bool {
	if opts.kind == "namespace" && opts.attestationPolicyOpts.Namespace == "" {
		slog.Error("flag \"namespace\" must be provided for Namespace attestation policy kind")
		return false
	}

	if opts.kind == "annotation" && (opts.attestationPolicyOpts.PodKey == "" || opts.attestationPolicyOpts.PodValue == "") {
		slog.Error("flags \"annotation-key\" and \"annotation-value\" must be provided for Annotation attestation policy kind")
		return false
	}

	return true
}

func GetAttestationPolicyKind(s string) (attestation_policy_proto.AttestionPolicyKind, error) {
	switch s {
	case "annotated":
		return attestation_policy_proto.AttestionPolicyKind_ATTESTION_POLICY_KIND_ANNOTATED, nil
	case "namespace":
		return attestation_policy_proto.AttestionPolicyKind_ATTESTION_POLICY_KIND_NAMESPACE, nil
	}

	// TODO: Update error message.
	return attestation_policy_proto.AttestionPolicyKind_ATTESTION_POLICY_KIND_UNSPECIFIED, fmt.Errorf(fmt.Sprintf("unknown attestation policy kind %s", s))
}
