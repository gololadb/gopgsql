package parser

// --- INSERT ---

// parseInsertStmt parses INSERT INTO ... VALUES/SELECT ...
func (p *Parser) parseInsertStmt(w *WithClause) Stmt {
	pos := p.pos
	p.wantKeyword("insert")
	p.wantKeyword("into")

	s := &InsertStmt{
		baseStmt:   baseStmt{baseNode{pos}},
		WithClause: w,
	}

	s.Relation = p.parseRangeVar()

	// Optional alias
	if p.gotKeyword("as") {
		s.Relation.Alias = p.parseAlias()
	} else if p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
		if !p.isAnyKeyword("values", "select", "default", "overriding", "on") &&
			p.tok != Token('(') {
			s.Relation.Alias = p.parseAlias()
		}
	}

	// Optional column list
	if p.tok == Token('(') && !p.isKeyword("select") && !p.isKeyword("values") {
		// Peek: is this a column list or a subquery?
		// Column list starts with ( followed by identifiers
		// We'll try column list first
		p.next() // consume (
		if p.tok == IDENT || p.tok == KEYWORD {
			// Column list
			s.Cols = p.parseInsertColumnList()
			p.wantSelf(')')
		} else {
			// Probably a subquery — put back by parsing as select
			sub := p.parseSelectNoParens()
			p.wantSelf(')')
			s.SelectStmt = sub
			goto afterSource
		}
	}

	// OVERRIDING
	if p.isKeyword("overriding") {
		p.next()
		if p.gotKeyword("system") {
			s.Override = OVERRIDING_SYSTEM_VALUE
		} else if p.gotKeyword("user") {
			s.Override = OVERRIDING_USER_VALUE
		}
		p.wantKeyword("value")
	}

	// DEFAULT VALUES or SELECT/VALUES
	if p.isKeyword("default") {
		p.next()
		p.wantKeyword("values")
		// No SelectStmt — means DEFAULT VALUES
	} else {
		s.SelectStmt = p.parseSelectStmt()
	}

afterSource:
	// ON CONFLICT
	if p.isKeyword("on") {
		p.next()
		if p.gotKeyword("conflict") {
			s.OnConflict = p.parseOnConflict()
		}
	}

	// RETURNING
	if p.gotKeyword("returning") {
		s.ReturningList = p.parseTargetList()
	}

	return s
}

func (p *Parser) parseInsertColumnList() []*ResTarget {
	var list []*ResTarget
	list = append(list, p.parseInsertColumn())
	for p.gotSelf(',') {
		list = append(list, p.parseInsertColumn())
	}
	return list
}

func (p *Parser) parseInsertColumn() *ResTarget {
	pos := p.pos
	name := p.colId()
	return &ResTarget{
		baseNode: baseNode{pos},
		Name:     name,
	}
}

func (p *Parser) parseOnConflict() *OnConflictClause {
	oc := &OnConflictClause{baseNode: baseNode{p.pos}}

	// Optional conflict target
	if p.tok == Token('(') || p.isKeyword("on") {
		oc.Infer = p.parseInferClause()
	}

	// DO NOTHING or DO UPDATE
	p.wantKeyword("do")
	if p.gotKeyword("nothing") {
		oc.Action = ONCONFLICT_NOTHING
	} else if p.gotKeyword("update") {
		oc.Action = ONCONFLICT_UPDATE
		p.wantKeyword("set")
		oc.TargetList = p.parseSetClauseList()
		if p.gotKeyword("where") {
			oc.WhereClause = p.parseExpr()
		}
	} else {
		p.syntaxError("expected NOTHING or UPDATE after DO")
	}

	return oc
}

