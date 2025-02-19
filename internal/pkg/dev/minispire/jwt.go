// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package minispire

import (
	"crypto"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

type JWTKey struct {
	// The signer used to sign keys
	Signer crypto.Signer

	// Kid is the JWT key ID (i.e. "kid" claim)
	Kid string

	// NotAfter is the expiration time of the JWT key.
	NotAfter time.Time
}

// WorkloadJWTSVIDParams are parameters relevant to workload JWT-SVID creation
type WorkloadJWTSVIDParams struct {
	// SPIFFE ID of the SVID
	SPIFFEID spiffeid.ID

	// TTL is the desired time-to-live of the SVID. Regardless of the TTL, the
	// lifetime of the token will be capped to that of the signing key.
	TTL time.Duration

	// Audience is used for audience claims
	Audience []string
}

// WorkloadJWTSVIDParams are parameters relevant to workload JWT PoP creation
type WorkloadJWTPOParams struct {
	// SPIFFE ID of the SVID
	SPIFFEID spiffeid.ID

	// TTL is the desired time-to-live of the SVID. Regardless of the TTL, the
	// lifetime of the token will be capped to that of the signing key.
	TTL time.Duration

	// Audience is used for audience claims
	Audience string

	Key jose.JSONWebKey
}
