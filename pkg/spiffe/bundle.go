// Copyright 2025 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spiffe

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/spiffe/go-spiffe/v2/bundle/spiffebundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/spire-api-sdk/proto/spire/api/types"
)

// GetSPIFFETrustBundle converts a trust bundle from the given *types.Bundle to
// *spiffebundle.Bundle.
func GetSPIFFETrustBundle(trustBundle *types.Bundle) (*spiffebundle.Bundle, error) {
	trustDomain, err := spiffeid.TrustDomainFromString(trustBundle.TrustDomain)
	if err != nil {
		return nil, err
	}

	x509Authorities, jwtAuthorities, err := getAuthorities(trustBundle)
	if err != nil {
		return nil, err
	}

	newTrustBundle := spiffebundle.New(trustDomain)
	newTrustBundle.SetX509Authorities(x509Authorities)
	newTrustBundle.SetJWTAuthorities(jwtAuthorities)
	if trustBundle.RefreshHint > 0 {
		newTrustBundle.SetRefreshHint(time.Duration(trustBundle.RefreshHint) * time.Second)
	}

	if trustBundle.SequenceNumber > 0 {
		newTrustBundle.SetSequenceNumber(trustBundle.SequenceNumber)
	}

	return newTrustBundle, nil
}

// getAuthorities gets the X.509 authorities and JWT authorities from the
// provided *types.Bundle.
func getAuthorities(trustBundle *types.Bundle) ([]*x509.Certificate, map[string]crypto.PublicKey, error) {
	x509Authorities, err := convertX509Certificates(trustBundle.X509Authorities)
	if err != nil {
		return nil, nil, err
	}

	jwtAuthorities, err := convertJWTKeys(trustBundle.JwtAuthorities)
	if err != nil {
		return nil, nil, err
	}

	return x509Authorities, jwtAuthorities, nil
}

// convertX509Certificates converts X.509 certificates from the given
// []*types.X509Certificate to []*x509.Certificate.
func convertX509Certificates(proto []*types.X509Certificate) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate

	for i, auth := range proto {
		if auth == nil {
			return nil, fmt.Errorf("auth at index %d is nil", i)
		}
		if auth.Asn1 == nil {
			return nil, fmt.Errorf("ASN.1 data at index %d is nil", i)
		}
		cert, err := x509.ParseCertificate(auth.Asn1)
		if err != nil {
			return nil, fmt.Errorf("unable to parse root CA %d: %w", i, err)
		}

		certs = append(certs, cert)
	}
	return certs, nil
}

// convertJWTKeys converts JWT keys from the given []*types.JWTKey to
// map[string]crypto.PublicKey.
// The key ID of the public key is used as the key in the returned map.
func convertJWTKeys(proto []*types.JWTKey) (map[string]crypto.PublicKey, error) {
	keys := make(map[string]crypto.PublicKey)

	for i, publicKey := range proto {
		jwtSigningKey, err := x509.ParsePKIXPublicKey(publicKey.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("unable to parse JWT signing key %d: %w", i, err)
		}

		keys[publicKey.KeyId] = jwtSigningKey
	}

	return keys, nil
}
