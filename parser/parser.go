package parser

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jespino/gopgsql/scanner"
)

// Parser is a recursive-descent parser for PostgreSQL SQL.
type Parser struct {
	scan scanner.Scanner
	errh    func(pos int, msg string)
	errcnt  int
	first   error

	// current token state (copied from scanner after each advance)
	tok   Token
	lit   string
	kind  LitKind
	kwcat KeywordCategory
	pos   int // byte offset of current token
}

// Parse parses the SQL source and returns a list of raw statements.
func Parse(src io.Reader, errh func(pos int, msg string)) ([]*RawStmt, error) {
	var p Parser
	p.init(src, errh)
	return p.parseStmtList(), p.first
}

func (p *Parser) init(src io.Reader, errh func(pos int, msg string)) {
	p.errh = errh
	p.scan.Init(src, func(line, col uint, msg string) {
		// Convert line/col to a synthetic position for error reporting.
		// For simplicity we just use the scanner's byte offset.
		if errh != nil {
			errh(-1, msg)
		}
	})
	p.next() // prime the first token
}

// next advances to the next token.
func (p *Parser) next() {
	p.scan.Next()
	p.tok = p.scan.Tok
	p.lit = p.scan.Lit
	p.kind = p.scan.Kind
	p.kwcat = p.scan.KwCat
	// Compute byte offset from line/col (approximate; we use line*10000+col as a proxy)
	p.pos = int(p.scan.Line)*10000 + int(p.scan.Col)
}

// error reports a parse error at the current position.
func (p *Parser) error(msg string) {
	err := fmt.Errorf("at position %d: %s", p.pos, msg)
	if p.first == nil {
		p.first = err
	}
	p.errcnt++
	if p.errh != nil {
		p.errh(p.pos, msg)
	}
}

// errorf reports a formatted parse error.
func (p *Parser) errorf(format string, args ...any) {
	p.error(fmt.Sprintf(format, args...))
}

// syntaxError reports a syntax error about an unexpected token.
func (p *Parser) syntaxError(msg string) {
	if msg == "" {
		p.errorf("syntax error at %s", p.tokDesc())
	} else {
		p.errorf("syntax error: %s (got %s)", msg, p.tokDesc())
	}
}

func (p *Parser) tokDesc() string {
	switch {
	case p.tok == EOF:
		return "end of input"
	case p.tok == IDENT:
		return fmt.Sprintf("identifier %q", p.lit)
	case p.tok == KEYWORD:
		return fmt.Sprintf("keyword %q", p.lit)
	case p.tok == ICONST:
		return fmt.Sprintf("integer %q", p.lit)
	case p.tok == FCONST:
		return fmt.Sprintf("number %q", p.lit)
	case p.tok == SCONST:
		return fmt.Sprintf("string '%s'", p.lit)
	case p.tok == PARAM:
		return fmt.Sprintf("parameter %s", p.lit)
	case p.tok == Op:
		return fmt.Sprintf("operator %q", p.lit)
	case p.tok < 128:
		return fmt.Sprintf("'%c'", rune(p.tok))
	default:
		return fmt.Sprintf("token %v", p.tok)
	}
}

// --- Token matching helpers (modeled after Go's parser) ---

// got consumes the current token if it matches tok and returns true.
func (p *Parser) got(tok Token) bool {
	if p.tok == tok {
		p.next()
		return true
	}
	return false
}

// want consumes the current token if it matches tok, or reports an error.
func (p *Parser) want(tok Token) {
	if !p.got(tok) {
		p.syntaxError(fmt.Sprintf("expected %s", tokName(tok)))
	}
}

// gotKeyword returns true and advances if the current token is the given keyword.
func (p *Parser) gotKeyword(kw string) bool {
	if p.tok == KEYWORD && p.lit == kw {
		p.next()
		return true
	}
	return false
}

// wantKeyword consumes the given keyword or reports an error.
func (p *Parser) wantKeyword(kw string) {
	if !p.gotKeyword(kw) {
		p.syntaxError(fmt.Sprintf("expected %q", kw))
	}
}

// isKeyword returns true if the current token is the given keyword.
func (p *Parser) isKeyword(kw string) bool {
	return p.tok == KEYWORD && p.lit == kw
}

