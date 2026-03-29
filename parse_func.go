package pgscan

// parseCreateFunction parses CREATE [OR REPLACE] FUNCTION/PROCEDURE name(params) RETURNS type options...
func (p *Parser) parseCreateFunction(replace bool) *CreateFunctionStmt {
	pos := p.pos
	cs := &CreateFunctionStmt{
		baseStmt: baseStmt{baseNode{pos}},
		Replace:  replace,
	}

	if p.gotKeyword("procedure") {
		cs.IsProcedure = true
	} else {
		p.wantKeyword("function")
	}

	cs.Funcname = p.parseQualifiedName()

	// Parameter list
	p.wantSelf('(')
	if p.tok != Token(')') {
		cs.Parameters = p.parseFuncParams()
	}
	p.wantSelf(')')

	// RETURNS type (functions only)
	if !cs.IsProcedure && p.gotKeyword("returns") {
		if p.gotKeyword("table") {
			// RETURNS TABLE (col type, ...)
			p.wantSelf('(')
			for {
				param := &FunctionParameter{baseNode: baseNode{p.pos}, Mode: FUNC_PARAM_OUT}
				param.Name = p.colId()
				param.ArgType = p.parseTypeName()
				cs.Parameters = append(cs.Parameters, param)
				if !p.gotSelf(',') {
					break
				}
			}
			p.wantSelf(')')
		} else if p.gotKeyword("setof") {
			cs.ReturnType = p.parseTypeName()
			cs.ReturnType.Setof = true
		} else if p.gotKeyword("void") {
			// RETURNS VOID — no return type needed
		} else {
			cs.ReturnType = p.parseTypeName()
		}
	}

	// Function options: LANGUAGE, AS, IMMUTABLE, STABLE, VOLATILE, etc.
	cs.Options = p.parseFuncOptions()

	return cs
}

// parseFuncParams parses a comma-separated list of function parameters.
func (p *Parser) parseFuncParams() []*FunctionParameter {
	var params []*FunctionParameter
	params = append(params, p.parseFuncParam())
	for p.gotSelf(',') {
		params = append(params, p.parseFuncParam())
	}
	return params
}

// parseFuncParam parses a single function parameter: [mode] [name] type [DEFAULT expr]
func (p *Parser) parseFuncParam() *FunctionParameter {
	pos := p.pos
	fp := &FunctionParameter{baseNode: baseNode{pos}, Mode: FUNC_PARAM_DEFAULT}

	// Optional mode
	switch {
	case p.gotKeyword("in"):
		if p.gotKeyword("out") {
			fp.Mode = FUNC_PARAM_INOUT
		} else {
			fp.Mode = FUNC_PARAM_IN
		}
	case p.gotKeyword("out"):
		fp.Mode = FUNC_PARAM_OUT
	case p.gotKeyword("inout"):
		fp.Mode = FUNC_PARAM_INOUT
	case p.gotKeyword("variadic"):
		fp.Mode = FUNC_PARAM_VARIADIC
	}

	// Name and type — tricky because name is optional.
	// If we see two identifiers in a row, first is name, second starts type.
	// If we see one identifier followed by comma/close-paren/default, it's a type.
	// Heuristic: try to parse as type. If the next token after the first identifier
	// is another identifier or a type keyword, the first was a name.
	// Simplified: always try name + type, fall back to just type.
	if p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
		// Save the identifier
		savedName := p.lit
		savedPos := p.pos

		// Check if this looks like a type keyword
		if p.isAnyKeyword("int", "integer", "smallint", "bigint", "real", "float",
			"double", "decimal", "numeric", "boolean", "bool", "text", "varchar",
			"character", "char", "timestamp", "time", "interval", "json", "jsonb",
			"uuid", "xml", "bytea", "bit") {
			// It's a type — no name
			fp.ArgType = p.parseTypeName()
		} else {
			// Could be name or type. Consume it and check what follows.
			p.next()
			if p.tok == IDENT || p.tok == KEYWORD {
				// Two identifiers — first was name
				fp.Name = savedName
				fp.ArgType = p.parseTypeName()
			} else if p.tok == Token(',') || p.tok == Token(')') || p.isKeyword("default") {
				// Single identifier followed by delimiter — it was a type name
				fp.ArgType = &TypeName{baseNode: baseNode{savedPos}, Names: []string{savedName}}
			} else {
				// Assume it was a name and what follows is a type
				fp.Name = savedName
				fp.ArgType = p.parseTypeName()
			}
		}
	} else {
		fp.ArgType = p.parseTypeName()
	}

	// DEFAULT expr
	if p.gotKeyword("default") || p.gotSelf('=') {
		fp.DefExpr = p.parseExpr()
	}

	return fp
}

