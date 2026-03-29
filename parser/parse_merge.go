package parser

// parseMergeStmt parses:
//
//	MERGE INTO relation [AS alias]
//	USING table_ref ON condition
//	merge_when_list
//	[RETURNING ...]
func (p *Parser) parseMergeStmt(w *WithClause) *MergeStmt {
	p.wantKeyword("merge")
	p.wantKeyword("into")

	m := &MergeStmt{
		baseStmt:   baseStmt{baseNode{p.pos}},
		WithClause: w,
	}

	// Target relation with optional alias
	m.Relation = p.parseRangeVar()
	if p.gotKeyword("as") {
		m.Relation.Alias = &Alias{baseNode: baseNode{p.pos}, Aliasname: p.colId()}
	} else if p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
		// Alias without AS, but not if next keyword is USING
		if !p.isKeyword("using") {
			m.Relation.Alias = &Alias{baseNode: baseNode{p.pos}, Aliasname: p.colId()}
		}
	}

	// USING source_relation
	p.wantKeyword("using")
	m.SourceRelation = p.parseTableRef()

	// ON join_condition
	p.wantKeyword("on")
	m.JoinCondition = p.parseExpr()

	// merge_when_list (one or more WHEN clauses)
	for p.isKeyword("when") {
		m.WhenClauses = append(m.WhenClauses, p.parseMergeWhenClause())
	}

	// Optional RETURNING
	if p.gotKeyword("returning") {
		m.ReturningList = p.parseTargetList()
	}

	return m
}

// parseMergeWhenClause parses a single WHEN [NOT] MATCHED ... THEN action.
func (p *Parser) parseMergeWhenClause() *MergeWhenClause {
	pos := p.pos
	p.wantKeyword("when")

	wc := &MergeWhenClause{baseNode: baseNode{pos}}

	if p.gotKeyword("matched") {
		// WHEN MATCHED
		wc.MatchKind = MERGE_WHEN_MATCHED
	} else {
		// WHEN NOT MATCHED [BY SOURCE | BY TARGET]
		p.wantKeyword("not")
		p.wantKeyword("matched")
		if p.gotKeyword("by") {
			if p.gotKeyword("source") {
				wc.MatchKind = MERGE_WHEN_NOT_MATCHED_BY_SOURCE
			} else {
				p.wantKeyword("target")
				wc.MatchKind = MERGE_WHEN_NOT_MATCHED_BY_TARGET
			}
		} else {
			wc.MatchKind = MERGE_WHEN_NOT_MATCHED_BY_TARGET
		}
	}

	// Optional AND condition
	if p.gotKeyword("and") {
		wc.Condition = p.parseExpr()
	}

	p.wantKeyword("then")

	// Action: UPDATE SET ... | DELETE | INSERT ... | DO NOTHING
	switch {
	case p.isKeyword("update"):
		p.next()
		p.wantKeyword("set")
		wc.CommandType = MERGE_CMD_UPDATE
		wc.TargetList = p.parseSetClauseList()

	case p.isKeyword("delete"):
		p.next()
		wc.CommandType = MERGE_CMD_DELETE

	case p.isKeyword("insert"):
		p.next()
		wc.CommandType = MERGE_CMD_INSERT
		p.parseMergeInsert(wc)

	case p.isKeyword("do"):
		p.next()
		p.wantKeyword("nothing")
		wc.CommandType = MERGE_CMD_NOTHING

	default:
		p.syntaxError("expected UPDATE, DELETE, INSERT, or DO NOTHING")
	}

	return wc
}

// parseMergeInsert parses the INSERT action of a MERGE WHEN clause:
//
//	INSERT [(column_list)] [OVERRIDING ...] VALUES (expr_list)
//	INSERT DEFAULT VALUES
func (p *Parser) parseMergeInsert(wc *MergeWhenClause) {
	// Optional OVERRIDING before column list or VALUES
	if p.gotKeyword("overriding") {
		if p.gotKeyword("user") {
			wc.Override = OVERRIDING_USER_VALUE
		} else {
			p.wantKeyword("system")
			wc.Override = OVERRIDING_SYSTEM_VALUE
		}
		p.wantKeyword("value")
	}

	// Optional column list
	if p.tok == Token('(') {
		p.next()
		// Could be column list or VALUES
		wc.TargetList = p.parseInsertColumnList()
		p.wantSelf(')')

		// Check for OVERRIDING after column list
		if p.gotKeyword("overriding") {
			if p.gotKeyword("user") {
				wc.Override = OVERRIDING_USER_VALUE
			} else {
				p.wantKeyword("system")
				wc.Override = OVERRIDING_SYSTEM_VALUE
			}
			p.wantKeyword("value")
		}
	}

	// DEFAULT VALUES or VALUES (expr_list)
	if p.gotKeyword("default") {
		p.wantKeyword("values")
		return
	}

	p.wantKeyword("values")
	p.wantSelf('(')
	wc.Values = p.parseExprList()
	p.wantSelf(')')
}
