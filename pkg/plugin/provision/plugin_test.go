// Copyright 2025 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test_valuesToStruct(t *testing.T) {
	tests := []struct {
		name         string
		values       map[string]any
		want         *structpb.Struct
		breaksStruct bool
	}{
		{
			name:   "empty map",
			values: map[string]any{},
			want: func() *structpb.Struct {
				s, err := structpb.NewStruct(nil)
				require.NoError(t, err)
				return s
			}(),
		},
		{
			name: "breaks struct",
			values: map[string]any{
				"foo": "bar",
				// []string is not supported by Struct. JSON round trip converts this to []any.
				"baz": []string{"qux"},
			},
			want: func() *structpb.Struct {
				s, err := structpb.NewStruct(map[string]any{
					"foo": "bar",
					"baz": []any{"qux"},
				})
				require.NoError(t, err)
				return s
			}(),
			breaksStruct: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valuesToStruct(tt.values)
			require.NoError(t, err)
			assert.EqualExportedValues(t, tt.want, got)

			if tt.breaksStruct {
				// Confirm that these values are not natively supported by Struct.
				_, err := structpb.NewStruct(tt.values)
				require.Error(t, err)
			}
		})
	}
}
