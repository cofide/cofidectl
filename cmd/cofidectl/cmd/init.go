package cmd

import (
	"fmt"
	"log"
	"os"

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
This command initialises a new Cofide config file in the current working
directory
`

type Opts struct {
	enableConnect bool
}

func (i *InitCommand) GetRootCommand() *cobra.Command {
	opts := Opts{}
	cmd := &cobra.Command{
		Use:   "init [ARGS]",
		Short: "Initialises the Cofide config file",
		Long:  initRootCmdDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.enableConnect {
				fmt.Println("👀 get in touch with us at hello@cofide.io to find out more")
				os.Exit(1)
			}

			if err := i.source.Init(); err != nil {
				log.Fatal(err)
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&opts.enableConnect, "enable-connect", false, "Enables Cofide Connect")

	return cmd
}