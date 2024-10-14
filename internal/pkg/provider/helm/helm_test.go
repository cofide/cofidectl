package helm

import (
	"testing"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/stretchr/testify/assert"
)

func TestHelmSPIREProvider(t *testing.T) {
	trustZone := &trust_zone_proto.TrustZone{TrustDomain: "foo.bar"}
	spireValues := map[string]interface{}{}
	spireCRDsValues := map[string]interface{}{}

	p := NewHelmSPIREProvider(trustZone, spireValues, spireCRDsValues)
	assert.Equal(t, p.SPIREVersion, "0.21.0")
	assert.Equal(t, p.SPIRECRDsVersion, "0.4.0")
	assert.Equal(t, trustZone.TrustDomain, p.trustZone.TrustDomain)
}
