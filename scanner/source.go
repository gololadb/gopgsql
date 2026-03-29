package scanner

import (
	"io"
	"unicode/utf8"
)

// source is a buffered rune reader optimized for scanning SQL source.
// It is modeled after the Go compiler's source reader, with a sentinel-based
// ASCII fast path and segment capture for zero-copy literal extraction.
//
// The buffer uses three indices b, r, e:
//
//   - b (>= 0) marks the beginning of the current segment being captured.
//   - r points to the byte after the most recently read character ch.
//   - e points to the byte after the last valid byte in the buffer.
//
// buf[e] is always the sentinel (utf8.RuneSelf), enabling single-comparison
// ASCII fast path in nextch().
//
//	buf [...read...|...segment...|ch|...unread...|s|...free...]
//	     ^          ^             ^  ^            ^
//	     |          |             |  |            |
//	     b      r-chw             r  e         len(buf)
type source struct {
	in   io.Reader
	errh func(line, col uint, msg string)

	buf       []byte // source buffer
	ioerr     error  // pending I/O error, or nil
	b, r, e   int    // buffer indices
	line, col uint   // source position of ch (0-based)
	ch        rune   // most recently read character
	chw       int    // width of ch in bytes
}

const sentinel = utf8.RuneSelf

func (s *source) init(in io.Reader, errh func(line, col uint, msg string)) {
	s.in = in
	s.errh = errh

	if s.buf == nil {
		s.buf = make([]byte, nextSize(0))
	}
	s.buf[0] = sentinel
	s.ioerr = nil
	s.b, s.r, s.e = -1, 0, 0
	s.line, s.col = 0, 0
	s.ch = ' '
	s.chw = 0
}

const linebase = 1
const colbase = 1

// pos returns the 1-based (line, col) of the current character.
func (s *source) pos() (line, col uint) {
	return linebase + s.line, colbase + s.col
}

// error reports an error at the current character position.
func (s *source) error(msg string) {
	line, col := s.pos()
	s.errh(line, col, msg)
}

// start begins capturing a new segment (including s.ch).
func (s *source) start() { s.b = s.r - s.chw }

// stop ends segment capture.
func (s *source) stop() { s.b = -1 }

// segment returns the bytes of the current segment (excluding s.ch).
func (s *source) segment() []byte { return s.buf[s.b : s.r-s.chw] }

// rewind resets the read position to the start of the active segment.
// The segment must not contain newlines.
func (s *source) rewind() {
	if s.b < 0 {
		panic("no active segment")
	}
	s.col -= uint(s.r - s.b)
	s.r = s.b
	s.nextch()
}

// nextch advances to the next character.
func (s *source) nextch() {
redo:
	s.col += uint(s.chw)
	if s.ch == '\n' {
		s.line++
		s.col = 0
	}

	// Fast path: ASCII character
	if s.ch = rune(s.buf[s.r]); s.ch < sentinel {
		s.r++
		s.chw = 1
		if s.ch == 0 {
			s.error("invalid NUL character")
			goto redo
		}
		return
	}

	// Ensure we have enough bytes for a full rune
	for s.e-s.r < utf8.UTFMax && !utf8.FullRune(s.buf[s.r:s.e]) && s.ioerr == nil {
		s.fill()
	}

	// EOF
	if s.r == s.e {
		if s.ioerr != io.EOF {
			s.error("I/O error: " + s.ioerr.Error())
			s.ioerr = nil
		}
		s.ch = -1
		s.chw = 0
		return
	}

	s.ch, s.chw = utf8.DecodeRune(s.buf[s.r:s.e])
	s.r += s.chw

	if s.ch == utf8.RuneError && s.chw == 1 {
		s.error("invalid UTF-8 encoding")
		goto redo
	}

	// BOM only allowed at start of input
	const BOM = 0xfeff
	if s.ch == BOM {
		if s.line > 0 || s.col > 0 {
			s.error("invalid BOM in the middle of the file")
		}
		goto redo
	}
}

// fill reads more bytes into the buffer.
func (s *source) fill() {
	b := s.r
	if s.b >= 0 {
		b = s.b
		s.b = 0
	}
	content := s.buf[b:s.e]

	if len(content)*2 > len(s.buf) {
		s.buf = make([]byte, nextSize(len(s.buf)))
		copy(s.buf, content)
	} else if b > 0 {
		copy(s.buf, content)
	}
	s.r -= b
	s.e -= b

	for i := 0; i < 10; i++ {
		var n int
		n, s.ioerr = s.in.Read(s.buf[s.e : len(s.buf)-1])
		if n < 0 {
			panic("negative read")
		}
		if n > 0 || s.ioerr != nil {
			s.e += n
			s.buf[s.e] = sentinel
			return
		}
	}

	s.buf[s.e] = sentinel
	s.ioerr = io.ErrNoProgress
}

func nextSize(size int) int {
	const min = 4 << 10 // 4K
	const max = 1 << 20 // 1M
	if size < min {
		return min
	}
	if size <= max {
		return size << 1
	}
	return size + max
}
