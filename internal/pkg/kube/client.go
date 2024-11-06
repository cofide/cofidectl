// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package kube

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type Client struct {
	CmdConfig  *api.Config
	Clientset  kubernetes.Interface
	RestConfig *rest.Config
}

func NewKubeClient(configPath string) (*Client, error) {
	apiConfig, err := clientcmd.LoadFromFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("load from file: %w", err)
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, fmt.Errorf("build config: %w", err)
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("new for config: %w", err)
	}

	return &Client{
		CmdConfig:  apiConfig,
		Clientset:  client,
		RestConfig: restConfig,
	}, nil
}

func NewKubeClientFromSpecifiedContext(configPath string, context string) (*Client, error) {
	apiConfig, err := clientcmd.LoadFromFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("load from file: %w", err)
	}

	restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: configPath},
		&clientcmd.ConfigOverrides{CurrentContext: context},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build config: %w", err)
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("new for config: %w", err)
	}

	apiConfig.CurrentContext = context

	return &Client{
		CmdConfig:  apiConfig,
		Clientset:  client,
		RestConfig: restConfig,
	}, nil
}
