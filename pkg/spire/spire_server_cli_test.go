// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spire

import (
	"fmt"
	"reflect"
	"testing"
	"time"
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
