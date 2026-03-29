package parser

// ---------------------------------------------------------------------------
// DefineStmt: CREATE AGGREGATE, CREATE OPERATOR, CREATE TYPE (range/shell),
// CREATE TEXT SEARCH ..., CREATE COLLATION
// ---------------------------------------------------------------------------

// parseCreateAggregate parses CREATE [OR REPLACE] AGGREGATE name (args) (definition).
func (p *Parser) parseCreateAggregate(replace bool) *DefineStmt {
	p.wantKeyword("aggregate")
	pos := p.pos

	ds := &DefineStmt{
		baseStmt: baseStmt{baseNode{pos}},
		Kind:     OBJECT_AGGREGATE,
		Replace:  replace,
	}
	ds.Defnames = p.parseQualifiedName()

	// Arguments
	p.wantSelf('(')
	if p.tok != Token(')') {
		// Old-style: CREATE AGGREGATE name (sfunc = ..., stype = ..., ...)
		// New-style: CREATE AGGREGATE name (basetype) (sfunc = ..., ...)
		// Detect: if first token after ( looks like name = value, it's old-style definition
		if p.isDefElemStart() {
			ds.OldStyle = true
			ds.Definition = p.parseDefElemList()
			p.wantSelf(')')
			return ds
		}
		// New-style: parse argument types
		for p.tok != Token(')') && p.tok != EOF {
			ds.Args = append(ds.Args, p.parseTypeName())
			if !p.gotSelf(',') {
				break
			}
		}
	}
	p.wantSelf(')')

	// Definition list
	p.wantSelf('(')
	ds.Definition = p.parseDefElemList()
	p.wantSelf(')')

	return ds
}

// isDefElemStart checks if the current position looks like name = value (a DefElem).
func (p *Parser) isDefElemStart() bool {
	// A DefElem starts with an identifier/keyword followed by '='
	if p.tok != IDENT && p.tok != KEYWORD {
		return false
	}
	// We can't peek ahead easily, so use a heuristic:
	// Common DefElem names for aggregates
	switch p.lit {
	case "sfunc", "stype", "initcond", "finalfunc", "combinefunc",
		"serialfunc", "deserialfunc", "msfunc", "mstype", "minitcond",
		"mfinalfunc", "sortop", "parallel", "hypothetical":
		return true
	}
	return false
}

// parseDefElemList parses comma-separated name = value pairs.
func (p *Parser) parseDefElemList() []*DefElem {
	var defs []*DefElem
	defs = append(defs, p.parseDefElem())
	for p.gotSelf(',') {
		defs = append(defs, p.parseDefElem())
	}
	return defs
}

// parseDefElem parses name = value.
func (p *Parser) parseDefElem() *DefElem {
	pos := p.pos
	name := p.colLabel()
	de := &DefElem{baseNode: baseNode{pos}, Defname: name}

	if p.gotSelf('=') {
		de.Arg = p.parseDefArg()
	}

	return de
}

// parseDefArg parses the value part of a DefElem.
func (p *Parser) parseDefArg() Node {
	pos := p.pos
	switch {
	case p.tok == SCONST:
		s := p.lit
		p.next()
		return &String{baseNode: baseNode{pos}, Str: s}
	case p.tok == ICONST:
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: p.parseInt()}}
	case p.tok == FCONST:
		s := p.lit
		p.next()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValFloat, Str: s}}
	case p.tok == Token('-'):
		p.next()
		if p.tok == ICONST {
			return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: -p.parseInt()}}
		}
		s := p.lit
		p.next()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValFloat, Str: "-" + s}}
	default:
		// Use colLabel to accept reserved keywords (like "default") as values
		name := p.colLabel()
		if p.gotSelf('.') {
			schema := name
			name2 := p.colLabel()
			return &TypeName{baseNode: baseNode{pos}, Names: []string{schema, name2}}
		}
		return &String{baseNode: baseNode{pos}, Str: name}
	}
}

// parseCreateOperator parses CREATE OPERATOR name (definition).
// Called after OPERATOR keyword has been consumed.
func (p *Parser) parseCreateOperator() *DefineStmt {
	pos := p.pos

	ds := &DefineStmt{
		baseStmt: baseStmt{baseNode{pos}},
		Kind:     OBJECT_OPERATOR,
	}

	// Operator name can be schema-qualified: schema.operator_symbol
	ds.Defnames = p.parseOperatorName()

	p.wantSelf('(')
	ds.Definition = p.parseDefElemList()
	p.wantSelf(')')

	return ds
}

