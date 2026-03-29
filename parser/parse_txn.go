package parser

// parseTransactionStmt parses BEGIN, START TRANSACTION, COMMIT, END, ROLLBACK,
// SAVEPOINT, RELEASE SAVEPOINT, ROLLBACK TO SAVEPOINT.
func (p *Parser) parseTransactionStmt() *TransactionStmt {
	pos := p.pos
	ts := &TransactionStmt{baseStmt: baseStmt{baseNode{pos}}}

	switch {
	case p.gotKeyword("begin"):
		ts.Kind = TRANS_STMT_BEGIN
		p.gotKeyword("work")       // optional
		p.gotKeyword("transaction") // optional
		ts.Options = p.parseTransactionModes()

	case p.isKeyword("start"):
		p.next()
		p.wantKeyword("transaction")
		ts.Kind = TRANS_STMT_START
		ts.Options = p.parseTransactionModes()

	case p.gotKeyword("commit"):
		ts.Kind = TRANS_STMT_COMMIT
		p.gotKeyword("work")       // optional
		p.gotKeyword("transaction") // optional
		if p.gotKeyword("prepared") {
			ts.Kind = TRANS_STMT_COMMIT_PREPARED
			if p.tok == SCONST {
				ts.Options = []string{p.lit}
				p.next()
			}
		}

	case p.gotKeyword("end"):
		ts.Kind = TRANS_STMT_COMMIT // END = COMMIT
		p.gotKeyword("work")
		p.gotKeyword("transaction")

	case p.gotKeyword("abort"):
		ts.Kind = TRANS_STMT_ROLLBACK // ABORT = ROLLBACK
		p.gotKeyword("work")
		p.gotKeyword("transaction")

	case p.gotKeyword("rollback"):
		ts.Kind = TRANS_STMT_ROLLBACK
		p.gotKeyword("work")       // optional
		p.gotKeyword("transaction") // optional
		if p.gotKeyword("prepared") {
			ts.Kind = TRANS_STMT_ROLLBACK_PREPARED
			if p.tok == SCONST {
				ts.Options = []string{p.lit}
				p.next()
			}
		} else if p.gotKeyword("to") {
			ts.Kind = TRANS_STMT_ROLLBACK_TO
			p.gotKeyword("savepoint") // optional
			ts.Options = []string{p.colId()}
		}

	case p.gotKeyword("savepoint"):
		ts.Kind = TRANS_STMT_SAVEPOINT
		ts.Options = []string{p.colId()}

	case p.isKeyword("release"):
		p.next()
		p.gotKeyword("savepoint") // optional
		ts.Kind = TRANS_STMT_RELEASE
		ts.Options = []string{p.colId()}

	case p.isKeyword("prepare"):
		p.next()
		p.wantKeyword("transaction")
		ts.Kind = TRANS_STMT_PREPARE
		if p.tok == SCONST {
			ts.Options = []string{p.lit}
			p.next()
		}
	}

	return ts
}

// parseTransactionModes parses optional ISOLATION LEVEL ..., READ ONLY/WRITE, [NOT] DEFERRABLE.
func (p *Parser) parseTransactionModes() []string {
	var modes []string
	for {
		switch {
		case p.isKeyword("isolation"):
			p.next()
			p.wantKeyword("level")
			switch {
			case p.gotKeyword("serializable"):
				modes = append(modes, "SERIALIZABLE")
			case p.isKeyword("repeatable"):
				p.next()
				p.wantKeyword("read")
				modes = append(modes, "REPEATABLE READ")
			case p.isKeyword("read"):
				p.next()
				if p.gotKeyword("committed") {
					modes = append(modes, "READ COMMITTED")
				} else {
					p.wantKeyword("uncommitted")
					modes = append(modes, "READ UNCOMMITTED")
				}
			}
		case p.isKeyword("read"):
			p.next()
			if p.gotKeyword("only") {
				modes = append(modes, "READ ONLY")
			} else {
				p.wantKeyword("write")
				modes = append(modes, "READ WRITE")
			}
		case p.gotKeyword("deferrable"):
			modes = append(modes, "DEFERRABLE")
		case p.isKeyword("not"):
			p.next()
			p.wantKeyword("deferrable")
			modes = append(modes, "NOT DEFERRABLE")
		default:
			return modes
		}
		p.gotSelf(',') // optional comma between modes
	}
}

