// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/provision_plugin/v1alpha1"
)

// StatusOk returns a Status message with Done set to false and no Error.
func StatusOk(stage, message string) *provisionpb.Status {
	done := false
	return &provisionpb.Status{Stage: &stage, Message: &message, Done: &done}
}

// StatusOk returns a Status message with Done set to true and no Error.
func StatusDone(stage, message string) *provisionpb.Status {
	done := true
	return &provisionpb.Status{Stage: &stage, Message: &message, Done: &done}
}

// StatusOk returns a Status message with Done set to true and an Error.
func StatusError(stage, message string, err error) *provisionpb.Status {
	done := true
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return &provisionpb.Status{Stage: &stage, Message: &message, Done: &done, Error: &errMsg}
}
