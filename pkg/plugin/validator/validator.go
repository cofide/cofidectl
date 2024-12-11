// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package validator

import "context"

// Validator is a common interface embedded into other plugin interfaces.
type Validator interface {
	// Validate checks whether the plugin is configured correctly.
	Validate(context.Context) error
}
