// Copyright 2025 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package spirehelm

import (
	"context"

	kubeutil "github.com/cofide/cofidectl/pkg/kube"
	"github.com/cofide/cofidectl/pkg/spire"
	spiretypes "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
)

// Type check that SPIREAPIFactoryImpl implements the SPIREAPIFactory interface.
var _ SPIREAPIFactory = &SPIREAPIFactoryImpl{}

// SPIREAPIFactory is an interface that abstracts the construction of SPIREAPI objects.
type SPIREAPIFactory interface {
	// Build returns a SPIREAPI.
	Build(kubeCfgFile, kubeContext string) (SPIREAPI, error)
}

// SPIREAPI is an interface that abstracts a subset of the SPIRE server API for use by the SpireHelm plugin.
type SPIREAPI interface {
	// WaitForServerIP waits for a SPIRE server pod and service to become ready, then returns the external IP of the service.
	WaitForServerIP(ctx context.Context) (string, error)

	// GetBundle retrieves a SPIFFE bundle for the local trust zone.
	GetBundle(ctx context.Context) (*spiretypes.Bundle, error)
}

// SPIREAPIFactoryImpl implements the SPIREAPIFactory interface, building a SPIREAPIImpl.
type SPIREAPIFactoryImpl struct{}

func (f *SPIREAPIFactoryImpl) Build(kubeCfgFile, kubeContext string) (SPIREAPI, error) {
	client, err := kubeutil.NewKubeClientFromSpecifiedContext(kubeCfgFile, kubeContext)
	if err != nil {
		return nil, err
	}

	return &SPIREAPIImpl{client: client}, nil
}

// SPIREAPIImpl implements the SPIREAPI interface using the Kubernetes API to interact with a
// SPIRE server.
type SPIREAPIImpl struct {
	client *kubeutil.Client
}

func (s *SPIREAPIImpl) WaitForServerIP(ctx context.Context) (string, error) {
	return spire.WaitForServerIP(ctx, s.client)
}

func (s *SPIREAPIImpl) GetBundle(ctx context.Context) (*spiretypes.Bundle, error) {
	return spire.GetBundle(ctx, s.client)
}
