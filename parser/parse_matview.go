package parser

// parseCreateMatView parses CREATE MATERIALIZED VIEW [IF NOT EXISTS] name
//   [USING method] [(cols)] [TABLESPACE ts] [WITH (opts)] AS query [WITH [NO] DATA].
func (p *Parser) parseCreateMatView() *CreateMatViewStmt {
	p.wantKeyword("materialized")
	p.wantKeyword("view")
	pos := p.pos

	cm := &CreateMatViewStmt{
		baseStmt: baseStmt{baseNode{pos}},
		WithData: true, // default
	}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("not")
		p.wantKeyword("exists")
		cm.IfNotExists = true
	}

	cm.Relation = p.parseRangeVar()

	// Optional USING access_method
	if p.isKeyword("using") {
		p.next()
		cm.AccessMethod = p.colId()
	}

	// Optional WITH (options)
	if p.isKeyword("with") {
		p.next()
		if p.tok == Token('(') {
			cm.Options = p.parseParenDefElemList()
		}
	}

	// Optional TABLESPACE
	if p.gotKeyword("tablespace") {
		cm.TableSpace = p.colId()
	}

	p.wantKeyword("as")
	cm.Query = p.parseSelectStmt()

	// Optional WITH [NO] DATA
	if p.isKeyword("with") {
		p.next()
		if p.gotKeyword("no") {
			p.wantKeyword("data")
			cm.WithData = false
		} else {
			p.wantKeyword("data")
			cm.WithData = true
		}
	}

	return cm
}

// parseRefreshMatView parses REFRESH MATERIALIZED VIEW [CONCURRENTLY] name [WITH [NO] DATA].
func (p *Parser) parseRefreshMatView() *RefreshMatViewStmt {
	p.wantKeyword("refresh")
	pos := p.pos
	p.wantKeyword("materialized")
	p.wantKeyword("view")

	rm := &RefreshMatViewStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.gotKeyword("concurrently") {
		rm.Concurrent = true
	}

	rm.Relation = p.parseRangeVar()

	// Optional WITH [NO] DATA
	if p.isKeyword("with") {
		p.next()
		if p.gotKeyword("no") {
			p.wantKeyword("data")
			rm.SkipData = true
		} else {
			p.wantKeyword("data")
		}
	}

	return rm
}

// parseCreateStatistics parses CREATE STATISTICS [IF NOT EXISTS] name [(types)] ON exprs FROM table.
func (p *Parser) parseCreateStatistics() *CreateStatsStmt {
	p.wantKeyword("statistics")
	pos := p.pos

	cs := &CreateStatsStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("not")
		p.wantKeyword("exists")
		cs.IfNotExists = true
	}

	cs.Defnames = p.parseQualifiedName()

	// Optional (stat_types)
	if p.gotSelf('(') {
		cs.StatTypes = append(cs.StatTypes, p.colId())
		for p.gotSelf(',') {
			cs.StatTypes = append(cs.StatTypes, p.colId())
		}
		p.wantSelf(')')
	}

	p.wantKeyword("on")

	// Expression list
	cs.Exprs = append(cs.Exprs, p.parseExpr())
	for p.gotSelf(',') {
		cs.Exprs = append(cs.Exprs, p.parseExpr())
	}

	p.wantKeyword("from")

	// Table list
	cs.Relations = append(cs.Relations, p.parseQualifiedName())
	for p.gotSelf(',') {
		cs.Relations = append(cs.Relations, p.parseQualifiedName())
	}

	return cs
}

// parseAlterStatistics parses ALTER STATISTICS name SET STATISTICS n | RENAME TO | SET SCHEMA | OWNER TO.
func (p *Parser) parseAlterStatistics() Stmt {
	p.wantKeyword("statistics")
	pos := p.pos

	name := p.parseQualifiedName()

	switch {
	case p.isKeyword("set"):
		p.next()
		if p.isKeyword("schema") {
			p.next()
			newSchema := p.colId()
			return &RenameStmt{
				baseStmt:   baseStmt{baseNode{pos}},
				RenameType: OBJECT_STATISTICS,
				Subname:    joinName(name),
				Newname:    newSchema,
			}
		}
		// SET STATISTICS n
		p.wantKeyword("statistics")
		val := p.parseInt()
		return &AlterStatsStmt{
			baseStmt:      baseStmt{baseNode{pos}},
			Defnames:      name,
			Stxstattarget: int(val),
		}
	case p.isKeyword("rename"):
		p.next()
		p.wantKeyword("to")
		newName := p.colId()
		return &RenameStmt{
			baseStmt:   baseStmt{baseNode{pos}},
			RenameType: OBJECT_STATISTICS,
			Subname:    joinName(name),
			Newname:    newName,
		}
	case p.isKeyword("owner"):
		p.next()
		p.wantKeyword("to")
		newOwner := p.colId()
		return &AlterOwnerStmt{
			baseStmt:   baseStmt{baseNode{pos}},
			ObjectType: OBJECT_STATISTICS,
			Object:     name,
			NewOwner:   newOwner,
		}
	default:
		p.syntaxError("expected SET, RENAME, or OWNER after ALTER STATISTICS name")
		return nil
	}
}

// joinName joins a qualified name into a dot-separated string.
func joinName(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += "." + p
	}
	return result
}
