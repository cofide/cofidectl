package minispire

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/cryptosigner"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/spire/pkg/common/cryptoutil"
	"github.com/spiffe/spire/pkg/common/jwtsvid"
)

// This package implements a simple in-memory CA for SPIRE.
// It generates a CA and signs SVIDs on the fly for local development purposes.

type InMemoryCA struct {
	caKey       crypto.Signer
	caCert      *x509.Certificate
	caCertBytes []byte
	jwtKey      JWTKey

	KeyType KeyType
}

type KeyType int

const (
	KeyTypeRSA KeyType = iota
	KeyTypeECDSAP256
)

func NewInMemoryCA(kt KeyType) (*InMemoryCA, error) {
	var caKey crypto.Signer
	if kt == KeyTypeRSA {
		var err error
		caKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("failed to generate CA key: %v", err)
		}
	} else if kt == KeyTypeECDSAP256 {
		var err error
		caKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate CA key: %v", err)
		}
	}
	caSerial, err := rand.Int(rand.Reader, big.NewInt(100000))
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA serial: %v", err)
	}
	caCert := &x509.Certificate{
		Subject: pkix.Name{
			Organization:  []string{"Cofide Development"},
			Country:       []string{"Earth"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotAfter:              time.Now().Add(time.Hour * 24 * 30), // setting 30 days to avoid people using this outside of development, no laptop lasts that long
		SerialNumber:          caSerial,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caCertBytes, err := x509.CreateCertificate(rand.Reader, caCert, caCert, caKey.Public(), caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA cert: %v", err)
	}

	return &InMemoryCA{
		KeyType:     kt,
		caKey:       caKey,
		caCert:      caCert,
		caCertBytes: caCertBytes,

		jwtKey: JWTKey{
			Signer:   caKey,
			Kid:      "kid",
			NotAfter: caCert.NotAfter,
		},
	}, nil
}

func (i *InMemoryCA) Sign(csrBytes []byte) ([]byte, time.Time, error) {
	svidSerial, err := rand.Int(rand.Reader, big.NewInt(100000))
	if err != nil {
		return nil, time.Now(), fmt.Errorf("failed to generate SVID serial: %v", err)
	}

	// parse CSR
	csr, err := x509.ParseCertificateRequest(csrBytes)
	if err != nil {
		return nil, time.Now(), fmt.Errorf("failed to parse CSR: %v", err)
	}

	// create SVID
	svid := &x509.Certificate{
		URIs:         csr.URIs,
		NotAfter:     time.Now().Add(4 * time.Minute), // set to 4 minutes for testing of rotation
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		SerialNumber: svidSerial,
	}

	// sign SVID
	svidBytes, err := x509.CreateCertificate(rand.Reader, svid, i.caCert, csr.PublicKey, i.caKey)
	if err != nil {
		return nil, time.Now(), fmt.Errorf("failed to sign SVID: %v", err)
	}

	return svidBytes, svid.NotAfter, nil
}

func (i *InMemoryCA) GetCACert() []byte {
	return i.caCertBytes
}

func (i *InMemoryCA) SignWorkloadJWTSVID(ctx context.Context, params WorkloadJWTSVIDParams) (string, error) {
	if params.TTL == 0 {
		params.TTL = time.Minute * 5
	}

	claims := map[string]any{
		"sub": params.SPIFFEID,
		"exp": jwt.NewNumericDate(time.Now().Add(params.TTL)),
		"aud": params.Audience,
		"iat": jwt.NewNumericDate(time.Now()),
		"iss": "spire",
	}

	alg, err := cryptoutil.JoseAlgFromPublicKey(i.jwtKey.Signer.Public())
	if err != nil {
		return "", fmt.Errorf("failed to determine JWT key algorithm: %w", err)
	}

	jwtSigner, err := jose.NewSigner(
		jose.SigningKey{
			Algorithm: alg,
			Key: jose.JSONWebKey{
				Key:   cryptosigner.Opaque(i.jwtKey.Signer),
				KeyID: i.jwtKey.Kid,
			},
		},
		new(jose.SignerOptions).WithType("JWT"),
	)
	if err != nil {
		return "", fmt.Errorf("failed to configure JWT signer: %w", err)
	}

	signedToken, err := jwt.Signed(jwtSigner).Claims(claims).Serialize()
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT SVID: %w", err)
	}

	if _, err := i.ValidateWorkloadJWTSVID(signedToken, params.SPIFFEID); err != nil {
		return "", err
	}

	return signedToken, nil
}

func (i *InMemoryCA) ValidateWorkloadJWTSVID(rawToken string, id spiffeid.ID) (*jwt.Claims, error) {
	token, err := jwt.ParseSigned(rawToken, jwtsvid.AllowedSignatureAlgorithms)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT-SVID for validation: %w", err)
	}

	var claims jwt.Claims
	if err := token.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return nil, fmt.Errorf("failed to extract JWT-SVID claims for validation: %w", err)
	}

	now := time.Now()
	switch {
	case claims.Subject != id.String():
		return nil, fmt.Errorf(`invalid JWT-SVID "sub" claim: expected %q but got %q`, id, claims.Subject)
	case claims.Expiry == nil:
		return nil, errors.New(`invalid JWT-SVID "exp" claim: required but missing`)
	case !claims.Expiry.Time().After(now):
		return nil, fmt.Errorf(`invalid JWT-SVID "exp" claim: already expired as of %s`, claims.Expiry.Time().Format(time.RFC3339))
	case claims.NotBefore != nil && claims.NotBefore.Time().After(now):
		return nil, fmt.Errorf(`invalid JWT-SVID "nbf" claim: not yet valid until %s`, claims.NotBefore.Time().Format(time.RFC3339))
	case len(claims.Audience) == 0:
		return nil, errors.New(`invalid JWT-SVID "aud" claim: required but missing`)
	}
	return &claims, nil
}

