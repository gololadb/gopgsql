package scanner

import (
	"strings"
	"testing"
)

// helper to scan all tokens from input
func scanAll(t *testing.T, input string) []Scanner {
	t.Helper()
	var s Scanner
	var errs []string
	s.Init(strings.NewReader(input), func(line, col uint, msg string) {
		errs = append(errs, msg)
	})
	var tokens []Scanner
	for {
		s.Next()
		// snapshot the state
		snap := s
		tokens = append(tokens, snap)
		if s.Tok == EOF {
			break
		}
	}
	if len(errs) > 0 {
		t.Logf("scanner errors: %v", errs)
	}
	return tokens
}

func expectToken(t *testing.T, got Scanner, tok Token, lit string) {
	t.Helper()
	if got.Tok != tok {
		t.Errorf("expected token %v, got %v (lit=%q)", tok, got.Tok, got.Lit)
	}
	if got.Lit != lit {
		t.Errorf("expected lit %q, got %q (tok=%v)", lit, got.Lit, got.Tok)
	}
}

func expectSelfToken(t *testing.T, got Scanner, ch rune) {
	t.Helper()
	if got.Tok != Token(ch) {
		t.Errorf("expected self token %q, got tok=%v lit=%q", string(ch), got.Tok, got.Lit)
	}
}

// --- Whitespace and comments ---

func TestWhitespace(t *testing.T) {
	tokens := scanAll(t, "   \t\n\r  ")
	if len(tokens) != 1 || tokens[0].Tok != EOF {
		t.Errorf("expected only EOF, got %d tokens", len(tokens))
	}
}

func TestLineComment(t *testing.T) {
	tokens := scanAll(t, "-- this is a comment\n42")
	expectToken(t, tokens[0], ICONST, "42")
}

func TestBlockComment(t *testing.T) {
	tokens := scanAll(t, "/* comment */ 42")
	expectToken(t, tokens[0], ICONST, "42")
}

func TestNestedBlockComment(t *testing.T) {
	tokens := scanAll(t, "/* outer /* inner */ still comment */ 42")
	expectToken(t, tokens[0], ICONST, "42")
}

// --- Identifiers and keywords ---

func TestSimpleIdent(t *testing.T) {
	tokens := scanAll(t, "foo bar_baz _x")
	expectToken(t, tokens[0], IDENT, "foo")
	expectToken(t, tokens[1], IDENT, "bar_baz")
	expectToken(t, tokens[2], IDENT, "_x")
}

func TestIdentCaseFolding(t *testing.T) {
	tokens := scanAll(t, "FooBar")
	expectToken(t, tokens[0], IDENT, "foobar")
}

func TestKeyword(t *testing.T) {
	tokens := scanAll(t, "SELECT from WHERE")
	expectToken(t, tokens[0], KEYWORD, "select")
	if tokens[0].KwCat != ReservedKeyword {
		t.Errorf("expected ReservedKeyword, got %v", tokens[0].KwCat)
	}
	expectToken(t, tokens[1], KEYWORD, "from")
	expectToken(t, tokens[2], KEYWORD, "where")
}

func TestUnreservedKeyword(t *testing.T) {
	tokens := scanAll(t, "abort")
	expectToken(t, tokens[0], KEYWORD, "abort")
	if tokens[0].KwCat != UnreservedKeyword {
		t.Errorf("expected UnreservedKeyword, got %v", tokens[0].KwCat)
	}
}

// --- String literals ---

func TestSimpleString(t *testing.T) {
	tokens := scanAll(t, "'hello world'")
	expectToken(t, tokens[0], SCONST, "hello world")
}

func TestStringWithEscapedQuote(t *testing.T) {
	tokens := scanAll(t, "'it''s'")
	expectToken(t, tokens[0], SCONST, "it's")
}

func TestEmptyString(t *testing.T) {
	tokens := scanAll(t, "''")
	expectToken(t, tokens[0], SCONST, "")
}

func TestExtendedString(t *testing.T) {
	tokens := scanAll(t, `E'hello\nworld'`)
	expectToken(t, tokens[0], SCONST, "hello\nworld")
}

