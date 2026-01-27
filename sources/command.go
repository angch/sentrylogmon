package sources

import (
	"fmt"
	"io"
	"log"
	"os/exec"
)

type CommandSource struct {
	name    string
	command string
	args    []string
	cmd     *exec.Cmd
}

func NewCommandSource(name string, command string, args ...string) *CommandSource {
	return &CommandSource{
		name:    name,
		command: command,
		args:    args,
	}
}

func (s *CommandSource) Stream() (io.Reader, error) {
	// Create a new command instance for each stream start (allows restart)
	s.cmd = exec.Command(s.command, s.args...)

	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	if err := s.cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %v", err)
	}

	// Launch a goroutine to wait for the command to finish and reap the process
	go func() {
		if err := s.cmd.Wait(); err != nil {
			// Log the error if the command exits with an error
			// This helps debug why a monitor source might be restarting or failing
			log.Printf("Command source '%s' (%s) exited with error: %v", s.name, s.command, err)
		}
	}()

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