// isAnyKeyword returns true if the current token is any of the given keywords.
func (p *Parser) isAnyKeyword(kws ...string) bool {
	if p.tok != KEYWORD {
		return false
	}
	for _, kw := range kws {
		if p.lit == kw {
			return true
		}
	}
	return false
}

// gotSelf consumes a single-character token and returns true.
func (p *Parser) gotSelf(ch rune) bool {
	if p.tok == Token(ch) {
		p.next()
		return true
	}
	return false
}

// wantSelf consumes a single-character token or reports an error.
func (p *Parser) wantSelf(ch rune) {
	if !p.gotSelf(ch) {
		p.syntaxError(fmt.Sprintf("expected '%c'", ch))
	}
}

// ident consumes an identifier or unreserved keyword and returns its name.
// In PostgreSQL, unreserved keywords can be used as identifiers.
func (p *Parser) ident() string {
	switch {
	case p.tok == IDENT:
		name := p.lit
		p.next()
		return name
	case p.tok == KEYWORD && p.kwcat != ReservedKeyword:
		// Unreserved, column-name, and type/func-name keywords can be identifiers
		name := p.lit
		p.next()
		return name
	default:
		p.syntaxError("expected identifier")
		p.next()
		return ""
	}
}

// colId consumes a ColId (identifier or unreserved/col-name keyword).
func (p *Parser) colId() string {
	return p.ident()
}

// colLabel consumes a column label (any keyword or identifier can be a label).
func (p *Parser) colLabel() string {
	if p.tok == IDENT || p.tok == KEYWORD {
		name := p.lit
		p.next()
		return name
	}
	p.syntaxError("expected column label")
	p.next()
	return ""
}

// tokName returns a human-readable name for a token.
func tokName(tok Token) string {
	if tok < 128 {
		return fmt.Sprintf("'%c'", rune(tok))
	}
	switch tok {
	case EOF:
		return "end of input"
	case IDENT:
		return "identifier"
	case ICONST:
		return "integer"
	case FCONST:
		return "number"
	case SCONST:
		return "string"
	case PARAM:
		return "parameter"
	case TYPECAST:
		return "'::'  "
	case LESS_EQUALS:
		return "'<='"
	case GREATER_EQUALS:
		return "'>='"
	case NOT_EQUALS:
		return "'!='"
	default:
		return fmt.Sprintf("token(%d)", tok)
	}
}

// --- Parsing entry points ---

// parseStmtList parses: stmt { ';' stmt }
func (p *Parser) parseStmtList() []*RawStmt {
	var stmts []*RawStmt
	for p.tok != EOF {
		s := p.parseStmt()
		if s != nil {
			stmts = append(stmts, &RawStmt{
				baseNode: baseNode{Location: s.Pos()},
				Stmt:     s,
				StmtEnd:  p.pos,
			})
		}
		if !p.gotSelf(';') {
			if p.tok != EOF {
				p.syntaxError("expected ';' or end of input")
				// skip to next semicolon or EOF
				for p.tok != EOF && !p.gotSelf(';') {
					p.next()
				}
			}
		}
	}
	return stmts
}

// parseStmt dispatches to the appropriate statement parser.
func (p *Parser) parseStmt() Stmt {
	if p.tok == EOF {
		return nil
	}
	// Check for WITH (CTE) prefix
	if p.isKeyword("with") {
		return p.parseWithStmt()
	}
	return p.parseSimpleStmt()
}

