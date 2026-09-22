package file

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aretext/aretext/text"
)

func TestSaveNewFile(t *testing.T) {
	tmpDir := t.TempDir()

	path := filepath.Join(tmpDir, "test.txt")
	saveAndAssertContents(t, path, "abcd1234", expectedPerm(0644))
}

func TestSaveModifyExistingFile(t *testing.T) {
	path := createTestFile(t, "old contents")
	saveAndAssertContents(t, path, "new contents", expectedPerm(0644))
}

func TestSaveModifyExistingFilePreservePermissions(t *testing.T) {
	path := createTestFile(t, "old contents")

	err := os.Chmod(path, 0600)
	require.NoError(t, err)
	saveAndAssertContents(t, path, "new contents", expectedPerm(0600))
}

func TestSavePathToSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")
	symlinkPath := filepath.Join(tmpDir, "testsymlink")

	// Create the target file.
	f, err := os.Create(targetPath)
	require.NoError(t, err)
	defer f.Close()
	_, err = io.WriteString(f, "test")
	require.NoError(t, err)

	// Create symlink to the target file.
	err = os.Symlink(targetPath, symlinkPath)
	if err != nil && runtime.GOOS == "windows" {
		// Creating a symlink on Windows requires either administrator
		// privileges or developer mode, so skip instead of failing.
		t.Skipf("Could not create symlink: %s", err)
	}
	require.NoError(t, err)

	// Save to the symlink path.
	saveAndAssertContents(t, symlinkPath, "new contents", expectedPerm(0644))

	// Verify that the symlink is still a symlink.
	fileInfo, err := os.Lstat(symlinkPath)
	require.NoError(t, err)
	assert.True(t, fileInfo.Mode()&os.ModeSymlink != 0)

	// Verify that the target file was modified.
	fileBytes, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, "new contents\n", string(fileBytes))
}

func TestSavePathToHardLink(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")
	hardlinkPath := filepath.Join(tmpDir, "testhardlink")

	// Create the target file.
	f, err := os.Create(targetPath)
	require.NoError(t, err)
	defer f.Close()
	_, err = io.WriteString(f, "test")
	require.NoError(t, err)

	// Create hardlink to the target file.
	err = os.Link(targetPath, hardlinkPath)
	require.NoError(t, err)

	// Save to the hardlink path.
	saveAndAssertContents(t, hardlinkPath, "new contents", expectedPerm(0644))

	// Verify that the target file was modified.
	fileBytes, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, "new contents\n", string(fileBytes))
}

func TestSaveLineEndings(t *testing.T) {
	testCases := []struct {
		name             string
		treeContents     string
		lineEndings      LineEndings
		expectedContents string
	}{
		{
			name:             "lf, empty",
			treeContents:     "",
			lineEndings:      LineEndingsLF,
			expectedContents: "\n",
		},
		{
			name:             "lf, multiple lines",
			treeContents:     "ab\ncd",
			lineEndings:      LineEndingsLF,
			expectedContents: "ab\ncd\n",
		},
		{
			name:             "crlf, empty",
			treeContents:     "",
			lineEndings:      LineEndingsCRLF,
			expectedContents: "\r\n",
		},
		{
			name:             "crlf, single line",
			treeContents:     "abcd",
			lineEndings:      LineEndingsCRLF,
			expectedContents: "abcd\r\n",
		},
		{
			name:             "crlf, multiple lines",
			treeContents:     "ab\ncd",
			lineEndings:      LineEndingsCRLF,
			expectedContents: "ab\r\ncd\r\n",
		},
		{
			name:             "crlf, empty lines",
			treeContents:     "ab\n\n\ncd",
			lineEndings:      LineEndingsCRLF,
			expectedContents: "ab\r\n\r\n\r\ncd\r\n",
		},
		{
			name:             "crlf, carriage return in content",
			treeContents:     "ab\rcd",
			lineEndings:      LineEndingsCRLF,
			expectedContents: "ab\rcd\r\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "test.txt")

			tree, err := text.NewTreeFromString(tc.treeContents)
			require.NoError(t, err)

			watcher, err := Save(path, tree, tc.lineEndings, testWatcherPollInterval)
			require.NoError(t, err)
			defer watcher.Stop()

			fileBytes, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedContents, string(fileBytes))

			// The watcher checksum must match the bytes on disk, otherwise
			// the editor would immediately report the file as changed.
			changed, err := watcher.CheckFileContentsChanged()
			require.NoError(t, err)
			assert.False(t, changed)
		})
	}
}

// TestSaveLoadRoundTripPreservesLineEndings checks that saving a document
// loaded from a CRLF file reproduces the original bytes exactly.
func TestSaveLoadRoundTripPreservesLineEndings(t *testing.T) {
	originalContents := "first\r\nsecond\r\n\r\nfourth\r\n"
	path := createTestFile(t, originalContents)

	tree, loadWatcher, lineEndings, err := Load(path, testWatcherPollInterval)
	require.NoError(t, err)
	defer loadWatcher.Stop()
	require.Equal(t, LineEndingsCRLF, lineEndings)
	require.Equal(t, "first\nsecond\n\nfourth", tree.String())

	saveWatcher, err := Save(path, tree, lineEndings, testWatcherPollInterval)
	require.NoError(t, err)
	defer saveWatcher.Stop()

	fileBytes, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, originalContents, string(fileBytes))
}

func saveAndAssertContents(t *testing.T, path string, contents string, perms os.FileMode) {
	tree, err := text.NewTreeFromString(contents)
	require.NoError(t, err)

	watcher, err := Save(path, tree, LineEndingsLF, testWatcherPollInterval)
	require.NoError(t, err)
	assert.Equal(t, path, watcher.Path())
	defer watcher.Stop()

	fileBytes, err := os.ReadFile(path)
	require.NoError(t, err)

	expectedContents := contents + "\n" // Append POSIX EOF
	assert.Equal(t, expectedContents, string(fileBytes))

	fileInfo, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, perms, fileInfo.Mode().Perm())
}

// expectedPerm translates unix file permissions to the permissions reported by the current OS.
func expectedPerm(unixPerm fs.FileMode) fs.FileMode {
	if runtime.GOOS == "windows" {
		// Windows distinguishes only read-write from read-only, and the
		// files in these tests are all writable, so os.Stat reports 0666.
		return fs.FileMode(0666)
	}
	return unixPerm
}
