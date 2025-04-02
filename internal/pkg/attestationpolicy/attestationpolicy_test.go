// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package attestationpolicy

import (
	"testing"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	types "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_findTrustZone(t *testing.T) {
	tests := []struct {
		name          string
		trustZoneID   string
		configFunc    func(*config.Config)
		want          *trust_zone_proto.TrustZone
		wantErr       bool
		wantErrString string
	}{
		{
			name:        "trust zone found",
			trustZoneID: "tz1-id",
			want: &trust_zone_proto.TrustZone{
				Id:   ptrOf("tz1-id"),
				Name: "tz1",
			},
		},
		{
			name:          "trust zone not found",
			trustZoneID:   "tz2-id",
			wantErr:       true,
			wantErrString: "trust zone not found with ID: tz2-id",
		},
		{
			name:          "empty trust zone ID",
			trustZoneID:   "",
			wantErr:       true,
			wantErrString: "trust zone ID is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := newMockDataSource()
			got, err := findTrustZone(source, tt.trustZoneID)
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

type mockDataSource struct {
	*local.LocalDataSource
}

func newMockDataSource() datasource.DataSource {
	configLoader, _ := config.NewMemoryLoader(&config.Config{})
	localDS, _ := local.NewLocalDataSource(configLoader)
	return &mockDataSource{LocalDataSource: localDS}
}

func (m *mockDataSource) ListTrustZones() ([]*trust_zone_proto.TrustZone, error) {
	trustZones := []*trust_zone_proto.TrustZone{
		{
			Id:   ptrOf("tz1-id"),
			Name: "tz1",
		},
	}

	return trustZones, nil
}

func ptrOf[T any](x T) *T {
	return &x
}
