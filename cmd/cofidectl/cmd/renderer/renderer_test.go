// Copyright 2026 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package renderer

import (
	"bytes"
	"encoding/json"
	"testing"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
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

 COL 1  | COL 2  
--------+--------
 Row1-1 | Row1-2 
 Row2-1 | Row2-2 
`
	tests := []struct {
		name         string
		tables       []Table
		wantOutput   string
		wantRendered bool
	}{
		{
			name:         "Single Table",
			tables:       []Table{table1},
			wantOutput:   expectedTable1,
			wantRendered: true,
		},
		{
			name:   "Multiple Tables",
			tables: []Table{table1, table2, table3},
			wantOutput: `Table 1

 COL 1  | COL 2  
--------+--------
 Row1-1 | Row1-2 
 Row2-1 | Row2-2 

Table 2

 COL A | COL B | COL C 
-------+-------+-------
 A1    | B1    | C1    

Table 3

 X  | Y  
----+----
 X1 | Y1 
 X2 | Y2 
 X3 | Y3 
`,
			wantRendered: true,
		},
		{
			name:         "Empty first table",
			tables:       []Table{empty, table1},
			wantOutput:   expectedTable1,
			wantRendered: true,
		},
		{
			name:         "Empty last table",
			tables:       []Table{table1, empty},
			wantOutput:   expectedTable1,
			wantRendered: true,
		},
		{
			name:         "All empty tables",
			tables:       []Table{empty, empty},
			wantOutput:   "",
			wantRendered: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tr := NewTableRenderer(&buf)
			rendered, err := tr.RenderTables(tt.tables...)
			require.NoError(t, err)
			assert.Equal(t, tt.wantRendered, rendered)
			assert.Equal(t, tt.wantOutput, buf.String())
		})
	}
}

func TestTableRenderer_renderTable(t *testing.T) {
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
			wantOutput: ` HEADER 1 | HEADER 2 
----------+----------
 Data1A   | Data1B   
 Data2A   | Data2B   
`,
			wantRendered: true,
		},
		{
			name:  "With title and data",
			table: Table{Title: title, Header: header, Data: data},
			wantOutput: `Test Table

 HEADER 1 | HEADER 2 
----------+----------
 Data1A   | Data1B   
 Data2A   | Data2B   
`,
			wantRendered: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tr := NewTableRenderer(&buf)
			rendered, err := tr.renderTable(tt.table)
			require.NoError(t, err)
			assert.Equal(t, tt.wantOutput, buf.String())
			assert.Equal(t, tt.wantRendered, rendered)
		})
	}
}

func TestJSONRenderer_RenderTables(t *testing.T) {
	tz1 := &trust_zone_proto.TrustZone{Name: "tz1", TrustDomain: "example.org"}
	tz2 := &trust_zone_proto.TrustZone{Name: "tz2", TrustDomain: "other.org"}

	tableWithObjects := Table{
		Title:   "Trust Zones",
		Header:  []string{"Name", "Trust Domain"},
		Data:    [][]string{{"tz1", "example.org"}},
		Objects: []proto.Message{tz1},
	}
	tableWithTwoObjects := Table{
		Title:   "Trust Zones",
		Header:  []string{"Name", "Trust Domain"},
		Data:    [][]string{{"tz1", "example.org"}, {"tz2", "other.org"}},
		Objects: []proto.Message{tz1, tz2},
	}
	tableNoObjects := Table{
		Title:  "Empty",
		Header: []string{"Name"},
		Data:   [][]string{},
	}
	tableNoData := Table{
		Title:  "No data",
		Header: []string{"Name"},
	}
	tableWithTitle := Table{
		Title:   "Zone A",
		Header:  []string{"Name"},
		Data:    [][]string{{"a"}},
		Objects: []proto.Message{tz1},
	}
	tableWithTitle2 := Table{
		Title:   "Zone B",
		Header:  []string{"Name"},
		Data:    [][]string{{"b"}},
		Objects: []proto.Message{tz2},
	}

	tests := []struct {
		name         string
		tables       []Table
		wantRendered bool
		checkOutput  func(t *testing.T, output string)
	}{
		{
			name:         "No tables",
			tables:       []Table{},
			wantRendered: false,
			checkOutput: func(t *testing.T, output string) {
				assert.Empty(t, output)
			},
		},
		{
			name:         "All empty tables",
			tables:       []Table{tableNoObjects, tableNoData},
			wantRendered: false,
			checkOutput: func(t *testing.T, output string) {
				assert.Empty(t, output)
			},
		},
		{
			name:         "Single table with one object",
			tables:       []Table{tableWithObjects},
			wantRendered: true,
			checkOutput: func(t *testing.T, output string) {
				var arr []json.RawMessage
				require.NoError(t, json.Unmarshal([]byte(output), &arr))
				assert.Len(t, arr, 1)
				var obj map[string]any
				require.NoError(t, json.Unmarshal(arr[0], &obj))
				assert.Equal(t, "tz1", obj["name"])
				assert.Equal(t, "example.org", obj["trustDomain"])
			},
		},
		{
			name:         "Single table with two objects",
			tables:       []Table{tableWithTwoObjects},
			wantRendered: true,
			checkOutput: func(t *testing.T, output string) {
				var arr []json.RawMessage
				require.NoError(t, json.Unmarshal([]byte(output), &arr))
				assert.Len(t, arr, 2)
			},
		},
		{
			name:         "Empty table skipped",
			tables:       []Table{tableNoObjects, tableWithObjects},
			wantRendered: true,
			checkOutput: func(t *testing.T, output string) {
				var arr []json.RawMessage
				require.NoError(t, json.Unmarshal([]byte(output), &arr))
				assert.Len(t, arr, 1)
			},
		},
		{
			name:         "Multiple non-empty tables keyed by title",
			tables:       []Table{tableWithTitle, tableWithTitle2},
			wantRendered: true,
			checkOutput: func(t *testing.T, output string) {
				var obj map[string][]json.RawMessage
				require.NoError(t, json.Unmarshal([]byte(output), &obj))
				assert.Contains(t, obj, "Zone A")
				assert.Contains(t, obj, "Zone B")
				assert.Len(t, obj["Zone A"], 1)
				assert.Len(t, obj["Zone B"], 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			jr := NewJSONRenderer(&buf)
			rendered, err := jr.RenderTables(tt.tables...)
			require.NoError(t, err)
			assert.Equal(t, tt.wantRendered, rendered)
			tt.checkOutput(t, buf.String())
		})
	}
}
