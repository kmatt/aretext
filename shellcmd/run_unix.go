//go:build !windows

package shellcmd

// defaultShellProg is the shell used when neither ARETEXT_SHELL nor SHELL is set.
const defaultShellProg = "sh"

// clearTerminalCmd returns the command used to clear the terminal.
func clearTerminalCmd() (string, []string) {
	return "clear", nil
}
