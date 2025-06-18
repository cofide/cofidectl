// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"os"
	"testing"

	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestHelmSPIREProvider(t *testing.T) {
	cluster := &clusterpb.Cluster{Name: fixtures.StringPtr("fake-cluster")}
	spireValues := map[string]any{}
	spireCRDsValues := map[string]any{}
	kubeConfig := "fake-kube-config"

	p, err := NewHelmSPIREProvider(context.Background(), cluster, spireValues, spireCRDsValues, kubeConfig)
	assert.Nil(t, err)
	assert.Equal(t, p.SPIREVersion, "0.24.5")
	assert.Equal(t, p.SPIRECRDsVersion, "0.5.0")
	assert.Equal(t, cluster.GetName(), p.cluster.GetName())
	assert.Equal(t, kubeConfig, p.settings.KubeConfig)
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
