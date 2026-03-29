package parser

// ---------------------------------------------------------------------------
// JSON_SCALAR, JSON_SERIALIZE
// ---------------------------------------------------------------------------

// parseJsonReturning parses RETURNING type into a JsonOutput, if present.
func (p *Parser) parseJsonReturning() *JsonOutput {
	if !p.gotKeyword("returning") {
		return nil
	}
	return &JsonOutput{
		baseNode: baseNode{p.pos},
		TypeName: p.parseTypeName(),
	}
}

// parseJsonScalar parses JSON_SCALAR(expr).
// Called after JSON_SCALAR keyword has been consumed.
func (p *Parser) parseJsonScalar() Expr {
	pos := p.pos
	p.wantSelf('(')
	expr := p.parseExpr()
	js := &JsonScalarExpr{baseExpr: baseExpr{baseNode{pos}}, Expr: expr}
	js.Output = p.parseJsonReturning()
	p.wantSelf(')')
	return js
}

// parseJsonSerialize parses JSON_SERIALIZE(expr [FORMAT JSON [ENCODING UTF8]] [RETURNING type]).
// Called after JSON_SERIALIZE keyword has been consumed.
func (p *Parser) parseJsonSerialize() Expr {
	pos := p.pos
	p.wantSelf('(')
	expr := p.parseExpr()
	js := &JsonSerializeExpr{baseExpr: baseExpr{baseNode{pos}}, Expr: expr}
	// Optional FORMAT JSON
	if p.gotKeyword("format") {
		p.wantKeyword("json")
		// Optional ENCODING UTF8
		if p.isKeyword("encoding") {
			p.next()
			p.colId() // consume encoding name
		}
	}
	js.Output = p.parseJsonReturning()
	p.wantSelf(')')
	return js
}

// ---------------------------------------------------------------------------
// JSON_OBJECTAGG, JSON_ARRAYAGG
// ---------------------------------------------------------------------------

// parseJsonObjectAgg parses JSON_OBJECTAGG(key_expr : value_expr ... [RETURNING type]).
// Called after JSON_OBJECTAGG keyword has been consumed.
func (p *Parser) parseJsonObjectAgg() Expr {
	pos := p.pos
	p.wantSelf('(')

	agg := &JsonObjectAgg{baseExpr: baseExpr{baseNode{pos}}}

	// key : value  or  key VALUE value
	key := p.parseExpr()
	kv := &JsonKeyValue{baseNode: baseNode{pos}, Key: key}
	if p.gotSelf(':') {
		kv.Value = p.parseExpr()
	} else if p.gotKeyword("value") {
		kv.Value = p.parseExpr()
	}
	agg.Arg = kv

	// Optional NULL ON NULL / ABSENT ON NULL
	if p.isKeyword("null") {
		p.next()
		p.wantKeyword("on")
		p.wantKeyword("null")
		agg.AbsentOnNull = false
	} else if p.isKeyword("absent") {
		p.next()
		p.wantKeyword("on")
		p.wantKeyword("null")
		agg.AbsentOnNull = true
	}

	// Optional WITH UNIQUE KEYS / WITHOUT UNIQUE KEYS
	if p.isKeyword("with") {
		p.next()
		if p.gotKeyword("unique") {
			p.gotKeyword("keys")
			agg.UniqueKeys = true
		}
	} else if p.isKeyword("without") {
		p.next()
		p.gotKeyword("unique")
		p.gotKeyword("keys")
		agg.UniqueKeys = false
	}

	agg.Output = p.parseJsonReturning()
	p.wantSelf(')')

	return agg
}

// parseJsonArrayAgg parses JSON_ARRAYAGG(expr [ORDER BY ...] ... [RETURNING type]).
// Called after JSON_ARRAYAGG keyword has been consumed.
func (p *Parser) parseJsonArrayAgg() Expr {
	pos := p.pos
	p.wantSelf('(')

	agg := &JsonArrayAgg{baseExpr: baseExpr{baseNode{pos}}}
	agg.Arg = p.parseExpr()

	// Optional ORDER BY
	if p.isKeyword("order") {
		p.next()
		p.wantKeyword("by")
		agg.Order = p.parseSortClause()
	}

	// Optional NULL ON NULL / ABSENT ON NULL
	if p.isKeyword("null") {
		p.next()
		p.wantKeyword("on")
		p.wantKeyword("null")
		agg.AbsentOnNull = false
	} else if p.isKeyword("absent") {
		p.next()
		p.wantKeyword("on")
		p.wantKeyword("null")
		agg.AbsentOnNull = true
	}

	agg.Output = p.parseJsonReturning()
	p.wantSelf(')')

	return agg
}

