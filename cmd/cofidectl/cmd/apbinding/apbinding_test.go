// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package apbinding

import (
	"errors"
	"testing"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPBindingCommand_updateAPBinding(t *testing.T) {
	tests := []struct {
		name                     string
		trustZoneName            string
		attestationPolicyName    string
		flags                    map[string]string
		injectFailure            bool
		wantErr                  bool
		wantErrMessage           string
		nonExistentTrustZone     bool
		nonExistentPolicy        bool
		nonExistentBinding       bool
		wantCheck                func(t *testing.T, binding *ap_binding_proto.APBinding)
	}{
		{
			name:                  "update federates-with",
			trustZoneName:         "tz1",
			attestationPolicyName: "ap1",
			flags:                 map[string]string{"federates-with": "tz2"},
			wantCheck: func(t *testing.T, binding *ap_binding_proto.APBinding) {
				require.Len(t, binding.GetFederations(), 1)
				assert.Equal(t, "tz2-id", binding.GetFederations()[0].GetTrustZoneId())
			},
		},
		{
			name:                  "clear federations with --clear-federations",
			trustZoneName:         "tz1",
			attestationPolicyName: "ap1",
			flags:                 map[string]string{"clear-federations": "true"},
			wantCheck: func(t *testing.T, binding *ap_binding_proto.APBinding) {
				assert.Empty(t, binding.GetFederations())
			},
		},
		{
			name:                  "conflicting flags returns error",
			trustZoneName:         "tz1",
			attestationPolicyName: "ap1",
			flags:                 map[string]string{"federates-with": "tz2", "clear-federations": "true"},
			wantErr:               true,
			wantErrMessage:        "cannot simultaneously specify --federates-with and --clear-federations",
		},
		{
			name:                  "no flags set leaves binding unchanged",
			trustZoneName:         "tz1",
			attestationPolicyName: "ap1",
			flags:                 map[string]string{},
			wantCheck: func(t *testing.T, binding *ap_binding_proto.APBinding) {
				require.Len(t, binding.GetFederations(), 1)
				assert.Equal(t, "tz2-id", binding.GetFederations()[0].GetTrustZoneId())
			},
		},
		{
			name:                  "non-existent trust zone",
			trustZoneName:         "tz-missing",
			attestationPolicyName: "ap1",
			flags:                 map[string]string{"federates-with": "tz2"},
			wantErr:               true,
			wantErrMessage:        "failed to get trust zone tz-missing",
			nonExistentTrustZone:  true,
		},
		{
			name:                  "non-existent attestation policy",
			trustZoneName:         "tz1",
			attestationPolicyName: "ap-missing",
			flags:                 map[string]string{"federates-with": "tz2"},
			wantErr:               true,
			wantErrMessage:        "failed to get attestation policy ap-missing",
			nonExistentPolicy:     true,
		},
		{
			name:                  "non-existent binding",
			trustZoneName:         "tz1",
			attestationPolicyName: "ap2",
			flags:                 map[string]string{"federates-with": "tz2"},
			wantErr:               true,
			wantErrMessage:        "no binding found",
			nonExistentBinding:    true,
		},
		{
			name:                  "invalid federated trust zone",
			trustZoneName:         "tz1",
			attestationPolicyName: "ap1",
			flags:                 map[string]string{"federates-with": "tz-missing"},
			wantErr:               true,
			wantErrMessage:        "federated trust zone not found: tz-missing",
		},
		{
			name:                  "datastore failure",
			trustZoneName:         "tz1",
			attestationPolicyName: "ap1",
			flags:                 map[string]string{"federates-with": "tz2"},
			injectFailure:         true,
			wantErr:               true,
			wantErrMessage:        "fake update failure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := newFakeDataSource(t, defaultConfig())
			if tt.injectFailure {
				ds = &failingUpdateDS{LocalDataSource: ds.(*local.LocalDataSource)}
			}

			cmd := buildUpdateCmd(tt.flags)

			opts := updateOpts{
				trustZone:         tt.trustZoneName,
				attestationPolicy: tt.attestationPolicyName,
			}
			if val, ok := tt.flags["federates-with"]; ok {
				opts.federatesWith = []string{val}
				if val == "" {
					opts.federatesWith = []string{}
				}
			}
			if val, ok := tt.flags["clear-federations"]; ok && val == "true" {
				opts.clearFederations = true
			}

			c := APBindingCommand{}
			err := c.updateAPBinding(opts, cmd, ds)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrMessage)
			} else {
				require.NoError(t, err)
				tz, err := ds.GetTrustZoneByName(tt.trustZoneName)
				require.NoError(t, err)
				ap, err := ds.GetAttestationPolicyByName(tt.attestationPolicyName)
				require.NoError(t, err)
				bindings, err := ds.ListAPBindings(nil)
				require.NoError(t, err)
				var binding *ap_binding_proto.APBinding
				for _, b := range bindings {
					if b.GetTrustZoneId() == tz.GetId() && b.GetPolicyId() == ap.GetId() {
						binding = b
						break
					}
				}
				require.NotNil(t, binding)
				tt.wantCheck(t, binding)
			}
		})
	}
}

func buildUpdateCmd(flags map[string]string) *cobra.Command {
	opts := updateOpts{}
	cmd := &cobra.Command{}
	f := cmd.Flags()
	f.StringVar(&opts.trustZone, "trust-zone", "", "")
	f.StringVar(&opts.attestationPolicy, "attestation-policy", "", "")
	f.StringSliceVar(&opts.federatesWith, "federates-with", nil, "")
	f.BoolVar(&opts.clearFederations, "clear-federations", false, "")
	for name, val := range flags {
		cobra.CheckErr(cmd.Flags().Set(name, val))
	}
	return cmd
}

type failingUpdateDS struct {
	*local.LocalDataSource
}

// UpdateAPBinding fails unconditionally.
func (f *failingUpdateDS) UpdateAPBinding(_ *ap_binding_proto.APBinding) (*ap_binding_proto.APBinding, error) {
	return nil, errors.New("fake update failure")
}

func newFakeDataSource(t *testing.T, cfg *config.Config) datasource.DataSource {
	configLoader, err := config.NewMemoryLoader(cfg)
	require.Nil(t, err)
	lds, err := local.NewLocalDataSource(configLoader)
	require.Nil(t, err)
	return lds
}

func defaultConfig() *config.Config {
	return &config.Config{
		TrustZones: []*trust_zone_proto.TrustZone{
			fixtures.TrustZone("tz1"),
			fixtures.TrustZone("tz2"),
		},
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{
			fixtures.AttestationPolicy("ap1"),
			fixtures.AttestationPolicy("ap2"),
		},
		APBindings: []*ap_binding_proto.APBinding{
			fixtures.APBinding("apb1"),
		},
		Federations: []*federation_proto.Federation{
			fixtures.Federation("fed1"),
			fixtures.Federation("fed2"),
		},
		Plugins: fixtures.Plugins("plugins1"),
	}
}
