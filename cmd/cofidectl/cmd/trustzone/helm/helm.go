// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	cmdcontext "github.com/cofide/cofidectl/cmd/cofidectl/cmd/context"
	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/provider/helm"
)

type HelmCommand struct {
	cmdCtx *cmdcontext.CommandContext
}

func NewHelmCommand(cmdCtx *cmdcontext.CommandContext) *HelmCommand {
	return &HelmCommand{
		cmdCtx: cmdCtx,
	}
}

var helmRootCmdDesc = `
This command consists of multiple sub-commands to administer Cofide trust zone Helm configuration.
`

func (c *HelmCommand) GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "helm override|values [ARGS]",
		Short: "Manage trust zone Helm configuration",
		Long:  helmRootCmdDesc,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		c.GetOverrideCommand(),
		c.GetValuesCommand(),
	)

	return cmd
}

var helmOverrideCmdDesc = `
This command will override Helm values for a trust zone in the Cofide configuration state.
`

type overrideOpts struct {
	inputPath string
}

func (c *HelmCommand) GetOverrideCommand() *cobra.Command {
	opts := overrideOpts{}
	cmd := &cobra.Command{
		Use:   "override [ARGS]",
		Short: "Override Helm values for a trust zone",
		Long:  helmOverrideCmdDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource()
			if err != nil {
				return err
			}

			var reader io.Reader
			if opts.inputPath == "-" {
				reader = os.Stdin
			} else {
				f, err := os.Open(opts.inputPath)
				if err != nil {
					return err
				}
				defer f.Close()
				reader = f
			}
			values, err := readValues(reader)
			if err != nil {
				return err
			}

			return c.overrideValues(ds, args[0], values)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.inputPath, "input-file", "values.yaml", "Path of a file to read YAML values from, or - for stdin")

	return cmd
}

// overrideValues overrides Helm values for a trust zone.
func (c *HelmCommand) overrideValues(ds plugin.DataSource, tzName string, values map[string]interface{}) error {
	trustZone, err := ds.GetTrustZone(tzName)
	if err != nil {
		return err
	}

	trustZone.ExtraHelmValues, err = structpb.NewStruct(values)
	if err != nil {
		return err
	}

	// Check that the values are acceptable.
	generator := helm.NewHelmValuesGenerator(trustZone, ds)
	if _, err = generator.GenerateValues(); err != nil {
		return err
	}

	return ds.UpdateTrustZone(trustZone)
}

// readValues reads values in YAML format from the specified reader.
func readValues(reader io.Reader) (map[string]interface{}, error) {
	decoder := yaml.NewDecoder(reader)
	var values map[string]interface{}
	err := decoder.Decode(&values)
	return values, err
}

var helmValuesCmdDesc = `
This command will generate Helm values for a trust zone in the Cofide configuration state.
`

type valuesOpts struct {
	outputPath string
}

func (c *HelmCommand) GetValuesCommand() *cobra.Command {
	opts := valuesOpts{}
	cmd := &cobra.Command{
		Use:   "values [ARGS]",
		Short: "Generate Helm values for a trust zone",
		Long:  helmValuesCmdDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ds, err := c.cmdCtx.PluginManager.GetDataSource()
			if err != nil {
				return err
			}

			values, err := c.getValues(ds, args[0])
			if err != nil {
				return err
			}

			var writer io.Writer
			if opts.outputPath == "-" {
				writer = os.Stdout
			} else {
				f, err := os.Create(opts.outputPath)
				if err != nil {
					return err
				}
				defer f.Close()
				writer = f
			}
			if err := writeValues(values, writer); err != nil {
				return err
			}
			if opts.outputPath != "-" {
				fmt.Printf("Wrote Helm values to %s\n", opts.outputPath)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.outputPath, "output-file", "values.yaml", "Path of a file to write YAML values to, or - for stdout")

	return cmd
}

// getValues returns the Helm values for a trust zone.
func (c *HelmCommand) getValues(ds plugin.DataSource, tzName string) (map[string]interface{}, error) {
	trustZone, err := ds.GetTrustZone(tzName)
	if err != nil {
		return nil, err
	}

	generator := helm.NewHelmValuesGenerator(trustZone, ds)
	values, err := generator.GenerateValues()
	if err != nil {
		return nil, err
	}
	return values, nil
}

// writeValues writes values in YAML format to the specified writer.
func writeValues(values map[string]interface{}, writer io.Writer) error {
	encoder := yaml.NewEncoder(writer)
	defer encoder.Close()
	return encoder.Encode(values)
}
