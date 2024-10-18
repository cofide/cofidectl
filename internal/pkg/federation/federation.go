package federation

import (
	"fmt"

	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
)

type Federation struct {
	FromTrustDomain   string
	ToTrustDomain     string
	BundleEndpointURL string
	BootstrapBundle   string
}

func NewFederation(federation *federation_proto.Federation) *Federation {
	return &Federation{
		FromTrustDomain: federation.Left.TrustDomain,
		ToTrustDomain:   federation.Right.TrustDomain,
	}
}

func (fed *Federation) GetHelmConfig() map[string]interface{} {
	clusterFederatedTrustDomain := map[string]interface{}{
		"bundleEndpointURL": fmt.Sprintf("https://%s/bundle", fed.BundleEndpointURL),
		"bundleEndpointProfile": map[string]interface{}{
			"type":             "https_spiffe",
			"endpointSPIFFEID": fmt.Sprintf("spiffe://%s/spire/server", fed.ToTrustDomain),
		},
		"trustDomain":       fed.ToTrustDomain,
		"trustDomainBundle": fed.BootstrapBundle,
	}

	return clusterFederatedTrustDomain
}
