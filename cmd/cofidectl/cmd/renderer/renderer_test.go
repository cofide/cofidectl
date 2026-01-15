// Copyright 2026 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package renderer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableRenderer_RenderTables(t *testing.T) {
	table1 := Table{
		Title:  "Table 1",
		Header: []string{"Col1", "Col2"},
		Data: [][]string{
			{"Row1-1", "Row1-2"},
			{"Row2-1", "Row2-2"},
		},
	}
	table2 := Table{
		Title:  "Table 2",
		Header: []string{"ColA", "ColB", "ColC"},
		Data: [][]string{
			{"A1", "B1", "C1"},
		},
	}
	table3 := Table{
		Title:  "Table 3",
		Header: []string{"X", "Y"},
		Data: [][]string{
			{"X1", "Y1"},
			{"X2", "Y2"},
			{"X3", "Y3"},
		},
	}
	empty := Table{}
	expectedTable1 := `Table 1

   COL1  |  COL2   
---------+---------
  Row1-1 | Row1-2  
  Row2-1 | Row2-2  
`
	tests := []struct {
		name     string
		tables   []Table
		expected string
	}{
		{
			name:     "Single Table",
			tables:   []Table{table1},
			expected: expectedTable1,
		},
		{
			name:   "Multiple Tables",
			tables: []Table{table1, table2, table3},
			expected: `Table 1

   COL1  |  COL2   
---------+---------
  Row1-1 | Row1-2  
  Row2-1 | Row2-2  

Table 2

  COLA | COLB | COLC  
-------+------+-------
  A1   | B1   | C1    

Table 3

  X  | Y   
-----+-----
  X1 | Y1  
  X2 | Y2  
  X3 | Y3  
`,
		},
		{
			name:     "Empty first table",
			tables:   []Table{empty, table1},
			expected: expectedTable1,
		},
		{
			name:     "Empty last table",
			tables:   []Table{table1, empty},
			expected: expectedTable1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tr := NewTableRenderer(&buf)
			tr.RenderTables(tt.tables...)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestTableRenderer_RenderTable(t *testing.T) {
	title := "Test Table"
	header := []string{"Header1", "Header2"}
	data := [][]string{
		{"Data1A", "Data1B"},
		{"Data2A", "Data2B"},
	}
	tests := []struct {
		name         string
		table        Table
		wantOutput   string
		wantRendered bool
	}{
		{
			name:         "No Data",
			table:        Table{Title: title, Header: header},
			wantOutput:   "",
			wantRendered: false,
		},
		{
			name:  "No title",
			table: Table{Header: header, Data: data},
			wantOutput: `  HEADER1 | HEADER2  
----------+----------
  Data1A  | Data1B   
  Data2A  | Data2B   
`,
			wantRendered: true,
		},
		{
			name:  "With title and data",
			table: Table{Title: title, Header: header, Data: data},
			wantOutput: `Test Table

  HEADER1 | HEADER2  
----------+----------
  Data1A  | Data1B   
  Data2A  | Data2B   
`,
			wantRendered: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tr := NewTableRenderer(&buf)
			rendered := tr.RenderTable(tt.table)
			assert.Equal(t, tt.wantOutput, buf.String())
			assert.Equal(t, tt.wantRendered, rendered)
		})
	}
}
