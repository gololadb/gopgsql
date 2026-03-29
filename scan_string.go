package pgscan

import (
	"strings"
	"unicode/utf8"
)

// scanStdString scans a standard single-quoted string ('...').
// The opening quote has already been consumed.
func (s *Scanner) scanStdString() {
	ok := true
	var buf strings.Builder

	for {
		switch {
		case s.ch == '\'':
			s.nextch()
			if s.ch == '\'' {
				// Doubled quote — escaped single quote
				buf.WriteByte('\'')
				s.nextch()
				continue
			}
			s.Tok = SCONST
			s.Lit = buf.String()
			s.Bad = !ok
			s.Kind = StringLit
			return
		case s.ch < 0:
			s.errorf("unterminated quoted string")
			s.Tok = SCONST
			s.Lit = buf.String()
			s.Bad = true
			s.Kind = StringLit
			return
		default:
			buf.WriteRune(s.ch)
			s.nextch()
		}
	}
}

// scanExtString scans an E'...' extended string with backslash escapes.
// The E and opening quote have already been consumed.
func (s *Scanner) scanExtString() {
	ok := true
	var buf strings.Builder

	for {
		switch {
		case s.ch == '\'':
			s.nextch()
			if s.ch == '\'' {
				buf.WriteByte('\'')
				s.nextch()
				continue
			}
			s.Tok = SCONST
			s.Lit = buf.String()
			s.Bad = !ok
			s.Kind = StringLit
			return
		case s.ch == '\\':
			s.nextch()
			ch := s.unescapeChar(&ok)
			if ch != utf8.RuneError {
				buf.WriteRune(ch)
			}
		case s.ch < 0:
			s.errorf("unterminated quoted string")
			s.Tok = SCONST
			s.Lit = buf.String()
			s.Bad = true
			s.Kind = StringLit
			return
		default:
			buf.WriteRune(s.ch)
			s.nextch()
		}
	}
}

// unescapeChar processes a backslash escape in an E'' string.
// The backslash has been consumed; s.ch is the character after it.
func (s *Scanner) unescapeChar(ok *bool) rune {
	ch := s.ch
	s.nextch()
	switch ch {
	case 'b':
		return '\b'
	case 'f':
		return '\f'
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case 'v':
		return '\v'
	case '\'', '\\':
		return ch
	case '0', '1', '2', '3', '4', '5', '6', '7':
		val := int(ch - '0')
		for i := 1; i < 3 && isOctDig(s.ch); i++ {
			val = val*8 + int(s.ch-'0')
			s.nextch()
		}
		if val > 255 {
			s.errorf("octal escape value %d > 255", val)
			*ok = false
			return utf8.RuneError
		}
		return rune(val)
	case 'x':
		if !isHexDig(s.ch) {
			s.errorf("invalid hexadecimal escape")
			*ok = false
			return utf8.RuneError
		}
		val := hexVal(s.ch)
		s.nextch()
		if isHexDig(s.ch) {
			val = val*16 + hexVal(s.ch)
			s.nextch()
		}
		return rune(val)
	case 'u':
		return s.scanUnicodeEscape(4, ok)
	case 'U':
		return s.scanUnicodeEscape(8, ok)
	default:
		if ch < 0 {
			return utf8.RuneError
		}
		return ch
	}
}

func (s *Scanner) scanUnicodeEscape(ndigits int, ok *bool) rune {
	var val rune
	for i := 0; i < ndigits; i++ {
		if !isHexDig(s.ch) {
			s.errorf("invalid Unicode escape")
			*ok = false
			return utf8.RuneError
		}
		val = val*16 + rune(hexVal(s.ch))
		s.nextch()
	}
	if val > 0x10FFFF || (0xD800 <= val && val < 0xE000) {
		s.errorf("invalid Unicode escape value")
		*ok = false
		return utf8.RuneError
	}
	return val
}

func hexVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= lower(ch) && lower(ch) <= 'f':
		return int(lower(ch)-'a') + 10
	}
	return 0
}

// scanUnicodeString scans a U&'...' string. The U&' has been consumed.
func (s *Scanner) scanUnicodeString() {
	ok := true
	var buf strings.Builder

	for {
		switch {
		case s.ch == '\'':
			s.nextch()
			if s.ch == '\'' {
				buf.WriteByte('\'')
				s.nextch()
				continue
			}
			s.Tok = USCONST
			s.Lit = buf.String()
			s.Bad = !ok
			s.Kind = UStringLit
			return
		case s.ch < 0:
			s.errorf("unterminated quoted string")
			s.Tok = USCONST
			s.Lit = buf.String()
			s.Bad = true
			s.Kind = UStringLit
			return
		default:
			buf.WriteRune(s.ch)
			s.nextch()
		}
	}
}

