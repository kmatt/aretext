package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aretext/aretext/file"
)

func TestDefaultConfigYamlValid(t *testing.T) {
	rs, err := unmarshalRuleSet(DefaultConfigYaml)
	require.NoError(t, err)
	assert.Greater(t, len(rs), 1)
	require.NoError(t, rs.Validate())

	c := rs.ConfigForPath("test.go")
	assert.Equal(t, "go", c.SyntaxLanguage)
	assert.Equal(t, 4, c.TabSize)
	assert.True(t, c.AutoIndent)
	assert.Equal(t, "olive", c.Styles["lineNum"].Color)
}

func TestSaveDefaultConfigUsesPlatformLineEndings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aretext", "config.yaml")
	require.NoError(t, saveDefaultConfig(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	// The written config must still parse and validate.
	rs, err := unmarshalRuleSet(data)
	require.NoError(t, err)
	require.NoError(t, rs.Validate())

	// Line endings must match the platform, so that other editors on the
	// same machine display the config correctly.
	expected, err := file.NormalizeLineEndings(DefaultConfigYaml, file.PlatformLineEndings)
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(data))

	if runtime.GOOS == "windows" {
		assert.Contains(t, string(data), "\r\n")
	} else {
		assert.NotContains(t, string(data), "\r")
	}
}

// TestDefaultConfigYamlValidWithCrlf checks that the config written on Windows
// still parses. This runs on every platform so the Windows output is covered
// without needing a Windows host.
func TestDefaultConfigYamlValidWithCrlf(t *testing.T) {
	data, err := file.NormalizeLineEndings(DefaultConfigYaml, file.LineEndingsCRLF)
	require.NoError(t, err)
	require.Contains(t, string(data), "\r\n")

	rs, err := unmarshalRuleSet(data)
	require.NoError(t, err)
	assert.Greater(t, len(rs), 1)
	require.NoError(t, rs.Validate())

	c := rs.ConfigForPath("test.go")
	assert.Equal(t, "go", c.SyntaxLanguage)
	assert.Equal(t, 4, c.TabSize)
}
