package cmd

import (
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/spf13/cobra"
)

type InitCommand struct {
	source cofidectl_plugin.DataSource
}

func NewInitCommand(source cofidectl_plugin.DataSource) *InitCommand {
	return &InitCommand{
		source: source,
	}
}

var initRootCmdDesc = `
This command initialises a new Cofide planfile in the current working
directory
`

func (i *InitCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [ARGS]",
		Short: "Initialises the Cofide planfile",
		Long:  initRootCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}
