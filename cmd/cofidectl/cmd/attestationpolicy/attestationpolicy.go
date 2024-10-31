package attestationpolicy

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	cmd_context "github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"
	"github.com/cofide/cofidectl/internal/pkg/attestationpolicy"
	"github.com/gobeam/stringy"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
)

type AttestationPolicyCommand struct {
	cmdCtx *cmd_context.CommandContext
}

func NewAttestationPolicyCommand(cmdCtx *cmd_context.CommandContext) *AttestationPolicyCommand {
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
			ds, err := c.cmdCtx.PluginManager.GetPlugin()
			if err != nil {
				return err
			}

			if err := ds.Validate(); err != nil {
				return err
			}

			attestationPolicies, err := ds.ListAttestationPolicies()
			if err != nil {
				return err
			}

			data := make([][]string, len(attestationPolicies))
			for i, policy := range attestationPolicies {
				data[i] = []string{
					policy.Kind.String(),
					policy.Namespace,
					policy.PodKey,
					policy.PodValue,
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
	Name          string
	FederatesWith string

	// annotated
	PodKey   string
	PodValue string

	// namespace
	Namespace string
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

			if !validateOpts(opts) {
				return errors.New("unset flags for attestation policy")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetPlugin()
			if err != nil {
				return err
			}

			if err := ds.Validate(); err != nil {
				return err
			}

			kind, err := attestationpolicy.GetAttestationPolicyKind(opts.kind)
			if err != nil {
				return err
			}

			newAttestationPolicy := &attestation_policy_proto.AttestationPolicy{
				Kind:      kind,
				Name:      opts.attestationPolicyOpts.Name,
				Namespace: opts.attestationPolicyOpts.Namespace,
				PodKey:    opts.attestationPolicyOpts.PodKey,
				PodValue:  opts.attestationPolicyOpts.PodValue,
			}
			err = ds.AddAttestationPolicy(newAttestationPolicy)
			if err != nil {
				return err
			}

			trustZone, err := ds.GetTrustZone(opts.trustZoneName)
			if err != nil {
				return err
			}
			return ds.BindAttestationPolicy(newAttestationPolicy, trustZone)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.trustZoneName, "trust-zone", "", "Name of the trust zone to attach this attestation policy to")
	f.StringVar(&opts.attestationPolicyOpts.Name, "name", "", "Name to use for the attestation policy")
	f.StringVar(&opts.attestationPolicyOpts.Namespace, "namespace", "", "Namespace to use in Namespace attestation policy")
	f.StringVar(&opts.attestationPolicyOpts.PodKey, "annotation-key", "", "Key of Pod annotation to use in Annotated attestation policy")
	f.StringVar(&opts.attestationPolicyOpts.PodValue, "annotation-value", "", "Value of Pod annotation to use in Annotated attestation policy")
	f.StringVar(&opts.attestationPolicyOpts.FederatesWith, "federates-with", "", "Defines a trust domain to federate identity with")

	cobra.CheckErr(cmd.MarkFlagRequired("trust-zone"))
	cobra.CheckErr(cmd.MarkFlagRequired("name"))

	return cmd
}

func validateOpts(opts Opts) bool {
	if opts.kind == "namespace" && opts.attestationPolicyOpts.Namespace == "" {
		slog.Error("flag \"namespace\" must be provided for Namespace attestation policy kind")
		return false
	}

	if opts.kind == "annotated" && (opts.attestationPolicyOpts.PodKey == "" || opts.attestationPolicyOpts.PodValue == "") {
		slog.Error("flags \"annotation-key\" and \"annotation-value\" must be provided for annotated attestation policy kind")
		return false
	}

	return true
}

func GetAttestationPolicyKind(s string) (attestation_policy_proto.AttestationPolicyKind, error) {
	switch s {
	case "annotated":
		return attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_ANNOTATED, nil
	case "namespace":
		return attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_NAMESPACE, nil
	}

	// TODO: Update error message.
	return attestation_policy_proto.AttestationPolicyKind_ATTESTATION_POLICY_KIND_UNSPECIFIED, fmt.Errorf("unknown attestation policy kind %s", s)
}