func (p *Parser) parseInferClause() *InferClause {
	ic := &InferClause{baseNode: baseNode{p.pos}}

	if p.isKeyword("on") {
		p.next()
		p.wantKeyword("constraint")
		ic.Conname = p.colId()
		return ic
	}

	// Index expressions
	p.wantSelf('(')
	for {
		ic.IndexElems = append(ic.IndexElems, &String{
			baseNode: baseNode{p.pos},
			Str:      p.colId(),
		})
		if !p.gotSelf(',') {
			break
		}
	}
	p.wantSelf(')')

	if p.gotKeyword("where") {
		ic.WhereClause = p.parseExpr()
	}

	return ic
}

// --- UPDATE ---

// parseUpdateStmt parses UPDATE ... SET ... [FROM ...] [WHERE ...] [RETURNING ...]
func (p *Parser) parseUpdateStmt(w *WithClause) Stmt {
	pos := p.pos
	p.wantKeyword("update")

	s := &UpdateStmt{
		baseStmt:   baseStmt{baseNode{pos}},
		WithClause: w,
	}

	s.Relation = p.parseRangeVar()

	// Optional alias
	if p.gotKeyword("as") {
		s.Relation.Alias = p.parseAlias()
	} else if p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
		if !p.isKeyword("set") {
			s.Relation.Alias = p.parseAlias()
		}
	}

	p.wantKeyword("set")
	s.TargetList = p.parseSetClauseList()

	// FROM
	if p.gotKeyword("from") {
		s.FromClause = p.parseFromClause()
	}

	// WHERE
	if p.gotKeyword("where") {
		s.WhereClause = p.parseExpr()
	}

	// RETURNING
	if p.gotKeyword("returning") {
		s.ReturningList = p.parseTargetList()
	}

	return s
}

// parseSetClauseList parses col = expr, col = expr, ...
func (p *Parser) parseSetClauseList() []*ResTarget {
	var list []*ResTarget
	list = append(list, p.parseSetClause())
	for p.gotSelf(',') {
		list = append(list, p.parseSetClause())
	}
	return list
}

func (p *Parser) parseSetClause() *ResTarget {
	pos := p.pos
	rt := &ResTarget{baseNode: baseNode{pos}}

	// Column name (possibly with indirection: col[1] = ...)
	rt.Name = p.colId()

	// Optional indirection
	for p.tok == Token('[') || p.tok == Token('.') {
		if p.tok == Token('.') {
			p.next()
			rt.Indirection = append(rt.Indirection, &String{
				baseNode: baseNode{p.pos},
				Str:      p.colLabel(),
			})
		} else {
			p.next() // [
			idx := p.parseExpr()
			p.wantSelf(']')
			rt.Indirection = append(rt.Indirection, &A_Indices{
				baseNode: baseNode{pos},
				Uidx:     idx,
			})
		}
	}

	p.wantSelf('=')
	if p.isKeyword("default") {
		p.next()
		// DEFAULT — represented as nil Val
	} else {
		rt.Val = p.parseExpr()
	}

	return rt
}

// --- DELETE ---

// parseDeleteStmt parses DELETE FROM ... [USING ...] [WHERE ...] [RETURNING ...]
func (p *Parser) parseDeleteStmt(w *WithClause) Stmt {
	pos := p.pos
	p.wantKeyword("delete")
	p.wantKeyword("from")

	s := &DeleteStmt{
		baseStmt:   baseStmt{baseNode{pos}},
		WithClause: w,
	}

	s.Relation = p.parseRangeVar()

	// Optional alias
	if p.gotKeyword("as") {
		s.Relation.Alias = p.parseAlias()
	} else if p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
		if !p.isAnyKeyword("using", "where", "returning") {
			s.Relation.Alias = p.parseAlias()
		}
	}

	// USING
	if p.gotKeyword("using") {
		s.UsingClause = p.parseFromClause()
	}

	// WHERE
	if p.gotKeyword("where") {
		if p.isKeyword("current") {
			// WHERE CURRENT OF cursor — not supported in this parser
			p.syntaxError("WHERE CURRENT OF not supported")
		} else {
			s.WhereClause = p.parseExpr()
		}
	}

	// RETURNING
	if p.gotKeyword("returning") {
		s.ReturningList = p.parseTargetList()
	}

	return s
}
