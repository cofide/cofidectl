package cmd

import (
	"encoding/json"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/cofide/cofidectl/internal/pkg/provider"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/spf13/cobra"
)

type UpCommand struct {
	source cofidectl_plugin.DataSource
}

func NewUpCommand(source cofidectl_plugin.DataSource) *UpCommand {
	return &UpCommand{
		source: source,
	}
}

var upDesc = `
This command deploys a Cofide configuration
`

func (u *UpCommand) UpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [ARGS]",
		Short: "Deploy a Cofide configuration",
		Long:  upDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			spireValues, err := u.generateSPIREValues()
			if err != nil {
				return err
			}
			spireCRDsValues := map[string]interface{}{}

			prov := provider.NewHelmSPIREProvider(spireValues, spireCRDsValues)
			prov.Execute()

			return nil
		},
	}
	return cmd
}

func (u *UpCommand) generateSPIREValues() (map[string]interface{}, error) {
	trustZones, err := u.source.ListTrustZones()
	if err != nil {
		return nil, err
	}

	if len(trustZones) < 1 {
		return nil, fmt.Errorf("no trust zones have been defined")
	}

	ctx := cuecontext.New()
	valuesCUE := ctx.CompileBytes([]byte{})

	// TODO: This should gracefully handle the case where more than one trust zone has been defined.
	valuesCUE = valuesCUE.FillPath(cue.ParsePath("global.spire.trustDomain"), trustZones[0].TrustDomain)
	valuesJSON, err := valuesCUE.MarshalJSON()
	if err != nil {
		// TODO: Improve error messaging.
		return nil, err
	}

	var values map[string]interface{}

	err = json.Unmarshal([]byte(valuesJSON), &values)
	if err != nil {
		// TODO: Improve error messaging.
		return nil, err
	}

	return values, nil
}