func TestExtendedStringOctal(t *testing.T) {
	tokens := scanAll(t, `E'\101'`) // \101 = 'A'
	expectToken(t, tokens[0], SCONST, "A")
}

func TestExtendedStringHex(t *testing.T) {
	tokens := scanAll(t, `E'\x41'`) // \x41 = 'A'
	expectToken(t, tokens[0], SCONST, "A")
}

func TestExtendedStringUnicode(t *testing.T) {
	tokens := scanAll(t, `E'\u0041'`) // \u0041 = 'A'
	expectToken(t, tokens[0], SCONST, "A")
}

func TestDollarQuotedString(t *testing.T) {
	tokens := scanAll(t, "$$hello world$$")
	expectToken(t, tokens[0], SCONST, "hello world")
}

func TestDollarQuotedStringWithTag(t *testing.T) {
	tokens := scanAll(t, "$body$hello world$body$")
	expectToken(t, tokens[0], SCONST, "hello world")
}

func TestDollarQuotedStringNested(t *testing.T) {
	tokens := scanAll(t, "$outer$contains $$inner$$ text$outer$")
	expectToken(t, tokens[0], SCONST, "contains $$inner$$ text")
}

func TestUnicodeString(t *testing.T) {
	tokens := scanAll(t, "U&'hello'")
	expectToken(t, tokens[0], USCONST, "hello")
}

// --- Bit and hex strings ---

func TestBitString(t *testing.T) {
	tokens := scanAll(t, "B'1010'")
	expectToken(t, tokens[0], BCONST, "b1010")
}

func TestHexString(t *testing.T) {
	tokens := scanAll(t, "X'FF'")
	expectToken(t, tokens[0], XCONST, "xFF")
}

// --- Delimited identifiers ---

func TestDelimitedIdent(t *testing.T) {
	tokens := scanAll(t, `"MyTable"`)
	expectToken(t, tokens[0], IDENT, "MyTable")
}

func TestDelimitedIdentEscapedQuote(t *testing.T) {
	tokens := scanAll(t, `"say""hello"`)
	expectToken(t, tokens[0], IDENT, `say"hello`)
}

func TestUnicodeIdent(t *testing.T) {
	tokens := scanAll(t, `U&"foo"`)
	expectToken(t, tokens[0], UIDENT, "foo")
}

// --- Numbers ---

func TestDecimalInteger(t *testing.T) {
	tokens := scanAll(t, "42")
	expectToken(t, tokens[0], ICONST, "42")
	if tokens[0].Kind != IntLit {
		t.Errorf("expected IntLit, got %v", tokens[0].Kind)
	}
}

func TestHexInteger(t *testing.T) {
	tokens := scanAll(t, "0xFF")
	expectToken(t, tokens[0], ICONST, "0xFF")
}

func TestOctalInteger(t *testing.T) {
	tokens := scanAll(t, "0o77")
	expectToken(t, tokens[0], ICONST, "0o77")
}

func TestBinaryInteger(t *testing.T) {
	tokens := scanAll(t, "0b1010")
	expectToken(t, tokens[0], ICONST, "0b1010")
}

func TestIntegerWithUnderscores(t *testing.T) {
	tokens := scanAll(t, "1_000_000")
	expectToken(t, tokens[0], ICONST, "1_000_000")
}

func TestNumeric(t *testing.T) {
	tokens := scanAll(t, "3.14")
	expectToken(t, tokens[0], FCONST, "3.14")
	if tokens[0].Kind != FloatLit {
		t.Errorf("expected FloatLit, got %v", tokens[0].Kind)
	}
}

func TestNumericLeadingDot(t *testing.T) {
	tokens := scanAll(t, ".5")
	expectToken(t, tokens[0], FCONST, ".5")
}

func TestReal(t *testing.T) {
	tokens := scanAll(t, "1e10")
	expectToken(t, tokens[0], FCONST, "1e10")
}

