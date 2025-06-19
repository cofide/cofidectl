// Copyright 2025 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package utils

// PtrOf returns a pointer to a provided value.
func PtrOf[T any](v T) *T {
	return &v
}
