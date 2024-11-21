// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidator_ValidateValid(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{name: "empty", file: "empty.yaml"},
		{name: "defaults", file: "default.yaml"},
		{name: "full", file: "full.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := readTestConfig(t, tt.file)
			v := NewValidator()
			if err := v.Validate([]byte(data)); err != nil {
				t.Fatalf("Validator.Validate() error = %v", err)
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
			name:    "data source not a string",
			data:    "data_source: 123",
			wantErr: "data_source: conflicting values 123 and string",
		},
		{
			name:    "trust zones not a list",
			data:    "trust_zones: \"not-a-list\"",
			wantErr: "trust_zones: conflicting values \"not-a-list\" and [...#TrustZone]",
		},
		{
			name:    "attestation policies not a list",
			data:    "attestation_policies: \"not-a-list\"",
			wantErr: "attestation_policies: conflicting values \"not-a-list\" and [...#AttestationPolicy]",
		},
		{
			name:    "unexpected trust zone field",
			data:    "trust_zones: [foo: bar]",
			wantErr: "trust_zones.0.foo: field not allowed",
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
			name:    "missing attestation policy field",
			data:    string(readTestConfig(t, "missing_attestation_policy_field.yaml")),
			wantErr: "attestation_policies.0.kubernetes: field is required but not present",
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
