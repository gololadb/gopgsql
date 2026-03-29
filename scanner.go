package pgscan

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Scanner is a lexical tokenizer for PostgreSQL SQL source.
// After initialization, consecutive calls to Next advance one token at a time.
type Scanner struct {
	source

	// Current token, valid after calling Next().
	Line, Col uint
	Tok       Token
	Lit       string  // token text for IDENT, literals, Op, KEYWORD, PARAM
	Bad       bool    // true if a syntax error occurred in the current literal
	Kind      LitKind // valid when Tok is SCONST, ICONST, FCONST, BCONST, XCONST
	KwCat     KeywordCategory // valid when Tok is KEYWORD

	// pendingDotDot is set when we consumed a '.' after an integer but found
	// another '.', meaning the integer is followed by "..". We return the
	// integer first, then on the next call we synthesize DOT_DOT.
	pendingDotDot bool
}

func (s *Scanner) Init(src io.Reader, errh func(line, col uint, msg string)) {
	s.source.init(src, errh)
}

func (s *Scanner) errorf(format string, args ...any) {
	s.error(fmt.Sprintf(format, args...))
}

// setLit sets the scanner state for a literal token.
func (s *Scanner) setLit(tok Token, kind LitKind, ok bool) {
	s.Tok = tok
	s.Lit = string(s.segment())
	s.Bad = !ok
	s.Kind = kind
}

// --- Character classification ---

func isSpace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' || ch == '\v'
}

func isDigit(ch rune) bool  { return '0' <= ch && ch <= '9' }
func isHexDig(ch rune) bool { return isDigit(ch) || 'a' <= lower(ch) && lower(ch) <= 'f' }
func isOctDig(ch rune) bool { return '0' <= ch && ch <= '7' }
func isBinDig(ch rune) bool { return ch == '0' || ch == '1' }

func lower(ch rune) rune { return ('a' - 'A') | ch }

func isIdentStart(ch rune) bool {
	if ch >= utf8.RuneSelf {
		return unicode.IsLetter(ch)
	}
	return 'a' <= lower(ch) && lower(ch) <= 'z' || ch == '_'
}

func isIdentCont(ch rune) bool {
	if ch >= utf8.RuneSelf {
		return unicode.IsLetter(ch) || unicode.IsDigit(ch)
	}
	return isIdentStart(ch) || isDigit(ch) || ch == '$'
}

// isSelf returns true for single-character self tokens.
func isSelf(ch rune) bool {
	return strings.ContainsRune(",()[].;:|+-*/%^<>=", ch)
}

// isOpChar returns true for characters that can appear in operators.
func isOpChar(ch rune) bool {
	return strings.ContainsRune("~!@#^&|`?+-*/%<>=", ch)
}

// isNonSQLOpChar returns true for op_chars that are NOT standard SQL operators.
// Used to decide whether trailing +/- should be stripped.
func isNonSQLOpChar(ch byte) bool {
	return strings.ContainsRune("~!@#^&|`?%", rune(ch))
}

