// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package provider

// WorkloadIdentityProvider is the interface to drive downstream workload identity methodologies for the Cofide stack
type WorkloadIdentityProvider interface {
	Execute() (<-chan ProviderStatus, error)
}

type ProviderStatus struct {
	Stage   string
	Message string
	Done    bool
	Error   error
}
