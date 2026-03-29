package parser

// parseXmlConcat parses XMLCONCAT(expr, expr, ...).
func (p *Parser) parseXmlConcat() Expr {
	pos := p.pos
	p.wantSelf('(')
	var args []Expr
	args = append(args, p.parseExpr())
	for p.gotSelf(',') {
		args = append(args, p.parseExpr())
	}
	p.wantSelf(')')
	return &XmlExpr{baseExpr: baseExpr{baseNode{pos}}, Op: IS_XMLCONCAT, Args: args}
}

// parseXmlElement parses XMLELEMENT(NAME name [, XMLATTRIBUTES(...)] [, content...]).
func (p *Parser) parseXmlElement() Expr {
	pos := p.pos
	p.wantSelf('(')
	p.wantKeyword("name")
	name := p.colLabel()

	xe := &XmlExpr{baseExpr: baseExpr{baseNode{pos}}, Op: IS_XMLELEMENT, Name: name}

	if p.gotSelf(',') {
		// Check for XMLATTRIBUTES
		if p.isKeyword("xmlattributes") {
			p.next()
			p.wantSelf('(')
			xe.NamedArgs = p.parseXmlAttributeList()
			p.wantSelf(')')
			// More content args?
			for p.gotSelf(',') {
				xe.Args = append(xe.Args, p.parseExpr())
			}
		} else {
			// Content args
			xe.Args = append(xe.Args, p.parseExpr())
			for p.gotSelf(',') {
				xe.Args = append(xe.Args, p.parseExpr())
			}
		}
	}

	p.wantSelf(')')
	return xe
}

// parseXmlAttributeList parses expr AS name, ... inside XMLATTRIBUTES/XMLFOREST.
func (p *Parser) parseXmlAttributeList() []Node {
	var attrs []Node
	for {
		expr := p.parseExpr()
		var name string
		if p.gotKeyword("as") {
			name = p.colLabel()
		}
		attrs = append(attrs, &ResTarget{baseNode: baseNode{p.pos}, Name: name, Val: expr})
		if !p.gotSelf(',') {
			break
		}
	}
	return attrs
}

// parseXmlForest parses XMLFOREST(expr AS name, ...).
func (p *Parser) parseXmlForest() Expr {
	pos := p.pos
	p.wantSelf('(')
	namedArgs := p.parseXmlAttributeList()
	p.wantSelf(')')
	return &XmlExpr{baseExpr: baseExpr{baseNode{pos}}, Op: IS_XMLFOREST, NamedArgs: namedArgs}
}

// parseXmlParse parses XMLPARSE(DOCUMENT|CONTENT expr [STRIP WHITESPACE]).
func (p *Parser) parseXmlParse() Expr {
	pos := p.pos
	p.wantSelf('(')

	var xmlopt XmlOptionType
	if p.gotKeyword("document") {
		xmlopt = XMLOPTION_DOCUMENT
	} else {
		p.wantKeyword("content")
		xmlopt = XMLOPTION_CONTENT
	}

	expr := p.parseExpr()

	xe := &XmlExpr{
		baseExpr:  baseExpr{baseNode{pos}},
		Op:        IS_XMLPARSE,
		Xmloption: xmlopt,
		Args:      []Expr{expr},
	}

	// Optional STRIP WHITESPACE / PRESERVE WHITESPACE
	if p.gotKeyword("strip") {
		p.wantKeyword("whitespace")
	} else if p.gotKeyword("preserve") {
		p.wantKeyword("whitespace")
	}

	p.wantSelf(')')
	return xe
}

// parseXmlPi parses XMLPI(NAME name [, expr]).
func (p *Parser) parseXmlPi() Expr {
	pos := p.pos
	p.wantSelf('(')
	p.wantKeyword("name")
	name := p.colLabel()

	xe := &XmlExpr{baseExpr: baseExpr{baseNode{pos}}, Op: IS_XMLPI, Name: name}

	if p.gotSelf(',') {
		xe.Args = append(xe.Args, p.parseExpr())
	}

	p.wantSelf(')')
	return xe
}

