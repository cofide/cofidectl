package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"
	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/spf13/cobra"
)

type InitCommand struct {
	cmdCtx *context.CommandContext
}

func NewInitCommand(cmdCtx *context.CommandContext) *InitCommand {
	return &InitCommand{
		cmdCtx: cmdCtx,
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
			ds, err := i.cmdCtx.PluginManager.GetPlugin()
			if err != nil {
				return err
			}

			if err := ds.Init(); err != nil {
				log.Fatal(err)
			}

			if opts.enableConnect {
				if ok, _ := plugin.PluginExists("cofidectl-connect"); ok {
					cfg, err := i.cmdCtx.ConfigLoader.Read()
					if err != nil {
						return fmt.Errorf("could not open local config")
					}
					cfg.Plugins = append(cfg.Plugins, "cofidectl-connect")
					err = i.cmdCtx.ConfigLoader.Write(cfg)
					if err != nil {
						return fmt.Errorf("could not enable Connect plugin")
					}
					fmt.Println("cofidectl is now Connect-enabled")
					return nil
				} else {
					fmt.Println("👀 get in touch with us at hello@cofide.io to find out more")
					os.Exit(1)
				}
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&opts.enableConnect, "enable-connect", false, "Enables Cofide Connect")

	return cmd
}
