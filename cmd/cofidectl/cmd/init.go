package cmd

import (
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/spf13/cobra"
)

var initDesc = `
This command initialises a new Cofide planfile in the current working
directory
`

type InitCommand struct {
	source cofidectl_plugin.DataSource
}

func NewInitCommand(source cofidectl_plugin.DataSource) *InitCommand {
	return &InitCommand{
		source: source,
	}
}

func (i *InitCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [ARGS]",
		Short: "Initialises the Cofide planfile",
		Long:  initDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}
