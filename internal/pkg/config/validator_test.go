// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator_ValidateValid(t *testing.T) {
	tests := []struct {
		name           string
		file           string
		wantErr        bool
		wantErrMessage string
	}{
		{name: "empty", file: "empty.yaml", wantErr: true, wantErrMessage: "plugins: field is required but not present"},
		{name: "defaults", file: "default.yaml"},
		{name: "full", file: "full.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := readTestConfig(t, tt.file)
			v := NewValidator()
			err := v.Validate([]byte(data))
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidateInvalid(t *testing.T) {
	tests := []struct {
		name string
		data string
		// wantErr is a substring of the expected error message.
		wantErr string
	}{
		{
			name:    "unexpected top-level type",
			data:    "top-level-string",
			wantErr: "conflicting values \"top-level-string\" and ",
		},
		{
			name:    "unexpected top-level field",
			data:    "foo: bar",
			wantErr: "foo: field not allowed",
		},
		{
			name:    "trust zones not a list",
			data:    "trust_zones: \"not-a-list\"",
			wantErr: "trust_zones: conflicting values \"not-a-list\" and [...#TrustZone]",
		},
		{
			name:    "clusters not a list",
			data:    "clusters: \"not-a-list\"",
			wantErr: "clusters: conflicting values \"not-a-list\" and [...#Cluster]",
		},
		{
			name:    "attestation policies not a list",
			data:    "attestation_policies: \"not-a-list\"",
			wantErr: "attestation_policies: conflicting values \"not-a-list\" and [...#AttestationPolicy]",
		},
		{
			name:    "plugin config not a map",
			data:    "plugin_config: \"not-a-map\"",
			wantErr: "plugin_config: conflicting values \"not-a-map\" and {[string]:_}",
		},
		{
			name:    "unexpected trust zone field",
			data:    "trust_zones: [foo: bar]",
			wantErr: "trust_zones.0.foo: field not allowed",
		},
		{
			name:    "unexpected cluster field",
			data:    "clusters: [foo: bar]",
			wantErr: "clusters.0.foo: field not allowed",
		},
		{
			name:    "unexpected attestation policy field",
			data:    "attestation_policies: [foo: bar]",
			wantErr: "attestation_policies.0.foo: field not allowed",
		},
		{
			name:    "missing trust zone field",
			data:    string(readTestConfig(t, "missing_trust_zone_field.yaml")),
			wantErr: "trust_zones.0.name: field is required but not present",
		},
		{
			name:    "missing cluster field",
			data:    string(readTestConfig(t, "missing_cluster_field.yaml")),
			wantErr: "clusters.0.name: field is required but not present",
		},
		{
			name:    "missing attestation policy field",
			data:    string(readTestConfig(t, "missing_attestation_policy_field.yaml")),
			wantErr: "attestation_policies.0: incomplete value {name:\"ap1\",id?:string,kubernetes?:{namespace_selector?:~(#APLabelSelector),pod_selector?:~(#APLabelSelector),dns_name_templates?:[]}} | {name:\"ap1\",id?:string,static?:{spiffe_id!:string,selectors!:[]}}",
		},
		{
			name:    "plugins not a map",
			data:    "plugins: 123",
			wantErr: "plugins: conflicting values 123 and ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			if err := v.Validate([]byte(tt.data)); err == nil {
				t.Fatal("Validator.Validate() did not return error")
			} else {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