// parseXmlRoot parses XMLROOT(xml, VERSION expr|NO VALUE [, STANDALONE YES|NO|NO VALUE]).
func (p *Parser) parseXmlRoot() Expr {
	pos := p.pos
	p.wantSelf('(')
	xmlExpr := p.parseExpr()
	p.wantSelf(',')

	xe := &XmlExpr{baseExpr: baseExpr{baseNode{pos}}, Op: IS_XMLROOT, Args: []Expr{xmlExpr}}

	p.wantKeyword("version")
	if p.isKeyword("no") {
		p.next()
		p.wantKeyword("value")
		// version = NO VALUE (nil)
	} else {
		xe.Args = append(xe.Args, p.parseExpr())
	}

	// Optional STANDALONE
	if p.gotSelf(',') {
		p.wantKeyword("standalone")
		switch {
		case p.gotKeyword("yes"):
			// standalone yes
		case p.gotKeyword("no"):
			if p.gotKeyword("value") {
				// standalone no value
			}
			// standalone no
		}
	}

	p.wantSelf(')')
	return xe
}

// parseXmlSerialize parses XMLSERIALIZE(DOCUMENT|CONTENT expr AS type [INDENT]).
func (p *Parser) parseXmlSerialize() Expr {
	pos := p.pos
	p.wantSelf('(')

	var xmlopt XmlOptionType
	if p.gotKeyword("document") {
		xmlopt = XMLOPTION_DOCUMENT
	} else {
		p.wantKeyword("content")
		xmlopt = XMLOPTION_CONTENT
	}

	expr := p.parseExpr()
	p.wantKeyword("as")
	typeName := p.parseTypeName()

	xe := &XmlExpr{
		baseExpr:  baseExpr{baseNode{pos}},
		Op:        IS_XMLSERIALIZE,
		Xmloption: xmlopt,
		Args:      []Expr{expr},
		TypeName:  typeName,
	}

	if p.tok == IDENT && p.lit == "indent" {
		p.next()
		xe.Indent = true
	}

	p.wantSelf(')')
	return xe
}

// parseXmlExists parses XMLEXISTS(expr PASSING [BY REF] expr [BY REF]).
func (p *Parser) parseXmlExists() Expr {
	pos := p.pos
	p.wantSelf('(')
	xpath := p.parseExpr()

	xe := &XmlExpr{baseExpr: baseExpr{baseNode{pos}}, Op: IS_XMLEXISTS, Args: []Expr{xpath}}

	if p.gotKeyword("passing") {
		p.gotKeyword("by")
		p.gotKeyword("ref")
		xe.Args = append(xe.Args, p.parseExpr())
		p.gotKeyword("by")
		p.gotKeyword("ref")
	}

	p.wantSelf(')')
	return xe
}

// parseJsonObject parses JSON_OBJECT(key: value, ... [RETURNING type]).
func (p *Parser) parseJsonObject() Expr {
	pos := p.pos
	p.wantSelf('(')

	jo := &JsonObjectConstructor{baseExpr: baseExpr{baseNode{pos}}}

	if p.tok != Token(')') {
		// Parse key-value pairs or key VALUE value pairs
		for {
			// Check for RETURNING before parsing as key
			if p.isKeyword("returning") {
				break
			}
			if p.isKeyword("null") || p.isKeyword("absent") {
				break
			}
			key := p.parseExpr()
			// Separator: colon or VALUE keyword
			if p.gotSelf(':') || p.gotKeyword("value") {
				val := p.parseExpr()
				jo.Exprs = append(jo.Exprs, &JsonKeyValue{
					baseNode: baseNode{p.pos},
					Key:      key,
					Value:    val,
				})
			}
			if !p.gotSelf(',') {
				break
			}
		}
	}

	// NULL ON NULL / ABSENT ON NULL
	if p.gotKeyword("null") {
		p.wantKeyword("on")
		p.wantKeyword("null")
	} else if p.gotKeyword("absent") {
		p.wantKeyword("on")
		p.wantKeyword("null")
		jo.AbsentOnNull = true
	}

	// WITH|WITHOUT UNIQUE KEYS
	if p.isKeyword("with") {
		p.next()
		p.wantKeyword("unique")
		p.wantKeyword("keys")
		jo.UniqueKeys = true
	} else if p.isKeyword("without") {
		p.next()
		p.wantKeyword("unique")
		p.wantKeyword("keys")
	}

	// RETURNING type
	if p.gotKeyword("returning") {
		jo.Output = &JsonOutput{
			baseNode: baseNode{p.pos},
			TypeName: p.parseTypeName(),
		}
	}

	p.wantSelf(')')
	return jo
}

