// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package federation

import (
	"context"
	"fmt"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	"github.com/cofide/cofidectl/pkg/spire"
)

const (
	bundleEndpointProfileHTTPSWeb    = "https_web"
	bundleEndpointProfileHTTPSSPIFFE = "https_spiffe"
)

type Federation struct {
	destTrustZone *trust_zone_proto.TrustZone
}

func NewFederation(trustZone *trust_zone_proto.TrustZone) *Federation {
	return &Federation{
		destTrustZone: trustZone,
	}
}

func (fed *Federation) GetHelmConfig() (map[string]any, error) {
	switch fed.destTrustZone.GetBundleEndpointProfile() {
	case trust_zone_proto.BundleEndpointProfile_BUNDLE_ENDPOINT_PROFILE_HTTPS_SPIFFE:
		client, err := kubeutil.NewKubeClientFromSpecifiedContext(".kube/config", fed.destTrustZone.GetKubernetesContext())
		if err != nil {
			return nil, err
		}

		ctx := context.Background()
		bundle, err := spire.GetBundle(ctx, client)
		if err != nil {
			return nil, fmt.Errorf("failed obtaining bundle: %w", err)
		}

		return map[string]any{
			"bundleEndpointURL": fed.destTrustZone.GetBundleEndpointUrl(),
			"bundleEndpointProfile": map[string]any{
				"type":             bundleEndpointProfileHTTPSSPIFFE,
				"endpointSPIFFEID": fmt.Sprintf("spiffe://%s/spire/server", fed.destTrustZone.TrustDomain),
			},
			"trustDomain":       fed.destTrustZone.TrustDomain,
			"trustDomainBundle": bundle,
		}, nil
	case trust_zone_proto.BundleEndpointProfile_BUNDLE_ENDPOINT_PROFILE_HTTPS_WEB:
		return map[string]any{
			"bundleEndpointURL": fed.destTrustZone.GetBundleEndpointUrl(),
			"bundleEndpointProfile": map[string]any{
				"type": bundleEndpointProfileHTTPSWeb,
			},
			"trustDomain": fed.destTrustZone.TrustDomain,
		}, nil
	default:
		return nil, fmt.Errorf("unexpected bundle endpoint profile %d", fed.destTrustZone.GetBundleEndpointProfile())
	}
}
