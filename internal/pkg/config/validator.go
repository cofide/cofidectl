// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	_ "embed"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cue_yaml "cuelang.org/go/encoding/yaml"
)

//go:embed schema.cue
var schemaCue string

// Validator validates YAML-encoded configuration against a schema using Cue.
type Validator struct {
	cueContext *cue.Context
}

func NewValidator() *Validator {
	return &Validator{cueContext: cuecontext.New()}
}

func (v *Validator) Validate(data []byte) error {
	// Validate the YAML using the Cue schema
	schema := v.cueContext.CompileString(schemaCue)
	if err := schema.Err(); err != nil {
		return fmt.Errorf("error compiling Cue schema: %w", err)
	}

	if err := cue_yaml.Validate(data, schema); err != nil {
		return fmt.Errorf("error validating configuration YAML: %w", err)
	}

	return nil
}
