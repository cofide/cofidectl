// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package attestationpolicy

import (
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/google/go-cmp/cmp"
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

func selectorDiffOpts() []cmp.Option {
	return []cmp.Option{
		protocmp.Transform(),
		protocmp.SortRepeatedFields(&attestation_policy_proto.APLabelSelector{}, "match_expressions"),
		protocmp.SortRepeatedFields(&attestation_policy_proto.APMatchExpression{}, "values"),
	}
}
