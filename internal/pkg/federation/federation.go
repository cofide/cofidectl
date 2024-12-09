// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package federation

import (
	"fmt"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
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

func (fed *Federation) GetHelmConfig() (map[string]interface{}, error) {
	switch fed.destTrustZone.GetBundleEndpointProfile() {
	case trust_zone_proto.BundleEndpointProfile_BUNDLE_ENDPOINT_PROFILE_HTTPS_SPIFFE:
		return map[string]interface{}{
			"bundleEndpointURL": fed.destTrustZone.GetBundleEndpointUrl(),
			"bundleEndpointProfile": map[string]interface{}{
				"type":             bundleEndpointProfileHTTPSSPIFFE,
				"endpointSPIFFEID": fmt.Sprintf("spiffe://%s/spire/server", fed.destTrustZone.TrustDomain),
			},
			"trustDomain":       fed.destTrustZone.TrustDomain,
			"trustDomainBundle": fed.destTrustZone.GetBundle(),
		}, nil
	case trust_zone_proto.BundleEndpointProfile_BUNDLE_ENDPOINT_PROFILE_HTTPS_WEB:
		return map[string]interface{}{
			"bundleEndpointURL": fed.destTrustZone.GetBundleEndpointUrl(),
			"bundleEndpointProfile": map[string]interface{}{
				"type": bundleEndpointProfileHTTPSWeb,
			},
			"trustDomain": fed.destTrustZone.TrustDomain,
		}, nil
	default:
		return nil, fmt.Errorf("unexpected bundle endpoint profile %d", fed.destTrustZone.GetBundleEndpointProfile())
	}
}
