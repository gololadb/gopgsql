package pgscan

// parseAlterRole parses ALTER ROLE/USER name [WITH] options or ALTER ROLE name SET/RESET config.
func (p *Parser) parseAlterRole() Stmt {
	p.next() // consume ROLE or USER
	pos := p.pos
	roleName := p.colId()

	// ALTER ROLE name SET/RESET
	if p.isKeyword("set") || p.isKeyword("reset") {
		var setStmt Stmt
		if p.isKeyword("set") {
			setStmt = p.parseSetStmt()
		} else {
			setStmt = p.parseResetStmt()
		}
		return &AlterRoleSetStmt{
			baseStmt: baseStmt{baseNode{pos}},
			RoleName: roleName,
			SetStmt:  setStmt,
		}
	}

	// ALTER ROLE name [WITH] options
	p.gotKeyword("with")
	return &AlterRoleStmt{
		baseStmt: baseStmt{baseNode{pos}},
		RoleName: roleName,
		Options:  p.parseRoleOptions(),
	}
}

// parseAlterDomain parses ALTER DOMAIN name action.
func (p *Parser) parseAlterDomain() *AlterDomainStmt {
	p.wantKeyword("domain")
	pos := p.pos

	ad := &AlterDomainStmt{baseStmt: baseStmt{baseNode{pos}}}
	ad.TypeName = p.parseQualifiedName()

	switch {
	case p.isKeyword("set"):
		p.next()
		if p.gotKeyword("not") {
			p.wantKeyword("null")
			ad.Subtype = 'O' // SET NOT NULL
		} else if p.gotKeyword("default") {
			ad.Subtype = 'T' // SET DEFAULT
			ad.Def = p.parseExpr()
		} else {
			p.syntaxError("expected NOT NULL or DEFAULT after SET")
		}
	case p.isKeyword("drop"):
		p.next()
		if p.gotKeyword("not") {
			p.wantKeyword("null")
			ad.Subtype = 'N' // DROP NOT NULL
		} else if p.gotKeyword("default") {
			ad.Subtype = 'N' // DROP DEFAULT
		} else if p.gotKeyword("constraint") {
			ad.Subtype = 'X' // DROP CONSTRAINT
			if p.isKeyword("if") {
				p.next()
				p.wantKeyword("exists")
				ad.MissingOk = true
			}
			ad.Name = p.colId()
		}
	case p.isKeyword("add"):
		p.next()
		ad.Subtype = 'C' // ADD CONSTRAINT
		ad.Constraint = p.parseTableConstraint()
	case p.isKeyword("validate"):
		p.next()
		p.wantKeyword("constraint")
		ad.Subtype = 'V'
		ad.Name = p.colId()
	default:
		p.syntaxError("expected SET, DROP, ADD, or VALIDATE after ALTER DOMAIN name")
	}

	return ad
}

// parseAlterType parses ALTER TYPE name action.
// Dispatches to AlterEnumStmt or AlterTypeStmt.
func (p *Parser) parseAlterType() Stmt {
	p.wantKeyword("type")
	pos := p.pos
	typeName := p.parseQualifiedName()

	switch {
	case p.isKeyword("add"):
		p.next()
		if p.gotKeyword("value") {
			// ALTER TYPE name ADD VALUE [IF NOT EXISTS] 'val' [BEFORE|AFTER 'neighbor']
			ae := &AlterEnumStmt{baseStmt: baseStmt{baseNode{pos}}, TypeName: typeName}
			if p.isKeyword("if") {
				p.next()
				p.wantKeyword("not")
				p.wantKeyword("exists")
				ae.IfNotExists = true
			}
			if p.tok == SCONST {
				ae.NewVal = p.lit
				p.next()
			}
			if p.gotKeyword("before") {
				if p.tok == SCONST {
					ae.NewValNeighbor = p.lit
					p.next()
				}
				ae.NewValIsAfter = false
			} else if p.gotKeyword("after") {
				if p.tok == SCONST {
					ae.NewValNeighbor = p.lit
					p.next()
				}
				ae.NewValIsAfter = true
			}
			return ae
		}
		// ALTER TYPE name ADD ATTRIBUTE ...
		return &AlterTypeStmt{
			baseStmt: baseStmt{baseNode{pos}},
			TypeName: typeName,
		}
	case p.isKeyword("rename"):
		p.next()
		if p.gotKeyword("value") {
			// ALTER TYPE name RENAME VALUE 'old' TO 'new'
			ae := &AlterEnumStmt{baseStmt: baseStmt{baseNode{pos}}, TypeName: typeName}
			if p.tok == SCONST {
				ae.RenameOldVal = p.lit
				p.next()
			}
			p.wantKeyword("to")
			if p.tok == SCONST {
				ae.NewVal = p.lit
				p.next()
			}
			return ae
		}
		// Other RENAME forms handled generically
		return &AlterTypeStmt{
			baseStmt: baseStmt{baseNode{pos}},
			TypeName: typeName,
		}
	default:
		// Generic ALTER TYPE (SET SCHEMA, OWNER TO, etc.)
		return &AlterTypeStmt{
			baseStmt: baseStmt{baseNode{pos}},
			TypeName: typeName,
		}
	}
}

