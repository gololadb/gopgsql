package pgscan

// SQL syntax functions — special-syntax function calls that desugar into
// FuncCall nodes with system function names.

func sysFuncCall(pos int, name string, args []Expr) *FuncCall {
	return &FuncCall{
		baseExpr:   baseExpr{baseNode{pos}},
		Funcname:   []string{"pg_catalog", name},
		Args:       args,
		FuncFormat: COERCE_SQL_SYNTAX,
	}
}

// parseExtract parses EXTRACT(field FROM expr).
func (p *Parser) parseExtract(pos int) Expr {
	p.next() // consume EXTRACT
	p.wantSelf('(')

	if p.tok == Token(')') {
		// EXTRACT() with no args — allow as plain function call
		p.next()
		return sysFuncCall(pos, "extract", nil)
	}

	// field is an identifier or keyword (YEAR, MONTH, DAY, HOUR, MINUTE, SECOND, etc.)
	field := p.colLabel()
	p.wantKeyword("from")
	arg := p.parseExpr()
	p.wantSelf(')')

	return sysFuncCall(pos, "extract", []Expr{
		&A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValStr, Str: field}},
		arg,
	})
}

// parsePosition parses POSITION(expr IN expr).
// Note: PG reverses the args: position(B, A) for "A IN B".
// Uses b_expr for the first argument to avoid ambiguity with the IN keyword.
func (p *Parser) parsePosition(pos int) Expr {
	p.next() // consume POSITION
	p.wantSelf('(')

	if p.tok == Token(')') {
		p.next()
		return sysFuncCall(pos, "position", nil)
	}

	sub := p.parseBExpr() // b_expr to avoid IN ambiguity
	p.wantKeyword("in")
	str := p.parseExpr()
	p.wantSelf(')')

	// PG convention: position(haystack, needle)
	return sysFuncCall(pos, "position", []Expr{str, sub})
}

// parseSubstring parses:
//   SUBSTRING(expr FROM expr FOR expr)
//   SUBSTRING(expr FROM expr)
//   SUBSTRING(expr FOR expr)
//   SUBSTRING(expr, expr, expr)  — plain function call syntax
func (p *Parser) parseSubstring(pos int) Expr {
	p.next() // consume SUBSTRING
	p.wantSelf('(')

	if p.tok == Token(')') {
		p.next()
		return sysFuncCall(pos, "substring", nil)
	}

	arg := p.parseExpr()

	// Check for SQL syntax (FROM/FOR) vs plain function call syntax (,)
	if p.gotKeyword("from") {
		from := p.parseExpr()
		if p.gotKeyword("for") {
			forExpr := p.parseExpr()
			p.wantSelf(')')
			return sysFuncCall(pos, "substring", []Expr{arg, from, forExpr})
		}
		p.wantSelf(')')
		return sysFuncCall(pos, "substring", []Expr{arg, from})
	}

	if p.gotKeyword("for") {
		forExpr := p.parseExpr()
		p.wantSelf(')')
		// SUBSTRING(str FOR len) = SUBSTRING(str, 1, len)
		return sysFuncCall(pos, "substring", []Expr{
			arg,
			&A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: 1}},
			forExpr,
		})
	}

	if p.gotKeyword("similar") {
		// SUBSTRING(str SIMILAR pattern ESCAPE escape)
		pattern := p.parseExpr()
		p.wantKeyword("escape")
		esc := p.parseExpr()
		p.wantSelf(')')
		return sysFuncCall(pos, "substring", []Expr{arg, pattern, esc})
	}

	// Plain function call: SUBSTRING(expr, expr [, expr])
	var args []Expr
	args = append(args, arg)
	for p.gotSelf(',') {
		args = append(args, p.parseExpr())
	}
	p.wantSelf(')')

	// Use explicit call form for plain syntax
	return &FuncCall{
		baseExpr:   baseExpr{baseNode{pos}},
		Funcname:   []string{"substring"},
		Args:       args,
		FuncFormat: COERCE_EXPLICIT_CALL,
	}
}

