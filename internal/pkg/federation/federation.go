package federation

import (
	"fmt"

	"github.com/cofide/cofidectl/internal/pkg/trustzone"
)

type Federation struct {
	destTrustZone *trustzone.TrustZone
}

func NewFederation(trustZone *trustzone.TrustZone) *Federation {
	return &Federation{
		destTrustZone: trustZone,
	}
}

func (fed *Federation) GetHelmConfig() map[string]interface{} {
	clusterFederatedTrustDomain := map[string]interface{}{
		"bundleEndpointURL": fmt.Sprintf("https://%s:8443", fed.destTrustZone.TrustZoneProto.BundleEndpointUrl),
		"bundleEndpointProfile": map[string]interface{}{
			"type":             "https_spiffe",
			"endpointSPIFFEID": fmt.Sprintf("spiffe://%s/spire/server", fed.destTrustZone.TrustZoneProto.TrustDomain),
		},
		"trustDomain":       fed.destTrustZone.TrustZoneProto.TrustDomain,
		"trustDomainBundle": fed.destTrustZone.TrustZoneProto.Bundle,
	}

	return clusterFederatedTrustDomain
}
