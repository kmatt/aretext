//go:build windows

package shellcmd

// defaultShellProg is the shell used when neither ARETEXT_SHELL nor SHELL is set.
// PowerShell is available on every supported version of Windows.
const defaultShellProg = "powershell.exe"

// clearTerminalCmd returns the command used to clear the terminal.
// There is no "clear" program on Windows, and "cls" is a cmd.exe builtin
// rather than an executable, so it has to be invoked through cmd.exe.
func clearTerminalCmd() (string, []string) {
	return "cmd.exe", []string{"/c", "cls"}
}
