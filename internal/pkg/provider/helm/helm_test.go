// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"testing"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestHelmSPIREProvider(t *testing.T) {
	trustZoneProto := &trust_zone_proto.TrustZone{TrustDomain: "foo.bar"}
	spireValues := map[string]interface{}{}
	spireCRDsValues := map[string]interface{}{}

	p := NewHelmSPIREProvider(trustZoneProto, spireValues, spireCRDsValues)
	assert.Equal(t, p.SPIREVersion, "0.21.0")
	assert.Equal(t, p.SPIRECRDsVersion, "0.4.0")
	assert.Equal(t, trustZoneProto.TrustDomain, p.trustZone.TrustDomain)
}
