package backend

import (
	"bufio"
	"context"
	"os/exec"
)

// LineHandler is called for each line of combined output.
type LineHandler func(line string)

// StreamCmd runs a command and streams each line of combined output to handler.
// This gives real-time feedback for long-running install/build operations.
func StreamCmd(ctx context.Context, handler LineHandler, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Read stdout and stderr concurrently.
	done := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			handler(scanner.Text())
		}
		scanner = bufio.NewScanner(stderr)
		for scanner.Scan() {
			handler(scanner.Text())
		}
		close(done)
	}()

	<-done
	return cmd.Wait()
}