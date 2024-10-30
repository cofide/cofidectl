package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
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

	cmd := exec.Command(fmt.Sprintf("./%s", s.BinaryName), strings.Join(s.Args, ","))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Setup process group kill on timeout
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			// Kill the entire process group
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}()

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("executing %s: %w", s.BinaryName, err)
	}

	fmt.Println(string(out))
	return nil
}
