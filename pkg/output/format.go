// Copyright 2026 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package output

// Format represents an output format for CLI commands.
type Format string

const (
	TableFormat Format = "table"
	JSONFormat  Format = "json"
)

// ValidFormats is the list of supported output formats.
var ValidFormats = []Format{TableFormat, JSONFormat}
