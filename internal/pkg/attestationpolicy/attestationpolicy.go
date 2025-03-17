// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package attestationpolicy

import (
	"fmt"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
)

type AttestationPolicy struct {
	AttestationPolicyProto *attestation_policy_proto.AttestationPolicy
}

const (
	Annotated   = "annotated"
	Cluster     = "cluster"
	Namespace   = "namespace"
	Unspecified = "unspecified"
)

func NewAttestationPolicy(attestationPolicy *attestation_policy_proto.AttestationPolicy) *AttestationPolicy {
	return &AttestationPolicy{
		AttestationPolicyProto: attestationPolicy,
	}
}

func (ap *AttestationPolicy) GetHelmConfig(source datasource.DataSource, binding *ap_binding_proto.APBinding) (map[string]any, error) {
	var clusterSPIFFEID = make(map[string]any)
	switch policy := ap.AttestationPolicyProto.Policy.(type) {
	case *attestation_policy_proto.AttestationPolicy_Kubernetes:
		kubernetes := policy.Kubernetes
		if kubernetes.NamespaceSelector != nil {
			selector := getAPLabelSelectorHelmConfig(kubernetes.NamespaceSelector)
			if selector != nil {
				clusterSPIFFEID["namespaceSelector"] = selector
			}
		}
		if kubernetes.PodSelector != nil {
			selector := getAPLabelSelectorHelmConfig(kubernetes.PodSelector)
			if selector != nil {
				clusterSPIFFEID["podSelector"] = selector
			}
		}
	default:
		return nil, fmt.Errorf("unexpected attestation policy kind: %T", policy)
	}

	// nolint:staticcheck
	if len(binding.FederatesWith) > 0 {
		// Convert from trust zones to trust domains.
		federatesWith := []string{}
		// nolint:staticcheck
		for _, tzName := range binding.FederatesWith {
			if trustZone, err := source.GetTrustZone(tzName); err != nil {
				return nil, err
			} else {
				federatesWith = append(federatesWith, trustZone.TrustDomain)
			}
		}
		clusterSPIFFEID["federatesWith"] = federatesWith
	}

	return clusterSPIFFEID, nil
}

func getAPLabelSelectorHelmConfig(selector *attestation_policy_proto.APLabelSelector) map[string]any {
	if len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0 {
		return nil
	}

	matchLabels := map[string]any{}
	for k, v := range selector.MatchLabels {
		matchLabels[k] = v
	}

	matchExpressions := []map[string]any{}
	for _, me := range selector.MatchExpressions {
		matchExpressions = append(matchExpressions, map[string]any{
			"key":      me.GetKey(),
			"operator": me.GetOperator(),
			"values":   me.GetValues(),
		})
	}

	return map[string]any{
		"matchLabels":      matchLabels,
		"matchExpressions": matchExpressions,
	}
}