func (i *InMemoryCA) SignWorkloadJWTSVIDPOP(ctx context.Context, params WorkloadJWTPOParams) (string, error) {
	if params.TTL == 0 {
		params.TTL = time.Minute * 5
	}

	claims := map[string]any{
		"sub": params.SPIFFEID,
		"aud": params.Audience,
		"exp": jwt.NewNumericDate(time.Now().Add(params.TTL)),
		"iat": jwt.NewNumericDate(time.Now()),
		"iss": fmt.Sprintf("spiffe://%s", params.SPIFFEID.TrustDomain()),
		"cnf": map[string]any{
			"jwk": params.Key,
		},
	}

	claims["jti"] = generateJTI(claims, params.SPIFFEID.String())

	alg, err := cryptoutil.JoseAlgFromPublicKey(i.jwtKey.Signer.Public())
	if err != nil {
		return "", fmt.Errorf("failed to determine JWT key algorithm: %w", err)
	}

	jwtSigner, err := jose.NewSigner(
		jose.SigningKey{
			Algorithm: alg,
			Key: jose.JSONWebKey{
				Key:   cryptosigner.Opaque(i.jwtKey.Signer),
				KeyID: i.jwtKey.Kid,
			},
		},
		new(jose.SignerOptions).WithType("wimse-id+jwt"),
	)
	if err != nil {
		return "", fmt.Errorf("failed to configure JWT signer: %w", err)
	}

	signedToken, err := jwt.Signed(jwtSigner).Claims(claims).Serialize()
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT SVID: %w", err)
	}

	if _, err := i.ValidateWorkloadJWTSVID(signedToken, params.SPIFFEID); err != nil {
		return "", err
	}

	return signedToken, nil
}

func generateJTI(claims map[string]any, spiffeID string) string {
	// generate a unique identifier for the token using the claims, spiffeID and nonce in SHA256

	var claimsJSON []byte
	var err error
	if claimsJSON, err = json.Marshal(claims); err != nil {
		return ""
	}

	hash := crypto.SHA256.New()
	hash.Write([]byte(spiffeID))
	hash.Write(claimsJSON)

	// add 5 bytes of random data to avoid collisions
	nonce := make([]byte, 5)
	rand.Read(nonce)
	hash.Write(nonce)

	return fmt.Sprintf("%x", hash.Sum(nil))
}
