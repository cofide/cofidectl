// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spire

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/cofide/cofidectl/internal/pkg/test/utils"
	types "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var oneAgentList = `{
  "agents": [
    {
      "attestation_type": "k8s_psat",
      "banned": false,
      "can_reattest": true,
      "id": {
        "path": "/spire/agent/k8s_psat/connect/831b9aa2-de44-4f20-bd61-238e756600ce",
        "trust_domain": "cofide.test"
      },
      "selectors": [
        {
          "type": "k8s_psat",
          "value": "agent_node_ip:172.18.0.3"
        },
        {
          "type": "k8s_psat",
          "value": "agent_node_name:connect-control-plane"
        },
        {
          "type": "k8s_psat",
          "value": "agent_node_uid:831b9aa2-de44-4f20-bd61-238e756600ce"
        },
        {
          "type": "k8s_psat",
          "value": "agent_ns:spire-system"
        },
        {
          "type": "k8s_psat",
          "value": "agent_pod_name:spire-agent-52plm"
        },
        {
          "type": "k8s_psat",
          "value": "agent_pod_uid:5ca6358e-9c57-4b26-84e8-fbbb57dd4c6f"
        },
        {
          "type": "k8s_psat",
          "value": "agent_sa:spire-agent"
        },
        {
          "type": "k8s_psat",
          "value": "cluster:connect"
        }
      ],
      "x509svid_expires_at": "1729275243",
      "x509svid_serial_number": "281715470147913728055350377728086773688"
    }
  ],
  "next_page_token": ""
}`

func Test_parseAgentList(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    []Agent
		wantErr bool
	}{
		{
			name:    "empty",
			output:  `{"agents": []}`,
			want:    []Agent{},
			wantErr: false,
		},
		{
			name:   "one agent",
			output: oneAgentList,
			want: []Agent{
				{
					Name:            "spire-agent-52plm",
					Status:          "unknown",
					Id:              "spiffe://cofide.test/spire/agent/k8s_psat/connect/831b9aa2-de44-4f20-bd61-238e756600ce",
					AttestationType: "k8s_psat",
					ExpirationTime:  time.Unix(1729275243, 0),
					Serial:          "281715470147913728055350377728086773688",
					CanReattest:     true,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAgentList([]byte(tt.output))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAgentList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				fmt.Println(got)
				t.Errorf("parseAgentList() = %v, want %v", got, tt.want)
			}
		})
	}
}

var bundleShow = `{
  "jwt_authorities": [
    {
      "expires_at": "1738987145",
      "key_id": "sHYIGH99d7NhlAVufX9a9e0D9HMPGCQw",
      "public_key": "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0mg3S/3z/NlFHhqvd49RibgQpgsWvVBs66pC27AsJIh9UFs5jW17QQJkaBRt/LtA4jhQIQErj3g1ZPyv2JCfLOA+rFHcGFdsnuf8xTgKQfmp4v/xpvUQVmA9rzoFLx5DTDxLe0tU0lgGhJxPJcoSGzAae/Tn/1jenWkIvyPX1W5TMFiIJkpPpqASOUCOnkdwwZ+XeLo+7XWGUAjNtHVsEIOjiIRFkeZCwKSXJvXy9T5OMjCtGsQFaF6+fg5wE0VJBXCDXMr/uPIbVmozGC75opOOPJXcV8daVbEpCKm2BFDcm0MNchNijGGCR0JhYEhb04YSAhN8tmyjxeHHJiblmwIDAQAB",
      "tainted": false
    }
  ],
  "refresh_hint": "2",
  "sequence_number": "1",
  "trust_domain": "td2",
  "x509_authorities": [
    {
      "asn1": "MIIDrjCCApagAwIBAgIRAL6Ru792Wi5AhHhh387STRIwDQYJKoZIhvcNAQELBQAwZDELMAkGA1UEBhMCVUsxDzANBgNVBAoTBkNvZmlkZTESMBAGA1UEAxMJY29maWRlLmlvMTAwLgYDVQQFEycyNTMzMTAwMTAyMjM0MjQ3NDE4NDYzOTczNzY0MDQzMTM0OTI3NTQwHhcNMjUwMjA3MTU1ODU1WhcNMjUwMjA4MDM1OTA1WjBkMQswCQYDVQQGEwJVSzEPMA0GA1UEChMGQ29maWRlMRIwEAYDVQQDEwljb2ZpZGUuaW8xMDAuBgNVBAUTJzI1MzMxMDAxMDIyMzQyNDc0MTg0NjM5NzM3NjQwNDMxMzQ5Mjc1NDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAM0IjG8AFER3+u7njyJqVyHWnGNqEWkOWGXmUmEAx87fpJr4U5X8piXZwPHPVIfcrH1jINpBAOuCBihrAbhwAX0HmtkPt3LFWMUp47zHS7+sSy2TReuEHTLtqxgEG7iwBG2sby0YTotZnb3q1XjnuydOzYBuLXCghNiIkS+NRe2koOv5QeUZJN7IoDuG6bGg6R4CwmHFhLeA2ZMY9QO/X7PhI9PcL6yDurOxgt43qjjGPrkUVVb4v4ju5iz8COaFp1oGchAq+3Tkd0Pl9Vclv8vllDBDMxMjkXjKO1P0ueomldaBJQ5nP/OpmVjhEZ5S9EOKTcfJ7qqS33TAJnBnp00CAwEAAaNbMFkwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFGCz3aiUExK4+2cTKGFcJpxBcAexMBcGA1UdEQQQMA6GDHNwaWZmZTovL3RkMjANBgkqhkiG9w0BAQsFAAOCAQEAfhzGZqw3UC+uJGsOLFQ0v7EWS35UB8PvgWABDd+2cRABnSSsNciaszN0Fz9t1qJcP20eldna5b0eZNJLOH89BEqWGTiXD37B3qAqKsT/pAU0eglMtDCNW+KipDpAoo9dFlbF+cSk9dJlH0gNYsMwO1vMFdrRK/4O79sRkxKn2JMf082EXsFpDzPORDsZ1FidOkWT3kTKbH469zFz8a0El7Tq58/2aELkF9qUnP3ZfN6H9CGiES7OV7kNuzuTadVIiFQpeYxd+U/ro6jKeyUdY83FZ6Qfx/bRTRqXStrbutDcdetWWQvRGRCHRoa0uMNmz8fkqLDRkc+emcJGyGSLAQ==",
      "tainted": false
    }
  ]
}`

func Test_parseBundleShow(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    *types.Bundle
		wantErr bool
	}{
		{
			name:   "empty",
			output: `{}`,
			want: &types.Bundle{
				JwtAuthorities:  []*types.JWTKey{},
				X509Authorities: []*types.X509Certificate{},
			},
			wantErr: false,
		},
		{
			name:   "full",
			output: bundleShow,
			want: &types.Bundle{
				JwtAuthorities: []*types.JWTKey{
					{
						ExpiresAt: 1738987145,
						KeyId:     "sHYIGH99d7NhlAVufX9a9e0D9HMPGCQw",
						PublicKey: utils.Base64Decode("MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0mg3S/3z/NlFHhqvd49RibgQpgsWvVBs66pC27AsJIh9UFs5jW17QQJkaBRt/LtA4jhQIQErj3g1ZPyv2JCfLOA+rFHcGFdsnuf8xTgKQfmp4v/xpvUQVmA9rzoFLx5DTDxLe0tU0lgGhJxPJcoSGzAae/Tn/1jenWkIvyPX1W5TMFiIJkpPpqASOUCOnkdwwZ+XeLo+7XWGUAjNtHVsEIOjiIRFkeZCwKSXJvXy9T5OMjCtGsQFaF6+fg5wE0VJBXCDXMr/uPIbVmozGC75opOOPJXcV8daVbEpCKm2BFDcm0MNchNijGGCR0JhYEhb04YSAhN8tmyjxeHHJiblmwIDAQAB"),
						Tainted:   false,
					},
				},
				RefreshHint:    2,
				SequenceNumber: 1,
				TrustDomain:    "td2",
				X509Authorities: []*types.X509Certificate{
					{
						Asn1:    utils.Base64Decode("MIIDrjCCApagAwIBAgIRAL6Ru792Wi5AhHhh387STRIwDQYJKoZIhvcNAQELBQAwZDELMAkGA1UEBhMCVUsxDzANBgNVBAoTBkNvZmlkZTESMBAGA1UEAxMJY29maWRlLmlvMTAwLgYDVQQFEycyNTMzMTAwMTAyMjM0MjQ3NDE4NDYzOTczNzY0MDQzMTM0OTI3NTQwHhcNMjUwMjA3MTU1ODU1WhcNMjUwMjA4MDM1OTA1WjBkMQswCQYDVQQGEwJVSzEPMA0GA1UEChMGQ29maWRlMRIwEAYDVQQDEwljb2ZpZGUuaW8xMDAuBgNVBAUTJzI1MzMxMDAxMDIyMzQyNDc0MTg0NjM5NzM3NjQwNDMxMzQ5Mjc1NDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAM0IjG8AFER3+u7njyJqVyHWnGNqEWkOWGXmUmEAx87fpJr4U5X8piXZwPHPVIfcrH1jINpBAOuCBihrAbhwAX0HmtkPt3LFWMUp47zHS7+sSy2TReuEHTLtqxgEG7iwBG2sby0YTotZnb3q1XjnuydOzYBuLXCghNiIkS+NRe2koOv5QeUZJN7IoDuG6bGg6R4CwmHFhLeA2ZMY9QO/X7PhI9PcL6yDurOxgt43qjjGPrkUVVb4v4ju5iz8COaFp1oGchAq+3Tkd0Pl9Vclv8vllDBDMxMjkXjKO1P0ueomldaBJQ5nP/OpmVjhEZ5S9EOKTcfJ7qqS33TAJnBnp00CAwEAAaNbMFkwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFGCz3aiUExK4+2cTKGFcJpxBcAexMBcGA1UdEQQQMA6GDHNwaWZmZTovL3RkMjANBgkqhkiG9w0BAQsFAAOCAQEAfhzGZqw3UC+uJGsOLFQ0v7EWS35UB8PvgWABDd+2cRABnSSsNciaszN0Fz9t1qJcP20eldna5b0eZNJLOH89BEqWGTiXD37B3qAqKsT/pAU0eglMtDCNW+KipDpAoo9dFlbF+cSk9dJlH0gNYsMwO1vMFdrRK/4O79sRkxKn2JMf082EXsFpDzPORDsZ1FidOkWT3kTKbH469zFz8a0El7Tq58/2aELkF9qUnP3ZfN6H9CGiES7OV7kNuzuTadVIiFQpeYxd+U/ro6jKeyUdY83FZ6Qfx/bRTRqXStrbutDcdetWWQvRGRCHRoa0uMNmz8fkqLDRkc+emcJGyGSLAQ=="),
						Tainted: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid type",
			output:  `{"jwt_authorities": "invalid type"}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBundleShow([]byte(tt.output))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
