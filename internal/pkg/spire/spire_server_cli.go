package spire

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	kubeutil "github.com/cofide/cofidectl/internal/pkg/kube"
	types "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
)

const (
	k8sPsatSelectorType  = "k8s_psat"
	agentPodNameSelector = "agent_pod_name:"
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
		namespace,
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
		if selector.Type == k8sPsatSelectorType && strings.HasPrefix(selector.Value, agentPodNameSelector) {
			return strings.TrimPrefix(selector.Value, agentPodNameSelector)
		}
	}
	return "unknown"
}
