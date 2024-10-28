package config

import (
	_ "embed"
	"strings"
	"testing"
)

var emptyListsYAML string = `
plugins: []
trustzones: []
attestationpolicies: []
`

var fullConfigYAML string = `
plugins:
    - test-plugin
trustzones:
    - name: tz1
      trustdomain: td1
      kubernetescluster: local1
      kubernetescontext: kind-local1
      trustprovider:
        name: ""
        kind: kubernetes
      bundleendpointurl: 127.0.0.1
      bundle: ""
      federations:
        - left: tz1
          right: tz2
      attestationpolicies:
        - name: ap1
          kind: 2
          podkey: ""
          podvalue: ""
          namespace: ns1
    - name: tz2
      trustdomain: td2
      kubernetescluster: local2
      kubernetescontext: kind-local2
      trustprovider:
        name: ""
        kind: kubernetes
      bundleendpointurl: 127.0.0.2
      bundle: ""
      federations:
        - left: tz2
          right: tz1
      attestationpolicies:
        - name: ap2
          kind: 1
          podkey: foo
          podvalue: bar
          namespace: ""
attestationpolicies:
    - name: ap1
      kind: 2
      podkey: ""
      podvalue: ""
      namespace: ns1
    - name: ap2
      kind: 1
      podkey: foo
      podvalue: bar
      namespace: ""
`

func TestValidator_ValidateValid(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "empty", data: ""},
		{name: "empty lists", data: emptyListsYAML},
		{name: "basic", data: fullConfigYAML},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			if err := v.Validate([]byte(tt.data)); err != nil {
				t.Fatalf("Validator.Validate() error = %v", err)
			}
		})
	}
}

var missingTrustZoneField string = `
trustzones:
    - trustdomain: td1
      kubernetescluster: local1
      kubernetescontext: kind-local1
      trustprovider:
        name: ""
        kind: kubernetes
      bundleendpointurl: 127.0.0.1
      bundle: ""
      federations:
        - left: tz1
          right: tz2
      attestationpolicies: []
`

var missingAttestationPolicyField string = `
attestationpolicies:
    - name: ap1
      kind: 2
      podkey: ""
      podvalue: ""
`

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
			data:    missingTrustZoneField,
			wantErr: "trustzones.0.name: incomplete value string",
		},
		{
			name:    "missing attestation policy field",
			data:    missingAttestationPolicyField,
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
