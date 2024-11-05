// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package kube

type Repository interface {
	GetContexts() ([]string, error)
}
