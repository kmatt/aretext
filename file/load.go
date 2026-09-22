package file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aretext/aretext/text"
)

// Load reads a file from disk and starts a watcher to detect changes.
// This will remove the POSIX end-of-file indicator (line feed at end of file).
// CRLF line endings are translated to LF, and the detected line endings are
// returned so they can be restored when the document is saved.
func Load(path string, watcherPollInterval time.Duration) (*text.Tree, *Watcher, LineEndings, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, LineEndingsLF, fmt.Errorf("filepath.Abs: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, nil, LineEndingsLF, fmt.Errorf("os.Open: %w", err)
	}
	defer f.Close()

	lastModifiedTime, size, err := lastModifiedTimeAndSize(f)
	if err != nil {
		return nil, nil, LineEndingsLF, fmt.Errorf("lastModifiedTime: %w", err)
	}

	tree, lineEndings, checksum, err := readContentsAndChecksum(f)
	if err != nil {
		return nil, nil, LineEndingsLF, fmt.Errorf("readContentsAndChecksum: %w", err)
	}

	// POSIX files end with a single line feed to indicate the end of the file.
	// We remove it from the tree to simplify editor operations; we'll add it back when saving the file.
	removePosixEof(tree)

	watcher := NewWatcherForExistingFile(watcherPollInterval, path, lastModifiedTime, size, checksum)

	return tree, watcher, lineEndings, nil
}

func readContentsAndChecksum(f *os.File) (*text.Tree, LineEndings, string, error) {
	// The checksum is calculated from the bytes on disk, before line endings
	// are translated, so the watcher can compare it with the file's contents.
	checksummer := NewChecksummer()
	lr := newLfReader(io.TeeReader(f, checksummer))
	tree, err := text.NewTreeFromReader(lr)
	if err != nil {
		return nil, LineEndingsLF, "", fmt.Errorf("text.NewTreeFromReader: %w", err)
	}
	return tree, lr.LineEndings(), checksummer.Checksum(), nil
}

func lastModifiedTimeAndSize(f *os.File) (time.Time, int64, error) {
	fileInfo, err := f.Stat()
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("f.Stat: %w", err)
	}

	return fileInfo.ModTime(), fileInfo.Size(), nil
}

func removePosixEof(tree *text.Tree) {
	if endsWithLineFeed(tree) {
		lastPos := tree.NumChars() - 1
		tree.DeleteAtPosition(lastPos)
	}
}

func endsWithLineFeed(tree *text.Tree) bool {
	reader := tree.ReverseReaderAtPosition(tree.NumChars())
	var buf [1]byte
	if n, err := reader.Read(buf[:]); err != nil || n == 0 {
		return false
	}
	return buf[0] == '\n'
}
