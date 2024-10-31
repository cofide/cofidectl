package kube

import (
	"context"
	"fmt"
	"io"

	v1 "k8s.io/api/core/v1"
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

	exec, err := remotecommand.NewWebSocketExecutor(config, "POST", req.URL().String())
	if err != nil {
		return fmt.Errorf("new executor: %w", err)
	}
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return fmt.Errorf("stream exec: %w", err)
	}

	return nil
}
