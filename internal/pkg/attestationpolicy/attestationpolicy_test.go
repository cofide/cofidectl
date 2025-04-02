// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package attestationpolicy

import (
	"testing"

	types "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_formatSelectors(t *testing.T) {
	tests := []struct {
		name          string
		selectors     []*types.Selector
		want          []string
		wantErr       bool
		wantErrString string
	}{
		{
			name: "valid selector",
			selectors: []*types.Selector{
				{
					Type:  "k8s",
					Value: "ns:foo",
				},
			},
			want: []string{"k8s:ns:foo"},
		},
		{
			name: "multiple selectors",
			selectors: []*types.Selector{
				{
					Type:  "k8s",
					Value: "ns:foo",
				},
				{
					Type:  "k8s",
					Value: "ns:bar",
				},
			},
			want: []string{"k8s:ns:foo", "k8s:ns:bar"},
		},
		{
			name: "multiple selectors with different types",
			selectors: []*types.Selector{
				{
					Type:  "k8s",
					Value: "ns:foo",
				},
				{
					Type:  "k8s_psat",
					Value: "cluster:bar",
				},
			},
			want: []string{"k8s:ns:foo", "k8s_psat:cluster:bar"},
		},
		{
			name:      "no selectors",
			selectors: []*types.Selector{},
			want:      []string{},
		},
		{
			name: "selector with empty type",
			selectors: []*types.Selector{
				{
					Type:  "",
					Value: "ns:foo",
				},
			},
			want:          nil,
			wantErr:       true,
			wantErrString: "invalid selector type=\"\", value=\"ns:foo\"",
		},
		{
			name: "selector with empty value",
			selectors: []*types.Selector{
				{
					Type:  "k8s",
					Value: "",
				},
			},
			want:          nil,
			wantErr:       true,
			wantErrString: "invalid selector type=\"k8s\", value=\"\"",
		},
		{
			name:      "nil selector list",
			selectors: nil,
			want:      []string{},
		},
		{
			name: "mixed valid and empty selectors",
			selectors: []*types.Selector{
				{
					Type:  "k8s",
					Value: "ns:foo",
				},
				{
					Type:  "",
					Value: "",
				},
				{
					Type:  "k8s",
					Value: "ns:bar",
				},
			},
			want:          nil,
			wantErr:       true,
			wantErrString: "invalid selector type=\"\", value=\"\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatSelectors(tt.selectors)
			if !tt.wantErr {
				require.Nil(t, err, "unexpected error")
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
