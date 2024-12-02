// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package statusspinner

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"

	"github.com/cofide/cofidectl/pkg/provider"
)

// statusSpinner implements a CLI spinner that displays messages from `provider.ProviderStatus`.
type statusSpinner struct {
	spinner *spinner.Spinner
}

func New() *statusSpinner {
	return &statusSpinner{spinner: spinner.New(spinner.CharSets[9], 100*time.Millisecond)}
}

// watch starts the spinner and updates it status info read from `statusCh`.
// The spinner is stopped before returning and any error status is returned.
func (ss *statusSpinner) Watch(statusCh <-chan provider.ProviderStatus) error {
	ss.start()
	defer ss.stop()
	for status := range statusCh {
		ss.update(&status)
		if status.Error != nil {
			return status.Error
		}
	}
	return nil
}

func (ss *statusSpinner) start() {
	ss.spinner.Start()
}

func (ss *statusSpinner) stop() {
	ss.spinner.Stop()
}

func (ss *statusSpinner) update(status *provider.ProviderStatus) {
	ss.spinner.Suffix = fmt.Sprintf(" %s: %s\n", status.Stage, status.Message)
	if status.Done {
		ss.spinner.Stop()
		if status.Error != nil {
			fmt.Printf("❌ %s: %s\n", status.Stage, status.Message)
		} else {
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Printf("%s %s: %s\n\n", green("✅"), status.Stage, status.Message)
		}
	}
}