// Next advances the scanner by one token.
func (s *Scanner) Next() {
	// Handle synthetic DOT_DOT from "integer.." disambiguation.
	if s.pendingDotDot {
		s.pendingDotDot = false
		s.Line, s.Col = s.pos()
		s.Tok = DOT_DOT
		s.Lit = ".."
		// s.ch is currently the second '.'; consume it.
		s.nextch()
		return
	}

redo:
	s.stop()
	// skip whitespace
	for isSpace(s.ch) {
		s.nextch()
	}

	// skip -- line comments
	if s.ch == '-' {
		s.nextch()
		if s.ch == '-' {
			s.skipLineComment()
			goto redo
		}
		// not a comment — handle as operator or self token starting with '-'
		s.Line, s.Col = s.pos()
		s.start()
		if isOpChar(s.ch) {
			s.operatorFrom('-')
		} else {
			s.Tok = Token('-')
			s.Lit = "-"
		}
		return
	}

	// skip /* block comments */
	if s.ch == '/' {
		s.nextch()
		if s.ch == '*' {
			s.nextch()
			s.skipBlockComment()
			goto redo
		}
		// not a comment — handle as operator or self token starting with '/'
		s.Line, s.Col = s.pos()
		s.start()
		if isOpChar(s.ch) {
			s.operatorFrom('/')
		} else {
			s.Tok = Token('/')
			s.Lit = "/"
		}
		return
	}

	// record token position
	s.Line, s.Col = s.pos()
	s.start()

	// EOF
	if s.ch < 0 {
		s.Tok = EOF
		s.Lit = ""
		return
	}

	// Identifiers and keywords; also handles prefixed literals (B, X, E, N, U&)
	if isIdentStart(s.ch) {
		s.scanIdentOrPrefixed()
		return
	}

	// Numbers (starting with digit or leading dot)
	if isDigit(s.ch) {
		s.scanNumber(false)
		return
	}
	if s.ch == '.' {
		s.nextch()
		if isDigit(s.ch) {
			s.scanNumber(true)
			return
		}
		if s.ch == '.' {
			s.nextch()
			s.Tok = DOT_DOT
			s.Lit = ".."
			return
		}
		s.Tok = Token('.')
		s.Lit = "."
		return
	}

	// String literals
	if s.ch == '\'' {
		s.nextch() // consume opening quote
		s.scanStdString()
		return
	}

	// Double-quoted identifiers
	if s.ch == '"' {
		s.scanDelimitedIdent()
		return
	}

	// Dollar-quoted strings
	if s.ch == '$' {
		s.nextch()
		// Positional parameter: $1, $2, ...
		if isDigit(s.ch) {
			s.scanParam()
			return
		}
		// Dollar-quote delimiter: $$ or $tag$
		if s.ch == '$' || isIdentStart(s.ch) {
			s.scanDollarString()
			return
		}
		// bare $ is returned as single-char token
		s.Tok = Token('$')
		s.Lit = "$"
		return
	}

	// Typecast ::
	if s.ch == ':' {
		s.nextch()
		if s.ch == ':' {
			s.nextch()
			s.Tok = TYPECAST
			s.Lit = "::"
			return
		}
		if s.ch == '=' {
			s.nextch()
			s.Tok = COLON_EQUALS
			s.Lit = ":="
			return
		}
		s.Tok = Token(':')
		s.Lit = ":"
		return
	}

	// Multi-char operators starting with specific chars
	switch s.ch {
	case '=':
		s.nextch()
		if s.ch == '>' {
			s.nextch()
			s.Tok = EQUALS_GREATER
			s.Lit = "=>"
			return
		}
		if s.ch == '=' || isOpChar(s.ch) {
			s.operatorFrom('=')
			return
		}
		s.Tok = Token('=')
		s.Lit = "="
		return

	case '<':
		s.nextch()
		if s.ch == '=' {
			s.nextch()
			if !isOpChar(s.ch) {
				s.Tok = LESS_EQUALS
				s.Lit = "<="
				return
			}
			s.operatorFrom2('<', '=')
			return
		}
		if s.ch == '>' {
			s.nextch()
			if !isOpChar(s.ch) {
				s.Tok = LESS_GREATER
				s.Lit = "<>"
				return
			}
			s.operatorFrom2('<', '>')
			return
		}
		if isOpChar(s.ch) {
			s.operatorFrom('<')
			return
		}
		s.Tok = Token('<')
		s.Lit = "<"
		return

	case '>':
		s.nextch()
		if s.ch == '=' {
			s.nextch()
			if !isOpChar(s.ch) {
				s.Tok = GREATER_EQUALS
				s.Lit = ">="
				return
			}
			s.operatorFrom2('>', '=')
			return
		}
		if isOpChar(s.ch) {
			s.operatorFrom('>')
			return
		}
		s.Tok = Token('>')
		s.Lit = ">"
		return

	case '!':
		s.nextch()
		if s.ch == '=' {
			s.nextch()
			if !isOpChar(s.ch) {
				s.Tok = NOT_EQUALS
				s.Lit = "!="
				return
			}
			s.operatorFrom2('!', '=')
			return
		}
		if isOpChar(s.ch) {
			s.operatorFrom('!')
			return
		}
		// lone '!' — not a self token, return as Op
		s.Tok = Op
		s.Lit = "!"
		return
	}

	// Generic operator chars (~ @ # ^ & | ` ?)
	if isOpChar(s.ch) && !isSelf(s.ch) {
		first := s.ch
		s.nextch()
		if isOpChar(s.ch) {
			s.operatorFrom(byte(first))
			return
		}
		s.Tok = Op
		s.Lit = string(first)
		return
	}

	// Self tokens: , ( ) [ ] ; + - * / % ^ | (single char)
	if isSelf(s.ch) {
		ch := s.ch
		s.nextch()
		// Check for multi-char operator starting with a self char
		// that also happens to be an op_char (e.g. +, -, *, /, %, ^, <, >, =, |)
		if isOpChar(ch) && isOpChar(s.ch) {
			s.operatorFrom(byte(ch))
			return
		}
		s.Tok = Token(ch)
		s.Lit = string(ch)
		return
	}

	// Anything else
	s.errorf("invalid character %#U", s.ch)
	s.nextch()
	goto redo
}

