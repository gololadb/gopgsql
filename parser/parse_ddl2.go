package parser

// ---------------------------------------------------------------------------
// CREATE INDEX
// ---------------------------------------------------------------------------

// parseCreateIndex parses CREATE [UNIQUE] INDEX ...
func (p *Parser) parseCreateIndex(concurrent bool) *IndexStmt {
	idx := &IndexStmt{baseStmt: baseStmt{baseNode{p.pos}}}

	if p.gotKeyword("unique") {
		idx.Unique = true
	}
	p.wantKeyword("index")

	if p.gotKeyword("concurrently") {
		idx.Concurrent = true
	}

	// IF NOT EXISTS
	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("not")
		p.wantKeyword("exists")
		idx.IfNotExists = true
	}

	// Optional index name (may be omitted)
	if (p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword)) && !p.isKeyword("on") {
		idx.Idxname = p.colId()
	}

	p.wantKeyword("on")
	idx.Relation = p.parseRangeVar()

	// Optional USING access_method
	if p.gotKeyword("using") {
		idx.AccessMethod = p.colId()
	}

	// (index_params)
	p.wantSelf('(')
	idx.IndexParams = p.parseIndexParams()
	p.wantSelf(')')

	// Optional NULLS [NOT] DISTINCT
	if p.isKeyword("nulls") {
		p.next()
		p.wantKeyword("not")
		p.wantKeyword("distinct")
		idx.NullsNotDistinct = true
	}

	// Optional WHERE clause
	if p.gotKeyword("where") {
		idx.WhereClause = p.parseExpr()
	}

	return idx
}

// parseIndexParams parses a comma-separated list of index elements.
func (p *Parser) parseIndexParams() []*IndexElem {
	var params []*IndexElem
	params = append(params, p.parseIndexElem())
	for p.gotSelf(',') {
		params = append(params, p.parseIndexElem())
	}
	return params
}

// parseIndexElem parses a single index element.
func (p *Parser) parseIndexElem() *IndexElem {
	pos := p.pos
	elem := &IndexElem{baseNode: baseNode{pos}}

	if p.tok == Token('(') {
		p.next()
		elem.Expr = p.parseExpr()
		p.wantSelf(')')
	} else {
		elem.Name = p.colId()
	}

	if p.gotKeyword("asc") {
		elem.Ordering = SORTBY_ASC
	} else if p.gotKeyword("desc") {
		elem.Ordering = SORTBY_DESC
	}

	if p.isKeyword("nulls") {
		p.next()
		if p.gotKeyword("first") {
			elem.NullsOrder = SORTBY_NULLS_FIRST
		} else {
			p.wantKeyword("last")
			elem.NullsOrder = SORTBY_NULLS_LAST
		}
	}

	return elem
}

// ---------------------------------------------------------------------------
// ALTER dispatch
// ---------------------------------------------------------------------------

func (p *Parser) parseAlterStmt() Stmt {
	p.wantKeyword("alter")

	switch {
	case p.isKeyword("table"):
		return p.parseAlterTableStmt()
	case p.isKeyword("sequence"):
		return p.parseAlterSequence()
	case p.isKeyword("extension"):
		return p.parseAlterExtension()
	case p.isKeyword("database"):
		return p.parseAlterDatabase()
	case p.isAnyKeyword("role", "user"):
		return p.parseAlterRole()
	case p.isKeyword("domain"):
		return p.parseAlterDomain()
	case p.isKeyword("type"):
		return p.parseAlterType()
	case p.isAnyKeyword("function", "procedure"):
		return p.parseAlterFunction()
	case p.isKeyword("policy"):
		return p.parseAlterPolicy()
	case p.isKeyword("publication"):
		return p.parseAlterPublication()
	case p.isKeyword("subscription"):
		return p.parseAlterSubscription()
	case p.isKeyword("event"):
		p.next()
		return p.parseAlterEventTrigger()
	case p.isKeyword("system"):
		return p.parseAlterSystem()
	case p.isKeyword("default"):
		return p.parseAlterDefaultPrivileges()
	case p.isKeyword("statistics"):
		return p.parseAlterStatistics()
	case p.isKeyword("schema"):
		return p.parseAlterSchema()
	case p.isKeyword("aggregate"):
		return p.parseAlterAggregate()
	default:
		p.syntaxError("expected object type after ALTER")
		p.next()
		return nil
	}
}

