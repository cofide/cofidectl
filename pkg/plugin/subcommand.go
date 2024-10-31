package plugin

import (
	"bytes"
	"context"
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

// ExecuteSubCommand handles the execution of a subprocess with timeout and cleanup
func ExecuteSubCommand(binary string, args []string) error {
	cmd := &SubCommand{
		BinaryName: binary,
		Args:       args,
		Timeout:    defaultTimeout,
	}
	return cmd.Execute()
}

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

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	signal.Stop(sigChan)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ProcessState.Exited() {
			return fmt.Errorf("process terminated by signal: %v", exitErr.ProcessState.String())
		}
		return fmt.Errorf("error executing %s: %w", s.BinaryName, err)
	}

	output := buf.String()
	fmt.Println(output)

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
