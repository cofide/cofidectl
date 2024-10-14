package dev

import (
	"github.com/spf13/cobra"
)

var federationDesc = `
This command consists of multiple subcommands to administer the Cofide local development environment
`

func NewDevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev mini-spire [ARGS]",
		Short: "setup a local development spire",
		Long:  federationDesc,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(devMiniSpireCmd())
	return cmd
}
