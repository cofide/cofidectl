// Copyright 2025 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spiffe

import (
	"strings"
	"testing"

	"github.com/cofide/cofidectl/internal/pkg/test/utils"
	"github.com/spiffe/go-spiffe/v2/bundle/spiffebundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSPIFFETrustBundle(t *testing.T) {
	tests := []struct {
		name    string
		bundle  *types.Bundle
		want    *spiffebundle.Bundle
		wantErr bool
	}{
		{
			name: "success",
			bundle: &types.Bundle{
				JwtAuthorities: []*types.JWTKey{
					{
						PublicKey: utils.Base64Decode("MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0mg3S/3z/NlFHhqvd49RibgQpgsWvVBs66pC27AsJIh9UFs5jW17QQJkaBRt/LtA4jhQIQErj3g1ZPyv2JCfLOA+rFHcGFdsnuf8xTgKQfmp4v/xpvUQVmA9rzoFLx5DTDxLe0tU0lgGhJxPJcoSGzAae/Tn/1jenWkIvyPX1W5TMFiIJkpPpqASOUCOnkdwwZ+XeLo+7XWGUAjNtHVsEIOjiIRFkeZCwKSXJvXy9T5OMjCtGsQFaF6+fg5wE0VJBXCDXMr/uPIbVmozGC75opOOPJXcV8daVbEpCKm2BFDcm0MNchNijGGCR0JhYEhb04YSAhN8tmyjxeHHJiblmwIDAQAB"),
						KeyId:     "sHYIGH99d7NhlAVufX9a9e0D9HMPGCQw",
						ExpiresAt: 1738987145,
						Tainted:   false,
					},
				},
				RefreshHint:    2,
				SequenceNumber: 3,
				TrustDomain:    "td1",
				X509Authorities: []*types.X509Certificate{
					{
						Asn1:    utils.Base64Decode("MIIDrjCCApagAwIBAgIRAL6Ru792Wi5AhHhh387STRIwDQYJKoZIhvcNAQELBQAwZDELMAkGA1UEBhMCVUsxDzANBgNVBAoTBkNvZmlkZTESMBAGA1UEAxMJY29maWRlLmlvMTAwLgYDVQQFEycyNTMzMTAwMTAyMjM0MjQ3NDE4NDYzOTczNzY0MDQzMTM0OTI3NTQwHhcNMjUwMjA3MTU1ODU1WhcNMjUwMjA4MDM1OTA1WjBkMQswCQYDVQQGEwJVSzEPMA0GA1UEChMGQ29maWRlMRIwEAYDVQQDEwljb2ZpZGUuaW8xMDAuBgNVBAUTJzI1MzMxMDAxMDIyMzQyNDc0MTg0NjM5NzM3NjQwNDMxMzQ5Mjc1NDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAM0IjG8AFER3+u7njyJqVyHWnGNqEWkOWGXmUmEAx87fpJr4U5X8piXZwPHPVIfcrH1jINpBAOuCBihrAbhwAX0HmtkPt3LFWMUp47zHS7+sSy2TReuEHTLtqxgEG7iwBG2sby0YTotZnb3q1XjnuydOzYBuLXCghNiIkS+NRe2koOv5QeUZJN7IoDuG6bGg6R4CwmHFhLeA2ZMY9QO/X7PhI9PcL6yDurOxgt43qjjGPrkUVVb4v4ju5iz8COaFp1oGchAq+3Tkd0Pl9Vclv8vllDBDMxMjkXjKO1P0ueomldaBJQ5nP/OpmVjhEZ5S9EOKTcfJ7qqS33TAJnBnp00CAwEAAaNbMFkwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFGCz3aiUExK4+2cTKGFcJpxBcAexMBcGA1UdEQQQMA6GDHNwaWZmZTovL3RkMjANBgkqhkiG9w0BAQsFAAOCAQEAfhzGZqw3UC+uJGsOLFQ0v7EWS35UB8PvgWABDd+2cRABnSSsNciaszN0Fz9t1qJcP20eldna5b0eZNJLOH89BEqWGTiXD37B3qAqKsT/pAU0eglMtDCNW+KipDpAoo9dFlbF+cSk9dJlH0gNYsMwO1vMFdrRK/4O79sRkxKn2JMf082EXsFpDzPORDsZ1FidOkWT3kTKbH469zFz8a0El7Tq58/2aELkF9qUnP3ZfN6H9CGiES7OV7kNuzuTadVIiFQpeYxd+U/ro6jKeyUdY83FZ6Qfx/bRTRqXStrbutDcdetWWQvRGRCHRoa0uMNmz8fkqLDRkc+emcJGyGSLAQ=="),
						Tainted: false,
					},
				},
			},
			want: func() *spiffebundle.Bundle {
				trustDomain := spiffeid.RequireTrustDomainFromString("td1")
				jwks := "{\"keys\":[{\"use\":\"x509-svid\",\"kty\":\"RSA\",\"n\":\"zQiMbwAURHf67uePImpXIdacY2oRaQ5YZeZSYQDHzt-kmvhTlfymJdnA8c9Uh9ysfWMg2kEA64IGKGsBuHABfQea2Q-3csVYxSnjvMdLv6xLLZNF64QdMu2rGAQbuLAEbaxvLRhOi1mdverVeOe7J07NgG4tcKCE2IiRL41F7aSg6_lB5Rkk3sigO4bpsaDpHgLCYcWEt4DZkxj1A79fs-Ej09wvrIO6s7GC3jeqOMY-uRRVVvi_iO7mLPwI5oWnWgZyECr7dOR3Q-X1VyW_y-WUMEMzEyOReMo7U_S56iaV1oElDmc_86mZWOERnlL0Q4pNx8nuqpLfdMAmcGenTQ\",\"e\":\"AQAB\",\"x5c\":[\"MIIDrjCCApagAwIBAgIRAL6Ru792Wi5AhHhh387STRIwDQYJKoZIhvcNAQELBQAwZDELMAkGA1UEBhMCVUsxDzANBgNVBAoTBkNvZmlkZTESMBAGA1UEAxMJY29maWRlLmlvMTAwLgYDVQQFEycyNTMzMTAwMTAyMjM0MjQ3NDE4NDYzOTczNzY0MDQzMTM0OTI3NTQwHhcNMjUwMjA3MTU1ODU1WhcNMjUwMjA4MDM1OTA1WjBkMQswCQYDVQQGEwJVSzEPMA0GA1UEChMGQ29maWRlMRIwEAYDVQQDEwljb2ZpZGUuaW8xMDAuBgNVBAUTJzI1MzMxMDAxMDIyMzQyNDc0MTg0NjM5NzM3NjQwNDMxMzQ5Mjc1NDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAM0IjG8AFER3+u7njyJqVyHWnGNqEWkOWGXmUmEAx87fpJr4U5X8piXZwPHPVIfcrH1jINpBAOuCBihrAbhwAX0HmtkPt3LFWMUp47zHS7+sSy2TReuEHTLtqxgEG7iwBG2sby0YTotZnb3q1XjnuydOzYBuLXCghNiIkS+NRe2koOv5QeUZJN7IoDuG6bGg6R4CwmHFhLeA2ZMY9QO/X7PhI9PcL6yDurOxgt43qjjGPrkUVVb4v4ju5iz8COaFp1oGchAq+3Tkd0Pl9Vclv8vllDBDMxMjkXjKO1P0ueomldaBJQ5nP/OpmVjhEZ5S9EOKTcfJ7qqS33TAJnBnp00CAwEAAaNbMFkwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFGCz3aiUExK4+2cTKGFcJpxBcAexMBcGA1UdEQQQMA6GDHNwaWZmZTovL3RkMjANBgkqhkiG9w0BAQsFAAOCAQEAfhzGZqw3UC+uJGsOLFQ0v7EWS35UB8PvgWABDd+2cRABnSSsNciaszN0Fz9t1qJcP20eldna5b0eZNJLOH89BEqWGTiXD37B3qAqKsT/pAU0eglMtDCNW+KipDpAoo9dFlbF+cSk9dJlH0gNYsMwO1vMFdrRK/4O79sRkxKn2JMf082EXsFpDzPORDsZ1FidOkWT3kTKbH469zFz8a0El7Tq58/2aELkF9qUnP3ZfN6H9CGiES7OV7kNuzuTadVIiFQpeYxd+U/ro6jKeyUdY83FZ6Qfx/bRTRqXStrbutDcdetWWQvRGRCHRoa0uMNmz8fkqLDRkc+emcJGyGSLAQ==\"]},{\"use\":\"jwt-svid\",\"kty\":\"RSA\",\"kid\":\"sHYIGH99d7NhlAVufX9a9e0D9HMPGCQw\",\"n\":\"0mg3S_3z_NlFHhqvd49RibgQpgsWvVBs66pC27AsJIh9UFs5jW17QQJkaBRt_LtA4jhQIQErj3g1ZPyv2JCfLOA-rFHcGFdsnuf8xTgKQfmp4v_xpvUQVmA9rzoFLx5DTDxLe0tU0lgGhJxPJcoSGzAae_Tn_1jenWkIvyPX1W5TMFiIJkpPpqASOUCOnkdwwZ-XeLo-7XWGUAjNtHVsEIOjiIRFkeZCwKSXJvXy9T5OMjCtGsQFaF6-fg5wE0VJBXCDXMr_uPIbVmozGC75opOOPJXcV8daVbEpCKm2BFDcm0MNchNijGGCR0JhYEhb04YSAhN8tmyjxeHHJiblmw\",\"e\":\"AQAB\"}],\"spiffe_sequence\":3,\"spiffe_refresh_hint\":2}"
				bundle, err := spiffebundle.Read(trustDomain, strings.NewReader(jwks))
				require.NoError(t, err)
				return bundle
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSPIFFETrustBundle(tt.bundle)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
