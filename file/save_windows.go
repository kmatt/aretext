//go:build windows

package file

import "io"

// saveFile writes the contents of the reader to the file at the given path.
//
// Unlike the unix implementation, this writes in-place instead of writing a
// temporary file and renaming it over the target. renameio does not support
// Windows (see https://github.com/google/renameio/pull/20), and MoveFileEx
// fails if another process has the target open without FILE_SHARE_DELETE,
// which is common on Windows. Writing in-place gives up atomicity, but it
// preserves hardlinks and follows symlinks to their target for free.
func saveFile(path string, r io.Reader) error {
	return saveDirectly(path, r)
}