// parseFuncOptions parses function body and attribute options.
func (p *Parser) parseFuncOptions() []*DefElem {
	var opts []*DefElem
	for {
		switch {
		case p.isKeyword("language"):
			p.next()
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "language",
				Arg:      &String{baseNode: baseNode{p.pos}, Str: p.colLabel()},
			})
		case p.isKeyword("as"):
			p.next()
			if p.tok == SCONST {
				body := p.lit
				p.next()
				opts = append(opts, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "as",
					Arg:      &String{baseNode: baseNode{p.pos}, Str: body},
				})
				// Optional second string (for C functions: library, symbol)
				if p.gotSelf(',') && p.tok == SCONST {
					opts = append(opts, &DefElem{
						baseNode: baseNode{p.pos},
						Defname:  "as_symbol",
						Arg:      &String{baseNode: baseNode{p.pos}, Str: p.lit},
					})
					p.next()
				}
			}
		case p.isAnyKeyword("immutable", "stable", "volatile"):
			opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: p.lit})
			p.next()
		case p.isAnyKeyword("strict", "called"):
			// STRICT or CALLED ON NULL INPUT
			name := p.lit
			p.next()
			if name == "called" {
				p.wantKeyword("on")
				p.wantKeyword("null")
				p.wantKeyword("input")
			}
			opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: name})
		case p.isKeyword("returns"):
			// RETURNS NULL ON NULL INPUT
			p.next()
			p.wantKeyword("null")
			p.wantKeyword("on")
			p.wantKeyword("null")
			p.wantKeyword("input")
			opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: "strict"})
		case p.isKeyword("security"):
			p.next()
			if p.gotKeyword("definer") {
				opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: "security_definer"})
			} else {
				p.wantKeyword("invoker")
				opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: "security_invoker"})
			}
		case p.isKeyword("parallel"):
			p.next()
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "parallel",
				Arg:      &String{baseNode: baseNode{p.pos}, Str: p.colLabel()},
			})
		case p.isKeyword("cost"):
			p.next()
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "cost",
				Arg:      &A_Const{baseExpr: baseExpr{baseNode{p.pos}}, Val: Value{Type: ValInt, Ival: p.parseInt()}},
			})
		case p.isKeyword("rows"):
			p.next()
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "rows",
				Arg:      &A_Const{baseExpr: baseExpr{baseNode{p.pos}}, Val: Value{Type: ValInt, Ival: p.parseInt()}},
			})
		case p.isKeyword("set"):
			p.next()
			name := p.colId()
			p.gotSelf('=')
			p.gotKeyword("to")
			val := p.colLabel()
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "set_" + name,
				Arg:      &String{baseNode: baseNode{p.pos}, Str: val},
			})
		default:
			return opts
		}
	}
}

// parseDoStmt parses DO $$ body $$ [LANGUAGE lang].
func (p *Parser) parseDoStmt() *DoStmt {
	p.wantKeyword("do")
	pos := p.pos

	ds := &DoStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.tok == SCONST {
		ds.Args = append(ds.Args, &DefElem{
			baseNode: baseNode{p.pos},
			Defname:  "as",
			Arg:      &String{baseNode: baseNode{p.pos}, Str: p.lit},
		})
		p.next()
	}

	if p.gotKeyword("language") {
		ds.Args = append(ds.Args, &DefElem{
			baseNode: baseNode{p.pos},
			Defname:  "language",
			Arg:      &String{baseNode: baseNode{p.pos}, Str: p.colLabel()},
		})
	}

	return ds
}

// parseCallStmt parses CALL procedure_name(args).
func (p *Parser) parseCallStmt() *CallStmt {
	p.wantKeyword("call")
	pos := p.pos

	// Parse as a function call expression
	parts := p.parseQualifiedName()
	p.wantSelf('(')
	var args []Expr
	if p.tok != Token(')') {
		args = p.parseExprList()
	}
	p.wantSelf(')')

	return &CallStmt{
		baseStmt: baseStmt{baseNode{pos}},
		FuncCall: &FuncCall{
			baseExpr: baseExpr{baseNode{pos}},
			Funcname: parts,
			Args:     args,
		},
	}
}

