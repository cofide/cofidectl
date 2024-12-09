// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"errors"
	"testing"

	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/provision_plugin/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestStatusBuilder_Ok(t *testing.T) {
	tests := []struct {
		name      string
		trustZone string
		cluster   string
		stage     string
		message   string
		want      *provisionpb.Status
	}{
		{
			name:      "trust zone and cluster",
			trustZone: "tz1",
			cluster:   "cluster1",
			stage:     "Testing",
			message:   "Fake message",
			want:      makeStatus("Testing", "Fake message for cluster1 in tz1", false, nil),
		},
		{
			name:      "no trust zone or cluster",
			trustZone: "",
			cluster:   "",
			stage:     "Testing",
			message:   "Fake message",
			want:      makeStatus("Testing", "Fake message", false, nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewStatusBuilder(tt.trustZone, tt.cluster)
			got := sb.Ok(tt.stage, tt.message)
			assert.EqualExportedValues(t, tt.want, got)
		})
	}
}

func TestStatusBuilder_Done(t *testing.T) {
	tests := []struct {
		name      string
		trustZone string
		cluster   string
		stage     string
		message   string
		want      *provisionpb.Status
	}{
		{
			name:      "trust zone and cluster",
			trustZone: "tz1",
			cluster:   "cluster1",
			stage:     "Testing",
			message:   "Fake message",
			want:      makeStatus("Testing", "Fake message for cluster1 in tz1", true, nil),
		},
		{
			name:      "no trust zone or cluster",
			trustZone: "",
			cluster:   "",
			stage:     "Testing",
			message:   "Fake message",
			want:      makeStatus("Testing", "Fake message", true, nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewStatusBuilder(tt.trustZone, tt.cluster)
			got := sb.Done(tt.stage, tt.message)
			assert.EqualExportedValues(t, tt.want, got)
		})
	}
}

func TestStatusBuilder_Error(t *testing.T) {
	tests := []struct {
		name      string
		trustZone string
		cluster   string
		stage     string
		message   string
		err       error
		want      *provisionpb.Status
	}{
		{
			name:      "trust zone and cluster",
			trustZone: "tz1",
			cluster:   "cluster1",
			stage:     "Testing",
			message:   "Fake message",
			err:       errors.New("Fake error"),
			want:      makeStatus("Testing", "Fake message for cluster1 in tz1", true, ptrOf("Fake error")),
		},
		{
			name:      "no trust zone or cluster",
			trustZone: "",
			cluster:   "",
			stage:     "Testing",
			message:   "Fake message",
			err:       errors.New("Fake error"),
			want:      makeStatus("Testing", "Fake message", true, ptrOf("Fake error")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewStatusBuilder(tt.trustZone, tt.cluster)
			got := sb.Error(tt.stage, tt.message, tt.err)
			assert.EqualExportedValues(t, tt.want, got)
		})
	}
}

func makeStatus(stage, message string, done bool, err *string) *provisionpb.Status {
	return &provisionpb.Status{Stage: &stage, Message: &message, Done: &done, Error: err}
}

func ptrOf[T any](x T) *T {
	return &x
}