// parseOverlay parses OVERLAY(expr PLACING expr FROM expr [FOR expr]).
func (p *Parser) parseOverlay(pos int) Expr {
	p.next() // consume OVERLAY
	p.wantSelf('(')

	if p.tok == Token(')') {
		p.next()
		return sysFuncCall(pos, "overlay", nil)
	}

	str := p.parseExpr()

	// Check for SQL syntax vs plain function call
	if p.gotKeyword("placing") {
		repl := p.parseExpr()
		p.wantKeyword("from")
		from := p.parseExpr()
		args := []Expr{str, repl, from}
		if p.gotKeyword("for") {
			forExpr := p.parseExpr()
			args = append(args, forExpr)
		}
		p.wantSelf(')')
		return sysFuncCall(pos, "overlay", args)
	}

	// Plain function call syntax
	var args []Expr
	args = append(args, str)
	for p.gotSelf(',') {
		args = append(args, p.parseExpr())
	}
	p.wantSelf(')')
	return &FuncCall{
		baseExpr:   baseExpr{baseNode{pos}},
		Funcname:   []string{"overlay"},
		Args:       args,
		FuncFormat: COERCE_EXPLICIT_CALL,
	}
}

// parseTrim parses TRIM([LEADING|TRAILING|BOTH] [chars FROM] string).
func (p *Parser) parseTrim(pos int) Expr {
	p.next() // consume TRIM
	p.wantSelf('(')

	funcName := "btrim" // default is BOTH

	if p.gotKeyword("both") {
		funcName = "btrim"
	} else if p.gotKeyword("leading") {
		funcName = "ltrim"
	} else if p.gotKeyword("trailing") {
		funcName = "rtrim"
	}

	if p.tok == Token(')') {
		p.next()
		return sysFuncCall(pos, funcName, nil)
	}

	first := p.parseExpr()

	if p.gotKeyword("from") {
		// TRIM([dir] chars FROM string)
		str := p.parseExpr()
		p.wantSelf(')')
		return sysFuncCall(pos, funcName, []Expr{str, first})
	}

	// Collect remaining args
	var args []Expr
	args = append(args, first)
	for p.gotSelf(',') {
		args = append(args, p.parseExpr())
	}
	p.wantSelf(')')
	return sysFuncCall(pos, funcName, args)
}

// parseTreat parses TREAT(expr AS type).
func (p *Parser) parseTreat(pos int) Expr {
	p.next() // consume TREAT
	p.wantSelf('(')
	arg := p.parseExpr()
	p.wantKeyword("as")
	tn := p.parseTypeName()
	p.wantSelf(')')

	// Convert to function call using the type name
	typFunc := tn.Names[len(tn.Names)-1]
	return &FuncCall{
		baseExpr:   baseExpr{baseNode{pos}},
		Funcname:   []string{"pg_catalog", typFunc},
		Args:       []Expr{arg},
		FuncFormat: COERCE_EXPLICIT_CALL,
	}
}

// parseNormalize parses NORMALIZE(expr [, form]).
func (p *Parser) parseNormalize(pos int) Expr {
	p.next() // consume NORMALIZE
	p.wantSelf('(')
	arg := p.parseExpr()

	if p.gotSelf(',') {
		// NORMALIZE(expr, form)
		form := p.colLabel()
		p.wantSelf(')')
		return sysFuncCall(pos, "normalize", []Expr{
			arg,
			&A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValStr, Str: form}},
		})
	}

	p.wantSelf(')')
	return sysFuncCall(pos, "normalize", []Expr{arg})
}

// parseCollationFor parses COLLATION FOR (expr).
func (p *Parser) parseCollationFor(pos int) Expr {
	p.next() // consume COLLATION
	p.wantKeyword("for")
	p.wantSelf('(')
	arg := p.parseExpr()
	p.wantSelf(')')
	return sysFuncCall(pos, "pg_collation_for", []Expr{arg})
}
