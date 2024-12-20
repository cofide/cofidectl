// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"os"
	"testing"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestHelmSPIREProvider(t *testing.T) {
	trustZoneProto := &trust_zone_proto.TrustZone{TrustDomain: "foo.bar"}
	spireValues := map[string]any{}
	spireCRDsValues := map[string]any{}

	p, err := NewHelmSPIREProvider(context.Background(), trustZoneProto, spireValues, spireCRDsValues)
	assert.Nil(t, err)
	assert.Equal(t, p.SPIREVersion, "0.21.0")
	assert.Equal(t, p.SPIRECRDsVersion, "0.4.0")
	assert.Equal(t, trustZoneProto.TrustDomain, p.trustZone.TrustDomain)
}

func TestGetChartRef(t *testing.T) {
	originalPath := os.Getenv("HELM_REPO_PATH")
	defer os.Setenv("HELM_REPO_PATH", originalPath)

	tests := []struct {
		name            string
		helmRepoPath    string
		helmRepoPathSet bool
		chartName       string
		want            string
		wantErr         bool
		errString       string
	}{
		{
			name:            "with HELM_REPO_PATH set",
			helmRepoPathSet: true,
			helmRepoPath:    "spire-local",
			chartName:       "spire",
			want:            "spire-local/spire",
			wantErr:         false,
		},
		{
			name:            "with HELM_REPO_PATH containing trailing slash",
			helmRepoPathSet: true,
			helmRepoPath:    "custom-repo/",
			chartName:       "spire",
			want:            "custom-repo/spire",
			wantErr:         false,
		},
		{
			name:            "with HELM_REPO_PATH containing trailing slashes",
			helmRepoPathSet: true,
			helmRepoPath:    "custom-repo//",
			chartName:       "spire",
			want:            "custom-repo/spire",
			wantErr:         false,
		},
		{
			name:            "with empty HELM_REPO_PATH",
			helmRepoPathSet: true,
			helmRepoPath:    "",
			chartName:       "spire",
			want:            "spire/spire",
			wantErr:         true,
			errString:       "HELM_REPO_PATH environment variable is set but empty",
		},
		{
			name:            "with empty chart name",
			helmRepoPathSet: false,
			helmRepoPath:    "spire-local",
			chartName:       "",
			wantErr:         true,
			errString:       "chart name cannot be empty",
		},
		{
			name:            "with HELM_REPO_PATH not set",
			helmRepoPathSet: false,
			chartName:       "spire",
			want:            "spire/spire",
			wantErr:         false,
		},
		{
			name:            "with empty HELM_REPO_PATH and an empty chart name",
			helmRepoPathSet: false,
			chartName:       "",
			wantErr:         true,
			errString:       "chart name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.helmRepoPathSet {
				os.Setenv("HELM_REPO_PATH", tt.helmRepoPath)
			} else {
				os.Unsetenv("HELM_REPO_PATH")
			}

			got, err := getChartRef(tt.chartName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
