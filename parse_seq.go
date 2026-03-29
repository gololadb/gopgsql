package pgscan

// parseCreateSequence parses CREATE [TEMP] SEQUENCE [IF NOT EXISTS] name [options].
func (p *Parser) parseCreateSequence(temp bool) *CreateSeqStmt {
	p.wantKeyword("sequence")
	pos := p.pos

	cs := &CreateSeqStmt{baseStmt: baseStmt{baseNode{pos}}, Temp: temp}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("not")
		p.wantKeyword("exists")
		cs.IfNotExists = true
	}

	cs.Name = p.parseQualifiedName()
	cs.Options = p.parseSeqOptions()
	return cs
}

// parseAlterSequence parses ALTER SEQUENCE [IF EXISTS] name [options].
func (p *Parser) parseAlterSequence() *AlterSeqStmt {
	p.wantKeyword("sequence")
	pos := p.pos

	as := &AlterSeqStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("exists")
		as.IfExists = true
	}

	as.Name = p.parseQualifiedName()
	as.Options = p.parseSeqOptions()
	return as
}

// parseSeqOptions parses sequence options: INCREMENT, START, MINVALUE, MAXVALUE, CACHE, CYCLE, OWNED BY, etc.
func (p *Parser) parseSeqOptions() []*DefElem {
	var opts []*DefElem
	for {
		switch {
		case p.gotKeyword("increment"):
			p.gotKeyword("by") // optional BY
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "increment",
				Arg:      p.parseSignedIconst(),
			})
		case p.isKeyword("start"):
			p.next()
			p.gotKeyword("with") // optional WITH
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "start",
				Arg:      p.parseSignedIconst(),
			})
		case p.gotKeyword("minvalue"):
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "minvalue",
				Arg:      p.parseSignedIconst(),
			})
		case p.gotKeyword("maxvalue"):
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "maxvalue",
				Arg:      p.parseSignedIconst(),
			})
		case p.gotKeyword("cache"):
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "cache",
				Arg:      p.parseSignedIconst(),
			})
		case p.gotKeyword("cycle"):
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "cycle",
				Arg:      &A_Const{baseExpr: baseExpr{baseNode{p.pos}}, Val: Value{Type: ValInt, Ival: 1}},
			})
		case p.isKeyword("no"):
			p.next()
			switch {
			case p.gotKeyword("minvalue"):
				opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: "minvalue"})
			case p.gotKeyword("maxvalue"):
				opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: "maxvalue"})
			case p.gotKeyword("cycle"):
				opts = append(opts, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "cycle",
					Arg:      &A_Const{baseExpr: baseExpr{baseNode{p.pos}}, Val: Value{Type: ValInt, Ival: 0}},
				})
			default:
				p.syntaxError("expected MINVALUE, MAXVALUE, or CYCLE after NO")
				p.next()
			}
		case p.isKeyword("owned"):
			p.next()
			p.wantKeyword("by")
			if p.gotKeyword("none") {
				opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: "owned_by"})
			} else {
				name := p.parseQualifiedName()
				nameStr := ""
				for i, n := range name {
					if i > 0 {
						nameStr += "."
					}
					nameStr += n
				}
				opts = append(opts, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "owned_by",
					Arg:      &String{baseNode: baseNode{p.pos}, Str: nameStr},
				})
			}
		case p.isKeyword("as"):
			p.next()
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "as",
				Arg:      &String{baseNode: baseNode{p.pos}, Str: p.colId()},
			})
		case p.gotKeyword("restart"):
			p.gotKeyword("with") // optional WITH
			if p.tok == ICONST {
				opts = append(opts, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "restart",
					Arg:      p.parseSignedIconst(),
				})
			} else {
				opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: "restart"})
			}
		default:
			return opts
		}
	}
}

// parseSignedIconst parses an optional sign followed by an integer constant.
func (p *Parser) parseSignedIconst() Expr {
	neg := false
	if p.tok == '-' {
		neg = true
		p.next()
	} else if p.tok == '+' {
		p.next()
	}
	val := p.parseInt()
	if neg {
		val = -val
	}
	return &A_Const{baseExpr: baseExpr{baseNode{p.pos}}, Val: Value{Type: ValInt, Ival: val}}
}

