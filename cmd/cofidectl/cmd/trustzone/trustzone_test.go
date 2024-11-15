// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package trustzone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateOpts(t *testing.T) {
	// https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE-ID.md#21-trust-domain
	tt := []struct {
		name        string
		domain      string
		errExpected bool
	}{
		{domain: "example.com", errExpected: false},
		{domain: "example-domain.com", errExpected: false},
		{domain: "example_domain.com", errExpected: false},
		{domain: "spiffe://example.com", errExpected: false},
		{domain: "EXAMPLE.COM", errExpected: true},
		{domain: "example.com:1234", errExpected: true},
		{domain: "user:password@example.com", errExpected: true},
		{domain: "example?.com", errExpected: true},
		{domain: "exam%3Aple.com", errExpected: true},
	}

	for _, tc := range tt {
		t.Run(tc.domain, func(t *testing.T) {
			err := validateOpts(addOpts{trustDomain: tc.domain})
			assert.Equal(t, tc.errExpected, err != nil)
		})
	}
}
