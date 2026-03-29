package parser

// parseSubqueryOp parses: ANY/ALL/SOME (subquery_or_expr) after an operator.
// The operator has already been consumed and is passed as 'op'.
func (p *Parser) parseSubqueryOp(left Expr, op string) Expr {
	pos := left.Pos()
	isAll := p.isKeyword("all")
	p.next() // consume ANY/ALL/SOME

	p.wantSelf('(')

	// Check for subquery
	if p.isKeyword("select") || p.isKeyword("values") || p.isKeyword("with") || p.isKeyword("table") {
		sub := p.parseSelectStmt()
		p.wantSelf(')')
		linkType := ANY_SUBLINK
		if isAll {
			linkType = ALL_SUBLINK
		}
		return &SubLink{
			baseExpr:    baseExpr{baseNode{pos}},
			SubLinkType: linkType,
			Testexpr:    left,
			OperName:    []string{op},
			Subselect:   sub,
		}
	}

	// Array expression: op ANY/ALL (expr)
	arg := p.parseExpr()
	p.wantSelf(')')

	kind := AEXPR_OP_ANY
	if isAll {
		kind = AEXPR_OP_ALL
	}
	return &A_Expr{
		baseExpr: baseExpr{baseNode{pos}},
		Kind:     kind,
		Name:     []string{op},
		Lexpr:    left,
		Rexpr:    arg,
	}
}

// parseQualOp parses OPERATOR(schema.opname) qualified operator syntax.
// Returns the operator name parts. The OPERATOR keyword has already been consumed.
// The operator name can be a symbol like =, <>, etc., not just an identifier.
func (p *Parser) parseQualOp() []string {
	p.wantSelf('(')
	var names []string

	// First token: could be schema name (ident/keyword) or operator symbol.
	// If it's an ident/keyword, consume it and check for '.'.
	if p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
		name := p.lit
		p.next()
		if p.gotSelf('.') {
			// schema.op
			names = append(names, name)
			names = append(names, p.parseOpSymbol())
		} else {
			// No dot — the ident itself is the operator name (unusual but valid)
			names = append(names, name)
		}
	} else {
		// Operator symbol directly
		names = append(names, p.parseOpSymbol())
	}

	p.wantSelf(')')
	return names
}

// parseOpSymbol consumes and returns an operator symbol token.
func (p *Parser) parseOpSymbol() string {
	var op string
	switch {
	case p.tok == Op:
		op = p.lit
	case p.tok == Token('='):
		op = "="
	case p.tok == Token('<'):
		op = "<"
	case p.tok == Token('>'):
		op = ">"
	case p.tok == LESS_EQUALS:
		op = "<="
	case p.tok == GREATER_EQUALS:
		op = ">="
	case p.tok == LESS_GREATER:
		op = "<>"
	case p.tok == NOT_EQUALS:
		op = "!="
	case p.tok == Token('+'):
		op = "+"
	case p.tok == Token('-'):
		op = "-"
	case p.tok == Token('*'):
		op = "*"
	case p.tok == Token('/'):
		op = "/"
	case p.tok == Token('%'):
		op = "%"
	case p.tok == Token('^'):
		op = "^"
	case p.tok == Token('|'):
		op = "|"
	default:
		p.syntaxError("expected operator symbol")
		return ""
	}
	p.next()
	return op
}
