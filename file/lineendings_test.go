package file

import (
	"io"
	"runtime"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLfReader(t *testing.T) {
	testCases := []struct {
		name                string
		input               string
		expectedOutput      string
		expectedLineEndings LineEndings
	}{
		{
			name:                "empty",
			input:               "",
			expectedOutput:      "",
			expectedLineEndings: LineEndingsLF,
		},
		{
			name:                "no line endings",
			input:               "abcd",
			expectedOutput:      "abcd",
			expectedLineEndings: LineEndingsLF,
		},
		{
			name:                "lf line endings",
			input:               "ab\ncd\n",
			expectedOutput:      "ab\ncd\n",
			expectedLineEndings: LineEndingsLF,
		},
		{
			name:                "crlf line endings",
			input:               "ab\r\ncd\r\n",
			expectedOutput:      "ab\ncd\n",
			expectedLineEndings: LineEndingsCRLF,
		},
		{
			name:                "crlf line ending at start",
			input:               "\r\nab",
			expectedOutput:      "\nab",
			expectedLineEndings: LineEndingsCRLF,
		},
		{
			name:                "consecutive crlf line endings",
			input:               "ab\r\n\r\n\r\ncd",
			expectedOutput:      "ab\n\n\ncd",
			expectedLineEndings: LineEndingsCRLF,
		},
		{
			name:                "mixed line endings, first is crlf",
			input:               "ab\r\ncd\nef",
			expectedOutput:      "ab\ncd\nef",
			expectedLineEndings: LineEndingsCRLF,
		},
		{
			name:                "mixed line endings, first is lf",
			input:               "ab\ncd\r\nef",
			expectedOutput:      "ab\ncd\nef",
			expectedLineEndings: LineEndingsLF,
		},
		{
			name:                "carriage return followed by character",
			input:               "ab\rcd",
			expectedOutput:      "ab\rcd",
			expectedLineEndings: LineEndingsLF,
		},
		{
			name:                "carriage return at end of input",
			input:               "abcd\r",
			expectedOutput:      "abcd\r",
			expectedLineEndings: LineEndingsLF,
		},
		{
			name:                "consecutive carriage returns before line feed",
			input:               "ab\r\r\ncd",
			expectedOutput:      "ab\r\ncd",
			expectedLineEndings: LineEndingsCRLF,
		},
		{
			name:                "only a carriage return",
			input:               "\r",
			expectedOutput:      "\r",
			expectedLineEndings: LineEndingsLF,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Read the input in a single chunk, then again one byte at a time
			// and in half-sized chunks, to exercise reads that split a CRLF pair.
			for wrapperName, wrapper := range readerWrappers() {
				t.Run(wrapperName, func(t *testing.T) {
					lr := newLfReader(wrapper(strings.NewReader(tc.input)))
					output, err := io.ReadAll(lr)
					require.NoError(t, err)
					assert.Equal(t, tc.expectedOutput, string(output))
					assert.Equal(t, tc.expectedLineEndings, lr.LineEndings())
				})
			}
		})
	}
}

func TestCrlfReader(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "empty",
			input:          "",
			expectedOutput: "",
		},
		{
			name:           "no line endings",
			input:          "abcd",
			expectedOutput: "abcd",
		},
		{
			name:           "single line feed",
			input:          "\n",
			expectedOutput: "\r\n",
		},
		{
			name:           "line feeds between characters",
			input:          "ab\ncd\n",
			expectedOutput: "ab\r\ncd\r\n",
		},
		{
			name:           "consecutive line feeds",
			input:          "ab\n\n\ncd",
			expectedOutput: "ab\r\n\r\n\r\ncd",
		},
		{
			name:           "carriage return in content",
			input:          "ab\rcd\n",
			expectedOutput: "ab\rcd\r\n",
		},
		{
			name:           "carriage return before line feed in content",
			input:          "ab\r\ncd",
			expectedOutput: "ab\r\r\ncd",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for wrapperName, wrapper := range readerWrappers() {
				t.Run(wrapperName, func(t *testing.T) {
					cr := newCrlfReader(wrapper(strings.NewReader(tc.input)))
					output, err := io.ReadAll(cr)
					require.NoError(t, err)
					assert.Equal(t, tc.expectedOutput, string(output))
				})
			}
		})
	}
}

