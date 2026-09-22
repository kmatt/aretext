package file

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/aretext/aretext/text"
)

const defaultPermForNewFile fs.FileMode = 0644

// Save writes the text to disk and starts a new watcher to detect subsequent changes.
// This adds the POSIX end-of-file indicator (line feed at the end of the file).
// The document is stored with LF line endings, so if lineEndings is
// LineEndingsCRLF every line feed is translated to CRLF on the way to disk.
func Save(path string, tree *text.Tree, lineEndings LineEndings, watcherPollInterval time.Duration) (*Watcher, error) {
	// Compose a reader that appends the POSIX EOF indicator, restores the
	// original line endings, then calculates the checksum. The checksum must be
	// calculated last so that it matches the bytes written to disk.
	textReader := tree.ReaderAtPosition(0)
	posixEofReader := strings.NewReader("\n")
	var r io.Reader = io.MultiReader(&textReader, posixEofReader)
	if lineEndings == LineEndingsCRLF {
		r = newCrlfReader(r)
	}
	checksummer := NewChecksummer()
	r = io.TeeReader(r, checksummer)

	// Save the file using the strategy for the current platform
	// (see save_unix.go and save_windows.go).
	err := saveFile(path, r)
	if err != nil {
		return nil, err
	}

	// Start a new watcher for subsequent changes to the file.
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("os.Stat: %w", err)
	}
	watcher := NewWatcherForExistingFile(watcherPollInterval, path, fileInfo.ModTime(), fileInfo.Size(), checksummer.Checksum())

	return watcher, nil
}

func saveDirectly(path string, r io.Reader) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, defaultPermForNewFile)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}
	defer f.Close()

	// Write to the file.
	_, err = io.Copy(f, r)
	if err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}

	// Sync the file to disk so the watcher calculates the checksum correctly later.
	err = f.Sync()
	if err != nil {
		return fmt.Errorf("file.Sync: %w", err)
	}

	return nil
}
