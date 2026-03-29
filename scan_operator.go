package pgscan

import "strings"

// operatorFrom builds a multi-character operator. The first character has
// already been consumed and is passed as 'first'. s.ch is the next char
// (already known to be an op_char).
func (s *Scanner) operatorFrom(first byte) {
	var buf strings.Builder
	buf.WriteByte(first)
	for isOpChar(s.ch) {
		buf.WriteRune(s.ch)
		s.nextch()
	}
	s.finishOperator(buf.String())
}

// operatorFrom2 builds a multi-character operator from two already-consumed
// characters plus any following op_chars.
func (s *Scanner) operatorFrom2(c1, c2 byte) {
	var buf strings.Builder
	buf.WriteByte(c1)
	buf.WriteByte(c2)
	for isOpChar(s.ch) {
		buf.WriteRune(s.ch)
		s.nextch()
	}
	s.finishOperator(buf.String())
}

// finishOperator applies PostgreSQL's operator post-processing:
//  1. Truncate at embedded /* or -- (comment starts).
//  2. Strip trailing +/- unless the operator contains non-SQL op chars.
//  3. If the result is a single self char or a known 2-char token, return that.
func (s *Scanner) finishOperator(op string) {
	nchars := len(op)

	// Truncate at embedded comment starts (not at position 0)
	if idx := strings.Index(op[1:], "/*"); idx >= 0 {
		nchars = idx + 1
	}
	if idx := strings.Index(op[1:], "--"); idx >= 0 && idx+1 < nchars {
		nchars = idx + 1
	}

	// Strip trailing +/- unless a non-SQL op char is present
	if nchars > 1 && (op[nchars-1] == '+' || op[nchars-1] == '-') {
		hasNonSQL := false
		for i := 0; i < nchars-1; i++ {
			if isNonSQLOpChar(op[i]) {
				hasNonSQL = true
				break
			}
		}
		if !hasNonSQL {
			for nchars > 1 && (op[nchars-1] == '+' || op[nchars-1] == '-') {
				nchars--
			}
		}
	}

	op = op[:nchars]

	// Single self char
	if nchars == 1 && isSelf(rune(op[0])) {
		s.Tok = Token(rune(op[0]))
		s.Lit = op
		return
	}

	// Known 2-char tokens
	if nchars == 2 {
		switch op {
		case "=>":
			s.Tok = EQUALS_GREATER
			s.Lit = op
			return
		case ">=":
			s.Tok = GREATER_EQUALS
			s.Lit = op
			return
		case "<=":
			s.Tok = LESS_EQUALS
			s.Lit = op
			return
		case "<>":
			s.Tok = LESS_GREATER
			s.Lit = op
			return
		case "!=":
			s.Tok = NOT_EQUALS
			s.Lit = op
			return
		case "->":
			s.Tok = RIGHT_ARROW
			s.Lit = op
			return
		}
	}

	if len(op) >= 63 { // NAMEDATALEN
		s.errorf("operator too long")
	}

	s.Tok = Op
	s.Lit = op
}
