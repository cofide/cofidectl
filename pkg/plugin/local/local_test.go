// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"slices"
	"testing"

	"github.com/cofide/cofidectl/pkg/plugin"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
)

func TestLocalDataSource_ImplementsDataSource(t *testing.T) {
	local := LocalDataSource{}
	var _ plugin.DataSource = &local
}

func TestNewLocalDataSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		config     *config.Config
		wantConfig *config.Config
		wantErr    bool
	}{
		{
			name:       "non-existent config",
			config:     nil,
			wantConfig: nil,
			wantErr:    true,
		},
		{
			name:       "default config",
			config:     config.NewConfig(),
			wantConfig: config.NewConfig(),
		},
		{
			name: "non-default config",
			config: &config.Config{
				DataSource:      "test-plugin",
				ProvisionPlugin: "test-provision-plugin",
			},
			wantConfig: &config.Config{
				DataSource:          "test-plugin",
				TrustZones:          []*trust_zone_proto.TrustZone{},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
				PluginConfig:        map[string]*structpb.Struct{},
				ProvisionPlugin:     "test-provision-plugin",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := config.NewMemoryLoader(tt.config)
			require.Nil(t, err)

			got, err := NewLocalDataSource(loader)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				want := &LocalDataSource{
					loader: loader,
					config: tt.wantConfig,
				}
				assert.Equal(t, want, got)
			}
		})
	}
}

func TestLocalDataSource_Validate(t *testing.T) {
	lds, _ := buildLocalDataSource(t, config.NewConfig())

	err := lds.Validate()
	require.Nil(t, err)
}

func TestLocalDataSource_AddTrustZone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		config        *config.Config
		trustZone     *trust_zone_proto.TrustZone
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "success",
			config:    config.NewConfig(),
			trustZone: fixtures.TrustZone("tz1"),
			wantErr:   false,
		},
		{
			name: "duplicate",
			config: &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
				},
			},
			trustZone:     fixtures.TrustZone("tz1"),
			wantErr:       true,
			wantErrString: "trust zone tz1 already exists in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lds, loader := buildLocalDataSource(t, tt.config)

			got, err := lds.AddTrustZone(tt.trustZone)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				assert.EqualExportedValues(t, tt.trustZone, got)
				assert.False(t, slices.Contains(lds.config.TrustZones, tt.trustZone), "Pointer to trust zone stored in config")
				// Check that the trust zone was persisted.
				gotConfig := readConfig(t, loader)
				gotTrustZone, ok := gotConfig.GetTrustZoneByName(tt.trustZone.Name)
				assert.True(t, ok)
				assert.EqualExportedValues(t, tt.trustZone, gotTrustZone)
			}
		})
	}
}

func TestLocalDataSource_GetTrustZone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		trustZone     string
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "success",
			trustZone: "tz1",
			wantErr:   false,
		},
		{
			name:          "non-existent",
			trustZone:     "tz2",
			wantErr:       true,
			wantErrString: "failed to find trust zone tz2 in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
				},
			}
			lds, _ := buildLocalDataSource(t, cfg)

			got, err := lds.GetTrustZone(tt.trustZone)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, cfg.TrustZones[0], got)
				assert.False(t, slices.Contains(lds.config.TrustZones, got), "Pointer to trust zone in config returned")
			}
		})
	}
}

func TestLocalDataSource_UpdateTrustZone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		trustZone     *trust_zone_proto.TrustZone
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "no changes",
			trustZone: fixtures.TrustZone("tz1"),
			wantErr:   false,
		},
		{
			name: "allowed changes",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.Bundle = fixtures.StringPtr("new bundle")
				tz.BundleEndpointUrl = fixtures.StringPtr("http://new.bundle")
				tz.KubernetesCluster = fixtures.StringPtr("new-cluster")
				tz.KubernetesContext = fixtures.StringPtr("new-context")
				return tz
			}(),
			wantErr: false,
		},
		{
			name:          "non-existent",
			trustZone:     &trust_zone_proto.TrustZone{Name: "tz2"},
			wantErr:       true,
			wantErrString: "failed to find trust zone tz2 in local config",
		},
		{
			name: "disallowed trust domain",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.TrustDomain = "new.domain"
				return tz
			}(),
			wantErr:       true,
			wantErrString: "cannot update trust domain for existing trust zone tz1",
		},
		{
			name: "disallowed nil trust provider",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.TrustProvider = nil
				return tz
			}(),
			wantErr:       true,
			wantErrString: "cannot remove trust provider for trust zone tz1",
		},
		{
			name: "disallowed trust provider kind",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.TrustProvider.Kind = fixtures.StringPtr("invalid")
				return tz
			}(),
			wantErr:       true,
			wantErrString: "cannot update trust provider kind for existing trust zone tz1",
		},
		{
			name: "disallowed federation",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.AttestationPolicies = []*ap_binding_proto.APBinding{
					{TrustZone: "tz1", Policy: "ap2"},
				}
				return tz
			}(),
			wantErr:       true,
			wantErrString: "cannot update attestation policies for existing trust zone tz1",
		},
		{
			name: "disallowed attestation policy",
			trustZone: func() *trust_zone_proto.TrustZone {
				tz := fixtures.TrustZone("tz1")
				tz.Federations = []*federation_proto.Federation{
					{From: "tz1", To: "tz3"},
				}
				return tz
			}(),
			wantErr:       true,
			wantErrString: "cannot update federations for existing trust zone tz1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
				},
			}
			lds, loader := buildLocalDataSource(t, cfg)

			err := lds.UpdateTrustZone(tt.trustZone)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, tt.trustZone, lds.config.TrustZones[0])
				assert.False(t, slices.Contains(lds.config.TrustZones, tt.trustZone), "Pointer to trust zone stored in config")
				// Check that the trust zone was persisted.
				gotConfig := readConfig(t, loader)
				gotTrustZone, ok := gotConfig.GetTrustZoneByName(tt.trustZone.Name)
				assert.True(t, ok)
				assert.EqualExportedValues(t, tt.trustZone, gotTrustZone)
			}
		})
	}
}

