// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"

	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/spf13/cobra"
)

type VersionCommand struct {
	name    string
	version string
	cmdCtx  *cmdcontext.CommandContext
}

func NewVersionCommand(name string, version string, cmdCtx *cmdcontext.CommandContext) *VersionCommand {
	return &VersionCommand{
		name:    name,
		version: version,
		cmdCtx:  cmdCtx,
	}
}

var versionCmdDesc = `
This command prints version information
`

type VersionOpts struct {
	printBuildInfo bool
}

func (v *VersionCommand) VersionCmd() *cobra.Command {
	opts := VersionOpts{}
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  versionCmdDesc,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(fmt.Sprintf("%s %s (%s/%s)", v.name, v.version, runtime.GOOS, runtime.GOARCH))

			if opts.printBuildInfo {
				buildInfo, ok := debug.ReadBuildInfo()

				cmd.Println("\nBuild information:")

				if ok {
					cmd.Println(buildInfo)
				} else {
					cmd.Println("Build information not available.")
				}
			}
		},
	}

	f := cmd.Flags()
	f.BoolVar(&opts.printBuildInfo, "build-info", false, "print build information")

	return cmd
}