// ---------------------------------------------------------------------------
// JSON_TABLE
// ---------------------------------------------------------------------------

// parseJsonTable parses JSON_TABLE(expr, path_spec COLUMNS (...)) as a FROM-clause source.
// Called after JSON_TABLE keyword has been consumed.
func (p *Parser) parseJsonTable() *JsonTable {
	pos := p.pos
	p.wantSelf('(')

	jt := &JsonTable{baseNode: baseNode{pos}}
	jt.Expr = p.parseExpr()
	p.wantSelf(',')

	// Path specification (string literal)
	jt.PathSpec = p.parseExpr()

	// Optional PASSING
	if p.gotKeyword("passing") {
		jt.Passing = p.parseJsonPassing()
	}

	// COLUMNS (...)
	p.wantKeyword("columns")
	p.wantSelf('(')
	jt.Columns = p.parseJsonTableColumns()
	p.wantSelf(')')

	// Optional ON EMPTY / ON ERROR at table level
	for p.isKeyword("error") || p.isKeyword("null") || p.isKeyword("empty") || p.isKeyword("default") {
		beh := p.parseJsonTableBehavior()
		if p.gotKeyword("on") {
			if p.gotKeyword("error") {
				jt.OnError = beh
			} else if p.gotKeyword("empty") {
				jt.OnEmpty = beh
			}
		}
	}

	p.wantSelf(')')

	return jt
}

func (p *Parser) parseJsonTableBehavior() *JsonBehavior {
	pos := p.pos
	switch {
	case p.gotKeyword("null"):
		return &JsonBehavior{baseNode: baseNode{pos}, BType: JSON_BEHAVIOR_NULL}
	case p.gotKeyword("error"):
		return &JsonBehavior{baseNode: baseNode{pos}, BType: JSON_BEHAVIOR_ERROR}
	case p.gotKeyword("empty"):
		return &JsonBehavior{baseNode: baseNode{pos}, BType: JSON_BEHAVIOR_EMPTY}
	case p.isKeyword("default"):
		p.next()
		expr := p.parseExpr()
		return &JsonBehavior{baseNode: baseNode{pos}, BType: JSON_BEHAVIOR_DEFAULT, Expr: expr}
	default:
		return &JsonBehavior{baseNode: baseNode{pos}, BType: JSON_BEHAVIOR_NULL}
	}
}

func (p *Parser) parseJsonTableColumns() []*JsonTableColumn {
	var cols []*JsonTableColumn
	cols = append(cols, p.parseJsonTableColumn())
	for p.gotSelf(',') {
		cols = append(cols, p.parseJsonTableColumn())
	}
	return cols
}

func (p *Parser) parseJsonTableColumn() *JsonTableColumn {
	pos := p.pos

	// NESTED [PATH] path_string COLUMNS (...)
	if p.gotKeyword("nested") {
		p.gotKeyword("path") // optional
		pathSpec := p.parseExpr()
		p.wantKeyword("columns")
		p.wantSelf('(')
		nested := p.parseJsonTableColumns()
		p.wantSelf(')')
		return &JsonTableColumn{
			baseNode: baseNode{pos},
			Coltype:  JTC_NESTED,
			PathSpec: pathSpec,
			Columns:  nested,
		}
	}

	// name FOR ORDINALITY
	// name type [PATH path] [ON EMPTY] [ON ERROR]
	// name type EXISTS [PATH path]
	name := p.colId()

	if p.gotKeyword("for") {
		p.wantKeyword("ordinality")
		return &JsonTableColumn{
			baseNode: baseNode{pos},
			Coltype:  JTC_FOR_ORDINALITY,
			Name:     name,
		}
	}

	col := &JsonTableColumn{baseNode: baseNode{pos}, Name: name}
	col.TypeName = p.parseTypeName()

	if p.gotKeyword("exists") {
		col.Coltype = JTC_EXISTS
	} else {
		col.Coltype = JTC_REGULAR
	}

	// Optional PATH path_string
	if p.gotKeyword("path") {
		col.PathSpec = p.parseExpr()
	}

	// Optional wrapper (WITH [CONDITIONAL|UNCONDITIONAL] WRAPPER)
	if p.isKeyword("with") {
		p.next()
		if p.gotKeyword("conditional") {
			p.gotKeyword("wrapper")
			col.Wrapper = JSW_CONDITIONAL
		} else if p.gotKeyword("unconditional") {
			p.gotKeyword("wrapper")
			col.Wrapper = JSW_UNCONDITIONAL
		} else {
			p.gotKeyword("wrapper")
			col.Wrapper = JSW_UNCONDITIONAL
		}
	} else if p.isKeyword("without") {
		p.next()
		p.gotKeyword("wrapper")
		col.Wrapper = JSW_NONE
	}

	// Optional ON EMPTY / ON ERROR behaviors
	for i := 0; i < 2; i++ {
		if p.isAnyKeyword("null", "error", "empty", "default") {
			beh := p.parseJsonTableBehavior()
			if p.gotKeyword("on") {
				if p.gotKeyword("error") {
					col.OnError = beh
				} else if p.gotKeyword("empty") {
					col.OnEmpty = beh
				}
			}
		}
	}

	return col
}