// parseCreateTrigger parses CREATE [OR REPLACE] TRIGGER name ...
func (p *Parser) parseCreateTrigger(replace bool) *CreateTrigStmt {
	p.wantKeyword("trigger")
	pos := p.pos

	ct := &CreateTrigStmt{
		baseStmt: baseStmt{baseNode{pos}},
		Replace:  replace,
	}
	ct.Trigname = p.colId()

	// BEFORE | AFTER | INSTEAD OF
	switch {
	case p.gotKeyword("before"):
		ct.Timing = TRIGGER_TYPE_BEFORE
	case p.gotKeyword("after"):
		ct.Timing = TRIGGER_TYPE_AFTER
	case p.isKeyword("instead"):
		p.next()
		p.wantKeyword("of")
		ct.Timing = TRIGGER_TYPE_INSTEAD
	}

	// Events: INSERT | UPDATE [OF col, ...] | DELETE | TRUNCATE (OR-separated)
	ct.Events = p.parseTriggerEvents(ct)

	p.wantKeyword("on")
	ct.Relation = p.parseRangeVar()

	// FOR [EACH] ROW | STATEMENT
	if p.gotKeyword("for") {
		p.gotKeyword("each") // optional
		if p.gotKeyword("row") {
			ct.Row = true
		} else {
			p.wantKeyword("statement")
			ct.Row = false
		}
	}

	// WHEN (condition)
	if p.gotKeyword("when") {
		p.wantSelf('(')
		ct.WhenClause = p.parseExpr()
		p.wantSelf(')')
	}

	// EXECUTE FUNCTION|PROCEDURE func_name(args)
	p.wantKeyword("execute")
	p.gotKeyword("function")  // or
	p.gotKeyword("procedure") // either is accepted
	ct.Funcname = p.parseQualifiedName()
	p.wantSelf('(')
	if p.tok != Token(')') {
		ct.Args = p.parseExprList()
	}
	p.wantSelf(')')

	return ct
}

// parseTriggerEvents parses INSERT | UPDATE [OF cols] | DELETE | TRUNCATE separated by OR.
func (p *Parser) parseTriggerEvents(ct *CreateTrigStmt) int {
	events := p.parseSingleTriggerEvent(ct)
	for p.gotKeyword("or") {
		events |= p.parseSingleTriggerEvent(ct)
	}
	return events
}

func (p *Parser) parseSingleTriggerEvent(ct *CreateTrigStmt) int {
	switch {
	case p.gotKeyword("insert"):
		return TRIGGER_TYPE_INSERT
	case p.gotKeyword("update"):
		if p.gotKeyword("of") {
			ct.Columns = p.parseNameList()
		}
		return TRIGGER_TYPE_UPDATE
	case p.gotKeyword("delete"):
		return TRIGGER_TYPE_DELETE
	case p.gotKeyword("truncate"):
		return TRIGGER_TYPE_TRUNCATE
	}
	p.syntaxError("expected INSERT, UPDATE, DELETE, or TRUNCATE")
	return 0
}

// parseCreateRule parses CREATE [OR REPLACE] RULE name AS ON event TO table [WHERE cond] DO [INSTEAD|ALSO] action.
func (p *Parser) parseCreateRule(replace bool) *RuleStmt {
	p.wantKeyword("rule")
	pos := p.pos

	rs := &RuleStmt{
		baseStmt: baseStmt{baseNode{pos}},
		Replace:  replace,
	}
	rs.Rulename = p.colId()

	p.wantKeyword("as")
	p.wantKeyword("on")

	switch {
	case p.gotKeyword("select"):
		rs.Event = CMD_SELECT
	case p.gotKeyword("insert"):
		rs.Event = CMD_INSERT
	case p.gotKeyword("update"):
		rs.Event = CMD_UPDATE
	case p.gotKeyword("delete"):
		rs.Event = CMD_DELETE
	}

	p.wantKeyword("to")
	rs.Relation = p.parseRangeVar()

	if p.gotKeyword("where") {
		rs.WhereClause = p.parseExpr()
	}

	p.wantKeyword("do")

	if p.gotKeyword("instead") {
		rs.Instead = true
	} else {
		p.gotKeyword("also")
	}

	// Action: NOTHING | stmt | (stmts)
	if p.gotKeyword("nothing") {
		// No actions
	} else if p.tok == Token('(') {
		p.next()
		for p.tok != Token(')') && p.tok != EOF {
			rs.Actions = append(rs.Actions, p.parseSimpleStmt())
			p.gotSelf(';')
		}
		p.wantSelf(')')
	} else {
		rs.Actions = append(rs.Actions, p.parseSimpleStmt())
	}

	return rs
}