// ALTER TABLE
// ---------------------------------------------------------------------------

func (p *Parser) parseAlterTableStmt() *AlterTableStmt {
	p.wantKeyword("table")

	stmt := &AlterTableStmt{baseStmt: baseStmt{baseNode{p.pos}}}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("exists")
		stmt.MissingOk = true
	}

	inh := true
	if p.gotKeyword("only") {
		inh = false
	}

	stmt.Relation = p.parseRangeVar()
	stmt.Relation.Inh = inh

	if p.isKeyword("rename") {
		return p.parseAlterTableRename(stmt)
	}

	stmt.Cmds = p.parseAlterTableCmds()
	return stmt
}

func (p *Parser) parseAlterTableRename(stmt *AlterTableStmt) *AlterTableStmt {
	p.wantKeyword("rename")

	if p.gotKeyword("column") {
		oldName := p.colId()
		p.wantKeyword("to")
		newName := p.colId()
		stmt.Cmds = []*AlterTableCmd{{
			baseNode: baseNode{p.pos},
			Subtype:  AT_RenameColumn,
			Name:     oldName,
			Def:      &String{baseNode: baseNode{p.pos}, Str: newName},
		}}
	} else if p.gotKeyword("to") {
		newName := p.colId()
		stmt.Cmds = []*AlterTableCmd{{
			baseNode: baseNode{p.pos},
			Subtype:  AT_RenameTable,
			Name:     newName,
		}}
	} else {
		oldName := p.colId()
		p.wantKeyword("to")
		newName := p.colId()
		stmt.Cmds = []*AlterTableCmd{{
			baseNode: baseNode{p.pos},
			Subtype:  AT_RenameColumn,
			Name:     oldName,
			Def:      &String{baseNode: baseNode{p.pos}, Str: newName},
		}}
	}
	return stmt
}

func (p *Parser) parseAlterTableCmds() []*AlterTableCmd {
	var cmds []*AlterTableCmd
	cmds = append(cmds, p.parseAlterTableCmd())
	for p.gotSelf(',') {
		cmds = append(cmds, p.parseAlterTableCmd())
	}
	return cmds
}

