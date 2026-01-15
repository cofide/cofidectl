// Copyright 2026 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package renderer

import (
	"fmt"
	"io"

	"github.com/olekukonko/tablewriter"
)

// Renderer provides an interface for rendering columnar data.
type Renderer interface {
	Render(tables ...Table)
}

// TableRenderer provides a Renderer implementation to render to an io.Writer as a set of tables.
type TableRenderer struct {
	writer io.Writer
}

// Table defines a single table with an optional title.
type Table struct {
	Title  string
	Header []string
	Data   [][]string
}

// NewTableRenderer returns a new TableRenderer for the specified writer.
// It renders tables using the tablewriter module.
func NewTableRenderer(writer io.Writer) *TableRenderer {
	return &TableRenderer{
		writer: writer,
	}
}

// Render renders the specified tables to the table renderer's writer.
// It returns whether any tables were rendered.
func (tr *TableRenderer) RenderTables(tables ...Table) bool {
	rendered := false
	for _, table := range tables {
		if !table.IsEmpty() && rendered {
			_, _ = fmt.Fprintln(tr.writer)
		}
		if tr.RenderTable(table) {
			rendered = true
		}
	}
	return rendered
}

// Render renders the specified table to the renderer's writer.
// It returns whether the table was rendered.
func (tr *TableRenderer) RenderTable(table Table) bool {
	if table.IsEmpty() {
		return false
	}
	tw := tablewriter.NewWriter(tr.writer)
	if table.Title != "" {
		_, _ = fmt.Fprintln(tr.writer, table.Title)
		_, _ = fmt.Fprintln(tr.writer)
	}
	tw.SetHeader(table.Header)
	tw.SetBorder(false)
	tw.AppendBulk(table.Data)
	tw.Render()
	return true
}

// IsEmpty returns true if the table has no data.
func (t Table) IsEmpty() bool {
	return len(t.Data) == 0
}
