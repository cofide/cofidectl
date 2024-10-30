package config

import (
	"strings"
	"testing"
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
			name:    "plugins not a list",
			data:    "plugins: \"not-a-list\"",
			wantErr: "plugins: conflicting values \"not-a-list\" and [...#Plugin]",
		},
		{
			name:    "trust zones not a list",
			data:    "trustzones: \"not-a-list\"",
			wantErr: "trustzones: conflicting values \"not-a-list\" and [...#TrustZone]",
		},
		{
			name:    "attestation policies not a list",
			data:    "attestationpolicies: \"not-a-list\"",
			wantErr: "attestationpolicies: conflicting values \"not-a-list\" and [...#AttestationPolicy]",
		},
		{
			name:    "unexpected trust zone field",
			data:    "trustzones: [foo: bar]",
			wantErr: "trustzones.0.foo: field not allowed",
		},
		{
			name:    "unexpected attestation policy field",
			data:    "attestationpolicies: [foo: bar]",
			wantErr: "attestationpolicies.0.foo: field not allowed",
		},
		{
			name:    "missing trust zone field",
			data:    string(readTestConfig(t, "missing_trust_zone_field.yaml")),
			wantErr: "trustzones.0.name: incomplete value string",
		},
		{
			name:    "missing attestation policy field",
			data:    string(readTestConfig(t, "missing_attestation_policy_field.yaml")),
			wantErr: "attestationpolicies.0.namespace: incomplete value string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			if err := v.Validate([]byte(tt.data)); err == nil {
				t.Fatal("Validator.Validate() did not return error")
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validator.Validate() error string = %v, wantErr %v", err.Error(), tt.wantErr)
			}
		})
	}
}