// parseOperatorName parses an operator name which may be schema-qualified.
func (p *Parser) parseOperatorName() []string {
	var names []string
	// Could be: schema . operator_symbol  or just operator_symbol
	if (p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword)) {
		name := p.lit
		p.next()
		if p.gotSelf('.') {
			// schema.operator
			names = append(names, name)
			names = append(names, p.parseOpSymbol())
		} else {
			names = append(names, name)
		}
		return names
	}
	// Bare operator symbol
	names = append(names, p.parseOpSymbol())
	return names
}

// parseCreateTypeShellOrRange handles CREATE TYPE name (shell) or CREATE TYPE name AS RANGE (...).
// Called when the standard parseCreateType doesn't match (no AS ENUM or AS (...)).
func (p *Parser) parseCreateTypeRange(typeName []string) *DefineStmt {
	// AS RANGE already consumed
	pos := p.pos
	ds := &DefineStmt{
		baseStmt: baseStmt{baseNode{pos}},
		Kind:     OBJECT_TYPE,
		Defnames: typeName,
	}
	p.wantSelf('(')
	ds.Definition = p.parseDefElemList()
	p.wantSelf(')')
	return ds
}

// parseCreateTypeShell handles CREATE TYPE name (shell type, no body).
func (p *Parser) parseCreateTypeShell(typeName []string) *DefineStmt {
	return &DefineStmt{
		baseStmt: baseStmt{baseNode{0}},
		Kind:     OBJECT_TYPE,
		Defnames: typeName,
	}
}

// ---------------------------------------------------------------------------
// CREATE TEXT SEARCH {PARSER|DICTIONARY|TEMPLATE|CONFIGURATION}
// ---------------------------------------------------------------------------

func (p *Parser) parseCreateTextSearch() *DefineStmt {
	p.wantKeyword("search")
	pos := p.pos

	ds := &DefineStmt{baseStmt: baseStmt{baseNode{pos}}}

	switch {
	case p.gotKeyword("parser"):
		ds.Kind = OBJECT_TSPARSER
	case p.gotKeyword("dictionary"):
		ds.Kind = OBJECT_TSDICTIONARY
	case p.gotKeyword("template"):
		ds.Kind = OBJECT_TSTEMPLATE
	case p.gotKeyword("configuration"):
		ds.Kind = OBJECT_TSCONFIGURATION
	default:
		p.syntaxError("expected PARSER, DICTIONARY, TEMPLATE, or CONFIGURATION")
	}

	ds.Defnames = p.parseQualifiedName()

	p.wantSelf('(')
	ds.Definition = p.parseDefElemList()
	p.wantSelf(')')

	return ds
}

// ---------------------------------------------------------------------------
// CREATE COLLATION
// ---------------------------------------------------------------------------

func (p *Parser) parseCreateCollation() *DefineStmt {
	p.wantKeyword("collation")
	pos := p.pos

	ds := &DefineStmt{
		baseStmt: baseStmt{baseNode{pos}},
		Kind:     OBJECT_COLLATION,
	}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("not")
		p.wantKeyword("exists")
		ds.IfNotExists = true
	}

	ds.Defnames = p.parseQualifiedName()

	// FROM existing_collation  or  (LOCALE = ..., ...)
	if p.gotKeyword("from") {
		fromName := p.parseQualifiedName()
		ds.Definition = []*DefElem{{
			baseNode: baseNode{p.pos},
			Defname:  "from",
			Arg:      &String{baseNode: baseNode{p.pos}, Str: joinName(fromName)},
		}}
	} else {
		p.wantSelf('(')
		ds.Definition = p.parseDefElemList()
		p.wantSelf(')')
	}

	return ds
}

// ---------------------------------------------------------------------------
// CREATE CAST
// ---------------------------------------------------------------------------

