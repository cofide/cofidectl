// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package attestationpolicy

import (
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/google/go-cmp/cmp"
	types "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
)

func Test_formatLabelSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector *attestation_policy_proto.APLabelSelector
		want     string
	}{
		{
			name:     "namespace",
			selector: fixtures.AttestationPolicy("ap1").GetKubernetes().NamespaceSelector,
			want:     "kubernetes.io/metadata.name=ns1",
		},
		{
			name:     "pod label in",
			selector: fixtures.AttestationPolicy("ap2").GetKubernetes().PodSelector,
			want:     "foo in (bar)",
		},
		{
			name:     "all the selectors",
			selector: fixtures.AttestationPolicy("ap3").GetKubernetes().PodSelector,
			want:     "bar,!baz,foo in (bar,baz),foo notin (quux,qux),label1=value1,label2=value2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLabelSelector(tt.selector)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_parseLabelSelector(t *testing.T) {
	tests := []struct {
		name          string
		selector      string
		want          *attestation_policy_proto.APLabelSelector
		wantErr       bool
		wantErrString string
	}{
		{
			name:     "pod label in",
			selector: "foo in (bar)",
			want:     fixtures.AttestationPolicy("ap2").GetKubernetes().PodSelector,
			wantErr:  false,
		},
		{
			name:     "all the selectors",
			selector: "bar,!baz,foo in (bar,baz),foo notin (quux,qux),label1=value1,label2=value2",
			want:     fixtures.AttestationPolicy("ap3").GetKubernetes().PodSelector,
			wantErr:  false,
		},
		{
			name:          "invalid",
			selector:      "not a valid selector",
			wantErr:       true,
			wantErrString: "--pod-label argument \"not a valid selector\" invalid: couldn't parse the selector string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLabelSelector(tt.selector)
			if !tt.wantErr {
				require.Nil(t, err, "unexpected error")
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			}
			if diff := cmp.Diff(tt.want, got, selectorDiffOpts()...); diff != "" {
				t.Errorf("parseLabelSelector() mismatch (-want,+got):\n%s", diff)
			}
		})
	}
}

func Test_formatSelectors(t *testing.T) {
	tests := []struct {
		name          string
		selectors     []*types.Selector
		want          string
		wantErr       bool
		wantErrString string
	}{
		{
			name: "valid selector",
			selectors: []*types.Selector{
				{
					Type:  "k8s",
					Value: "ns:foo",
				},
			},
			want: "k8s:ns:foo",
		},
		{
			name: "multiple selectors",
			selectors: []*types.Selector{
				{
					Type:  "k8s",
					Value: "ns:foo",
				},
				{
					Type:  "k8s",
					Value: "ns:bar",
				},
			},
			want: "k8s:ns:foo,k8s:ns:bar",
		},
		{
			name: "multiple selectors with different types",
			selectors: []*types.Selector{
				{
					Type:  "k8s",
					Value: "ns:foo",
				},
				{
					Type:  "k8s_psat",
					Value: "cluster:bar",
				},
			},
			want: "k8s:ns:foo,k8s_psat:cluster:bar",
		},
		{
			name:          "no selectors",
			selectors:     []*types.Selector{},
			want:          "",
			wantErr:       true,
			wantErrString: "no selectors provided",
		},
		{
			name: "selector with empty type",
			selectors: []*types.Selector{
				{
					Type:  "",
					Value: "ns:foo",
				},
			},
			want:          "",
			wantErr:       true,
			wantErrString: "invalid selector type=\"\", value=\"ns:foo\"",
		},
		{
			name: "selector with empty value",
			selectors: []*types.Selector{
				{
					Type:  "k8s",
					Value: "",
				},
			},
			want:          "",
			wantErr:       true,
			wantErrString: "invalid selector type=\"k8s\", value=\"\"",
		},
		{
			name:          "nil selector list",
			selectors:     nil,
			want:          "",
			wantErr:       true,
			wantErrString: "no selectors provided",
		},
		{
			name: "mixed valid and empty selectors",
			selectors: []*types.Selector{
				{
					Type:  "k8s",
					Value: "ns:foo",
				},
				{
					Type:  "",
					Value: "",
				},
				{
					Type:  "k8s",
					Value: "ns:bar",
				},
			},
			want:          "",
			wantErr:       true,
			wantErrString: "invalid selector type=\"\", value=\"\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatSelectors(tt.selectors)
			if !tt.wantErr {
				require.Nil(t, err, "unexpected error")
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_parseSelectors(t *testing.T) {
	tests := []struct {
		name            string
		selectorStrings []string
		want            []*types.Selector
		wantErr         bool
		wantErrString   string
	}{
		{
			name:            "valid selector",
			selectorStrings: []string{"k8s:ns:foo"},
			want: []*types.Selector{
				{
					Type:  "k8s",
					Value: "ns:foo",
				},
			},
			wantErr: false,
		},
		{
			name:            "multiple selectors",
			selectorStrings: []string{"k8s:ns:foo", "k8s:ns:bar"},
			want: []*types.Selector{
				{
					Type:  "k8s",
					Value: "ns:foo",
				},
				{
					Type:  "k8s",
					Value: "ns:bar",
				},
			},
			wantErr: false,
		},
		{
			name:            "invalid selector format - too many colons",
			selectorStrings: []string{"k8s:ns:foo:bar:baz"},
			wantErr:         true,
			wantErrString:   "invalid selector format \"k8s:ns:foo:bar:baz\", too many ':' characters, expected 'type:key:value'",
		},
		{
			name:            "invalid selector format - too few parts",
			selectorStrings: []string{"k8s:ns"},
			wantErr:         true,
			wantErrString:   "invalid selector format \"k8s:ns\", expected 'type:key:value'",
		},
		{
			name:            "invalid selector format - empty type",
			selectorStrings: []string{":ns:foo"},
			wantErr:         true,
			wantErrString:   "invalid selector format, type is empty: \":ns:foo\"",
		},
		{
			name:            "invalid selector format - empty key",
			selectorStrings: []string{"k8s::foo"},
			wantErr:         true,
			wantErrString:   "invalid selector format, key is empty: \"k8s::foo\"",
		},
		{
			name:            "invalid selector format - empty value",
			selectorStrings: []string{"k8s:ns:"},
			wantErr:         true,
			wantErrString:   "invalid selector format, value is empty: \"k8s:ns:\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSelectors(tt.selectorStrings)
			if !tt.wantErr {
				require.Nil(t, err, "unexpected error")
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrString)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func selectorDiffOpts() []cmp.Option {
	return []cmp.Option{
		protocmp.Transform(),
		protocmp.SortRepeatedFields(&attestation_policy_proto.APLabelSelector{}, "match_expressions"),
		protocmp.SortRepeatedFields(&attestation_policy_proto.APMatchExpression{}, "values"),
	}
}
