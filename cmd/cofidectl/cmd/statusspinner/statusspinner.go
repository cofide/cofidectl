// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package statusspinner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"

	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/provision_plugin/v1alpha2"
)

// statusSpinner implements a CLI spinner that displays messages from `provider.ProviderStatus`.
type statusSpinner struct {
	spinner *spinner.Spinner
}

func new() *statusSpinner {
	return &statusSpinner{spinner: spinner.New(spinner.CharSets[9], 100*time.Millisecond)}
}

func (ss *statusSpinner) start() {
	ss.spinner.Start()
}

func (ss *statusSpinner) stop() {
	ss.spinner.Stop()
}

func (ss *statusSpinner) update(status *provisionpb.Status) {
	ss.spinner.Suffix = fmt.Sprintf(" %s: %s\n", status.GetStage(), status.GetMessage())
	if status.GetDone() {
		ss.spinner.Stop()
		if status.GetError() != "" {
			fmt.Printf("❌ %s: %s\n", status.GetStage(), status.GetMessage())
		} else {
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Printf("%s %s: %s\n\n", green("✅"), status.GetStage(), status.GetMessage())
		}
	}
}

// WatchProvisionStatus reads Status objects from a channel and manages status spinners to consume the events.
// The channel may receive status objects for multiple sequential operations, each of which should use its own spinner.
func WatchProvisionStatus(ctx context.Context, statusCh <-chan *provisionpb.Status, quiet bool) error {
	var spinner *statusSpinner
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case status, ok := <-statusCh:
			if !ok {
				return nil
			}

			if spinner == nil {
				spinner = new()
				if !quiet {
					spinner.start()
				}
			}

			spinner.update(status)
			if status.GetError() != "" {
				return errors.New(status.GetError())
			}
			if status.GetDone() {
				spinner.stop()
				spinner = nil
			}
		}
	}
}