// parseAlterFunction parses ALTER FUNCTION/PROCEDURE name (args) action.
func (p *Parser) parseAlterFunction() *AlterFunctionStmt {
	objType := byte('f')
	if p.isKeyword("procedure") {
		objType = 'p'
	}
	p.next() // consume FUNCTION or PROCEDURE
	pos := p.pos

	af := &AlterFunctionStmt{
		baseStmt: baseStmt{baseNode{pos}},
		ObjType:  objType,
	}

	funcName := p.parseQualifiedName()
	af.Func = &FuncWithArgs{baseNode: baseNode{pos}, Funcname: funcName}

	// Parse argument types if present
	if p.gotSelf('(') {
		if p.tok != Token(')') {
			af.Func.Funcargs = append(af.Func.Funcargs, p.parseTypeName())
			for p.gotSelf(',') {
				af.Func.Funcargs = append(af.Func.Funcargs, p.parseTypeName())
			}
		}
		p.wantSelf(')')
	}

	// Parse actions
	for {
		switch {
		case p.gotKeyword("rename"):
			p.wantKeyword("to")
			af.Actions = append(af.Actions, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "rename",
				Arg:      &String{baseNode: baseNode{p.pos}, Str: p.colId()},
			})
		case p.isKeyword("owner"):
			p.next()
			p.wantKeyword("to")
			af.Actions = append(af.Actions, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "owner",
				Arg:      &String{baseNode: baseNode{p.pos}, Str: p.colId()},
			})
		case p.isKeyword("set"):
			p.next()
			if p.gotKeyword("schema") {
				af.Actions = append(af.Actions, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "schema",
					Arg:      &String{baseNode: baseNode{p.pos}, Str: p.colId()},
				})
			} else {
				// SET config_param = value
				name := p.colId()
				var val string
				if p.gotKeyword("to") || p.gotSelf('=') {
					val = p.colId()
				}
				af.Actions = append(af.Actions, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  name,
					Arg:      &String{baseNode: baseNode{p.pos}, Str: val},
				})
			}
		case p.isKeyword("security"):
			// SECURITY DEFINER / SECURITY INVOKER
			p.next()
			af.Actions = append(af.Actions, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "security",
				Arg:      &String{baseNode: baseNode{p.pos}, Str: p.colId()},
			})
		case p.tok == IDENT:
			// Identifiers like STRICT, IMMUTABLE, STABLE, VOLATILE, COST, ROWS, PARALLEL, etc.
			name := p.lit
			p.next()
			switch name {
			case "strict", "immutable", "stable", "volatile", "leakproof":
				af.Actions = append(af.Actions, &DefElem{baseNode: baseNode{p.pos}, Defname: name})
			case "cost":
				af.Actions = append(af.Actions, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "cost",
					Arg:      &String{baseNode: baseNode{p.pos}, Str: p.lit},
				})
				p.next()
			case "rows":
				af.Actions = append(af.Actions, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "rows",
					Arg:      &String{baseNode: baseNode{p.pos}, Str: p.lit},
				})
				p.next()
			case "parallel":
				af.Actions = append(af.Actions, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "parallel",
					Arg:      &String{baseNode: baseNode{p.pos}, Str: p.lit},
				})
				p.next()
			default:
				// Unknown action, stop
				return af
			}
		default:
			return af
		}
	}
}

// parseAlterPolicy parses ALTER POLICY name ON table [TO roles] [USING (expr)] [WITH CHECK (expr)].
func (p *Parser) parseAlterPolicy() *AlterPolicyStmt {
	p.wantKeyword("policy")
	pos := p.pos

	ap := &AlterPolicyStmt{baseStmt: baseStmt{baseNode{pos}}}
	ap.PolicyName = p.colId()
	p.wantKeyword("on")
	ap.Table = p.parseQualifiedName()

	if p.gotKeyword("to") {
		ap.Roles = p.parseRoleList()
	}

	if p.isKeyword("using") {
		p.next()
		p.wantSelf('(')
		ap.Qual = p.parseExpr()
		p.wantSelf(')')
	}

	if p.isKeyword("with") {
		p.next()
		p.wantKeyword("check")
		p.wantSelf('(')
		ap.WithCheck = p.parseExpr()
		p.wantSelf(')')
	}

	return ap
}

