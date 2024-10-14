// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/apbinding"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/attestationpolicy"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/dev"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/federation"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/trustzone"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/workload"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"

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
	var logLevel string

	cmd := &cobra.Command{
		Use:          "cofidectl",
		Short:        "Cofide CLI",
		Long:         rootCmdDesc,
		SilenceUsage: true,
		// This runs before any subcommand.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			slogLevel, err := slogLevelFromString(logLevel)
			if err != nil {
				return err
			}

			r.cmdCtx.SetLogLevel(slogLevel)
			slog.Debug("Set slog level", slog.String("level", slogLevel.String()))
			return nil
		},
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	pf := cmd.PersistentFlags()
	pf.StringVar(&kubeCfgFile, "kube-config", path.Join(home, ".kube/config"), "kubeconfig file location")
	pf.StringVar(&logLevel, "log-level", "ERROR", "log level")

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
		dev.NewDevCmd(),
	)

	return cmd, nil
}

// slogLevelFromString returns an slog.Level from a string log level.
// The string level is case-insensitive.
func slogLevelFromString(level string) (slog.Level, error) {
	var slogLevel slog.Level
	// slog accepts funky inputs like INFO-3, so restrict what we accept.
	validLevels := []string{"debug", "warn", "info", "error"}
	if !slices.Contains(validLevels, strings.ToLower(level)) {
		return slogLevel, fmt.Errorf("unexpected log level %s, valid levels: %s", level, strings.Join(validLevels, ", "))
	}

	if err := slogLevel.UnmarshalText([]byte(level)); err != nil {
		return slogLevel, fmt.Errorf("unexpected log level %s: %w", level, err)
	}
	return slogLevel, nil
}
