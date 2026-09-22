package file

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	testCases := []struct {
		name                 string
		fileContents         string
		expectedTreeContents string
		expectedLineEndings  LineEndings
	}{
		{
			name:                 "empty",
			fileContents:         "",
			expectedTreeContents: "",
			expectedLineEndings:  LineEndingsLF,
		},
		{
			name:                 "ends with character, no POSIX eof",
			fileContents:         "ab\ncd",
			expectedTreeContents: "ab\ncd",
			expectedLineEndings:  LineEndingsLF,
		},
		{
			name:                 "POSIX eof",
			fileContents:         "abcd\n",
			expectedTreeContents: "abcd",
			expectedLineEndings:  LineEndingsLF,
		},
		{
			name:                 "no line endings",
			fileContents:         "abcd",
			expectedTreeContents: "abcd",
			expectedLineEndings:  LineEndingsLF,
		},
		{
			name:                 "CRLF line endings",
			fileContents:         "ab\r\ncd\r\n",
			expectedTreeContents: "ab\ncd",
			expectedLineEndings:  LineEndingsCRLF,
		},
		{
			name:                 "CRLF line endings, no eof indicator",
			fileContents:         "ab\r\ncd",
			expectedTreeContents: "ab\ncd",
			expectedLineEndings:  LineEndingsCRLF,
		},
		{
			name:                 "mixed line endings, first is CRLF",
			fileContents:         "ab\r\ncd\nef\n",
			expectedTreeContents: "ab\ncd\nef",
			expectedLineEndings:  LineEndingsCRLF,
		},
		{
			name:                 "mixed line endings, first is LF",
			fileContents:         "ab\ncd\r\nef\n",
			expectedTreeContents: "ab\ncd\nef",
			expectedLineEndings:  LineEndingsLF,
		},
		{
			name:                 "carriage return without line feed is content",
			fileContents:         "ab\rcd\n",
			expectedTreeContents: "ab\rcd",
			expectedLineEndings:  LineEndingsLF,
		},
		{
			name:                 "carriage return at end of file is content",
			fileContents:         "abcd\r",
			expectedTreeContents: "abcd\r",
			expectedLineEndings:  LineEndingsLF,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := createTestFile(t, tc.fileContents)

			tree, watcher, lineEndings, err := Load(filePath, time.Second)
			require.NoError(t, err)
			defer watcher.Stop()

			assert.Equal(t, tc.expectedTreeContents, tree.String())
			assert.Equal(t, tc.expectedLineEndings, lineEndings)
		})
	}
}
