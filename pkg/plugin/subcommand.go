package plugin

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

const (
	defaultTimeout = 2 * time.Minute
)

type SubCommand struct {
	BinaryName string
	Args       []string
	Timeout    time.Duration
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
	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()

	cmd := exec.Command(s.BinaryName, s.Args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Setup process group kill on timeout
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			pgid, err := syscall.Getpgid(cmd.Process.Pid)
			if err == nil {
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
			}
		}
	}()

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error executing %s: %w", s.BinaryName, err)
	}

	output := buf.String()
	fmt.Println(output)

	return nil

}
