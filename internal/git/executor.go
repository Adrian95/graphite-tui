package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// DefaultTimeout is the default command execution timeout
const DefaultTimeout = 30 * time.Second

// ExecutionConfig defines parameters for command execution
type ExecutionConfig struct {
	// Command is the base command to execute (e.g., "gt submit")
	Command string

	// Args are additional arguments passed safely (no shell injection)
	Args []string

	// SkipHooks adds --no-verify flag if true
	SkipHooks bool

	// Timeout for command execution (defaults to DefaultTimeout)
	Timeout time.Duration

	// Interactive runs command with tea.ExecProcess for terminal access
	Interactive bool

	// UseShell wraps command in shell execution
	UseShell bool

	// AllowUserEnv uses user's environment (for commands needing prompts)
	AllowUserEnv bool
}

// CommandExecutor handles all git/graphite command execution
type CommandExecutor struct {
	nonInteractiveEnv []string
}

// NewCommandExecutor creates a new executor with standard environment
func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{
		nonInteractiveEnv: buildNonInteractiveEnv(),
	}
}

// buildNonInteractiveEnv creates environment variables that prevent editor popups
func buildNonInteractiveEnv() []string {
	return append(os.Environ(),
		"GIT_EDITOR=true",
		"EDITOR=true",
		"VISUAL=true",
		"GT_NON_INTERACTIVE=1",
		"GIT_TERMINAL_PROMPT=0",
	)
}

// Execute runs a command synchronously and returns the result
func (e *CommandExecutor) Execute(ctx context.Context, cfg ExecutionConfig) CmdFinishedMsg {
	// Apply default timeout
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := e.buildCommand(cfg)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run with context for timeout support
	err := e.runWithContext(ctx, cmd)

	return CmdFinishedMsg{
		Output:  strings.TrimSpace(stdout.String()),
		Stderr:  strings.TrimSpace(stderr.String()),
		Err:     e.wrapError(cfg.Command, err, ctx.Err()),
		Command: cfg.Command,
	}
}

// ExecuteAsync returns a tea.Cmd that executes the command asynchronously
func (e *CommandExecutor) ExecuteAsync(cfg ExecutionConfig) tea.Cmd {
	return func() tea.Msg {
		return e.Execute(context.Background(), cfg)
	}
}

// ExecuteInteractive runs a command that needs terminal access
func (e *CommandExecutor) ExecuteInteractive(cfg ExecutionConfig) tea.Cmd {
	cmd := e.buildCommand(cfg)

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return CmdFinishedMsg{
			Err:     err,
			Command: cfg.Command,
		}
	})
}

// ExecuteChain runs multiple commands in sequence, stopping on first error
func (e *CommandExecutor) ExecuteChain(ctx context.Context, configs []ExecutionConfig) CmdFinishedMsg {
	var lastResult CmdFinishedMsg

	for _, cfg := range configs {
		lastResult = e.Execute(ctx, cfg)
		if lastResult.Err != nil {
			return lastResult
		}
	}

	return lastResult
}

// buildCommand constructs the exec.Cmd based on configuration
func (e *CommandExecutor) buildCommand(cfg ExecutionConfig) *exec.Cmd {
	commandStr := cfg.Command

	// Add hooks flag if needed
	if cfg.SkipHooks && strings.Contains(commandStr, "gt ") {
		commandStr += " --no-verify"
	}

	var cmd *exec.Cmd

	if cfg.UseShell {
		// Shell execution for complex commands
		if len(cfg.Args) > 0 {
			// Use parameterized shell script for safety
			script := commandStr + " \"$1\""
			cmd = exec.Command("sh", "-c", script, "--", cfg.Args[0])
		} else {
			cmd = exec.Command("sh", "-c", commandStr)
		}
	} else {
		// Direct execution (preferred for safety)
		parts := strings.Fields(commandStr)
		if len(cfg.Args) > 0 {
			parts = append(parts, cfg.Args...)
		}
		if len(parts) == 0 {
			cmd = exec.Command("true") // No-op
		} else {
			cmd = exec.Command(parts[0], parts[1:]...)
		}
	}

	// Set environment
	if cfg.AllowUserEnv {
		cmd.Env = os.Environ()
	} else {
		cmd.Env = e.nonInteractiveEnv
	}

	return cmd
}

// runWithContext executes command with context support
func (e *CommandExecutor) runWithContext(ctx context.Context, cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Timeout or cancellation
		_ = cmd.Process.Kill()
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// wrapError enriches errors with context
func (e *CommandExecutor) wrapError(cmdName string, execErr, ctxErr error) error {
	if execErr == nil {
		return nil
	}

	if ctxErr == context.DeadlineExceeded {
		return fmt.Errorf("%s: command timed out", cmdName)
	}

	if ctxErr == context.Canceled {
		return fmt.Errorf("%s: command canceled", cmdName)
	}

	return execErr
}

// --- Convenience Builders ---

// GTCommand creates a config for graphite commands
func GTCommand(subcommand string, skipHooks bool) ExecutionConfig {
	return ExecutionConfig{
		Command:   "gt " + subcommand,
		SkipHooks: skipHooks,
	}
}

// GTCommandWithArgs creates a config for graphite commands with arguments
func GTCommandWithArgs(subcommand string, args []string, skipHooks bool) ExecutionConfig {
	return ExecutionConfig{
		Command:   "gt " + subcommand,
		Args:      args,
		SkipHooks: skipHooks,
	}
}

// GitCommand creates a config for git commands
func GitCommand(args ...string) ExecutionConfig {
	return ExecutionConfig{
		Command: "git " + strings.Join(args, " "),
	}
}

// ShellCommand creates a config for shell-executed commands
func ShellCommand(cmd string, skipHooks bool) ExecutionConfig {
	return ExecutionConfig{
		Command:   cmd,
		SkipHooks: skipHooks,
		UseShell:  true,
	}
}