func (p *Parser) parseJsonPassing() []*JsonArgument {
	var args []*JsonArgument
	args = append(args, p.parseJsonPassingArg())
	for p.gotSelf(',') {
		args = append(args, p.parseJsonPassingArg())
	}
	return args
}

func (p *Parser) parseJsonPassingArg() *JsonArgument {
	pos := p.pos
	val := p.parseExpr()
	arg := &JsonArgument{baseNode: baseNode{pos}, Val: val}
	if p.gotKeyword("as") {
		arg.Name = p.colId()
	}
	return arg
}

// ---------------------------------------------------------------------------
// XMLTABLE
// ---------------------------------------------------------------------------

// parseXmlTable parses XMLTABLE([XMLNAMESPACES(...),] xpath PASSING xml COLUMNS (...)).
// Called after XMLTABLE keyword has been consumed.
func (p *Parser) parseXmlTable() *XmlTable {
	pos := p.pos
	p.wantSelf('(')

	xt := &XmlTable{baseNode: baseNode{pos}}

	// Optional XMLNAMESPACES(...)
	if p.isKeyword("xmlnamespaces") {
		p.next()
		p.wantSelf('(')
		// Parse namespace declarations — simplified: skip to closing paren
		for p.tok != Token(')') && p.tok != EOF {
			ns := p.parseExpr()
			xt.Namespaces = append(xt.Namespaces, ns)
			if !p.gotSelf(',') {
				break
			}
		}
		p.wantSelf(')')
		p.wantSelf(',')
	}

	// Row xpath expression
	xt.Xmlexpr = p.parseExpr()

	// PASSING xml_expr [BY REF|BY VALUE]
	p.wantKeyword("passing")
	if p.gotKeyword("by") {
		p.colId() // REF or VALUE
	}
	xt.Docexpr = p.parseExpr()
	if p.gotKeyword("by") {
		p.colId() // REF or VALUE
	}

	// COLUMNS col_def, ...  (no parens in XMLTABLE, columns end at closing paren)
	p.wantKeyword("columns")
	xt.Columns = p.parseXmlTableColumns()

	p.wantSelf(')')

	return xt
}

func (p *Parser) parseXmlTableColumns() []*XmlTableColumn {
	var cols []*XmlTableColumn
	cols = append(cols, p.parseXmlTableColumn())
	for p.gotSelf(',') {
		cols = append(cols, p.parseXmlTableColumn())
	}
	return cols
}

func (p *Parser) parseXmlTableColumn() *XmlTableColumn {
	pos := p.pos
	name := p.colId()

	// FOR ORDINALITY
	if p.gotKeyword("for") {
		p.wantKeyword("ordinality")
		return &XmlTableColumn{
			baseNode:      baseNode{pos},
			Name:          name,
			ForOrdinality: true,
		}
	}

	col := &XmlTableColumn{baseNode: baseNode{pos}, Name: name}
	col.TypeName = p.parseTypeName()

	// Optional PATH xpath
	if p.gotKeyword("path") {
		col.PathExpr = p.parseExpr()
	}

	// Optional DEFAULT expr (use primary expr to avoid consuming NOT NULL)
	if p.gotKeyword("default") {
		col.DefExpr = p.parsePrimaryExpr()
	}

	// Optional NOT NULL / NULL
	if p.isKeyword("not") {
		p.next()
		p.wantKeyword("null")
		col.IsNotNull = true
	} else {
		p.gotKeyword("null") // optional NULL (nullable)
	}

	return col
}
