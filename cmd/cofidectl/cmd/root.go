package cmd

import (
	"os"
	"path"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/attestationpolicy"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/federation"
	"github.com/cofide/cofidectl/cmd/cofidectl/cmd/trustzone"

	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/spf13/cobra"
)

type RootCommand struct {
	source cofidectl_plugin.DataSource
	args   []string
}

func NewRootCommand(source cofidectl_plugin.DataSource, args []string) *RootCommand {
	return &RootCommand{
		source: source,
		args:   args,
	}
}

var cfgFile string
var kubeCfgFile string
var rootCmdDesc = `cofidectl - Workload identity for hybrid and multi-cloud security`

func (r *RootCommand) GetRootCommand() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:          "cofidectl",
		Short:        "Cofide CLI",
		Long:         rootCmdDesc,
		SilenceUsage: true,
	}

	//cobra.OnInitialize(initConfig)

	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	cmd.PersistentFlags().StringVar(&kubeCfgFile, "kube-config", path.Join(home, ".kube/config"), "kubeconfig file location")
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cofide.yaml)")

	upCmd := NewUpCommand(r.source)
	tzCmd := trustzone.NewTrustZoneCommand(r.source)
	apCmd := attestationpolicy.NewAttestationPolicyCommand(r.source)
	fedCmd := federation.NewFederationCommand(r.source)

	cmd.AddCommand(
		tzCmd.GetRootCommand(),
		apCmd.GetRootCommand(),
		fedCmd.GetRootCommand(),
		upCmd.UpCmd(),
	)

	return cmd, nil
}