func TestLocalDataSource_ListTrustZones(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name:    "none",
			config:  config.NewConfig(),
			wantErr: false,
		},
		{
			name: "two",
			config: &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lds, _ := buildLocalDataSource(t, tt.config)
			got, err := lds.ListTrustZones()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				want := tt.config.TrustZones
				if diff := cmp.Diff(got, want, protocmp.Transform()); diff != "" {
					t.Errorf("LocalDataSource.ListTrustZones() mismatch (-want,+got):\n%s", diff)
				}
				for _, gotTrustZone := range got {
					assert.False(t, slices.Contains(lds.config.TrustZones, gotTrustZone), "Pointer to trust zone in config returned")
				}
			}
		})
	}
}

func TestLocalDataSource_AddAttestationPolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		config        *config.Config
		policy        *attestation_policy_proto.AttestationPolicy
		wantErr       bool
		wantErrString string
	}{
		{
			name:    "success",
			config:  config.NewConfig(),
			policy:  fixtures.AttestationPolicy("ap1"),
			wantErr: false,
		},
		{
			name: "duplicate",
			config: &config.Config{
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
				},
			},
			policy:        fixtures.AttestationPolicy("ap1"),
			wantErr:       true,
			wantErrString: "attestation policy ap1 already exists in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lds, loader := buildLocalDataSource(t, tt.config)

			got, err := lds.AddAttestationPolicy(tt.policy)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				assert.EqualExportedValues(t, tt.policy, got)
				assert.False(t, slices.Contains(lds.config.AttestationPolicies, tt.policy), "Pointer to trust zone stored in config")
				// Check that the policy was persisted.
				gotConfig := readConfig(t, loader)
				gotPolicy, ok := gotConfig.GetAttestationPolicyByName(tt.policy.Name)
				assert.True(t, ok)
				assert.EqualExportedValues(t, tt.policy, gotPolicy)
			}
		})
	}
}

func TestLocalDataSource_GetAttestationPolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		policy        string
		wantErr       bool
		wantErrString string
	}{
		{
			name:    "success",
			policy:  "ap1",
			wantErr: false,
		},
		{
			name:          "non-existent",
			policy:        "ap2",
			wantErr:       true,
			wantErrString: "failed to find attestation policy ap2 in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
				},
			}
			lds, _ := buildLocalDataSource(t, cfg)

			got, err := lds.GetAttestationPolicy(tt.policy)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, cfg.AttestationPolicies[0], got)
				assert.False(t, slices.Contains(lds.config.AttestationPolicies, got), "Pointer to trust zone in config returned")
			}
		})
	}
}

func TestLocalDataSource_ListAttestationPolicies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name:    "none",
			config:  config.NewConfig(),
			wantErr: false,
		},
		{
			name: "two",
			config: &config.Config{
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
					fixtures.AttestationPolicy("ap2"),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lds, _ := buildLocalDataSource(t, tt.config)
			got, err := lds.ListAttestationPolicies()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				want := tt.config.AttestationPolicies
				if diff := cmp.Diff(got, want, protocmp.Transform()); diff != "" {
					t.Errorf("LocalDataSource.ListAttestationPolicies() mismatch (-want,+got):\n%s", diff)
				}
				for _, gotPolicy := range got {
					assert.False(t, slices.Contains(lds.config.AttestationPolicies, gotPolicy), "Pointer to attestation policy in config returned")
				}
			}
		})
	}
}

