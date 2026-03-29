package pgscan

// parseCreateView parses CREATE [OR REPLACE] [TEMP] VIEW name [(columns)] AS SELECT ...
func (p *Parser) parseCreateView(persistence RelPersistence, replace bool) *ViewStmt {
	p.wantKeyword("view")

	vs := &ViewStmt{
		baseStmt:    baseStmt{baseNode{p.pos}},
		Replace:     replace,
		Persistence: persistence,
	}

	vs.View = p.parseRangeVar()

	// Optional column aliases
	if p.tok == Token('(') {
		p.next()
		vs.Aliases = p.parseNameList()
		p.wantSelf(')')
	}

	p.wantKeyword("as")
	vs.Query = p.parseSelectStmt()

	// WITH [CASCADED | LOCAL] CHECK OPTION
	if p.isKeyword("with") {
		p.next()
		if p.gotKeyword("cascaded") {
			p.wantKeyword("check")
			p.wantKeyword("option")
			vs.WithCheckOption = CASCADED_CHECK_OPTION
		} else if p.gotKeyword("local") {
			p.wantKeyword("check")
			p.wantKeyword("option")
			vs.WithCheckOption = LOCAL_CHECK_OPTION
		} else {
			p.wantKeyword("check")
			p.wantKeyword("option")
			vs.WithCheckOption = CASCADED_CHECK_OPTION // default is CASCADED
		}
	}

	return vs
}

// parseExplainStmt parses EXPLAIN [ANALYZE] [VERBOSE] stmt or EXPLAIN (options) stmt.
func (p *Parser) parseExplainStmt() *ExplainStmt {
	p.wantKeyword("explain")

	es := &ExplainStmt{baseStmt: baseStmt{baseNode{p.pos}}}

	// EXPLAIN (option, ...) stmt
	if p.tok == Token('(') {
		p.next()
		es.Options = p.parseExplainOptions()
		p.wantSelf(')')
		es.Query = p.parseSimpleStmt()
		return es
	}

	// EXPLAIN ANALYZE [VERBOSE] stmt
	if p.isKeyword("analyze") || p.isKeyword("analyse") {
		p.next()
		es.Options = append(es.Options, &DefElem{
			baseNode: baseNode{p.pos},
			Defname:  "analyze",
		})
		if p.isKeyword("verbose") {
			p.next()
			es.Options = append(es.Options, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "verbose",
			})
		}
		es.Query = p.parseSimpleStmt()
		return es
	}

	// EXPLAIN VERBOSE stmt
	if p.isKeyword("verbose") {
		p.next()
		es.Options = append(es.Options, &DefElem{
			baseNode: baseNode{p.pos},
			Defname:  "verbose",
		})
		es.Query = p.parseSimpleStmt()
		return es
	}

	// Plain EXPLAIN stmt
	es.Query = p.parseSimpleStmt()
	return es
}

// parseExplainOptions parses comma-separated EXPLAIN options inside parens.
func (p *Parser) parseExplainOptions() []*DefElem {
	var opts []*DefElem
	opts = append(opts, p.parseExplainOption())
	for p.gotSelf(',') {
		opts = append(opts, p.parseExplainOption())
	}
	return opts
}

// parseExplainOption parses a single EXPLAIN option: name [value]
func (p *Parser) parseExplainOption() *DefElem {
	pos := p.pos
	name := p.colLabel() // ANALYZE, FORMAT, etc. are keywords
	de := &DefElem{baseNode: baseNode{pos}, Defname: name}

	// Optional value: boolean, string, or identifier
	if p.tok == IDENT || p.tok == KEYWORD || p.tok == SCONST || p.tok == ICONST {
		if p.tok == SCONST {
			de.Arg = &String{baseNode: baseNode{p.pos}, Str: p.lit}
			p.next()
		} else if p.tok == ICONST {
			de.Arg = &A_Const{baseExpr: baseExpr{baseNode{p.pos}}, Val: Value{Type: ValInt, Ival: p.parseInt()}}
		} else {
			// Boolean or identifier value
			de.Arg = &String{baseNode: baseNode{p.pos}, Str: p.lit}
			p.next()
		}
	}

	return de
}

// parseCopyStmt parses COPY table [(columns)] FROM/TO ... or COPY (query) TO ...
func (p *Parser) parseCopyStmt() *CopyStmt {
	p.wantKeyword("copy")

	cs := &CopyStmt{baseStmt: baseStmt{baseNode{p.pos}}}

	// COPY (query) TO ...
	if p.tok == Token('(') {
		p.next()
		cs.Query = p.parseSelectStmt()
		p.wantSelf(')')
		p.wantKeyword("to")
		cs.IsFrom = false
		p.parseCopyTarget(cs)
		p.parseCopyOptions(cs)
		return cs
	}

	// COPY table [(columns)] FROM/TO ...
	cs.Relation = p.parseRangeVar()

	// Optional column list
	if p.tok == Token('(') {
		p.next()
		cs.Attlist = p.parseNameList()
		p.wantSelf(')')
	}

	if p.gotKeyword("from") {
		cs.IsFrom = true
	} else {
		p.wantKeyword("to")
		cs.IsFrom = false
	}

	p.parseCopyTarget(cs)

	// Optional WHERE (only for COPY FROM)
	if cs.IsFrom && p.gotKeyword("where") {
		cs.WhereClause = p.parseExpr()
	}

	p.parseCopyOptions(cs)

	return cs
}

// parseCopyTarget parses STDIN/STDOUT/PROGRAM/filename.
func (p *Parser) parseCopyTarget(cs *CopyStmt) {
	if p.gotKeyword("program") {
		cs.IsProgram = true
		if p.tok == SCONST {
			cs.Filename = p.lit
			p.next()
		}
		return
	}
	if p.gotKeyword("stdin") || p.gotKeyword("stdout") {
		// STDIN/STDOUT — filename stays empty
		return
	}
	if p.tok == SCONST {
		cs.Filename = p.lit
		p.next()
	}
}

// parseCopyOptions parses WITH (option, ...) or legacy options.
func (p *Parser) parseCopyOptions(cs *CopyStmt) {
	if !p.isKeyword("with") {
		return
	}
	p.next()
	if p.tok != Token('(') {
		return
	}
	p.next()
	for p.tok != Token(')') && p.tok != EOF {
		opt := p.parseExplainOption() // same format: name [value]
		cs.Options = append(cs.Options, opt)
		p.gotSelf(',')
	}
	p.wantSelf(')')
}