func TestRealWithSign(t *testing.T) {
	tokens := scanAll(t, "1.5E-3")
	expectToken(t, tokens[0], FCONST, "1.5E-3")
}

func TestIntegerDotDot(t *testing.T) {
	// "1..10" should lex as ICONST(1), DOT_DOT, ICONST(10)
	tokens := scanAll(t, "1..10")
	expectToken(t, tokens[0], ICONST, "1")
	if tokens[1].Tok != DOT_DOT {
		t.Errorf("expected DOT_DOT, got %v", tokens[1].Tok)
	}
	expectToken(t, tokens[2], ICONST, "10")
}

// --- Parameters ---

func TestParam(t *testing.T) {
	tokens := scanAll(t, "$1")
	expectToken(t, tokens[0], PARAM, "$1")
}

func TestParamMultiDigit(t *testing.T) {
	tokens := scanAll(t, "$123")
	expectToken(t, tokens[0], PARAM, "$123")
}

// --- Operators ---

func TestSimpleOp(t *testing.T) {
	tokens := scanAll(t, "~")
	expectToken(t, tokens[0], Op, "~")
}

func TestMultiCharOp(t *testing.T) {
	tokens := scanAll(t, "~*")
	expectToken(t, tokens[0], Op, "~*")
}

// --- Multi-character fixed tokens ---

func TestTypecast(t *testing.T) {
	tokens := scanAll(t, "::")
	if tokens[0].Tok != TYPECAST {
		t.Errorf("expected TYPECAST, got %v", tokens[0].Tok)
	}
}

func TestColonEquals(t *testing.T) {
	tokens := scanAll(t, ":=")
	if tokens[0].Tok != COLON_EQUALS {
		t.Errorf("expected COLON_EQUALS, got %v", tokens[0].Tok)
	}
}

func TestEqualsGreater(t *testing.T) {
	tokens := scanAll(t, "=>")
	if tokens[0].Tok != EQUALS_GREATER {
		t.Errorf("expected EQUALS_GREATER, got %v", tokens[0].Tok)
	}
}

func TestLessEquals(t *testing.T) {
	tokens := scanAll(t, "<=")
	if tokens[0].Tok != LESS_EQUALS {
		t.Errorf("expected LESS_EQUALS, got %v", tokens[0].Tok)
	}
}

func TestGreaterEquals(t *testing.T) {
	tokens := scanAll(t, ">=")
	if tokens[0].Tok != GREATER_EQUALS {
		t.Errorf("expected GREATER_EQUALS, got %v", tokens[0].Tok)
	}
}

func TestNotEquals(t *testing.T) {
	tokens := scanAll(t, "!=")
	if tokens[0].Tok != NOT_EQUALS {
		t.Errorf("expected NOT_EQUALS, got %v", tokens[0].Tok)
	}
}

func TestLessGreater(t *testing.T) {
	tokens := scanAll(t, "<>")
	if tokens[0].Tok != LESS_GREATER {
		t.Errorf("expected LESS_GREATER, got %v", tokens[0].Tok)
	}
}

func TestRightArrow(t *testing.T) {
	tokens := scanAll(t, "->")
	if tokens[0].Tok != RIGHT_ARROW {
		t.Errorf("expected RIGHT_ARROW, got %v", tokens[0].Tok)
	}
}

func TestDotDot(t *testing.T) {
	tokens := scanAll(t, "..")
	if tokens[0].Tok != DOT_DOT {
		t.Errorf("expected DOT_DOT, got %v", tokens[0].Tok)
	}
}

// --- Self tokens ---

func TestSelfTokens(t *testing.T) {
	tokens := scanAll(t, "( ) [ ] , ; + - * / % ^ < > =")
	expected := []rune{'(', ')', '[', ']', ',', ';', '+', '-', '*', '/', '%', '^', '<', '>', '='}
	for i, ch := range expected {
		expectSelfToken(t, tokens[i], ch)
	}
}

// --- Complex SQL statements ---

