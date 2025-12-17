// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package attestationpolicy

import (
	"fmt"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	types "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
)

// spiffeTDIDTemplate is a Go template for a SPIFFE trust domain.
const spiffeTDIDTemplate = "spiffe://{{ .TrustDomain }}/"

// MakeClusterSPIFFEID returns a Helm value for a Kubernetes
func MakeClusterSPIFFEID(
	kubernetes *attestation_policy_proto.APKubernetes,
	source datasource.DataSource,
	binding *ap_binding_proto.APBinding,
) (map[string]any, error) {
	var clusterSPIFFEID = make(map[string]any)
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
	if kubernetes.GetSpiffeIdPathTemplate() != "" {
		clusterSPIFFEID["spiffeIDTemplate"] = spiffeTDIDTemplate + kubernetes.GetSpiffeIdPathTemplate()
	}
	if kubernetes.DnsNameTemplates != nil {
		dnsNameTemplates := getAPDNSNameTemplatesHelmConfig(kubernetes.DnsNameTemplates)
		if dnsNameTemplates != nil {
			clusterSPIFFEID["dnsNameTemplates"] = dnsNameTemplates
		}
	}
	federatesWith, err := makeFederatesWith(binding, source)
	if err != nil {
		return nil, err
	}
	if len(federatesWith) > 0 {
		clusterSPIFFEID["federatesWith"] = federatesWith
	}
	return clusterSPIFFEID, nil
}

func MakeClusterStaticEntry(
	static *attestation_policy_proto.APStatic,
	source datasource.DataSource,
	binding *ap_binding_proto.APBinding,
) (map[string]any, error) {
	selectors, err := formatSelectors(static.Selectors)
	if err != nil {
		return nil, err
	}
	trustZone, err := source.GetTrustZone(binding.GetTrustZoneId())
	if err != nil {
		return nil, err
	}
	spiffeID, err := renderSPIFFEID(trustZone.GetTrustDomain(), static.GetSpiffeIdPath())
	if err != nil {
		return nil, err
	}
	parentID, err := renderSPIFFEID(trustZone.GetTrustDomain(), static.GetParentIdPath())
	if err != nil {
		return nil, err
	}
	clusterStaticEntry := map[string]any{
		"spiffeID":  spiffeID,
		"parentID":  parentID,
		"selectors": selectors,
	}
	if len(static.GetDnsNames()) > 0 {
		clusterStaticEntry["dnsNames"] = static.GetDnsNames()
	}
	federatesWith, err := makeFederatesWith(binding, source)
	if err != nil {
		return nil, err
	}
	if len(federatesWith) > 0 {
		clusterStaticEntry["federatesWith"] = federatesWith
	}
	return clusterStaticEntry, nil
}

func makeFederatesWith(binding *ap_binding_proto.APBinding, source datasource.DataSource) ([]string, error) {
	// Convert from trust zones to trust domains.
	var federatesWith []string
	for _, fed := range binding.Federations {
		trustZone, err := source.GetTrustZone(fed.GetTrustZoneId())
		if err != nil {
			return nil, err
		}
		federatesWith = append(federatesWith, trustZone.TrustDomain)
	}
	return federatesWith, nil
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

func getAPDNSNameTemplatesHelmConfig(dnsNameTemplates []string) []string {
	// TODO: Consider validation of each dnsNameTemplate entry
	// here before adding to collection injected into Helm config
	if len(dnsNameTemplates) == 0 {
		return nil
	}
	return dnsNameTemplates
}

func renderSPIFFEID(trustDomain, path string) (string, error) {
	td, err := spiffeid.TrustDomainFromString(trustDomain)
	if err != nil {
		return "", err
	}
	spiffeID, err := spiffeid.FromPath(td, "/"+path)
	if err != nil {
		return "", err
	}
	return spiffeID.String(), nil
}
