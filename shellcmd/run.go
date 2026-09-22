package shellcmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// RunSilent runs the command and discards any output.
func RunSilent(ctx context.Context, cmd string, env []string) error {
	return Run(ctx, cmd, env, nil, nil, nil)
}

// RunInTerminal runs the command using inputs and outputs of the current process.
func RunInTerminal(ctx context.Context, cmd string, env []string) error {
	clearTerminal(ctx)
	return Run(ctx, cmd, env, os.Stdin, os.Stdout, os.Stderr)
}

// RunAndCaptureOutput runs the command and returns its stdout as a byte slice.
// If the output is not valid UTF-8 text, this returns an error.
func RunAndCaptureOutput(ctx context.Context, cmd string, env []string) (string, error) {
	var buf bytes.Buffer
	stdin, stdout, stderr := io.Reader(nil), &buf, io.Writer(nil)
	err := Run(ctx, cmd, env, stdin, stdout, stderr)
	if err != nil {
		return "", err
	}

	if !utf8.Valid(buf.Bytes()) {
		return "", fmt.Errorf("Shell command output is not valid UTF-8")
	}

	return buf.String(), nil
}

// Run executes a command with the configured shell.
func Run(ctx context.Context, shellCmd string, env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	prog := shellProg()
	args := append(shellArgs(prog), shellCmd)
	cmd := exec.CommandContext(ctx, prog, args...)
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// When stdout or stderr is not an *os.File, os/exec copies it through a
	// pipe on a background goroutine, and Wait blocks until that goroutine
	// finishes. A command that exits while a grandchild still holds the pipe
	// (say, a menu command that starts a background process) would otherwise
	// block the editor forever. WaitDelay bounds that wait, and also bounds
	// how long to wait for a child that ignores cancellation.
	//
	// This does not cover a Stdin reader that never reaches the end of its
	// input: os/exec cannot interrupt the goroutine copying from it, so Wait
	// blocks regardless of WaitDelay. Every caller here passes either nil,
	// an *os.File, or a finite reader.
	cmd.WaitDelay = waitDelay

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Cmd.Run: %w", err)
	}
	return nil
}

// waitDelay bounds how long to wait for a command's output to finish being
// copied after the command itself has exited or been cancelled. The timer
// starts only once the command is done, and draining a pipe takes microseconds,
// so this is short enough to keep the editor responsive without truncating
// the output of a command that exited normally.
const waitDelay = 2 * time.Second

func clearTerminal(ctx context.Context) {
	prog, args := clearTerminalCmd()
	clearCmd := exec.CommandContext(ctx, prog, args...)
	clearCmd.Stdout = os.Stdout
	clearCmd.Stderr = os.Stderr
	if err := clearCmd.Run(); err != nil {
		log.Printf("Error clearing screen: %v\n", err)
	}
}

// shellArgs returns the arguments that precede the command string.
// The shell is user-configurable through ARETEXT_SHELL and SHELL, and shells
// disagree about these arguments, so they are chosen from the program name
// rather than from the platform.
func shellArgs(prog string) []string {
	switch strings.TrimSuffix(strings.ToLower(filepath.Base(prog)), ".exe") {
	case "cmd":
		return []string{"/c"}

	case "powershell", "pwsh":
		// -NonInteractive is important: without it, PowerShell prompts for a
		// missing mandatory parameter instead of failing. The prompt is written
		// to the console rather than the command's stdout, so the editor would
		// appear to hang on a prompt the user cannot see or answer.
		//
		// -NoProfile skips the user's profile scripts. This matches "sh -c",
		// which does not read shell startup files either, and avoids paying
		// profile startup cost on every command.
		return []string{"-NoProfile", "-NonInteractive", "-Command"}

	default:
		// Assume a POSIX-compatible shell. "-c" works for sh, bash, zsh, fish,
		// and the bash bundled with Git for Windows.
		return []string{"-c"}
	}
}

// shellProg returns the shell program used to run commands.
// The platform-specific default can be overridden by the
// ARETEXT_SHELL or SHELL environment variables.
func shellProg() string {
	if s := os.Getenv("ARETEXT_SHELL"); s != "" {
		return s
	}

	if s := os.Getenv("SHELL"); s != "" {
		return s
	}

	return defaultShellProg
}