// parseCreateExtension parses CREATE EXTENSION [IF NOT EXISTS] name [WITH] [SCHEMA schema] [VERSION version] [CASCADE].
func (p *Parser) parseCreateExtension() *CreateExtensionStmt {
	p.wantKeyword("extension")
	pos := p.pos

	ce := &CreateExtensionStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("not")
		p.wantKeyword("exists")
		ce.IfNotExists = true
	}

	ce.Extname = p.colId()

	// Optional WITH
	p.gotKeyword("with")

	// Options
	for {
		switch {
		case p.gotKeyword("schema"):
			ce.Options = append(ce.Options, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "schema",
				Arg:      &String{baseNode: baseNode{p.pos}, Str: p.colId()},
			})
		case p.gotKeyword("version"):
			var ver string
			if p.tok == SCONST {
				ver = p.lit
				p.next()
			} else {
				ver = p.colId()
			}
			ce.Options = append(ce.Options, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "version",
				Arg:      &String{baseNode: baseNode{p.pos}, Str: ver},
			})
		case p.gotKeyword("cascade"):
			ce.Options = append(ce.Options, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "cascade",
			})
		default:
			return ce
		}
	}
}

// parseAlterExtension parses ALTER EXTENSION name UPDATE [TO version].
func (p *Parser) parseAlterExtension() *AlterExtensionStmt {
	p.wantKeyword("extension")
	pos := p.pos

	ae := &AlterExtensionStmt{baseStmt: baseStmt{baseNode{pos}}}
	ae.Extname = p.colId()

	if p.gotKeyword("update") {
		if p.gotKeyword("to") {
			var ver string
			if p.tok == SCONST {
				ver = p.lit
				p.next()
			} else {
				ver = p.colId()
			}
			ae.Options = append(ae.Options, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "version",
				Arg:      &String{baseNode: baseNode{p.pos}, Str: ver},
			})
		}
	}

	return ae
}

// parseCreatePolicy parses CREATE POLICY name ON table [AS PERMISSIVE|RESTRICTIVE]
// [FOR command] [TO roles] [USING (expr)] [WITH CHECK (expr)].
func (p *Parser) parseCreatePolicy() *CreatePolicyStmt {
	p.wantKeyword("policy")
	pos := p.pos

	cp := &CreatePolicyStmt{baseStmt: baseStmt{baseNode{pos}}, Permissive: true}
	cp.PolicyName = p.colId()
	p.wantKeyword("on")
	cp.Table = p.parseQualifiedName()

	// Optional AS PERMISSIVE|RESTRICTIVE
	if p.gotKeyword("as") {
		id := p.colId()
		if id == "restrictive" {
			cp.Permissive = false
		}
	}

	// Optional FOR command
	if p.gotKeyword("for") {
		switch {
		case p.gotKeyword("all"):
			cp.CmdName = "ALL"
		case p.gotKeyword("select"):
			cp.CmdName = "SELECT"
		case p.gotKeyword("insert"):
			cp.CmdName = "INSERT"
		case p.gotKeyword("update"):
			cp.CmdName = "UPDATE"
		case p.gotKeyword("delete"):
			cp.CmdName = "DELETE"
		default:
			cp.CmdName = p.colId()
		}
	}

	// Optional TO roles
	if p.gotKeyword("to") {
		cp.Roles = p.parseRoleList()
	}

	// Optional USING (expr)
	if p.isKeyword("using") {
		p.next()
		p.wantSelf('(')
		cp.Qual = p.parseExpr()
		p.wantSelf(')')
	}

	// Optional WITH CHECK (expr)
	if p.isKeyword("with") {
		p.next()
		p.wantKeyword("check")
		p.wantSelf('(')
		cp.WithCheck = p.parseExpr()
		p.wantSelf(')')
	}

	return cp
}

