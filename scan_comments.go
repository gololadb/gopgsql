package pgscan

// skipLineComment consumes everything after -- until end of line.
func (s *Scanner) skipLineComment() {
	for s.ch >= 0 && s.ch != '\n' {
		s.nextch()
	}
}

// skipBlockComment consumes a /* ... */ comment, supporting nesting.
// The opening /* has already been consumed.
func (s *Scanner) skipBlockComment() {
	depth := 1
	for s.ch >= 0 {
		if s.ch == '/' {
			s.nextch()
			if s.ch == '*' {
				s.nextch()
				depth++
			}
			continue
		}
		if s.ch == '*' {
			s.nextch()
			if s.ch == '/' {
				s.nextch()
				depth--
				if depth == 0 {
					return
				}
			}
			continue
		}
		s.nextch()
	}
	s.errorf("unterminated /* comment")
}
