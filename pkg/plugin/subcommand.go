// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"
	"time"
)

const (
	defaultTimeout    = 2 * time.Minute
	relativePluginDir = ".cofide/plugins"
)

type SubCommand struct {
	BinaryName string
	Args       []string
	Timeout    time.Duration
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

func NewSubCommand(binary string, args []string) *SubCommand {
	return &SubCommand{
		BinaryName: binary,
		Args:       args,
		Timeout:    defaultTimeout,
	}
}

// Execute handles the execution of a subprocess with timeout and cleanup
func (s *SubCommand) Execute() error {
	pluginDir, err := GetPluginDir()
	if err != nil {
		return err
	}

	killErrors := make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	pluginPath := filepath.Join(pluginDir, s.BinaryName)
	cmd := exec.Command(pluginPath, s.Args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Setup process group kill on timeout
	go func() {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				s.killProcessGroup(cmd, killErrors)
			}
		case sig := <-sigChan:
			// Forward the signal to the process group
			if cmd.Process != nil {
				pgid, err := syscall.Getpgid(cmd.Process.Pid)
				if err == nil {
					_ = syscall.Kill(-pgid, sig.(syscall.Signal))
				}
			}
		}
	}()

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	signal.Stop(sigChan)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ProcessState.Exited() {
			return fmt.Errorf("process terminated by signal: %v", exitErr.ProcessState.String())
		}
		return fmt.Errorf("error executing %s: %w", s.BinaryName, err)
	}

	return nil
}

func (s *SubCommand) killProcessGroup(cmd *exec.Cmd, killErrors chan<- error) {
	if cmd.Process == nil {
		killErrors <- fmt.Errorf("process not started")
		return
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		killErrors <- fmt.Errorf("failed to get pgid: %w", err)
		return
	}

	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
		killErrors <- fmt.Errorf("failed to kill process group: %w", err)
		return
	}

	killErrors <- nil
}