func TestSelectStatement(t *testing.T) {
	tokens := scanAll(t, "SELECT id, name FROM users WHERE id = $1")
	expectToken(t, tokens[0], KEYWORD, "select")
	expectToken(t, tokens[1], IDENT, "id")
	expectSelfToken(t, tokens[2], ',')
	expectToken(t, tokens[3], KEYWORD, "name") // "name" is an unreserved keyword in PG
	expectToken(t, tokens[4], KEYWORD, "from")
	expectToken(t, tokens[5], IDENT, "users")
	expectToken(t, tokens[6], KEYWORD, "where")
	expectToken(t, tokens[7], IDENT, "id")
	expectSelfToken(t, tokens[8], '=')
	expectToken(t, tokens[9], PARAM, "$1")
	if tokens[10].Tok != EOF {
		t.Errorf("expected EOF, got %v", tokens[10].Tok)
	}
}

func TestCreateFunction(t *testing.T) {
	input := `CREATE FUNCTION add(a integer, b integer) RETURNS integer AS $$
BEGIN
    RETURN a + b;
END;
$$ LANGUAGE plpgsql;`
	tokens := scanAll(t, input)
	// Just verify it doesn't crash and produces reasonable tokens
	if tokens[len(tokens)-1].Tok != EOF {
		t.Error("expected EOF at end")
	}
	// Find the dollar-quoted string
	found := false
	for _, tok := range tokens {
		if tok.Tok == SCONST && tok.Kind == DollarLit {
			found = true
			if !strings.Contains(tok.Lit, "RETURN a + b") {
				t.Errorf("dollar-quoted body doesn't contain expected text: %q", tok.Lit)
			}
		}
	}
	if !found {
		t.Error("did not find dollar-quoted string literal")
	}
}

func TestCastExpression(t *testing.T) {
	tokens := scanAll(t, "'42'::integer")
	expectToken(t, tokens[0], SCONST, "42")
	if tokens[1].Tok != TYPECAST {
		t.Errorf("expected TYPECAST, got %v", tokens[1].Tok)
	}
	expectToken(t, tokens[2], KEYWORD, "integer")
}

func TestJsonOperator(t *testing.T) {
	tokens := scanAll(t, "col1->>'key'")
	expectToken(t, tokens[0], IDENT, "col1")
	// ->> is a multi-char operator
	if tokens[1].Tok != Op {
		t.Errorf("expected Op, got %v (lit=%q)", tokens[1].Tok, tokens[1].Lit)
	}
	if tokens[1].Lit != "->>" {
		t.Errorf("expected ->> operator, got %q", tokens[1].Lit)
	}
	expectToken(t, tokens[2], SCONST, "key")
}

// --- Position tracking ---

func TestPositionTracking(t *testing.T) {
	tokens := scanAll(t, "SELECT\n  42")
	// SELECT is at line 1, col 1
	if tokens[0].Line != 1 || tokens[0].Col != 1 {
		t.Errorf("SELECT: expected (1,1), got (%d,%d)", tokens[0].Line, tokens[0].Col)
	}
	// 42 is at line 2, col 3
	if tokens[1].Line != 2 || tokens[1].Col != 3 {
		t.Errorf("42: expected (2,3), got (%d,%d)", tokens[1].Line, tokens[1].Col)
	}
}

// --- Error cases ---

func TestUnterminatedString(t *testing.T) {
	var errs []string
	var s Scanner
	s.Init(strings.NewReader("'unterminated"), func(line, col uint, msg string) {
		errs = append(errs, msg)
	})
	s.Next()
	if s.Tok != SCONST || !s.Bad {
		t.Errorf("expected bad SCONST, got tok=%v bad=%v", s.Tok, s.Bad)
	}
	if len(errs) == 0 {
		t.Error("expected error for unterminated string")
	}
}

func TestUnterminatedBlockComment(t *testing.T) {
	var errs []string
	var s Scanner
	s.Init(strings.NewReader("/* unterminated"), func(line, col uint, msg string) {
		errs = append(errs, msg)
	})
	s.Next()
	if len(errs) == 0 {
		t.Error("expected error for unterminated block comment")
	}
}
