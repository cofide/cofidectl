// Copyright 2026 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package renderer

import (
	"fmt"
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

var _ Renderer = (*TableRenderer)(nil)

// Renderer provides an interface for rendering columnar data.
type Renderer interface {
	RenderTables(tables ...Table) (bool, error)
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

// RenderTables renders the specified tables to the table renderer's writer.
// It returns whether any tables were rendered.
func (tr *TableRenderer) RenderTables(tables ...Table) (bool, error) {
	rendered := false
	for _, table := range tables {
		if !table.IsEmpty() && rendered {
			_, _ = fmt.Fprintln(tr.writer)
		}
		if r, err := tr.renderTable(table); err != nil {
			return false, err
		} else if r {
			rendered = true
		}
	}
	return rendered, nil
}

// renderTable renders the specified table to the renderer's writer.
// It returns whether the table was rendered.
func (tr *TableRenderer) renderTable(table Table) (bool, error) {
	if table.IsEmpty() {
		return false, nil
	}
	tw := tablewriter.NewTable(
		tr.writer,
		tablewriter.WithRenderer(
			renderer.NewBlueprint(
				tw.Rendition{
					Borders: tw.BorderNone,
					Symbols: tw.NewSymbols(tw.StyleASCII),
				},
			),
		),
		tablewriter.WithHeader(table.Header),
	)
	if table.Title != "" {
		if _, err := fmt.Fprintf(tr.writer, "%s\n\n", table.Title); err != nil {
			return false, err
		}
	}
	if err := tw.Bulk(table.Data); err != nil {
		return false, err
	}
	if err := tw.Render(); err != nil {
		return false, err
	}
	return true, nil
}

// IsEmpty returns true if the table has no data.
func (t Table) IsEmpty() bool {
	return len(t.Data) == 0
}