// parseSetStmt parses SET [SESSION|LOCAL] name = value | SET name TO value | RESET name.
// Also dispatches SET CONSTRAINTS to parseSetConstraintsStmt.
func (p *Parser) parseSetStmt() Stmt {
	p.wantKeyword("set")
	pos := p.pos

	// SET CONSTRAINTS {ALL|name,...} {DEFERRED|IMMEDIATE}
	if p.isKeyword("constraints") {
		return p.parseSetConstraintsStmt()
	}

	vs := &VariableSetStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.gotKeyword("local") {
		vs.IsLocal = true
	} else {
		p.gotKeyword("session") // optional
	}

	// SET TRANSACTION ...
	if p.isKeyword("transaction") {
		p.next()
		modes := p.parseTransactionModes()
		// Represent as SET transaction_isolation etc.
		vs.Name = "transaction"
		for _, m := range modes {
			vs.Args = append(vs.Args, &A_Const{
				baseExpr: baseExpr{baseNode{pos}},
				Val:      Value{Type: ValStr, Str: m},
			})
		}
		return vs
	}

	vs.Name = p.colId()

	// SET name TO/= value | SET name TO DEFAULT
	if p.gotSelf('=') || p.gotKeyword("to") {
		if p.gotKeyword("default") {
			vs.IsReset = true
		} else {
			vs.Args = p.parseSetValueList()
		}
	}

	return vs
}

// parseSetValueList parses a comma-separated list of SET values.
func (p *Parser) parseSetValueList() []Expr {
	var vals []Expr
	vals = append(vals, p.parseSetValue())
	for p.gotSelf(',') {
		vals = append(vals, p.parseSetValue())
	}
	return vals
}

// parseSetValue parses a single SET value (string, number, identifier, ON, OFF).
func (p *Parser) parseSetValue() Expr {
	pos := p.pos
	switch {
	case p.tok == SCONST:
		s := p.lit
		p.next()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValStr, Str: s}}
	case p.tok == ICONST:
		v := p.parseInt()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: v}}
	case p.tok == FCONST:
		s := p.lit
		p.next()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValStr, Str: s}}
	default:
		// Identifier or keyword value (e.g. ON, OFF, true, false, utf8)
		name := p.colLabel()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValStr, Str: name}}
	}
}

// parseResetStmt parses RESET name | RESET ALL.
func (p *Parser) parseResetStmt() *VariableSetStmt {
	p.wantKeyword("reset")
	pos := p.pos

	vs := &VariableSetStmt{baseStmt: baseStmt{baseNode{pos}}, IsReset: true}

	if p.gotKeyword("all") {
		vs.Name = "all"
	} else {
		vs.Name = p.colId()
	}

	return vs
}

// parseShowStmt parses SHOW name | SHOW ALL.
func (p *Parser) parseShowStmt() *VariableShowStmt {
	p.wantKeyword("show")
	pos := p.pos

	vs := &VariableShowStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.gotKeyword("all") {
		vs.Name = "all"
	} else {
		vs.Name = p.colId()
	}

	return vs
}

// parseListenStmt parses LISTEN channel.
func (p *Parser) parseListenStmt() *ListenStmt {
	p.wantKeyword("listen")
	return &ListenStmt{
		baseStmt:      baseStmt{baseNode{p.pos}},
		Conditionname: p.colId(),
	}
}

// parseNotifyStmt parses NOTIFY channel [, payload].
func (p *Parser) parseNotifyStmt() *NotifyStmt {
	p.wantKeyword("notify")
	pos := p.pos
	ns := &NotifyStmt{baseStmt: baseStmt{baseNode{pos}}}
	ns.Conditionname = p.colId()
	if p.gotSelf(',') {
		if p.tok == SCONST {
			ns.Payload = p.lit
			p.next()
		}
	}
	return ns
}

// parseUnlistenStmt parses UNLISTEN channel | UNLISTEN *.
func (p *Parser) parseUnlistenStmt() *UnlistenStmt {
	p.wantKeyword("unlisten")
	pos := p.pos
	us := &UnlistenStmt{baseStmt: baseStmt{baseNode{pos}}}
	if p.gotSelf('*') {
		us.Conditionname = ""
	} else {
		us.Conditionname = p.colId()
	}
	return us
}

