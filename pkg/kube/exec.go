// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package kube

import (
	"context"
	"fmt"
	"io"
	"net/url"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

func RunCommand(ctx context.Context, client kubernetes.Interface, config *restclient.Config, podName string,
	namespace string, container string, command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {

	opts := &v1.PodExecOptions{
		Command: command,
		Stdin:   true,
		Stdout:  true,
		Stderr:  true,
	}
	if container != "" {
		opts.Container = container
	}

	req := client.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(opts, scheme.ParameterCodec)

	exec, err := createExecutor(req.URL(), config)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return fmt.Errorf("failed to stream exec: %w", err)
	}

	return nil
}

// createExecutor returns the Executor or an error if one occurred.
// Adapted from a function of the same name in kubectl: https://github.com/kubernetes/kubectl/blob/d0bc9691f3166ac2586b3c948f455f78987e34de/pkg/cmd/exec/exec.go#L137
func createExecutor(url *url.URL, config *restclient.Config) (remotecommand.Executor, error) {
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", url)
	if err != nil {
		return nil, err
	}
	// WebSocketExecutor must be "GET" method as described in RFC 6455 Sec. 4.1 (page 17).
	websocketExec, err := remotecommand.NewWebSocketExecutor(config, "GET", url.String())
	if err != nil {
		return nil, err
	}
	exec, err = remotecommand.NewFallbackExecutor(websocketExec, exec, func(err error) bool {
		return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
	})
	if err != nil {
		return nil, err
	}
	return exec, nil
}
