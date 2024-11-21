// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"syscall"
)

const (
	relativePluginDir = ".cofide/plugins"
)

type CliPlugin struct {
	BinaryName string
	Args       []string
}

func GetPluginDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get user home: %w", err)
	}
	return filepath.Join(usr.HomeDir, relativePluginDir), nil
}

func GetPluginPath(name string) (string, error) {
	pluginDir, err := GetPluginDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(pluginDir, name), nil
}

func PluginExists(name string) (bool, error) {
	pluginPath, err := GetPluginPath(name)
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(pluginPath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func NewCliPlugin(binary string, args []string) *CliPlugin {
	return &CliPlugin{
		BinaryName: binary,
		Args:       args,
	}
}

// Execute executes a CLI plugin by exec'ing into it, replacing the current process.
// This may change to execute the plugin as a subprocess, but Exec keeps things simple for now (no signal handling etc.)
func (cp *CliPlugin) Execute() error {
	pluginPath, err := GetPluginPath(cp.BinaryName)
	if err != nil {
		return err
	}
	// syscall.Exec requires the binary to be the 0th element of the arguments.
	args := append([]string{cp.BinaryName}, cp.Args...)
	err = syscall.Exec(pluginPath, args, os.Environ())
	if err != nil {
		return err
	}
	return nil
}