func (p *Parser) parseAlterTableCmd() *AlterTableCmd {
	pos := p.pos

	switch {
	case p.isKeyword("add"):
		p.next()
		if p.gotKeyword("column") {
			cmd := &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_AddColumn}
			if p.isKeyword("if") {
				p.next()
				p.wantKeyword("not")
				p.wantKeyword("exists")
				cmd.MissingOk = true
			}
			cmd.Def = p.parseColumnDef()
			return cmd
		}
		if p.isKeyword("constraint") || p.isKeyword("check") || p.isKeyword("unique") ||
			p.isKeyword("primary") || p.isKeyword("foreign") {
			cmd := &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_AddConstraint}
			cmd.Def = p.parseTableConstraint()
			return cmd
		}
		cmd := &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_AddColumn}
		cmd.Def = p.parseColumnDef()
		return cmd

	case p.isKeyword("drop"):
		p.next()
		if p.gotKeyword("column") {
			cmd := &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DropColumn}
			if p.isKeyword("if") {
				p.next()
				p.wantKeyword("exists")
				cmd.MissingOk = true
			}
			cmd.Name = p.colId()
			if p.gotKeyword("cascade") {
				cmd.Behavior = DROP_CASCADE
			} else {
				p.gotKeyword("restrict")
			}
			return cmd
		}
		if p.gotKeyword("constraint") {
			cmd := &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DropConstraint}
			if p.isKeyword("if") {
				p.next()
				p.wantKeyword("exists")
				cmd.MissingOk = true
			}
			cmd.Name = p.colId()
			if p.gotKeyword("cascade") {
				cmd.Behavior = DROP_CASCADE
			} else {
				p.gotKeyword("restrict")
			}
			return cmd
		}
		cmd := &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DropColumn}
		cmd.Name = p.colId()
		if p.gotKeyword("cascade") {
			cmd.Behavior = DROP_CASCADE
		} else {
			p.gotKeyword("restrict")
		}
		return cmd

	case p.isKeyword("alter"):
		p.next()
		if p.gotKeyword("constraint") {
			name := p.colId()
			// optional DEFERRABLE / NOT DEFERRABLE / INITIALLY ...
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_AlterConstraint, Name: name}
		}
		p.gotKeyword("column")
		colName := p.colId()

		switch {
		case p.isKeyword("set"):
			p.next()
			if p.gotKeyword("not") {
				p.wantKeyword("null")
				return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_SetNotNull, Name: colName}
			}
			if p.gotKeyword("default") {
				expr := p.parseExpr()
				return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_SetDefault, Name: colName, Def: expr.(Node)}
			}
			if p.gotKeyword("data") {
				p.wantKeyword("type")
				tn := p.parseTypeName()
				return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_AlterColumnType, Name: colName, Def: tn}
			}
			if p.isKeyword("statistics") {
				p.next()
				return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_SetStatistics, Name: colName,
					Def: &A_Const{baseExpr: baseExpr{baseNode{p.pos}}, Val: Value{Type: ValInt, Ival: p.parseInt()}}}
			}
			if p.isKeyword("storage") {
				p.next()
				return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_SetStorage, Name: colName,
					Def: &String{baseNode: baseNode{p.pos}, Str: p.colId()}}
			}
			if p.isKeyword("compression") {
				p.next()
				return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_SetCompression, Name: colName,
					Def: &String{baseNode: baseNode{p.pos}, Str: p.colId()}}
			}
			if p.isKeyword("expression") {
				p.next()
				p.wantKeyword("as")
				p.wantSelf('(')
				expr := p.parseExpr()
				p.wantSelf(')')
				return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_SetExpression, Name: colName, Def: expr.(Node)}
			}

		case p.isKeyword("drop"):
			p.next()
			if p.gotKeyword("not") {
				p.wantKeyword("null")
				return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DropNotNull, Name: colName}
			}
			if p.gotKeyword("default") {
				return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DropDefault, Name: colName}
			}
			if p.isKeyword("identity") {
				p.next()
				cmd := &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DropIdentity, Name: colName}
				if p.isKeyword("if") {
					p.next()
					p.wantKeyword("exists")
					cmd.MissingOk = true
				}
				return cmd
			}
			if p.isKeyword("expression") {
				p.next()
				cmd := &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DropExpression, Name: colName}
				if p.isKeyword("if") {
					p.next()
					p.wantKeyword("exists")
					cmd.MissingOk = true
				}
				return cmd
			}

		case p.isKeyword("type"):
			p.next()
			tn := p.parseTypeName()
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_AlterColumnType, Name: colName, Def: tn}

		case p.isKeyword("add"):
			p.next()
			p.wantKeyword("generated")
			// ALWAYS or BY DEFAULT
			if p.gotKeyword("always") {
				// ALWAYS
			} else {
				p.wantKeyword("by")
				p.wantKeyword("default")
			}
			p.wantKeyword("as")
			p.wantKeyword("identity")
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_AddIdentity, Name: colName}
		}

	case p.isKeyword("validate"):
		p.next()
		p.wantKeyword("constraint")
		return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_ValidateConstraint, Name: p.colId()}

	case p.isKeyword("cluster"):
		p.next()
		p.wantKeyword("on")
		return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_ClusterOn, Name: p.colId()}

	case p.isKeyword("set"):
		p.next()
		switch {
		case p.isKeyword("schema"):
			p.next()
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_SetSchema, Name: p.colId()}
		case p.isKeyword("without"):
			p.next()
			if p.gotKeyword("cluster") {
				return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DropCluster}
			}
			// SET WITHOUT OIDS (legacy, ignore)
			p.wantKeyword("oids")
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DropCluster}
		case p.isKeyword("logged"):
			p.next()
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_SetLogged}
		case p.isKeyword("unlogged"):
			p.next()
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_SetUnLogged}
		case p.isKeyword("tablespace"):
			p.next()
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_SetTableSpace, Name: p.colId()}
		case p.isKeyword("access"):
			p.next()
			p.wantKeyword("method")
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_SetAccessMethod, Name: p.colId()}
		case p.tok == Token('('):
			// SET (reloptions)
			p.next()
			// Parse options as DefElem list — store in Def
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_SetRelOptions}
		}

	case p.isKeyword("reset"):
		p.next()
		if p.tok == Token('(') {
			p.next()
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_ResetRelOptions}
		}

	case p.isKeyword("enable"):
		p.next()
		return p.parseAlterTableEnableDisable(pos, true)

	case p.isKeyword("disable"):
		p.next()
		return p.parseAlterTableEnableDisable(pos, false)

	case p.isKeyword("inherit"):
		p.next()
		return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_AddInherit,
			Def: &RangeVar{baseNode: baseNode{p.pos}, Relname: p.colId()}}

	case p.isKeyword("no"):
		p.next()
		if p.gotKeyword("inherit") {
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DropInherit,
				Def: &RangeVar{baseNode: baseNode{p.pos}, Relname: p.colId()}}
		}
		if p.gotKeyword("force") {
			p.wantKeyword("row")
			p.wantKeyword("level")
			p.wantKeyword("security")
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_NoForceRowSecurity}
		}

	case p.isKeyword("of"):
		p.next()
		return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_AddOf, Name: p.colId()}

	case p.isKeyword("not"):
		p.next()
		p.wantKeyword("of")
		return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DropOf}

	case p.isKeyword("owner"):
		p.next()
		p.wantKeyword("to")
		return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_ChangeOwner, Name: p.colId()}

	case p.isKeyword("replica"):
		p.next()
		p.wantKeyword("identity")
		// DEFAULT, USING INDEX name, FULL, NOTHING
		switch {
		case p.gotKeyword("default"):
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_ReplicaIdentity, Name: "default"}
		case p.gotKeyword("full"):
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_ReplicaIdentity, Name: "full"}
		case p.gotKeyword("nothing"):
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_ReplicaIdentity, Name: "nothing"}
		case p.isKeyword("using"):
			p.next()
			p.wantKeyword("index")
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_ReplicaIdentity, Name: p.colId()}
		}

	case p.isKeyword("force"):
		p.next()
		p.wantKeyword("row")
		p.wantKeyword("level")
		p.wantKeyword("security")
		return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_ForceRowSecurity}
	}

	p.syntaxError("expected ADD, DROP, ALTER, SET, ENABLE, DISABLE, VALIDATE, CLUSTER, INHERIT, OWNER, REPLICA, or FORCE in ALTER TABLE command")
	p.next()
	return &AlterTableCmd{baseNode: baseNode{pos}}
}

