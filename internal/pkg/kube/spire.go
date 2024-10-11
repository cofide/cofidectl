package kube

import (
	"bytes"
	"context"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func GetSpireServerEntries(ctx context.Context, client kubernetes.Interface, restConfig *restclient.Config) ([]string, error) {
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	podExecOpts := &v1.PodExecOptions{
		Command:   []string{"/opt/spire/bin/spire-server", "entry", "show"},
		Container: "spire-server",
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
	}

	request := client.CoreV1().
		RESTClient().
		Post().
		Namespace("spire").
		Resource("pods").
		Name("spire-server-0").
		SubResource("exec").
		VersionedParams(podExecOpts, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", request.URL())
	if err != nil {
		return nil, err
	}

	err = exec.StreamWithContext(
		ctx,
		remotecommand.StreamOptions{
			Stdin:  stdin,
			Stdout: stdout,
			Stderr: stderr,
		},
	)

	if err != nil {
		return nil, err
	}

	var podUids []string

	for _, line := range strings.Split(stdout.String(), "\n") {
		if strings.HasPrefix(line, "Selector") {
			podUid := strings.Split(line, ":")[3]
			podUids = append(podUids, podUid)
		}
	}
	return podUids, nil
}
