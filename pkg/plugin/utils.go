// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"errors"
	"os"
	"path/filepath"
)

func PluginExists(name string) (bool, error) {
	pluginDir, err := GetPluginDir()
	if err != nil {
		return false, err
	}

	pluginPath := filepath.Join(pluginDir, name)

	if _, err := os.Stat(pluginPath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}