// ---------------------------------------------------------------------------
// DROP
// ---------------------------------------------------------------------------

// parseDropDispatch handles DROP, dispatching to specialized parsers for
// DATABASE and other types that return non-DropStmt nodes.
// parseAlterTableEnableDisable handles ENABLE/DISABLE TRIGGER/RULE/ROW LEVEL SECURITY.
func (p *Parser) parseAlterTableEnableDisable(pos int, enable bool) *AlterTableCmd {
	if p.gotKeyword("trigger") {
		if p.gotKeyword("all") {
			if enable {
				return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_EnableTrigAll}
			}
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DisableTrigAll}
		}
		if p.gotKeyword("user") {
			if enable {
				return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_EnableTrigUser}
			}
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DisableTrigUser}
		}
		name := p.colId()
		if enable {
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_EnableTrig, Name: name}
		}
		return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DisableTrig, Name: name}
	}

	if enable && p.gotKeyword("always") {
		if p.gotKeyword("trigger") {
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_EnableAlwaysTrig, Name: p.colId()}
		}
		if p.gotKeyword("rule") {
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_EnableAlwaysRule, Name: p.colId()}
		}
	}

	if enable && p.isKeyword("replica") {
		p.next()
		if p.gotKeyword("trigger") {
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_EnableReplicaTrig, Name: p.colId()}
		}
		if p.gotKeyword("rule") {
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_EnableReplicaRule, Name: p.colId()}
		}
	}

	if p.gotKeyword("rule") {
		name := p.colId()
		if enable {
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_EnableRule, Name: name}
		}
		return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DisableRule, Name: name}
	}

	if p.gotKeyword("row") {
		p.wantKeyword("level")
		p.wantKeyword("security")
		if enable {
			return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_EnableRowSecurity}
		}
		return &AlterTableCmd{baseNode: baseNode{pos}, Subtype: AT_DisableRowSecurity}
	}

	p.syntaxError("expected TRIGGER, RULE, or ROW LEVEL SECURITY after ENABLE/DISABLE")
	return &AlterTableCmd{baseNode: baseNode{pos}}
}

