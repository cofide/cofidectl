// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/apbinding"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/attestationpolicy"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/federation"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/trustzone"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/workload"

	"github.com/spf13/cobra"
)

type RootCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

var kubeCfgFile string

func NewRootCommand(cmdCtx *cmdcontext.CommandContext) *RootCommand {
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
	if err != nil {
		return nil, err
	}

	cmd.PersistentFlags().StringVar(&kubeCfgFile, "kube-config", path.Join(home, ".kube/config"), "kubeconfig file location")

	initCmd := NewInitCommand(r.cmdCtx)
	upCmd := NewUpCommand(r.cmdCtx)
	downCmd := NewDownCommand(r.cmdCtx)
	tzCmd := trustzone.NewTrustZoneCommand(r.cmdCtx)
	apCmd := attestationpolicy.NewAttestationPolicyCommand(r.cmdCtx)
	apbCmd := apbinding.NewAPBindingCommand(r.cmdCtx)
	fedCmd := federation.NewFederationCommand(r.cmdCtx)
	wlCmd := workload.NewWorkloadCommand(r.cmdCtx)

	cmd.AddCommand(
		initCmd.GetRootCommand(),
		tzCmd.GetRootCommand(),
		apCmd.GetRootCommand(),
		apbCmd.GetRootCommand(),
		fedCmd.GetRootCommand(),
		wlCmd.GetRootCommand(),
		upCmd.UpCmd(),
		downCmd.DownCmd(),
	)

	return cmd, nil
}
