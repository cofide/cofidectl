// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/statusspinner"
	"github.com/cofide/cofidectl/internal/pkg/provider/helm"
	"github.com/spf13/cobra"
)

type DownCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewDownCommand(cmdCtx *cmdcontext.CommandContext) *DownCommand {
	return &DownCommand{
		cmdCtx: cmdCtx}
}

var downCmdDesc = `
This command uninstalls a Cofide configuration
`

func (d *DownCommand) DownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down [ARGS]",
		Short: "Uninstalls a Cofide configuration",
		Long:  downCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := d.cmdCtx.PluginManager.GetDataSource()
			if err != nil {
				return err
			}

			trustZones, err := ds.ListTrustZones()
			if err != nil {
				return err
			}

			if len(trustZones) == 0 {
				fmt.Println("no trust zones have been configured")
				return nil
			}

			if err := uninstallSPIREStack(cmd.Context(), trustZones); err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}

func uninstallSPIREStack(ctx context.Context, trustZones []*trust_zone_proto.TrustZone) error {
	for _, trustZone := range trustZones {
		prov, err := helm.NewHelmSPIREProvider(ctx, trustZone, nil, nil)
		if err != nil {
			return err
		}

		s := statusspinner.New()
		statusCh := prov.ExecuteUninstall()
		if err := s.Watch(statusCh); err != nil {
			return fmt.Errorf("uninstallation failed: %w", err)
		}
	}
	return nil
}
