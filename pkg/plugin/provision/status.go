// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"fmt"

	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/provision_plugin/v1alpha1"
)

// StatusBuilder makes it easier to construct Status messages with context about the trust zone
// and cluster.
type StatusBuilder struct {
	trustZone string
	cluster   string
}

func NewStatusBuilder(trustZone, cluster string) *StatusBuilder {
	return &StatusBuilder{trustZone: trustZone, cluster: cluster}
}

// Ok returns a Status message with Done set to false and no Error
func (sb *StatusBuilder) Ok(stage, message string) *provisionpb.Status {
	return StatusOk(stage, sb.getMessage(message))
}

// Done returns a Status message with Done set to true and no Error
func (sb *StatusBuilder) Done(stage, message string) *provisionpb.Status {
	return StatusDone(stage, sb.getMessage(message))
}

// Error returns a Status message with Done set to true and an Error
func (sb *StatusBuilder) Error(stage, message string, err error) *provisionpb.Status {
	return StatusError(stage, sb.getMessage(message), err)
}

// getMessage returns the provided message with trust zone and cluster context appended.
func (sb *StatusBuilder) getMessage(message string) string {
	if sb.cluster != "" {
		message = fmt.Sprintf("%s for %s", message, sb.cluster)
	}
	if sb.trustZone != "" {
		message = fmt.Sprintf("%s in %s", message, sb.trustZone)
	}
	return message
}

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