func (p *Parser) parseCreateCast() *CreateCastStmt {
	p.wantKeyword("cast")
	pos := p.pos

	cs := &CreateCastStmt{baseStmt: baseStmt{baseNode{pos}}}
	cs.Context = COERCION_EXPLICIT // default

	p.wantSelf('(')
	cs.SourceType = p.parseTypeName()
	p.wantKeyword("as")
	cs.TargetType = p.parseTypeName()
	p.wantSelf(')')

	if p.gotKeyword("without") {
		// WITHOUT FUNCTION
		p.wantKeyword("function")
	} else {
		p.wantKeyword("with")
		if p.gotKeyword("function") {
			cs.Func = p.parseFuncWithArgsSimple()
		} else if p.gotKeyword("inout") {
			cs.Inout = true
		}
	}

	// Optional AS IMPLICIT | AS ASSIGNMENT
	if p.gotKeyword("as") {
		if p.gotKeyword("implicit") {
			cs.Context = COERCION_IMPLICIT
		} else if p.gotKeyword("assignment") {
			cs.Context = COERCION_ASSIGNMENT
		}
	}

	return cs
}

// parseFuncWithArgsSimple parses func_name(arg_types).
func (p *Parser) parseFuncWithArgsSimple() *FuncWithArgs {
	pos := p.pos
	fa := &FuncWithArgs{baseNode: baseNode{pos}}
	fa.Funcname = p.parseQualifiedName()
	if p.tok == Token('(') {
		p.next()
		for p.tok != Token(')') && p.tok != EOF {
			fa.Funcargs = append(fa.Funcargs, p.parseTypeName())
			if !p.gotSelf(',') {
				break
			}
		}
		p.wantSelf(')')
	}
	return fa
}

// ---------------------------------------------------------------------------
// CREATE TRANSFORM
// ---------------------------------------------------------------------------

func (p *Parser) parseCreateTransform(replace bool) *CreateTransformStmt {
	p.wantKeyword("transform")
	pos := p.pos

	ct := &CreateTransformStmt{
		baseStmt: baseStmt{baseNode{pos}},
		Replace:  replace,
	}

	p.wantKeyword("for")
	ct.TypeName = p.parseTypeName()
	p.wantKeyword("language")
	ct.Lang = p.colId()

	p.wantSelf('(')
	// FROM SQL WITH FUNCTION func, TO SQL WITH FUNCTION func
	for p.tok != Token(')') && p.tok != EOF {
		if p.gotKeyword("from") {
			p.wantKeyword("sql")
			p.wantKeyword("with")
			p.wantKeyword("function")
			ct.FromSQL = p.parseFuncWithArgsSimple()
		} else if p.gotKeyword("to") {
			p.wantKeyword("sql")
			p.wantKeyword("with")
			p.wantKeyword("function")
			ct.ToSQL = p.parseFuncWithArgsSimple()
		}
		p.gotSelf(',')
	}
	p.wantSelf(')')

	return ct
}

// ---------------------------------------------------------------------------
// CREATE ACCESS METHOD
// ---------------------------------------------------------------------------

func (p *Parser) parseCreateAccessMethod() *CreateAmStmt {
	p.wantKeyword("method")
	pos := p.pos

	am := &CreateAmStmt{baseStmt: baseStmt{baseNode{pos}}}
	am.AmName = p.colId()

	p.wantKeyword("type")
	if p.gotKeyword("index") {
		am.AmType = "INDEX"
	} else {
		p.wantKeyword("table")
		am.AmType = "TABLE"
	}

	p.wantKeyword("handler")
	am.HandlerName = p.parseQualifiedName()

	return am
}

// ---------------------------------------------------------------------------
// CREATE OPERATOR CLASS
// ---------------------------------------------------------------------------

func (p *Parser) parseCreateOpClass() *CreateOpClassStmt {
	p.wantKeyword("class")
	pos := p.pos

	oc := &CreateOpClassStmt{baseStmt: baseStmt{baseNode{pos}}}

	oc.OpClassName = p.parseQualifiedName()

	if p.gotKeyword("default") {
		oc.IsDefault = true
	}

	p.wantKeyword("for")
	p.wantKeyword("type")
	oc.DataType = p.parseTypeName()

	p.wantKeyword("using")
	oc.AmName = p.colId()

	// Optional FAMILY
	if p.gotKeyword("family") {
		oc.OpFamily = p.parseQualifiedName()
	}

	p.wantKeyword("as")

	// Parse items: OPERATOR n name, FUNCTION n name, STORAGE type
	for {
		item := p.parseOpClassItem()
		oc.Items = append(oc.Items, item)
		if !p.gotSelf(',') {
			break
		}
	}

	return oc
}

