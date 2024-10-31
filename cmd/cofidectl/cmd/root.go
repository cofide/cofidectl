package cmd

import (
	"os"
	"path"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/attestationpolicy"
	cmd_context "github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/federation"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/trustzone"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/workload"

	"github.com/spf13/cobra"
)

type RootCommand struct {
	cmdCtx *cmd_context.CommandContext
}

var kubeCfgFile string

func NewRootCommand(cmdCtx *cmd_context.CommandContext) *RootCommand {
	return &RootCommand{
		cmdCtx: cmdCtx,
	}
}

var rootCmdDesc = `cofidectl - Workload identity for hybrid and multi-cloud security`

func (r *RootCommand) GetRootCommand() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:          "cofidectl",
		Short:        "Cofide CLI",
		Long:         rootCmdDesc,
		SilenceUsage: true,
	}

	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	cmd.PersistentFlags().StringVar(&kubeCfgFile, "kube-config", path.Join(home, ".kube/config"), "kubeconfig file location")

	initCmd := NewInitCommand(r.cmdCtx)
	upCmd := NewUpCommand(r.cmdCtx)
	downCmd := NewDownCommand(r.cmdCtx)
	tzCmd := trustzone.NewTrustZoneCommand(r.cmdCtx)
	apCmd := attestationpolicy.NewAttestationPolicyCommand(r.cmdCtx)
	fedCmd := federation.NewFederationCommand(r.cmdCtx)
	wlCmd := workload.NewWorkloadCommand(r.cmdCtx)

	cmd.AddCommand(
		initCmd.GetRootCommand(),
		tzCmd.GetRootCommand(),
		apCmd.GetRootCommand(),
		fedCmd.GetRootCommand(),
		wlCmd.GetRootCommand(),
		upCmd.UpCmd(),
		downCmd.DownCmd(),
	)

	return cmd, nil
}
