// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/provision_plugin/v1alpha2"
)

// Provider is an interface that abstracts a Helm-based workload identity provider.
type Provider interface {
	// AddRepository adds the SPIRE Helm repository to the local repositories.yaml.
	// The action is performed synchronously and status is streamed through the provided status channel.
	// This function should be called once, not per-trust zone.
	// The SPIRE Helm repository is added to the local repositories.yaml, locking the repositories.lock
	// file while making changes.
	AddRepository(statusCh chan<- *provisionpb.Status) error

	// Execute installs the SPIRE Helm stack to the selected Kubernetes context.
	// The action is performed synchronously and status is streamed through the provided status channel.
	Execute(statusCh chan<- *provisionpb.Status) error

	// ExecutePostInstallUpgrade upgrades the SPIRE stack to the selected Kubernetes context.
	// The action is performed synchronously and status is streamed through the provided status channel.
	ExecutePostInstallUpgrade(statusCh chan<- *provisionpb.Status) error

	// ExecuteUpgrade upgrades the SPIRE stack to the selected Kubernetes context.
	// The action is performed synchronously and status is streamed through the provided status channel.
	ExecuteUpgrade(statusCh chan<- *provisionpb.Status) error

	// ExecuteUninstall uninstalls the SPIRE stack from the selected Kubernetes context.
	// The action is performed synchronously and status is streamed through the provided status channel.
	ExecuteUninstall(statusCh chan<- *provisionpb.Status) error

	// CheckIfAlreadyInstalled returns true if the SPIRE chart has previously been installed.
	CheckIfAlreadyInstalled() (bool, error)
}