func (p *Parser) parseOpClassItem() *CreateOpClassItem {
	pos := p.pos
	item := &CreateOpClassItem{baseNode: baseNode{pos}}

	switch {
	case p.gotKeyword("operator"):
		item.ItemType = 1
		item.Number = int(p.parseInt())
		item.Name = p.parseOperatorName()
		// Optional (left_type, right_type)
		if p.tok == Token('(') {
			p.next()
			for p.tok != Token(')') && p.tok != EOF {
				item.ClassArgs = append(item.ClassArgs, p.parseTypeName())
				if !p.gotSelf(',') {
					break
				}
			}
			p.wantSelf(')')
		}
		// Optional FOR ORDER BY / FOR SEARCH
		if p.gotKeyword("for") {
			if p.gotKeyword("order") {
				p.wantKeyword("by")
				item.OrderFamily = p.parseQualifiedName()
			} else {
				p.wantKeyword("search")
			}
		}
	case p.gotKeyword("function"):
		item.ItemType = 2
		item.Number = int(p.parseInt())
		// Optional (left_type, right_type)
		if p.tok == Token('(') {
			// Could be arg types or function args — disambiguate
			// If followed by type, type) it's class args
			// For simplicity, parse as function name + args
		}
		item.Name = p.parseQualifiedName()
		if p.tok == Token('(') {
			p.next()
			for p.tok != Token(')') && p.tok != EOF {
				item.ClassArgs = append(item.ClassArgs, p.parseTypeName())
				if !p.gotSelf(',') {
					break
				}
			}
			p.wantSelf(')')
		}
	case p.isKeyword("storage"):
		p.next()
		item.ItemType = 3
		item.StoredType = p.parseTypeName()
	default:
		p.syntaxError("expected OPERATOR, FUNCTION, or STORAGE")
	}

	return item
}

// ---------------------------------------------------------------------------
// CREATE OPERATOR FAMILY
// ---------------------------------------------------------------------------

func (p *Parser) parseCreateOpFamily() *CreateOpFamilyStmt {
	p.wantKeyword("family")
	pos := p.pos

	of := &CreateOpFamilyStmt{baseStmt: baseStmt{baseNode{pos}}}
	of.OpFamilyName = p.parseQualifiedName()
	p.wantKeyword("using")
	of.AmName = p.colId()

	return of
}

// ---------------------------------------------------------------------------
// CREATE [OR REPLACE] [TRUSTED] [PROCEDURAL] LANGUAGE
// ---------------------------------------------------------------------------

func (p *Parser) parseCreateLanguage(replace bool) *CreatePLangStmt {
	pos := p.pos

	pl := &CreatePLangStmt{
		baseStmt: baseStmt{baseNode{pos}},
		Replace:  replace,
	}

	if p.gotKeyword("trusted") {
		pl.Trusted = true
	}
	p.gotKeyword("procedural") // optional
	p.wantKeyword("language")

	pl.PLName = p.colId()

	// Optional HANDLER, INLINE, VALIDATOR
	for {
		switch {
		case p.gotKeyword("handler"):
			pl.PLHandler = p.parseQualifiedName()
		case p.isKeyword("inline"):
			p.next()
			pl.PLInline = p.parseQualifiedName()
		case p.gotKeyword("validator"):
			pl.PLValidator = p.parseQualifiedName()
		default:
			return pl
		}
	}
}

// ---------------------------------------------------------------------------
// CREATE [DEFAULT] CONVERSION
// ---------------------------------------------------------------------------

func (p *Parser) parseCreateConversion(isDefault bool) *CreateConversionStmt {
	p.wantKeyword("conversion")
	pos := p.pos

	cc := &CreateConversionStmt{
		baseStmt:  baseStmt{baseNode{pos}},
		IsDefault: isDefault,
	}

	cc.ConvName = p.parseQualifiedName()
	p.wantKeyword("for")

	if p.tok == SCONST {
		cc.ForEncoding = p.lit
		p.next()
	}

	p.wantKeyword("to")

	if p.tok == SCONST {
		cc.ToEncoding = p.lit
		p.next()
	}

	p.wantKeyword("from")
	cc.FuncName = p.parseQualifiedName()

	return cc
}