// TestCrlfReaderSmallBuffer checks that a CRLF pair split across two reads
// is emitted correctly, including when the buffer holds only one byte.
func TestCrlfReaderSmallBuffer(t *testing.T) {
	cr := newCrlfReader(strings.NewReader("a\nb"))

	var output []byte
	buf := make([]byte, 1)
	for {
		n, err := cr.Read(buf)
		output = append(output, buf[:n]...)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	assert.Equal(t, "a\r\nb", string(output))
}

func TestNormalizeLineEndings(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		lineEndings    LineEndings
		expectedOutput string
	}{
		{
			name:           "empty",
			input:          "",
			lineEndings:    LineEndingsLF,
			expectedOutput: "",
		},
		{
			name:           "lf input to lf",
			input:          "ab\ncd\n",
			lineEndings:    LineEndingsLF,
			expectedOutput: "ab\ncd\n",
		},
		{
			name:           "lf input to crlf",
			input:          "ab\ncd\n",
			lineEndings:    LineEndingsCRLF,
			expectedOutput: "ab\r\ncd\r\n",
		},
		{
			name:           "crlf input to lf",
			input:          "ab\r\ncd\r\n",
			lineEndings:    LineEndingsLF,
			expectedOutput: "ab\ncd\n",
		},
		{
			name:           "crlf input to crlf",
			input:          "ab\r\ncd\r\n",
			lineEndings:    LineEndingsCRLF,
			expectedOutput: "ab\r\ncd\r\n",
		},
		{
			name:           "mixed input to crlf",
			input:          "ab\ncd\r\nef\n",
			lineEndings:    LineEndingsCRLF,
			expectedOutput: "ab\r\ncd\r\nef\r\n",
		},
		{
			name:           "mixed input to lf",
			input:          "ab\ncd\r\nef\n",
			lineEndings:    LineEndingsLF,
			expectedOutput: "ab\ncd\nef\n",
		},
		{
			name:           "carriage return that is not a line ending",
			input:          "ab\rcd\n",
			lineEndings:    LineEndingsCRLF,
			expectedOutput: "ab\rcd\r\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := NormalizeLineEndings([]byte(tc.input), tc.lineEndings)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedOutput, string(output))
		})
	}
}

// TestNormalizeLineEndingsIsIdempotent checks that normalizing an
// already-normalized value does not add more carriage returns.
func TestNormalizeLineEndingsIsIdempotent(t *testing.T) {
	for _, lineEndings := range []LineEndings{LineEndingsLF, LineEndingsCRLF} {
		t.Run(lineEndings.String(), func(t *testing.T) {
			once, err := NormalizeLineEndings([]byte("ab\ncd\r\nef"), lineEndings)
			require.NoError(t, err)
			twice, err := NormalizeLineEndings(once, lineEndings)
			require.NoError(t, err)
			assert.Equal(t, string(once), string(twice))
		})
	}
}

func TestPlatformLineEndings(t *testing.T) {
	if runtime.GOOS == "windows" {
		assert.Equal(t, LineEndingsCRLF, PlatformLineEndings)
	} else {
		assert.Equal(t, LineEndingsLF, PlatformLineEndings)
	}
}

// readerWrappers returns reader decorators that vary how many bytes each
// read returns, so that translation is exercised across chunk boundaries.
func readerWrappers() map[string]func(io.Reader) io.Reader {
	return map[string]func(io.Reader) io.Reader{
		"whole input": func(r io.Reader) io.Reader { return r },
		"one byte per read": func(r io.Reader) io.Reader {
			return iotest.OneByteReader(r)
		},
		"half buffer per read": func(r io.Reader) io.Reader {
			return iotest.HalfReader(r)
		},
		"data with err": func(r io.Reader) io.Reader {
			return iotest.DataErrReader(r)
		},
	}
}
