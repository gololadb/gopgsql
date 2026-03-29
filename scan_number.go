package pgscan

import "strings"

// scanNumber scans an integer, numeric, or real literal.
// If seenPoint is true, a leading '.' was already consumed.
func (s *Scanner) scanNumber(seenPoint bool) {
	ok := true
	var buf strings.Builder

	if seenPoint {
		buf.WriteByte('.')
		s.scanDecDigits(&buf)
	} else {
		if s.ch == '0' {
			buf.WriteByte('0')
			s.nextch()
			switch lower(s.ch) {
			case 'x':
				buf.WriteRune(s.ch)
				s.nextch()
				if !isHexDig(s.ch) && s.ch != '_' {
					s.errorf("invalid hexadecimal integer")
					ok = false
				} else {
					s.scanHexDigits(&buf)
				}
				s.checkNumericJunk(&ok)
				s.Tok = ICONST
				s.Lit = buf.String()
				s.Bad = !ok
				s.Kind = IntLit
				return
			case 'o':
				buf.WriteRune(s.ch)
				s.nextch()
				if !isOctDig(s.ch) && s.ch != '_' {
					s.errorf("invalid octal integer")
					ok = false
				} else {
					s.scanOctDigits(&buf)
				}
				s.checkNumericJunk(&ok)
				s.Tok = ICONST
				s.Lit = buf.String()
				s.Bad = !ok
				s.Kind = IntLit
				return
			case 'b':
				// Disambiguate: 0b could be binary integer prefix OR
				// the digit 0 followed by identifier 'b'. Check if the
				// char after 'b' is a binary digit or underscore.
				// But we also need to not confuse with 0B'...' (not valid
				// in PG — B prefix is only standalone). So just check for
				// binary digits.
				buf.WriteRune(s.ch)
				s.nextch()
				if !isBinDig(s.ch) && s.ch != '_' {
					s.errorf("invalid binary integer")
					ok = false
				} else {
					s.scanBinDigits(&buf)
				}
				s.checkNumericJunk(&ok)
				s.Tok = ICONST
				s.Lit = buf.String()
				s.Bad = !ok
				s.Kind = IntLit
				return
			default:
				// Plain decimal starting with 0 — fall through
			}
		}

		// Decimal integer part
		s.scanDecDigits(&buf)

		// Check for decimal point
		if s.ch == '.' {
			s.nextch()
			if s.ch == '.' {
				// "integer.." — return integer, synthesize DOT_DOT next.
				s.Tok = ICONST
				s.Lit = buf.String()
				s.Bad = !ok
				s.Kind = IntLit
				s.pendingDotDot = true
				return
			}
			buf.WriteByte('.')
			seenPoint = true
			s.scanDecDigits(&buf)
		}
	}

	// Exponent
	if lower(s.ch) == 'e' {
		buf.WriteRune(s.ch)
		s.nextch()
		if s.ch == '+' || s.ch == '-' {
			buf.WriteRune(s.ch)
			s.nextch()
		}
		if !isDigit(s.ch) {
			s.errorf("trailing junk after numeric literal")
			ok = false
		} else {
			s.scanDecDigits(&buf)
		}
		s.checkNumericJunk(&ok)
		s.Tok = FCONST
		s.Lit = buf.String()
		s.Bad = !ok
		s.Kind = FloatLit
		return
	}

	s.checkNumericJunk(&ok)

	if seenPoint {
		s.Tok = FCONST
		s.Lit = buf.String()
		s.Bad = !ok
		s.Kind = FloatLit
	} else {
		s.Tok = ICONST
		s.Lit = buf.String()
		s.Bad = !ok
		s.Kind = IntLit
	}
}

func (s *Scanner) scanDecDigits(buf *strings.Builder) {
	for isDigit(s.ch) || s.ch == '_' {
		buf.WriteRune(s.ch)
		s.nextch()
	}
}

func (s *Scanner) scanHexDigits(buf *strings.Builder) {
	for isHexDig(s.ch) || s.ch == '_' {
		buf.WriteRune(s.ch)
		s.nextch()
	}
}

func (s *Scanner) scanOctDigits(buf *strings.Builder) {
	for isOctDig(s.ch) || s.ch == '_' {
		buf.WriteRune(s.ch)
		s.nextch()
	}
}

func (s *Scanner) scanBinDigits(buf *strings.Builder) {
	for isBinDig(s.ch) || s.ch == '_' {
		buf.WriteRune(s.ch)
		s.nextch()
	}
}

// checkNumericJunk reports an error if an identifier char follows a number.
func (s *Scanner) checkNumericJunk(ok *bool) {
	if isIdentStart(s.ch) {
		s.errorf("trailing junk after numeric literal")
		*ok = false
		for isIdentCont(s.ch) {
			s.nextch()
		}
	}
}

// scanParam scans a positional parameter $N. The '$' has been consumed,
// s.ch is the first digit.
func (s *Scanner) scanParam() {
	var buf strings.Builder
	buf.WriteByte('$')
	for isDigit(s.ch) {
		buf.WriteRune(s.ch)
		s.nextch()
	}
	if isIdentStart(s.ch) {
		s.errorf("trailing junk after parameter")
		for isIdentCont(s.ch) {
			s.nextch()
		}
		s.Tok = PARAM
		s.Lit = buf.String()
		s.Bad = true
		return
	}
	s.Tok = PARAM
	s.Lit = buf.String()
	s.Bad = false
}