// parseCreatePublication parses CREATE PUBLICATION name [FOR TABLE t1, t2 | FOR ALL TABLES] [WITH (options)].
func (p *Parser) parseCreatePublication() *CreatePublicationStmt {
	p.wantKeyword("publication")
	pos := p.pos

	cp := &CreatePublicationStmt{baseStmt: baseStmt{baseNode{pos}}}
	cp.Pubname = p.colId()

	if p.gotKeyword("for") {
		if p.gotKeyword("all") {
			p.wantKeyword("tables")
			cp.ForAllTables = true
		} else {
			p.gotKeyword("table") // optional TABLE keyword
			cp.Tables = append(cp.Tables, p.parseQualifiedName())
			for p.gotSelf(',') {
				cp.Tables = append(cp.Tables, p.parseQualifiedName())
			}
		}
	}

	if p.isKeyword("with") {
		p.next()
		cp.Options = p.parseParenDefElemList()
	}

	return cp
}

// parseCreateSubscription parses CREATE SUBSCRIPTION name CONNECTION 'conninfo' PUBLICATION pub1, pub2 [WITH (options)].
func (p *Parser) parseCreateSubscription() *CreateSubscriptionStmt {
	p.wantKeyword("subscription")
	pos := p.pos

	cs := &CreateSubscriptionStmt{baseStmt: baseStmt{baseNode{pos}}}
	cs.Subname = p.colId()

	p.wantKeyword("connection")
	if p.tok == SCONST {
		cs.Conninfo = p.lit
		p.next()
	}

	p.wantKeyword("publication")
	cs.Publication = append(cs.Publication, p.colId())
	for p.gotSelf(',') {
		cs.Publication = append(cs.Publication, p.colId())
	}

	if p.isKeyword("with") {
		p.next()
		cs.Options = p.parseParenDefElemList()
	}

	return cs
}

// parseParenDefElemList parses (name = value, ...) option lists.
func (p *Parser) parseParenDefElemList() []*DefElem {
	p.wantSelf('(')
	var opts []*DefElem
	for {
		name := p.colId()
		var arg Node
		if p.gotSelf('=') {
			// Value can be a string, number, boolean keyword, or identifier
			switch {
			case p.tok == SCONST:
				arg = &String{baseNode: baseNode{p.pos}, Str: p.lit}
				p.next()
			case p.tok == ICONST:
				arg = &A_Const{baseExpr: baseExpr{baseNode{p.pos}}, Val: Value{Type: ValInt, Ival: p.parseInt()}}
			case p.tok == FCONST:
				arg = &String{baseNode: baseNode{p.pos}, Str: p.lit}
				p.next()
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
	p.wantSelf(')')
	return opts
}

// parseCreateEventTrigger parses CREATE EVENT TRIGGER name ON event [WHEN filter] EXECUTE FUNCTION func().
func (p *Parser) parseCreateEventTrigger() *CreateEventTrigStmt {
	p.wantKeyword("trigger")
	pos := p.pos

	ct := &CreateEventTrigStmt{baseStmt: baseStmt{baseNode{pos}}}
	ct.Trigname = p.colId()
	p.wantKeyword("on")
	ct.Eventname = p.colId()

	// Optional WHEN tag IN (...)
	if p.isKeyword("when") {
		p.next()
		for {
			filterVar := p.colId()
			p.wantKeyword("in")
			p.wantSelf('(')
			var vals []string
			if p.tok == SCONST {
				vals = append(vals, p.lit)
				p.next()
				for p.gotSelf(',') {
					if p.tok == SCONST {
						vals = append(vals, p.lit)
						p.next()
					}
				}
			}
			p.wantSelf(')')
			for _, v := range vals {
				ct.WhenClause = append(ct.WhenClause, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  filterVar,
					Arg:      &String{baseNode: baseNode{p.pos}, Str: v},
				})
			}
			if !p.gotKeyword("and") {
				break
			}
		}
	}

	p.wantKeyword("execute")
	if !p.gotKeyword("function") {
		p.gotKeyword("procedure")
	}
	ct.Funcname = p.parseQualifiedName()
	p.wantSelf('(')
	p.wantSelf(')')

	return ct
}