// parseVacuumStmt parses VACUUM [options] [table_list].
func (p *Parser) parseVacuumStmt() *VacuumStmt {
	isVacuum := p.isKeyword("vacuum")
	p.next() // consume VACUUM or ANALYZE
	pos := p.pos

	vs := &VacuumStmt{baseStmt: baseStmt{baseNode{pos}}, IsVacuum: isVacuum}

	// Options in parens
	if p.tok == Token('(') {
		p.next()
		for p.tok != Token(')') && p.tok != EOF {
			opt := &DefElem{baseNode: baseNode{p.pos}, Defname: p.colLabel()}
			if p.tok == IDENT || p.tok == KEYWORD || p.tok == ICONST {
				if p.tok == ICONST {
					opt.Arg = &A_Const{baseExpr: baseExpr{baseNode{p.pos}}, Val: Value{Type: ValInt, Ival: p.parseInt()}}
				} else {
					opt.Arg = &String{baseNode: baseNode{p.pos}, Str: p.lit}
					p.next()
				}
			}
			vs.Options = append(vs.Options, opt)
			p.gotSelf(',')
		}
		p.wantSelf(')')
	} else if isVacuum {
		// Legacy options: FULL, FREEZE, VERBOSE, ANALYZE
		for p.isAnyKeyword("full", "freeze", "verbose", "analyze", "analyse") {
			vs.Options = append(vs.Options, &DefElem{baseNode: baseNode{p.pos}, Defname: p.lit})
			p.next()
		}
	} else {
		// ANALYZE: optional VERBOSE
		if p.gotKeyword("verbose") {
			vs.Options = append(vs.Options, &DefElem{baseNode: baseNode{p.pos}, Defname: "verbose"})
		}
	}

	// Optional table list
	if p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
		vs.Relations = append(vs.Relations, p.parseRangeVar())
		for p.gotSelf(',') {
			vs.Relations = append(vs.Relations, p.parseRangeVar())
		}
	}

	return vs
}

// parseLockStmt parses LOCK [TABLE] name [, ...] [IN mode MODE] [NOWAIT].
func (p *Parser) parseLockStmt() *LockStmt {
	p.wantKeyword("lock")
	pos := p.pos
	p.gotKeyword("table") // optional

	ls := &LockStmt{baseStmt: baseStmt{baseNode{pos}}}

	ls.Relations = append(ls.Relations, p.parseRangeVar())
	for p.gotSelf(',') {
		ls.Relations = append(ls.Relations, p.parseRangeVar())
	}

	if p.gotKeyword("in") {
		// Collect mode words until MODE keyword
		var modeWords []string
		for !p.isKeyword("mode") && p.tok != EOF {
			modeWords = append(modeWords, p.lit)
			p.next()
		}
		p.wantKeyword("mode")
		for i, w := range modeWords {
			if i > 0 {
				ls.Mode += " "
			}
			ls.Mode += w
		}
	}

	if p.gotKeyword("nowait") {
		ls.Nowait = true
	}

	return ls
}

// parsePrepareStmt parses PREPARE name [(types)] AS stmt.
func (p *Parser) parsePrepareStmt() Stmt {
	p.wantKeyword("prepare")
	pos := p.pos

	// Could be PREPARE TRANSACTION 'id' (two-phase commit)
	if p.isKeyword("transaction") {
		ts := &TransactionStmt{baseStmt: baseStmt{baseNode{pos}}, Kind: TRANS_STMT_PREPARE}
		p.next()
		if p.tok == SCONST {
			ts.Options = []string{p.lit}
			p.next()
		}
		return ts
	}

	ps := &PrepareStmt{baseStmt: baseStmt{baseNode{pos}}}
	ps.Name = p.colId()

	// Optional parameter types
	if p.tok == Token('(') {
		p.next()
		ps.Argtypes = append(ps.Argtypes, p.parseTypeName())
		for p.gotSelf(',') {
			ps.Argtypes = append(ps.Argtypes, p.parseTypeName())
		}
		p.wantSelf(')')
	}

	p.wantKeyword("as")
	ps.Query = p.parseSimpleStmt()
	return ps
}

// parseExecuteStmt parses EXECUTE name [(params)].
func (p *Parser) parseExecuteStmt() *ExecuteStmt {
	p.wantKeyword("execute")
	pos := p.pos

	es := &ExecuteStmt{baseStmt: baseStmt{baseNode{pos}}}
	es.Name = p.colId()

	if p.tok == Token('(') {
		p.next()
		es.Params = p.parseExprList()
		p.wantSelf(')')
	}

	return es
}

// parseDeallocateStmt parses DEALLOCATE [PREPARE] name | ALL.
func (p *Parser) parseDeallocateStmt() *DeallocateStmt {
	p.wantKeyword("deallocate")
	pos := p.pos
	p.gotKeyword("prepare") // optional

	ds := &DeallocateStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.gotKeyword("all") {
		ds.IsAll = true
	} else {
		ds.Name = p.colId()
	}

	return ds
}

// parseDiscardStmt parses DISCARD ALL | PLANS | SEQUENCES | TEMP.
func (p *Parser) parseDiscardStmt() *DiscardStmt {
	p.wantKeyword("discard")
	pos := p.pos
	ds := &DiscardStmt{baseStmt: baseStmt{baseNode{pos}}}
	ds.Target = p.colLabel()
	return ds
}
