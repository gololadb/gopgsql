package pgscan

import "strings"

// scanIdentOrPrefixed scans an identifier, keyword, or a prefixed literal
// (B'...', X'...', E'...', N'...', U&'...', U&"...").
func (s *Scanner) scanIdentOrPrefixed() {
	first := s.ch
	s.nextch()

	// Check for single-letter prefix + quote
	switch lower(first) {
	case 'b':
		if s.ch == '\'' {
			s.nextch()
			s.scanBitString()
			return
		}
	case 'x':
		if s.ch == '\'' {
			s.nextch()
			s.scanHexString()
			return
		}
	case 'e':
		if s.ch == '\'' {
			s.nextch()
			s.scanExtString()
			return
		}
	case 'n':
		if s.ch == '\'' {
			// National character string — Postgres treats N'...' as a standard string.
			s.scanStdString()
			return
		}
	case 'u':
		if s.ch == '&' {
			s.nextch()
			if s.ch == '\'' {
				s.nextch()
				s.scanUnicodeString()
				return
			}
			if s.ch == '"' {
				s.nextch()
				s.scanUnicodeIdent()
				return
			}
			// Not U&' or U&" — return "u" as identifier; '&' is already s.ch
			// and will be picked up on the next call to Next().
			s.returnIdent(string(first))
			return
		}
	}

	// Regular identifier — consume remaining ident chars
	for isIdentCont(s.ch) {
		s.nextch()
	}

	lit := string(s.segment())
	s.returnIdent(lit)
}

// returnIdent sets the token to IDENT or KEYWORD after lowercasing and
// checking the keyword table.
func (s *Scanner) returnIdent(lit string) {
	litLower := strings.ToLower(lit)
	if kw, ok := LookupKeyword(litLower); ok {
		s.Tok = KEYWORD
		s.Lit = litLower
		s.KwCat = kw.Category
		return
	}
	s.Tok = IDENT
	s.Lit = litLower
}