func TestLocalDataSource_AddAPBinding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		binding       *ap_binding_proto.APBinding
		wantErr       bool
		wantErrString string
	}{
		{
			name: "success",
			binding: &ap_binding_proto.APBinding{
				TrustZone: "tz1",
				Policy:    "ap2",
			},
			wantErr: false,
		},
		{
			name: "federates with",
			binding: &ap_binding_proto.APBinding{
				TrustZone:     "tz1",
				Policy:        "ap2",
				FederatesWith: []string{"tz2"},
			},
			wantErr: false,
		},
		{
			name: "invalid trust zone",
			binding: &ap_binding_proto.APBinding{
				TrustZone: "invalid",
				Policy:    "ap2",
			},
			wantErr:       true,
			wantErrString: "failed to find trust zone invalid in local config",
		},
		{
			name: "invalid policy",
			binding: &ap_binding_proto.APBinding{
				TrustZone: "tz1",
				Policy:    "invalid",
			},
			wantErr:       true,
			wantErrString: "failed to find attestation policy invalid in local config",
		},
		{
			name: "federates with self",
			binding: &ap_binding_proto.APBinding{
				TrustZone:     "tz1",
				Policy:        "ap2",
				FederatesWith: []string{"tz1"},
			},
			wantErr:       true,
			wantErrString: "attestation policy ap2 federates with its own trust zone tz1",
		},
		{
			name: "federates with invalid tz",
			binding: &ap_binding_proto.APBinding{
				TrustZone:     "tz1",
				Policy:        "ap2",
				FederatesWith: []string{"invalid"},
			},
			wantErr:       true,
			wantErrString: "attestation policy ap2 federates with unknown trust zone invalid",
		},
		{
			name: "federates with unfederated tz",
			binding: &ap_binding_proto.APBinding{
				TrustZone:     "tz1",
				Policy:        "ap2",
				FederatesWith: []string{"tz3"},
			},
			wantErr:       true,
			wantErrString: "attestation policy ap2 federates with tz3 but trust zone tz1 does not",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
					fixtures.TrustZone("tz3"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
					fixtures.AttestationPolicy("ap2"),
				},
			}
			lds, loader := buildLocalDataSource(t, cfg)
			got, err := lds.AddAPBinding(tt.binding)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, tt.binding, got)
				assert.False(t, slices.Contains(lds.config.TrustZones[0].AttestationPolicies, tt.binding), "Pointer to attestation policy binding stored in config")
				// Check that the binding was persisted.
				gotConfig := readConfig(t, loader)
				gotBinding := gotConfig.TrustZones[0].AttestationPolicies[1]
				assert.EqualExportedValues(t, tt.binding, gotBinding)
			}
		})
	}
}

func TestLocalDataSource_DestroyAPBinding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		binding       *ap_binding_proto.APBinding
		wantErr       bool
		wantErrString string
	}{
		{
			name: "success",
			binding: &ap_binding_proto.APBinding{
				TrustZone: "tz1",
				Policy:    "ap1",
			},
			wantErr: false,
		},
		{
			name: "invalid trust zone",
			binding: &ap_binding_proto.APBinding{
				TrustZone: "invalid",
				Policy:    "ap1",
			},
			wantErr:       true,
			wantErrString: "failed to find trust zone invalid in local config",
		},
		{
			name: "invalid policy",
			binding: &ap_binding_proto.APBinding{
				TrustZone: "tz1",
				Policy:    "invalid",
			},
			wantErr:       true,
			wantErrString: "failed to find attestation policy binding for invalid in trust zone tz1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
				},
				AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
					fixtures.AttestationPolicy("ap1"),
				},
			}
			lds, loader := buildLocalDataSource(t, cfg)
			err := lds.DestroyAPBinding(tt.binding)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.NotContains(t, lds.config.TrustZones[0].AttestationPolicies, tt.binding)
				// Check that the binding removal was persisted.
				gotConfig := readConfig(t, loader)
				assert.NotContains(t, gotConfig.TrustZones[0].AttestationPolicies, tt.binding)
			}
		})
	}
}

func TestLocalDataSource_ListAPBindingsByTrustZone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		trustZone     string
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "none",
			trustZone: "tz3",
			wantErr:   false,
		},
		{
			name:      "two",
			trustZone: "tz1",
			wantErr:   false,
		},
		{
			name:          "invalid trust zone",
			trustZone:     "invalid",
			wantErr:       true,
			wantErrString: "failed to find trust zone invalid in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz3"),
				},
			}
			lds, _ := buildLocalDataSource(t, cfg)
			got, err := lds.ListAPBindingsByTrustZone(tt.trustZone)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				want, ok := cfg.GetTrustZoneByName(tt.trustZone)
				require.True(t, ok)
				if diff := cmp.Diff(got, want.AttestationPolicies, protocmp.Transform()); diff != "" {
					t.Errorf("LocalDataSource.ListAPBindingsByTrustZone() mismatch (-want,+got):\n%s", diff)
				}
				for _, gotBinding := range got {
					assert.False(t, slices.Contains(lds.config.TrustZones[0].AttestationPolicies, gotBinding), "Pointer to attestation policy binding in config returned")
				}
			}
		})
	}
}

