package file

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

// LineEndings represents the characters that terminate each line in a file.
type LineEndings int

const (
	// LineEndingsLF terminates lines with a line feed ("\n").
	// This is the convention on Linux, macOS, and BSD.
	LineEndingsLF = LineEndings(iota)

	// LineEndingsCRLF terminates lines with a carriage return and line feed ("\r\n").
	// This is the convention on Windows.
	LineEndingsCRLF
)

// NormalizeLineEndings translates every line ending in data to the given convention.
//
// This is for files that aretext generates for the local machine, such as the
// default config file. Documents are not normalized; they keep the line endings
// they were loaded with. See Load and Save.
func NormalizeLineEndings(data []byte, lineEndings LineEndings) ([]byte, error) {
	// Translate to LF first so that the input can use either convention.
	normalized, err := io.ReadAll(newLfReader(bytes.NewReader(data)))
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}

	if lineEndings != LineEndingsCRLF {
		return normalized, nil
	}

	normalized, err = io.ReadAll(newCrlfReader(bytes.NewReader(normalized)))
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}
	return normalized, nil
}

// String returns the name of the line ending convention.
func (le LineEndings) String() string {
	switch le {
	case LineEndingsCRLF:
		return "crlf"
	default:
		return "lf"
	}
}

// lfReader translates CRLF line endings to LF while reading.
//
// Documents are stored in the text tree with LF line endings so that editor
// operations don't have to handle a carriage return as part of a line break.
// The original line endings are restored when the document is saved.
//
// A carriage return that is not followed by a line feed is not a line ending,
// so it is preserved as document content.
type lfReader struct {
	r           *bufio.Reader
	lineEndings LineEndings
	detected    bool
}

func newLfReader(r io.Reader) *lfReader {
	return &lfReader{r: bufio.NewReader(r)}
}

// LineEndings returns the line endings used by the first line in the input.
// If the input has no line endings, this returns LineEndingsLF.
// This is valid only after the input has been read.
func (lr *lfReader) LineEndings() LineEndings {
	return lr.lineEndings
}

// Read implements io.Reader#Read()
func (lr *lfReader) Read(p []byte) (int, error) {
	var n int
	for n < len(p) {
		b, err := lr.r.ReadByte()
		if err != nil {
			if n > 0 && errors.Is(err, io.EOF) {
				return n, nil
			}
			return n, err
		}

		if b == '\r' {
			// Look ahead one byte to check whether this is a CRLF line ending.
			// Peek returns an error at the end of the input, in which case
			// the carriage return is content rather than a line ending.
			if next, err := lr.r.Peek(1); err == nil && next[0] == '\n' {
				lr.detect(LineEndingsCRLF)
				continue // Drop the carriage return; the line feed is read next.
			}
		} else if b == '\n' {
			lr.detect(LineEndingsLF)
		}

		p[n] = b
		n++
	}
	return n, nil
}

// detect records the line endings of the first line in the input.
func (lr *lfReader) detect(lineEndings LineEndings) {
	if !lr.detected {
		lr.lineEndings = lineEndings
		lr.detected = true
	}
}

// crlfReader translates LF line endings to CRLF while reading.
// This restores the line endings of a document that was loaded from a file
// using the Windows convention.
type crlfReader struct {
	r         *bufio.Reader
	pendingLf bool
}

func newCrlfReader(r io.Reader) *crlfReader {
	return &crlfReader{r: bufio.NewReader(r)}
}

// Read implements io.Reader#Read()
func (cr *crlfReader) Read(p []byte) (int, error) {
	var n int
	for n < len(p) {
		if cr.pendingLf {
			// Emit the line feed of a CRLF pair that did not fit in the
			// buffer during the previous read.
			p[n] = '\n'
			cr.pendingLf = false
			n++
			continue
		}

		b, err := cr.r.ReadByte()
		if err != nil {
			if n > 0 && errors.Is(err, io.EOF) {
				return n, nil
			}
			return n, err
		}

		if b == '\n' {
			p[n] = '\r'
			n++
			cr.pendingLf = true
			continue
		}

		p[n] = b
		n++
	}
	return n, nil
}