func (p *Parser) parseSimpleStmt() Stmt {
	switch {
	case p.isKeyword("select"), p.isKeyword("values"), p.tok == Token('('):
		return p.parseSelectStmt()
	case p.isKeyword("table"):
		return p.parseTableStmt()
	case p.isKeyword("insert"):
		return p.parseInsertStmt(nil)
	case p.isKeyword("update"):
		return p.parseUpdateStmt(nil)
	case p.isKeyword("delete"):
		return p.parseDeleteStmt(nil)
	case p.isKeyword("merge"):
		return p.parseMergeStmt(nil)
	case p.isKeyword("create"):
		return p.parseCreateStmt()
	case p.isKeyword("alter"):
		return p.parseAlterStmt()
	case p.isKeyword("drop"):
		return p.parseDropDispatch()
	case p.isKeyword("truncate"):
		return p.parseTruncateStmt()
	case p.isKeyword("explain"):
		return p.parseExplainStmt()
	case p.isKeyword("copy"):
		return p.parseCopyStmt()
	case p.isAnyKeyword("begin", "start", "commit", "end", "abort", "rollback", "savepoint", "release"):
		return p.parseTransactionStmt()
	case p.isKeyword("set"):
		return p.parseSetStmt()
	case p.isKeyword("reset"):
		return p.parseResetStmt()
	case p.isKeyword("show"):
		return p.parseShowStmt()
	case p.isKeyword("listen"):
		return p.parseListenStmt()
	case p.isKeyword("notify"):
		return p.parseNotifyStmt()
	case p.isKeyword("unlisten"):
		return p.parseUnlistenStmt()
	case p.isAnyKeyword("vacuum", "analyze", "analyse"):
		return p.parseVacuumStmt()
	case p.isKeyword("lock"):
		return p.parseLockStmt()
	case p.isKeyword("prepare"):
		return p.parsePrepareStmt()
	case p.isKeyword("execute"):
		return p.parseExecuteStmt()
	case p.isKeyword("deallocate"):
		return p.parseDeallocateStmt()
	case p.isKeyword("discard"):
		return p.parseDiscardStmt()
	case p.isKeyword("do"):
		return p.parseDoStmt()
	case p.isKeyword("call"):
		return p.parseCallStmt()
	case p.isKeyword("grant"):
		return p.parseGrantStmt()
	case p.isKeyword("revoke"):
		return p.parseRevokeStmt()
	case p.isKeyword("reassign"):
		return p.parseReassignOwned()
	case p.isKeyword("import"):
		return p.parseImportForeignSchema()
	case p.isKeyword("refresh"):
		return p.parseRefreshMatView()
	case p.isKeyword("declare"):
		return p.parseDeclareCursorStmt()
	case p.isAnyKeyword("fetch", "move"):
		return p.parseFetchStmt()
	case p.isKeyword("close"):
		return p.parseClosePortalStmt()
	case p.isKeyword("comment"):
		return p.parseCommentStmt()
	case p.isKeyword("security"):
		return p.parseSecLabelStmt()
	case p.isKeyword("checkpoint"):
		return p.parseCheckpointStmt()
	case p.isKeyword("load"):
		return p.parseLoadStmt()
	case p.isKeyword("reindex"):
		return p.parseReindexStmt()
	default:
		p.syntaxError("expected statement")
		p.next()
		return nil
	}
}

// parseWithStmt parses WITH ... followed by a DML statement.
func (p *Parser) parseWithStmt() Stmt {
	w := p.parseWithClause()
	switch {
	case p.isKeyword("select"), p.isKeyword("values"), p.tok == Token('('):
		sel := p.parseSelectStmt()
		if s, ok := sel.(*SelectStmt); ok {
			s.WithClause = w
		}
		return sel
	case p.isKeyword("insert"):
		return p.parseInsertStmt(w)
	case p.isKeyword("update"):
		return p.parseUpdateStmt(w)
	case p.isKeyword("delete"):
		return p.parseDeleteStmt(w)
	case p.isKeyword("merge"):
		return p.parseMergeStmt(w)
	default:
		p.syntaxError("expected SELECT, INSERT, UPDATE, DELETE, or MERGE after WITH")
		p.next()
		return nil
	}
}

// --- Utility parsers ---

// parseQualifiedName parses: name { '.' name }
func (p *Parser) parseQualifiedName() []string {
	var names []string
	names = append(names, p.colId())
	for p.gotSelf('.') {
		names = append(names, p.colLabel())
	}
	return names
}

// parseNameList parses a comma-separated list of identifiers.
func (p *Parser) parseNameList() []string {
	var names []string
	names = append(names, p.colId())
	for p.gotSelf(',') {
		names = append(names, p.colId())
	}
	return names
}

// parseExprList parses a comma-separated list of expressions.
func (p *Parser) parseExprList() []Expr {
	var list []Expr
	list = append(list, p.parseExpr())
	for p.gotSelf(',') {
		list = append(list, p.parseExpr())
	}
	return list
}

// parseInt parses an integer literal and returns its value.
func (p *Parser) parseInt() int64 {
	if p.tok != ICONST {
		p.syntaxError("expected integer")
		return 0
	}
	// Remove underscores for parsing
	s := strings.ReplaceAll(p.lit, "_", "")
	val, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		p.errorf("invalid integer: %s", p.lit)
	}
	p.next()
	return val
}
