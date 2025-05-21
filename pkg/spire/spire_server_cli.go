// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spire

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	types "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
)

const (
	k8sPSATSelectorType     = "k8s_psat"
	agentPodNameSelector    = "agent_pod_name:"
	k8sSelectorType         = "k8s"
	k8sPodUIDSelectorPrefix = "pod-uid:"
)

// execInServerContainer executes a command in the SPIRE server container.
func execInServerContainer(ctx context.Context, client *kubeutil.Client, command []string) ([]byte, []byte, error) {
	executable := serverExecutable
	command = append([]string{executable}, command...)
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := kubeutil.RunCommand(
		ctx,
		client.Clientset,
		client.RestConfig,
		serverPodName,
		serverNamespace,
		serverContainerName,
		command,
		stdin,
		stdout,
		stderr,
	)
	if err != nil {
		return nil, nil, err
	}
	return stdout.Bytes(), stderr.Bytes(), nil
}

// agentListJson represents the JSON-formatted output of the SPIRE server agent list command.
type agentListJson struct {
	Agents []agentJson `json:"agents"`
}

type agentJson struct {
	AttestationType string            `json:"attestation_type"`
	ExpirationTime  string            `json:"x509svid_expires_at"`
	Serial          string            `json:"x509svid_serial_number"`
	CanReattest     bool              `json:"can_reattest"`
	Id              *types.SPIFFEID   `json:"id"`
	Selectors       []*types.Selector `json:"selectors"`
}

// parseAgentList parses the output of `spire-server agent list -output json` and returns a slice of `Agent`.
func parseAgentList(output []byte) ([]Agent, error) {
	agents := &agentListJson{}
	err := json.Unmarshal(output, agents)
	if err != nil {
		return nil, err
	}

	statuses := []Agent{}
	for _, agent := range agents.Agents {
		podName := getPodNameSelector(agent)
		id := fmt.Sprintf("spiffe://%s%s", agent.Id.TrustDomain, agent.Id.Path)
		expTime, err := strconv.ParseInt(agent.ExpirationTime, 10, 64)
		if err != nil {
			fmt.Println("unable to parse agent expiration timestamp:", agent.ExpirationTime)
		}
		status := Agent{
			Name:            podName,
			Status:          "unknown",
			Id:              id,
			AttestationType: agent.AttestationType,
			ExpirationTime:  time.Unix(expTime, 0),
			Serial:          agent.Serial,
			CanReattest:     agent.CanReattest,
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func getPodNameSelector(agent agentJson) string {
	for _, selector := range agent.Selectors {
		if selector.Type == k8sPSATSelectorType && strings.HasPrefix(selector.Value, agentPodNameSelector) {
			return strings.TrimPrefix(selector.Value, agentPodNameSelector)
		}
	}
	return "unknown"
}

type entryListJson struct {
	Entries []entryJson `json:"entries"`
}

type entryJson struct {
	Selectors []*types.Selector `json:"selectors"`
	Id        *types.SPIFFEID   `json:"spiffe_id"`
}

func parseEntryList(output []byte) (*entryListJson, error) {
	entries := &entryListJson{}
	err := json.Unmarshal(output, entries)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

type bundleJson struct {
	JwtAuthorities  []jwtAuthorityJson  `json:"jwt_authorities"`
	RefreshHint     string              `json:"refresh_hint"`
	SequenceNumber  string              `json:"sequence_number"`
	TrustDomain     string              `json:"trust_domain"`
	X509Authorities []x509AuthorityJson `json:"x509_authorities"`
}

func (b *bundleJson) toBundle() (*types.Bundle, error) {
	bundle := types.Bundle{
		JwtAuthorities:  []*types.JWTKey{},
		TrustDomain:     b.TrustDomain,
		X509Authorities: []*types.X509Certificate{},
	}
	var err error

	if b.RefreshHint != "" {
		bundle.RefreshHint, err = strconv.ParseInt(b.RefreshHint, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	if b.SequenceNumber != "" {
		bundle.SequenceNumber, err = strconv.ParseUint(b.SequenceNumber, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	for _, ja := range b.JwtAuthorities {
		jk, err := ja.toJWTKey()
		if err != nil {
			return nil, err
		}

		bundle.JwtAuthorities = append(bundle.JwtAuthorities, jk)
	}

	for _, xc := range b.X509Authorities {
		bundle.X509Authorities = append(bundle.X509Authorities, xc.toX509Certificate())
	}

	return &bundle, nil
}

type jwtAuthorityJson struct {
	ExpiresAt string `json:"expires_at"`
	KeyId     string `json:"key_id"`
	PublicKey []byte `json:"public_key"`
	Tainted   bool   `json:"tainted"`
}

func (ja *jwtAuthorityJson) toJWTKey() (*types.JWTKey, error) {
	expiresAt, err := strconv.ParseInt(ja.ExpiresAt, 10, 64)
	if err != nil {
		return nil, err
	}

	return &types.JWTKey{
		ExpiresAt: expiresAt,
		KeyId:     ja.KeyId,
		PublicKey: ja.PublicKey,
		Tainted:   ja.Tainted,
	}, nil
}

type x509AuthorityJson struct {
	Asn1    []byte `json:"asn1"`
	Tainted bool   `json:"tainted"`
}

func (xa *x509AuthorityJson) toX509Certificate() *types.X509Certificate {
	return &types.X509Certificate{
		Asn1:    xa.Asn1,
		Tainted: xa.Tainted,
	}
}

// parseBundleShow parses the output of the 'bundle show -output json' command.
func parseBundleShow(output []byte) (*types.Bundle, error) {
	bundle := &bundleJson{}
	err := json.Unmarshal(output, bundle)
	if err != nil {
		return nil, err
	}
	return bundle.toBundle()
}