// parseJsonArray parses JSON_ARRAY(expr, ... [RETURNING type]).
func (p *Parser) parseJsonArray() Expr {
	pos := p.pos
	p.wantSelf('(')

	ja := &JsonArrayConstructor{baseExpr: baseExpr{baseNode{pos}}}

	if p.tok != Token(')') {
		for {
			if p.isKeyword("returning") || p.isKeyword("null") || p.isKeyword("absent") {
				break
			}
			ja.Exprs = append(ja.Exprs, p.parseExpr())
			if !p.gotSelf(',') {
				break
			}
		}
	}

	// NULL ON NULL / ABSENT ON NULL
	if p.gotKeyword("null") {
		p.wantKeyword("on")
		p.wantKeyword("null")
	} else if p.gotKeyword("absent") {
		p.wantKeyword("on")
		p.wantKeyword("null")
		ja.AbsentOnNull = true
	}

	// RETURNING type
	if p.gotKeyword("returning") {
		ja.Output = &JsonOutput{
			baseNode: baseNode{p.pos},
			TypeName: p.parseTypeName(),
		}
	}

	p.wantSelf(')')
	return ja
}

// parseJsonQuery parses JSON_QUERY(expr, path [PASSING ...] [RETURNING type] [wrapper] [behavior]).
func (p *Parser) parseJsonQuery() Expr {
	pos := p.pos
	p.wantSelf('(')

	jf := &JsonFuncExpr{baseExpr: baseExpr{baseNode{pos}}, Op: JSON_QUERY_OP}
	jf.Expr = p.parseExpr()
	p.wantSelf(',')
	jf.PathSpec = p.parseExpr()

	// Optional PASSING
	if p.gotKeyword("passing") {
		jf.Passing = p.parseJsonPassingList()
	}

	// Optional RETURNING
	if p.gotKeyword("returning") {
		jf.Output = &JsonOutput{
			baseNode: baseNode{p.pos},
			TypeName: p.parseTypeName(),
		}
	}

	// Optional wrapper: WITH [CONDITIONAL|UNCONDITIONAL] WRAPPER / WITHOUT WRAPPER / OMIT QUOTES
	if p.isKeyword("with") {
		p.next()
		if p.tok == IDENT && p.lit == "conditional" {
			p.next()
			jf.Wrapper = JSW_CONDITIONAL
		} else if p.tok == IDENT && p.lit == "unconditional" {
			p.next()
			jf.Wrapper = JSW_UNCONDITIONAL
		} else {
			jf.Wrapper = JSW_UNCONDITIONAL
		}
		p.wantKeyword("wrapper")
	} else if p.isKeyword("without") {
		p.next()
		p.wantKeyword("wrapper")
		jf.Wrapper = JSW_NONE
	}

	// Optional ON EMPTY / ON ERROR
	p.parseJsonBehaviors(jf)

	p.wantSelf(')')
	return jf
}

// parseJsonValue parses JSON_VALUE(expr, path [PASSING ...] [RETURNING type] [behavior]).
func (p *Parser) parseJsonValue() Expr {
	pos := p.pos
	p.wantSelf('(')

	jf := &JsonFuncExpr{baseExpr: baseExpr{baseNode{pos}}, Op: JSON_VALUE_OP}
	jf.Expr = p.parseExpr()
	p.wantSelf(',')
	jf.PathSpec = p.parseExpr()

	if p.gotKeyword("passing") {
		jf.Passing = p.parseJsonPassingList()
	}

	if p.gotKeyword("returning") {
		jf.Output = &JsonOutput{
			baseNode: baseNode{p.pos},
			TypeName: p.parseTypeName(),
		}
	}

	p.parseJsonBehaviors(jf)

	p.wantSelf(')')
	return jf
}