func TestLocalDataSource_AddFederation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		federation    *federation_proto.Federation
		wantErr       bool
		wantErrString string
	}{
		{
			name: "success",
			federation: &federation_proto.Federation{
				From: "tz1",
				To:   "tz3",
			},
			wantErr: false,
		},
		{
			name: "invalid from trust zone",
			federation: &federation_proto.Federation{
				From: "invalid",
				To:   "tz2",
			},
			wantErr:       true,
			wantErrString: "failed to find trust zone invalid in local config",
		},
		{
			name: "invalid to trust zone",
			federation: &federation_proto.Federation{
				From: "tz1",
				To:   "invalid",
			},
			wantErr:       true,
			wantErrString: "failed to find trust zone invalid in local config",
		},
		{
			name: "federate with self",
			federation: &federation_proto.Federation{
				From: "tz1",
				To:   "tz1",
			},
			wantErr:       true,
			wantErrString: "cannot federate trust zone tz1 with itself",
		},
		{
			name: "duplicate",
			federation: &federation_proto.Federation{
				From: "tz1",
				To:   "tz2",
			},
			wantErr:       true,
			wantErrString: "federation already exists between tz1 and tz2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
					fixtures.TrustZone("tz3"),
				},
			}
			lds, loader := buildLocalDataSource(t, cfg)
			got, err := lds.AddFederation(tt.federation)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				assert.EqualExportedValues(t, tt.federation, got)
				assert.False(t, slices.Contains(lds.config.TrustZones[0].Federations, tt.federation), "Pointer to federation stored in config")
				// Check that the federation was persisted.
				gotConfig := readConfig(t, loader)
				gotFederation := gotConfig.TrustZones[0].Federations[1]
				assert.EqualExportedValues(t, tt.federation, gotFederation)
			}
		})
	}
}

func TestLocalDataSource_ListFederations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name:    "none",
			config:  config.NewConfig(),
			wantErr: false,
		},
		{
			name: "two",
			config: &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz2"),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lds, _ := buildLocalDataSource(t, tt.config)
			got, err := lds.ListFederations()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				want := []*federation_proto.Federation{}
				for _, tz := range tt.config.TrustZones {
					want = append(want, tz.Federations...)
				}
				if diff := cmp.Diff(got, want, protocmp.Transform()); diff != "" {
					t.Errorf("LocalDataSource.ListFederations() mismatch (-want,+got):\n%s", diff)
				}
				for _, gotFederation := range got {
					for _, tz := range tt.config.TrustZones {
						assert.False(t, slices.Contains(tz.Federations, gotFederation), "Pointer to federation in config returned")
					}
				}
			}
		})
	}
}

func TestLocalDataSource_ListFederationsByTrustZone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		trustZone     string
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "none",
			trustZone: "tz3",
			wantErr:   false,
		},
		{
			name:      "two",
			trustZone: "tz1",
			wantErr:   false,
		},
		{
			name:          "invalid trust zone",
			trustZone:     "invalid",
			wantErr:       true,
			wantErrString: "failed to find trust zone invalid in local config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TrustZones: []*trust_zone_proto.TrustZone{
					fixtures.TrustZone("tz1"),
					fixtures.TrustZone("tz3"),
				},
			}
			lds, _ := buildLocalDataSource(t, cfg)
			got, err := lds.ListFederationsByTrustZone(tt.trustZone)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			} else {
				require.Nil(t, err)
				want, ok := cfg.GetTrustZoneByName(tt.trustZone)
				require.True(t, ok)
				if diff := cmp.Diff(got, want.Federations, protocmp.Transform()); diff != "" {
					t.Errorf("LocalDataSource.ListFederationsByTrustZone() mismatch (-want,+got):\n%s", diff)
				}
				for _, gotFederation := range got {
					assert.False(t, slices.Contains(want.Federations, gotFederation), "Pointer to attestation policy binding in config returned")
				}
			}
		})
	}
}

func buildLocalDataSource(t *testing.T, cfg *config.Config) (*LocalDataSource, *config.MemoryLoader) {
	loader, err := config.NewMemoryLoader(cfg)
	require.Nil(t, err)

	lds, err := NewLocalDataSource(loader)
	require.Nil(t, err)
	return lds, loader
}

func readConfig(t *testing.T, loader config.Loader) *config.Config {
	config, err := loader.Read()
	require.Nil(t, err)
	return config
}