// parseAlterPublication parses ALTER PUBLICATION name action.
func (p *Parser) parseAlterPublication() *AlterPublicationStmt {
	p.wantKeyword("publication")
	pos := p.pos

	ap := &AlterPublicationStmt{baseStmt: baseStmt{baseNode{pos}}}
	ap.Pubname = p.colId()

	switch {
	case p.isKeyword("add"):
		p.next()
		p.wantKeyword("table")
		ap.TableAction = "add"
		ap.Tables = append(ap.Tables, p.parseQualifiedName())
		for p.gotSelf(',') {
			ap.Tables = append(ap.Tables, p.parseQualifiedName())
		}
	case p.isKeyword("drop"):
		p.next()
		p.wantKeyword("table")
		ap.TableAction = "drop"
		ap.Tables = append(ap.Tables, p.parseQualifiedName())
		for p.gotSelf(',') {
			ap.Tables = append(ap.Tables, p.parseQualifiedName())
		}
	case p.isKeyword("set"):
		p.next()
		if p.gotKeyword("table") {
			ap.TableAction = "set"
			ap.Tables = append(ap.Tables, p.parseQualifiedName())
			for p.gotSelf(',') {
				ap.Tables = append(ap.Tables, p.parseQualifiedName())
			}
		} else {
			// SET (options)
			p.wantSelf('(')
			ap.Options = p.parseParenDefElemListInner()
			p.wantSelf(')')
		}
	}

	return ap
}

// parseParenDefElemListInner parses name = value, ... without the outer parens.
func (p *Parser) parseParenDefElemListInner() []*DefElem {
	var opts []*DefElem
	for {
		name := p.colId()
		var arg Node
		if p.gotSelf('=') {
			switch {
			case p.tok == SCONST:
				arg = &String{baseNode: baseNode{p.pos}, Str: p.lit}
				p.next()
			case p.tok == ICONST:
				arg = &A_Const{baseExpr: baseExpr{baseNode{p.pos}}, Val: Value{Type: ValInt, Ival: p.parseInt()}}
			case p.isAnyKeyword("true", "false", "on", "off"):
				arg = &String{baseNode: baseNode{p.pos}, Str: p.lit}
				p.next()
			default:
				arg = &String{baseNode: baseNode{p.pos}, Str: p.colId()}
			}
		}
		opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: name, Arg: arg})
		if !p.gotSelf(',') {
			break
		}
	}
	return opts
}

// parseAlterSubscription parses ALTER SUBSCRIPTION name action.
func (p *Parser) parseAlterSubscription() *AlterSubscriptionStmt {
	p.wantKeyword("subscription")
	pos := p.pos

	as := &AlterSubscriptionStmt{baseStmt: baseStmt{baseNode{pos}}}
	as.Subname = p.colId()

	switch {
	case p.gotKeyword("connection"):
		as.Kind = "connection"
		if p.tok == SCONST {
			as.Conninfo = p.lit
			p.next()
		}
	case p.gotKeyword("set"):
		if p.gotKeyword("publication") {
			as.Kind = "publication"
			as.Publication = append(as.Publication, p.colId())
			for p.gotSelf(',') {
				as.Publication = append(as.Publication, p.colId())
			}
		} else {
			as.Kind = "options"
			p.wantSelf('(')
			as.Options = p.parseParenDefElemListInner()
			p.wantSelf(')')
		}
	case p.gotKeyword("add"):
		p.wantKeyword("publication")
		as.Kind = "add_publication"
		as.Publication = append(as.Publication, p.colId())
		for p.gotSelf(',') {
			as.Publication = append(as.Publication, p.colId())
		}
	case p.gotKeyword("drop"):
		p.wantKeyword("publication")
		as.Kind = "drop_publication"
		as.Publication = append(as.Publication, p.colId())
		for p.gotSelf(',') {
			as.Publication = append(as.Publication, p.colId())
		}
	case p.isKeyword("enable"):
		p.next()
		as.Kind = "enable"
	case p.isKeyword("disable"):
		p.next()
		as.Kind = "disable"
	case p.isKeyword("refresh"):
		p.next()
		p.wantKeyword("publication")
		as.Kind = "refresh"
	}

	// Optional WITH (options)
	if p.isKeyword("with") {
		p.next()
		p.wantSelf('(')
		as.Options = p.parseParenDefElemListInner()
		p.wantSelf(')')
	}

	return as
}

// parseAlterEventTrigger parses ALTER EVENT TRIGGER name action.
func (p *Parser) parseAlterEventTrigger() *AlterEventTrigStmt {
	p.wantKeyword("trigger")
	pos := p.pos

	ae := &AlterEventTrigStmt{baseStmt: baseStmt{baseNode{pos}}}
	ae.Trigname = p.colId()

	switch {
	case p.isKeyword("enable"):
		p.next()
		ae.Tgenabled = 'O'
		if p.gotKeyword("replica") {
			ae.Tgenabled = 'R'
		} else if p.gotKeyword("always") {
			ae.Tgenabled = 'A'
		}
	case p.isKeyword("disable"):
		p.next()
		ae.Tgenabled = 'D'
	}

	return ae
}

// parseAlterSystem parses ALTER SYSTEM SET/RESET config.
func (p *Parser) parseAlterSystem() *AlterSystemStmt {
	p.wantKeyword("system")
	pos := p.pos

	as := &AlterSystemStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.isKeyword("set") {
		as.SetStmt = p.parseSetStmt()
	} else if p.isKeyword("reset") {
		as.SetStmt = p.parseResetStmt()
	} else {
		p.syntaxError("expected SET or RESET after ALTER SYSTEM")
	}

	return as
}
