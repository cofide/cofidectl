// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTrustProviderKindFromProfile(t *testing.T) {
	tests := []struct {
		name      string
		profile   string
		want      string
		wantErr   bool
		errString string
	}{
		{
			name:    "valid kubernetes profile",
			profile: "kubernetes",
			want:    "kubernetes",
			wantErr: false,
		},
		{
			name:    "valid istio profile",
			profile: "istio",
			want:    "kubernetes",
			wantErr: false,
		},
		{
			name:      "invalid profile specified",
			profile:   "invalid",
			wantErr:   true,
			errString: "failed to get trust provider kind, an invalid profile was specified: invalid",
		},
		{
			name:      "invalid profile specified, Kubernetes",
			profile:   "Kubernetes",
			wantErr:   true,
			errString: "failed to get trust provider kind, an invalid profile was specified: Kubernetes",
		},
		{
			name:      "invalid profile specified, ISTIO",
			profile:   "ISTIO",
			wantErr:   true,
			errString: "failed to get trust provider kind, an invalid profile was specified: ISTIO",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := GetTrustProviderKindFromProfile(tt.profile)
			if tt.wantErr {
				assert.Equal(t, tt.errString, err.Error())
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, tt.want, resp)
		})
	}
}