func (p *Parser) parseDropDispatch() Stmt {
	p.wantKeyword("drop")

	// Specialized DROP forms with their own node types
	if p.isKeyword("database") {
		return p.parseDropDatabase()
	}
	if p.isAnyKeyword("role", "user", "group") {
		return p.parseDropRole()
	}
	if p.isKeyword("owned") {
		return p.parseDropOwned()
	}
	if p.isAnyKeyword("function", "procedure", "aggregate") {
		return p.parseDropFunction()
	}

	return p.parseDropStmtAfterDrop()
}

func (p *Parser) parseDropStmtAfterDrop() *DropStmt {
	pos := p.pos

	ds := &DropStmt{baseStmt: baseStmt{baseNode{pos}}}

	switch {
	case p.gotKeyword("table"):
		ds.RemoveType = OBJECT_TABLE
	case p.gotKeyword("index"):
		ds.RemoveType = OBJECT_INDEX
		if p.gotKeyword("concurrently") {
			ds.Concurrent = true
		}
	case p.gotKeyword("view"):
		ds.RemoveType = OBJECT_VIEW
	case p.isKeyword("materialized"):
		p.next()
		p.wantKeyword("view")
		ds.RemoveType = OBJECT_MATVIEW
	case p.gotKeyword("sequence"):
		ds.RemoveType = OBJECT_SEQUENCE
	case p.gotKeyword("type"):
		ds.RemoveType = OBJECT_TYPE
	case p.gotKeyword("schema"):
		ds.RemoveType = OBJECT_SCHEMA
	case p.gotKeyword("function"):
		ds.RemoveType = OBJECT_FUNCTION
	case p.gotKeyword("procedure"):
		ds.RemoveType = OBJECT_PROCEDURE
	case p.gotKeyword("extension"):
		ds.RemoveType = OBJECT_EXTENSION
	case p.gotKeyword("trigger"):
		ds.RemoveType = OBJECT_TRIGGER
	case p.gotKeyword("rule"):
		ds.RemoveType = OBJECT_RULE
	case p.gotKeyword("domain"):
		ds.RemoveType = OBJECT_DOMAIN
	default:
		p.syntaxError("expected object type after DROP")
	}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("exists")
		ds.MissingOk = true
	}

	ds.Objects = append(ds.Objects, p.parseQualifiedName())

	// TRIGGER and RULE have ON table_name
	if (ds.RemoveType == OBJECT_TRIGGER || ds.RemoveType == OBJECT_RULE) && p.gotKeyword("on") {
		ds.Objects = append(ds.Objects, p.parseQualifiedName())
	} else {
		for p.gotSelf(',') {
			ds.Objects = append(ds.Objects, p.parseQualifiedName())
		}
	}

	if p.gotKeyword("cascade") {
		ds.Behavior = DROP_CASCADE
	} else {
		p.gotKeyword("restrict")
	}

	return ds
}

// ---------------------------------------------------------------------------
// TRUNCATE
// ---------------------------------------------------------------------------

func (p *Parser) parseTruncateStmt() *TruncateStmt {
	p.wantKeyword("truncate")
	pos := p.pos

	ts := &TruncateStmt{baseStmt: baseStmt{baseNode{pos}}}

	p.gotKeyword("table")

	ts.Relations = append(ts.Relations, p.parseRangeVar())
	for p.gotSelf(',') {
		ts.Relations = append(ts.Relations, p.parseRangeVar())
	}

	if p.gotKeyword("restart") {
		p.wantKeyword("identity")
		ts.RestartSeqs = true
	} else if p.gotKeyword("continue") {
		p.wantKeyword("identity")
		ts.RestartSeqs = false
	}

	if p.gotKeyword("cascade") {
		ts.Behavior = DROP_CASCADE
	} else {
		p.gotKeyword("restrict")
	}

	return ts
}