// scanUnicodeIdent scans a U&"..." identifier. The U&" has been consumed.
func (s *Scanner) scanUnicodeIdent() {
	ok := true
	var buf strings.Builder

	for {
		switch {
		case s.ch == '"':
			s.nextch()
			if s.ch == '"' {
				buf.WriteByte('"')
				s.nextch()
				continue
			}
			if buf.Len() == 0 {
				s.errorf("zero-length delimited identifier")
				ok = false
			}
			s.Tok = UIDENT
			s.Lit = buf.String()
			s.Bad = !ok
			s.Kind = UIdentLit
			return
		case s.ch < 0:
			s.errorf("unterminated quoted identifier")
			s.Tok = UIDENT
			s.Lit = buf.String()
			s.Bad = true
			s.Kind = UIdentLit
			return
		default:
			buf.WriteRune(s.ch)
			s.nextch()
		}
	}
}

// scanDelimitedIdent scans a "..." double-quoted identifier.
// s.ch is '"' on entry.
func (s *Scanner) scanDelimitedIdent() {
	s.nextch() // consume opening "
	ok := true
	var buf strings.Builder

	for {
		switch {
		case s.ch == '"':
			s.nextch()
			if s.ch == '"' {
				buf.WriteByte('"')
				s.nextch()
				continue
			}
			if buf.Len() == 0 {
				s.errorf("zero-length delimited identifier")
				ok = false
			}
			s.Tok = IDENT
			s.Lit = buf.String()
			s.Bad = !ok
			s.Kind = IdentLit
			return
		case s.ch < 0:
			s.errorf("unterminated quoted identifier")
			s.Tok = IDENT
			s.Lit = buf.String()
			s.Bad = true
			s.Kind = IdentLit
			return
		default:
			buf.WriteRune(s.ch)
			s.nextch()
		}
	}
}

// scanBitString scans a B'...' bit string. B and opening ' already consumed.
func (s *Scanner) scanBitString() {
	ok := true
	var buf strings.Builder
	buf.WriteByte('b')

	for {
		switch {
		case s.ch == '\'':
			s.nextch()
			if s.ch == '\'' {
				buf.WriteByte('\'')
				s.nextch()
				continue
			}
			s.Tok = BCONST
			s.Lit = buf.String()
			s.Bad = !ok
			s.Kind = BitLit
			return
		case s.ch < 0:
			s.errorf("unterminated bit string literal")
			s.Tok = BCONST
			s.Lit = buf.String()
			s.Bad = true
			s.Kind = BitLit
			return
		default:
			buf.WriteRune(s.ch)
			s.nextch()
		}
	}
}

// scanHexString scans an X'...' hex string. X and opening ' already consumed.
func (s *Scanner) scanHexString() {
	ok := true
	var buf strings.Builder
	buf.WriteByte('x')

	for {
		switch {
		case s.ch == '\'':
			s.nextch()
			if s.ch == '\'' {
				buf.WriteByte('\'')
				s.nextch()
				continue
			}
			s.Tok = XCONST
			s.Lit = buf.String()
			s.Bad = !ok
			s.Kind = HexLit
			return
		case s.ch < 0:
			s.errorf("unterminated hexadecimal string literal")
			s.Tok = XCONST
			s.Lit = buf.String()
			s.Bad = true
			s.Kind = HexLit
			return
		default:
			buf.WriteRune(s.ch)
			s.nextch()
		}
	}
}

// isDolqIdent returns true for characters valid in a dollar-quote tag
// (letters, digits, underscore, high-byte — but NOT '$').
func isDolqIdent(ch rune) bool {
	if ch == '$' {
		return false
	}
	return isIdentStart(ch) || isDigit(ch)
}

// scanDollarString scans a $tag$...$tag$ string.
// The initial '$' has been consumed. s.ch is '$' (for $$) or an ident start.
func (s *Scanner) scanDollarString() {
	// Build opening delimiter
	var tag strings.Builder
	tag.WriteByte('$')
	if s.ch != '$' {
		for isDolqIdent(s.ch) {
			tag.WriteRune(s.ch)
			s.nextch()
		}
		if s.ch != '$' {
			// dolqfailed: not a valid dollar-quote, return bare '$'
			s.Tok = Token('$')
			s.Lit = "$"
			return
		}
	}
	tag.WriteByte('$')
	s.nextch() // consume closing '$' of opening delimiter
	delim := tag.String()

	var buf strings.Builder
	for s.ch >= 0 {
		if s.ch == '$' {
			// Potential closing delimiter — try to match
			s.nextch()
			var cand strings.Builder
			cand.WriteByte('$')
			for isDolqIdent(s.ch) {
				cand.WriteRune(s.ch)
				s.nextch()
			}
			if s.ch == '$' {
				cand.WriteByte('$')
				s.nextch()
				if cand.String() == delim {
					s.Tok = SCONST
					s.Lit = buf.String()
					s.Bad = false
					s.Kind = DollarLit
					return
				}
				// Not a match — add candidate text to body
				buf.WriteString(cand.String())
				continue
			}
			// No closing '$' — add partial match to body
			buf.WriteString(cand.String())
			continue
		}
		buf.WriteRune(s.ch)
		s.nextch()
	}

	s.errorf("unterminated dollar-quoted string")
	s.Tok = SCONST
	s.Lit = buf.String()
	s.Bad = true
	s.Kind = DollarLit
}
