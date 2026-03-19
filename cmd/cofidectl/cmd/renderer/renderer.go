// Copyright 2026 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/cofide/cofidectl/pkg/output"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ Renderer = (*TableRenderer)(nil)
var _ Renderer = (*JSONRenderer)(nil)

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
	Title   string
	Header  []string
	Data    [][]string    // string rows for table output
	Objects []proto.Message // full API objects for JSON output
}

// NewTableRenderer returns a new TableRenderer for the specified writer.
// It renders tables using the tablewriter module.
func NewTableRenderer(writer io.Writer) *TableRenderer {
	return &TableRenderer{
		writer: writer,
	}
}

// New returns a Renderer for the given format writing to w.
func New(format output.Format, w io.Writer) (Renderer, error) {
	switch format {
	case output.TableFormat:
		return NewTableRenderer(w), nil
	case output.JSONFormat:
		return NewJSONRenderer(w), nil
	default:
		return nil, fmt.Errorf("unrecognised output format %q", format)
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

// JSONRenderer provides a Renderer implementation that outputs JSON.
type JSONRenderer struct {
	writer io.Writer
}

// NewJSONRenderer returns a new JSONRenderer for the specified writer.
func NewJSONRenderer(writer io.Writer) *JSONRenderer {
	return &JSONRenderer{writer: writer}
}

// RenderTables marshals all non-empty tables' Objects to JSON and writes to the writer.
// Single non-empty table → JSON array; multiple non-empty tables → JSON object keyed by title.
// It returns whether any output was written.
func (jr *JSONRenderer) RenderTables(tables ...Table) (bool, error) {
	// Collect non-empty tables (those with Objects).
	type namedArray struct {
		title string
		items []json.RawMessage
	}
	nonEmpty := make([]namedArray, 0, len(tables))
	for _, table := range tables {
		if len(table.Objects) == 0 {
			continue
		}
		items := make([]json.RawMessage, 0, len(table.Objects))
		for _, obj := range table.Objects {
			b, err := protojson.Marshal(obj)
			if err != nil {
				return false, fmt.Errorf("failed to marshal object: %w", err)
			}
			items = append(items, json.RawMessage(b))
		}
		title := table.Title
		if title == "" {
			title = "items"
		}
		nonEmpty = append(nonEmpty, namedArray{title: title, items: items})
	}

	if len(nonEmpty) == 0 {
		return false, nil
	}

	var buf bytes.Buffer
	var err error
	if len(nonEmpty) == 1 {
		err = encodeJSON(&buf, nonEmpty[0].items)
	} else {
		m := make(map[string][]json.RawMessage, len(nonEmpty))
		for _, na := range nonEmpty {
			m[na.title] = na.items
		}
		err = encodeJSON(&buf, m)
	}
	if err != nil {
		return false, err
	}
	_, err = buf.WriteTo(jr.writer)
	return err == nil, err
}

func encodeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
