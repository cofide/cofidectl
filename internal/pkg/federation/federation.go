package federation

import (
	"fmt"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
)

type Federation struct {
	destTrustZone *trust_zone_proto.TrustZone
}

func NewFederation(trustZone *trust_zone_proto.TrustZone) *Federation {
	return &Federation{
		destTrustZone: trustZone,
	}
}

func (fed *Federation) GetHelmConfig() map[string]interface{} {
	clusterFederatedTrustDomain := map[string]interface{}{
		"bundleEndpointURL": fed.destTrustZone.BundleEndpointUrl,
		"bundleEndpointProfile": map[string]interface{}{
			"type":             "https_spiffe",
			"endpointSPIFFEID": fmt.Sprintf("spiffe://%s/spire/server", fed.destTrustZone.TrustDomain),
		},
		"trustDomain":       fed.destTrustZone.TrustDomain,
		"trustDomainBundle": fed.destTrustZone.Bundle,
	}

	return clusterFederatedTrustDomain
}
