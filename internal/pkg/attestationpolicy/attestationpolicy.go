// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package attestationpolicy

import (
	"fmt"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	datasource_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
	"github.com/cofide/cofidectl/internal/pkg/trustzone"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	types "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
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
	case *attestation_policy_proto.AttestationPolicy_Static:
		trustZoneID := binding.GetTrustZoneId()

		clusters, err := source.ListClusters(&datasource_proto.ListClustersRequest_Filter{
			TrustZoneId: &trustZoneID,
		})
		if err != nil {
			return nil, err
		}

		if len(clusters) < 1 {
			return nil, trustzone.ErrNoClustersInTrustZone
		}

		if len(clusters) > 1 {
			return nil, trustzone.ErrOneClusterPerTrustZone
		}

		trustZone, err := source.GetTrustZone(trustZoneID)
		if err != nil {
			return nil, err
		}

		static := policy.Static
		selectors, err := formatSelectors(static.Selectors)
		if err != nil {
			return nil, err
		}

		clusterStaticEntry := map[string]any{
			"parentID":  fmt.Sprintf("spiffe://%s/cluster/%s/spire/agents", trustZone.GetTrustDomain(), clusters[0].GetName()),
			"spiffeID":  static.GetSpiffeId(),
			"selectors": selectors,
		}

		return clusterStaticEntry, nil
	default:
		return nil, fmt.Errorf("unexpected attestation policy kind: %T", policy)
	}

	if len(binding.Federations) > 0 {
		// Convert from trust zones to trust domains.
		federatesWith := []string{}
		for _, fed := range binding.Federations {
			if trustZone, err := source.GetTrustZone(fed.GetTrustZoneId()); err != nil {
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

func formatSelectors(selectors []*types.Selector) ([]string, error) {
	result := make([]string, 0, len(selectors))

	for _, selector := range selectors {
		if selector.Type == "" || selector.Value == "" {
			return nil, fmt.Errorf("invalid selector type=%q, value=%q", selector.Type, selector.Value)
		}

		result = append(result, fmt.Sprintf("%s:%s", selector.Type, selector.Value))
	}

	return result, nil
}
