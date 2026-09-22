Windows Support
===============

This document records how aretext builds and runs on Windows, and why the platform-specific code is structured the way it is. It resolves [issue #183](https://github.com/aretext/aretext/issues/183).

Windows 10 and Windows 11 are the same target as far as the build is concerned (`GOOS=windows`). Windows 11 only matters at runtime, where Windows Terminal is the default console host.

Building
--------

```
make build GO_OS=windows
```

This produces `aretext.exe`. On Windows itself, `go install` or `go build -o aretext.exe` also work. See [Install](docs/install.md) for end-user instructions.

Windows is built and tested by a separate `build-windows` CI job rather than by adding it to the existing matrix. The default `make` target depends on unix tooling (`goimports`, `markdownfmt`, `staticcheck`) and the committed-changes check uses bash-only syntax, so the Windows job runs `go build`, `go vet`, and `go test` directly.

Saving Files
------------

`file/save.go` holds the portable logic; the save strategy lives in `file/save_unix.go` and `file/save_windows.go`.

On unix, aretext writes a temporary file and renames it over the target using [renameio](https://github.com/google/renameio), except when the path is a hardlink, in which case it writes in place to avoid changing the inode.

On Windows, aretext always writes in place. This was the original cause of issue #183: the entire renameio API is guarded by `//go:build !windows`, so `renameio.NewPendingFile`, `renameio.WithPermissions`, and `renameio.WithExistingPermissions` were undefined. Writing in place rather than porting the rename is a deliberate choice:

-	`MoveFileEx` fails when another process holds the target open without `FILE_SHARE_DELETE`, which is common on Windows.
-	Writing in place preserves hardlinks and follows symlinks to their target, so neither needs to be detected. Hardlink detection on unix relies on `syscall.Stat_t`, which does not exist on Windows.

The tradeoff is that a save is not atomic on Windows. A crash mid-write can leave a partially written file.

Line Endings
------------

Line endings are detected on load and restored on save, so a file authored on Windows round-trips unchanged. See [Files](docs/files.md) for the user-facing description of the behavior.

Documents are stored in the text tree with LF line endings only. This keeps carriage returns out of every editor operation — cursor movement, syntax highlighting, search, and selection all continue to treat a line break as a single character.

The translation happens in `file/lineendings.go`:

-	`lfReader` wraps the file on load. It drops the carriage return of each CRLF pair and records the convention used by the first line. A carriage return that is not followed by a line feed is content, not a line ending, so it is preserved.
-	`crlfReader` wraps the text tree on save, expanding each line feed back to CRLF. Because it wraps the reader that already appended the POSIX end-of-file indicator, that final line feed is expanded too.

Both readers translate a byte at a time over a `bufio.Reader`, which is straightforward to verify against reads that split a CRLF pair across a chunk boundary. Loading an 8 MB CRLF file measured about 21% slower than loading the same file with LF line endings (35 ms versus 29 ms on an M4 Pro), which is not worth a bulk implementation for an interactive editor.

Two ordering constraints are worth preserving if this code is changed:

-	On load, the checksum is taken from the raw bytes on disk, before translation, because `file.Watcher` compares it against the file's contents.
-	On save, the checksum is taken after translation, for the same reason. If these are reversed, the editor reports the file as externally modified immediately after every save.

A document loaded from a file with mixed line endings is normalized to the convention of its first line when saved. Classic Mac-style (CR) line endings are not recognized.

The default config file is the one file aretext generates rather than loads, so it is written with `file.PlatformLineEndings` — CRLF on Windows. It belongs to the machine it was written on rather than to a repository shared across platforms, so other editors on that machine should display it correctly. This also normalizes the embedded copy, which picks up CRLF line endings when the repository is checked out with git's `autocrlf` enabled. Documents are never normalized this way; they keep whatever line endings they already had.

Glob Patterns
-------------

Configuration rule patterns and `hidePatterns` are matched by `file.GlobMatch`, which split paths on `os.PathSeparator`. On Windows that is `\`, so every pattern in the default config (`**/*.go`, `**/.git`, and so on) silently failed to match.

`splitPathComponents` now treats both `/` and `\` as separators on Windows. A backslash is a valid character in a file name on unix, so only `/` is a separator there. This keeps a single `app/default-config.yaml` for all platforms and lets user configs be written with either separator.

Shell Commands
--------------

`shellcmd/run_unix.go` and `shellcmd/run_windows.go` supply two platform-specific values:

-	`defaultShellProg` — `sh` on unix, `powershell.exe` on Windows. It is still overridden by `$ARETEXT_SHELL`, then `$SHELL`.
-	`clearTerminalCmd()` — there is no `clear` program on Windows, and `cls` is a `cmd.exe` builtin rather than an executable, so it is invoked as `cmd.exe /c cls`.

`shellArgs(prog)` in `shellcmd/run.go` returns the arguments that precede the command string. It is deliberately not platform-specific: PowerShell runs on Linux and macOS too, and `$ARETEXT_SHELL` can select it there, so the arguments are chosen from the program name rather than from `GOOS`. `/c` for `cmd.exe`, `-NoProfile -NonInteractive -Command` for `powershell.exe` and `pwsh.exe`, and `-c` for anything else (including the bash bundled with Git for Windows).

`-NonInteractive` is load-bearing. Without it, PowerShell prompts for a missing mandatory parameter rather than failing, and writes that prompt to the console rather than to the command's stdout — so the editor hangs on a prompt the user can neither see nor answer. This first showed up as `go test ./...` hanging on the clipboard tests, which run `cat > file`; in PowerShell that is `Get-Content` with no path, so it waited at `Path[0]:` forever.

`TestRunInPowerShellDoesNotPromptForMissingArgs` pins this down. It gives the command an `os.Pipe` as stdin so that a prompt has no end of input to fail on, and asserts that the command fails well before the context deadline. Removing `-NonInteractive` makes it fail after 20 seconds with "PowerShell appears to have waited at a prompt". An earlier version of the test passed a finite reader as stdin and passed either way, because the prompt failed on EOF instead of blocking — a finite stdin cannot detect this bug.

`Run` also sets `Cmd.WaitDelay`, for an unrelated hang that has nothing to do with Windows. When stdout is not an `*os.File`, `os/exec` copies it through a pipe and `Wait` blocks until the copy finishes, so a menu command as ordinary as `mycommand &` blocks the editor forever once the command exits but its background process keeps the pipe open. `WaitDelay` bounds that wait and reports `exec.ErrWaitDelay` instead. It does not help when `Stdin` is a reader that never ends: `os/exec` cannot interrupt the goroutine copying from it, and `awaitGoroutines` waits for that goroutine even after the delay expires. Every caller here passes nil, an `*os.File`, or a finite reader.

Paths in the UI
---------------

Child directory menu items are built with `os.PathSeparator` instead of a literal `/`, so they read `.\a\b` on Windows rather than `./a\b`.

Tests that compare paths are written with `/` separators and converted with `filepath.FromSlash`, since `ListDir` and the menus return paths using the separator of the current OS.

Tests that depend on a POSIX shell are skipped on Windows rather than ported. `state/shellcmd_test.go` drives the editor's shell integration with `printf`, `printenv`, and output redirection, and two clipboard tests use `cat` as a stand-in for a clipboard program. Rewriting these for PowerShell would test the shell rather than the editor, so `setupShellCmdTest` and `requirePosixShell` skip them. As a result the editor's shell integration has no automated coverage on Windows.

The `shellcmd` package itself is covered on every platform. `TestRunInPowerShell` looks up `pwsh` or `powershell.exe` on `$PATH`, selects it through `$ARETEXT_SHELL`, and skips only when neither is installed — so the Windows shell path can be exercised from a Linux or macOS machine with PowerShell installed, and must run on Windows, where `lookPowerShell` fails the test rather than skipping if PowerShell is missing.

Tests that are platform-aware for reasons unrelated to path separators:

-	Windows distinguishes only read-write from read-only, so `os.Stat` reports `0666` for the writable files used in the save tests.
-	Creating a symlink on Windows requires administrator privileges or developer mode, so `TestSavePathToSymlink` skips instead of failing when `os.Symlink` is not permitted.
-	PowerShell aliases `cat` to `Get-Content`, which cannot read from stdin, so the shell command test uses `[Console]::Out.Write([Console]::In.ReadToEnd())` on Windows.

Known Limitations
-----------------

-	There are no official Windows binaries. `RELEASE_PLATFORMS` in the Makefile is unchanged, and the release archives are `.tar.gz`, so publishing Windows builds needs a separate decision.
-	Saving is not atomic on Windows, as described above.
-	A document's line endings are detected automatically and cannot be configured or overridden. There is no way to convert a document between conventions from within the editor, and no indicator of which convention is in use.
