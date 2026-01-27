package sources

import (
	"fmt"
	"io"
	"os/exec"
)

type CommandSource struct {
	name string
	cmd  *exec.Cmd
}

func NewCommandSource(name string, command string, args ...string) *CommandSource {
	return &CommandSource{
		name: name,
		cmd:  exec.Command(command, args...),
	}
}

func (s *CommandSource) Stream() (io.Reader, error) {
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	if err := s.cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %v", err)
	}
	return stdout, nil
}

func (s *CommandSource) Close() error {
	if s.cmd != nil && s.cmd.Process != nil {
		// Try to kill the process
		return s.cmd.Process.Kill()
	}
	return nil
}

func (s *CommandSource) Name() string {
	return s.name
}