// parseJsonExists parses JSON_EXISTS(expr, path [PASSING ...] [behavior]).
func (p *Parser) parseJsonExistsFunc() Expr {
	pos := p.pos
	p.wantSelf('(')

	jf := &JsonFuncExpr{baseExpr: baseExpr{baseNode{pos}}, Op: JSON_EXISTS_OP}
	jf.Expr = p.parseExpr()
	p.wantSelf(',')
	jf.PathSpec = p.parseExpr()

	if p.gotKeyword("passing") {
		jf.Passing = p.parseJsonPassingList()
	}

	p.parseJsonBehaviors(jf)

	p.wantSelf(')')
	return jf
}

// parseJsonPassingList parses PASSING expr AS name, ...
func (p *Parser) parseJsonPassingList() []*JsonArgument {
	var args []*JsonArgument
	for {
		val := p.parseExpr()
		var name string
		if p.gotKeyword("as") {
			name = p.colId()
		}
		args = append(args, &JsonArgument{baseNode: baseNode{p.pos}, Val: val, Name: name})
		if !p.gotSelf(',') {
			break
		}
	}
	return args
}

// parseJsonBehaviors parses ON EMPTY and ON ERROR clauses.
func (p *Parser) parseJsonBehaviors(jf *JsonFuncExpr) {
	for i := 0; i < 2; i++ {
		beh := p.tryParseJsonBehavior()
		if beh == nil {
			break
		}
		if p.gotKeyword("on") {
			if p.gotKeyword("empty") {
				jf.OnEmpty = beh
			} else if p.gotKeyword("error") {
				jf.OnError = beh
			}
		}
	}
}

// tryParseJsonBehavior tries to parse a JSON behavior (NULL, ERROR, EMPTY, DEFAULT expr, etc.).
func (p *Parser) tryParseJsonBehavior() *JsonBehavior {
	switch {
	case p.isKeyword("null"):
		p.next()
		return &JsonBehavior{baseNode: baseNode{p.pos}, BType: JSON_BEHAVIOR_NULL}
	case p.isKeyword("error"):
		p.next()
		return &JsonBehavior{baseNode: baseNode{p.pos}, BType: JSON_BEHAVIOR_ERROR}
	case p.isKeyword("default"):
		p.next()
		expr := p.parseExpr()
		return &JsonBehavior{baseNode: baseNode{p.pos}, BType: JSON_BEHAVIOR_DEFAULT, Expr: expr}
	case p.isKeyword("true"):
		p.next()
		return &JsonBehavior{baseNode: baseNode{p.pos}, BType: JSON_BEHAVIOR_TRUE}
	case p.isKeyword("false"):
		p.next()
		return &JsonBehavior{baseNode: baseNode{p.pos}, BType: JSON_BEHAVIOR_FALSE}
	}
	return nil
}

// parseIsJson parses IS [NOT] JSON [VALUE|ARRAY|OBJECT|SCALAR] [WITH|WITHOUT UNIQUE KEYS].
// Called after IS [NOT] has been consumed and "json" keyword detected.
func (p *Parser) parseIsJson(expr Expr, not bool) Expr {
	pos := p.pos
	p.next() // consume JSON keyword

	jp := &JsonIsPredicate{baseExpr: baseExpr{baseNode{pos}}, Expr: expr}

	// Optional type: VALUE, OBJECT, ARRAY, SCALAR
	// ARRAY is a reserved keyword; OBJECT and SCALAR are identifiers
	switch {
	case p.gotKeyword("value"):
		jp.ItemType = JS_TYPE_ANY
	case p.tok == IDENT && p.lit == "object":
		p.next()
		jp.ItemType = JS_TYPE_OBJECT
	case p.isKeyword("array"):
		p.next()
		jp.ItemType = JS_TYPE_ARRAY
	case p.tok == IDENT && p.lit == "scalar":
		p.next()
		jp.ItemType = JS_TYPE_SCALAR
	default:
		jp.ItemType = JS_TYPE_ANY
	}

	// WITH|WITHOUT UNIQUE KEYS
	if p.isKeyword("with") {
		p.next()
		p.wantKeyword("unique")
		p.wantKeyword("keys")
		jp.UniqueKeys = true
	} else if p.isKeyword("without") {
		p.next()
		p.wantKeyword("unique")
		p.wantKeyword("keys")
	}

	if not {
		return &BoolExpr{baseExpr: baseExpr{baseNode{pos}}, Op: NOT_EXPR, Args: []Expr{jp}}
	}
	return jp
}
