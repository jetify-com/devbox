package stream

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"

	"github.com/pkg/errors"
)

func RunCommand(w io.Writer, cmd *exec.Cmd) error {

	// Get a pipe to read from standard out
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return errors.New("unable to open stdout pipe")
	}

	// Use the same writer for standard error
	cmd.Stderr = cmd.Stdout

	// Make a new channel which will be used to ensure we get all output
	done := make(chan struct{})

	// Create a scanner which scans pipe in a line-by-line fashion
	scanner := bufio.NewScanner(pipe)

	// Use the scanner to scan the output line by line and log it
	// It's running in a goroutine so that it doesn't block
	go func() {

		// Read line by line and process it
		for scanner.Scan() {
			line := scanner.Text()
			// TODO savil. make the devbox.installNixProfile have a writer that inserts this tab.
			fmt.Fprintf(w, "\t%s\n", line)
		}

		// We're all done, unblock the channel
		done <- struct{}{}
	}()

	// Start the command and check for errors
	if err := cmd.Start(); err != nil {
		return errors.Errorf("error starting command %s: %v", cmd, err)
	}

	// Wait for all output to be processed
	<-done

	// Wait for the command to finish
	err = cmd.Wait()
	return errors.WithStack(err)
}
