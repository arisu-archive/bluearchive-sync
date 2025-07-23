package xdelta

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"
)

type Client struct {
	ExecutablePath string
	mu             sync.Mutex
	Stdin          io.Reader
	Stdout         io.Writer
	Stderr         io.Writer
}

func (c *Client) Encode(ctx context.Context, fromFile, toFile, patchFile string) error {
	encodeArgs := []string{"-e", "-s", fromFile, toFile, patchFile}
	slog.Debug("Encoding", "encodeArgs", encodeArgs)
	cmd, err := c.Command(ctx, encodeArgs...)
	if err != nil {
		return fmt.Errorf("failed to create encode command: %w", err)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute encode: %w", err)
	}
	return nil
}

func (c *Client) Decode(ctx context.Context, fromFile, toFile, patchFile string) error {
	decodeArgs := []string{"-d", "-f", "-s", fromFile, patchFile, toFile}
	slog.Debug("Decoding", "decodeArgs", decodeArgs)
	cmd, err := c.Command(ctx, decodeArgs...)
	if err != nil {
		return fmt.Errorf("failed to create decode command: %w", err)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute decode: %w", err)
	}
	return nil
}

func (c *Client) Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	slog.Debug("Executing xdelta command", "args", args)
	commandContext := exec.CommandContext

	var err error
	c.mu.Lock()
	if c.ExecutablePath == "" {
		c.ExecutablePath, err = resolveExecutablePath()
	}
	c.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve xdelta3 path: %w", err)
	}

	cmd := commandContext(ctx, c.ExecutablePath, args...)
	cmd.Stderr = c.Stderr
	cmd.Stdout = c.Stdout
	cmd.Stdin = c.Stdin
	return cmd, nil
}

func resolveExecutablePath() (string, error) {
	// List of executable names to try, in order of preference
	executables := []string{"xdelta3", "xdelta"}

	for _, executable := range executables {
		path, err := exec.LookPath(executable)
		if err == nil {
			return path, nil
		}
		if !errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("failed to resolve %s path: %w", executable, err)
		}
	}

	return "", ErrXdelta3NotFound
}
