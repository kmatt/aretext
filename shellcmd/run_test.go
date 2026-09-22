package shellcmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunWithStdinAndStdout(t *testing.T) {
	// Run the command in the default shell for this platform,
	// not whatever shell the developer happens to be using.
	t.Setenv("ARETEXT_SHELL", "")
	t.Setenv("SHELL", "")

	var stdout bytes.Buffer
	err := Run(testContext(t), copyStdinToStdoutCmd(defaultShellProg), nil, strings.NewReader("abcd"), &stdout, nil)
	require.NoError(t, err)
	assert.Equal(t, "abcd", stdout.String())
}

// TestRunInPowerShell exercises the PowerShell code path, which is the default
// on Windows but can be selected on any platform through ARETEXT_SHELL.
func TestRunInPowerShell(t *testing.T) {
	prog := lookPowerShell(t)
	t.Setenv("ARETEXT_SHELL", prog)

	var stdout bytes.Buffer
	err := Run(testContext(t), copyStdinToStdoutCmd(prog), nil, strings.NewReader("abcd"), &stdout, nil)
	require.NoError(t, err)
	assert.Equal(t, "abcd", stdout.String())
}

// TestRunInPowerShellDoesNotPromptForMissingArgs is a regression test for a
// hang. "cat > file" in PowerShell is Get-Content with no path, so without
// -NonInteractive it prompts on the console, which the editor cannot answer.
// The command must fail instead.
func TestRunInPowerShellDoesNotPromptForMissingArgs(t *testing.T) {
	prog := lookPowerShell(t)
	t.Setenv("ARETEXT_SHELL", prog)

	path := filepath.Join(t.TempDir(), "out.txt")
	cmd := fmt.Sprintf("Get-Content > %s", path)

	// Give the command a stdin that produces nothing and never reaches the
	// end of input, so that a prompt blocks instead of failing on EOF.
	// This is an *os.File, so os/exec passes the descriptor to the child
	// directly and Wait does not wait on a goroutine copying from it.
	stdinReader, stdinWriter, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() {
		stdinReader.Close()
		stdinWriter.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	start := time.Now()
	err = Run(ctx, cmd, nil, stdinReader, nil, nil)
	elapsed := time.Since(start)

	// The command must fail because the parameter is missing. If PowerShell
	// is allowed to prompt for it, the command instead sits at the prompt
	// until the context deadline.
	require.Error(t, err)
	assert.Less(t, elapsed, 10*time.Second, "PowerShell appears to have waited at a prompt")
}

// TestRunReturnsWhenCommandLeaksOutputPipe checks that Run returns when a
// command exits but leaves a background process holding its stdout. Without
// Cmd.WaitDelay, os/exec's Wait blocks until that process exits, which hangs
// the editor on a menu command as ordinary as "mycommand &".
func TestRunReturnsWhenCommandLeaksOutputPipe(t *testing.T) {
	requirePosixShell(t)
	t.Setenv("ARETEXT_SHELL", "")
	t.Setenv("SHELL", "")

	// Exit immediately, leaving a grandchild that holds stdout open for
	// much longer than waitDelay.
	cmd := "(sleep 60 >&1 &) ; exit 0"

	var stdout bytes.Buffer
	start := time.Now()
	err := Run(testContext(t), cmd, nil, nil, &stdout, nil)

	// The command itself succeeded, so the only error should be the one
	// reporting that output copying was cut short.
	require.ErrorIs(t, err, exec.ErrWaitDelay)
	assert.Less(t, time.Since(start), 20*time.Second)
}

// TestShellArgsPassCommandLast checks the invariant that the command string is
// appended after the shell's own arguments.
func TestShellArgsPassCommandLast(t *testing.T) {
	progs := []string{
		"sh", "bash", "zsh", "fish",
		"powershell.exe", "pwsh", "pwsh.exe", "cmd.exe",
		`C:\Program Files\Git\usr\bin\bash.exe`,
	}
	for _, prog := range progs {
		t.Run(prog, func(t *testing.T) {
			args := shellArgs(prog)
			require.NotEmpty(t, args)

			last := args[len(args)-1]
			assert.Contains(t, []string{"-c", "-Command", "/c"}, last)
		})
	}
}

// TestShellArgsNonInteractive checks that PowerShell is told not to prompt,
// on every platform, since ARETEXT_SHELL can select it anywhere.
func TestShellArgsNonInteractive(t *testing.T) {
	progs := []string{"powershell", "powershell.exe", "pwsh", "pwsh.exe", "PowerShell.EXE"}
	for _, prog := range progs {
		t.Run(prog, func(t *testing.T) {
			assert.Contains(t, shellArgs(prog), "-NonInteractive")
		})
	}
}

// testContext bounds a command so that a shell waiting on input fails the
// test instead of hanging until the go test timeout.
func testContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// lookPowerShell returns the path to PowerShell, skipping the test if it is
// not installed. PowerShell is always present on Windows.
func lookPowerShell(t *testing.T) string {
	for _, prog := range []string{"pwsh", "powershell.exe"} {
		if path, err := exec.LookPath(prog); err == nil {
			return path
		} else if runtime.GOOS == "windows" {
			t.Fatalf("Could not find %s on Windows: %s", prog, err)
		}
	}
	t.Skip("PowerShell is not installed")
	return ""
}

// requirePosixShell skips a test that depends on POSIX shell syntax.
func requirePosixShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Test requires a POSIX shell")
	}
}

// copyStdinToStdoutCmd returns a shell command that copies stdin to stdout,
// for the given shell program.
func copyStdinToStdoutCmd(prog string) string {
	if isPowerShell(prog) {
		// PowerShell aliases "cat" to Get-Content, which cannot read from stdin.
		return "[Console]::Out.Write([Console]::In.ReadToEnd())"
	}
	return "cat"
}

func isPowerShell(prog string) bool {
	prog = strings.ToLower(prog)
	return strings.Contains(prog, "powershell") || strings.Contains(prog, "pwsh")
}
