package pgscan

// parseIntervalLiteral parses:
//   INTERVAL 'string' [opt_interval]
//   INTERVAL '(' Iconst ')' 'string'
// Called when current token is INTERVAL keyword.
func (p *Parser) parseIntervalLiteral(pos int) Expr {
	p.next() // consume INTERVAL

	tn := &TypeName{
		baseNode: baseNode{pos},
		Names:    []string{"pg_catalog", "interval"},
	}

	// INTERVAL (p) Sconst — precision before the string
	if p.tok == Token('(') {
		p.next()
		prec := p.parseInt()
		p.wantSelf(')')
		if p.tok != SCONST {
			p.syntaxError("expected string literal after INTERVAL(precision)")
		}
		s := p.lit
		p.next()
		tn.Typmods = []Expr{
			&A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: IntervalFullRange}},
			&A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: prec}},
		}
		return &TypeCast{
			baseExpr: baseExpr{baseNode{pos}},
			Arg:      &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValStr, Str: s}},
			TypeName: tn,
		}
	}

	// INTERVAL Sconst [opt_interval]
	if p.tok == SCONST {
		s := p.lit
		p.next()
		tn.Typmods = p.parseIntervalFields(pos)
		return &TypeCast{
			baseExpr: baseExpr{baseNode{pos}},
			Arg:      &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValStr, Str: s}},
			TypeName: tn,
		}
	}

	// Bare INTERVAL used as a type expression (shouldn't normally appear
	// in expression context without a string, but handle gracefully).
	p.syntaxError("expected string literal or '(' after INTERVAL")
	return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValNull}}
}

// tryConstTypename attempts to parse a ConstTypename (Numeric, ConstBit,
// ConstCharacter, ConstDatetime, JsonType) that can precede a string literal.
// Returns nil if the current token doesn't start a const typename.
// This does NOT consume INTERVAL (handled separately above).
func (p *Parser) tryConstTypename() *TypeName {
	pos := p.pos

	switch {
	// Numeric types: INT, SMALLINT, BIGINT, REAL, FLOAT, DOUBLE PRECISION,
	// DECIMAL, NUMERIC — these are ConstTypename via Numeric production.
	case p.isAnyKeyword("int", "integer"):
		p.next()
		return &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", "int4"}}
	case p.isKeyword("smallint"):
		p.next()
		return &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", "int2"}}
	case p.isKeyword("bigint"):
		p.next()
		return &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", "int8"}}
	case p.isKeyword("real"):
		p.next()
		return &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", "float4"}}
	case p.isKeyword("float"):
		p.next()
		tn := &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", "float8"}}
		if p.tok == Token('(') {
			p.next()
			tn.Typmods = p.parseExprList()
			p.wantSelf(')')
		}
		return tn
	case p.isKeyword("double"):
		p.next()
		p.wantKeyword("precision")
		return &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", "float8"}}
	case p.isAnyKeyword("decimal", "numeric"):
		p.next()
		tn := &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", "numeric"}}
		if p.tok == Token('(') {
			p.next()
			tn.Typmods = p.parseExprList()
			p.wantSelf(')')
		}
		return tn
	case p.isKeyword("boolean"), p.isKeyword("bool"):
		p.next()
		return &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", "bool"}}

	// ConstBit: BIT [VARYING] [(n)]
	case p.isKeyword("bit"):
		p.next()
		var name string
		if p.gotKeyword("varying") {
			name = "varbit"
		} else {
			name = "bit"
		}
		tn := &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", name}}
		if p.tok == Token('(') {
			p.next()
			tn.Typmods = p.parseExprList()
			p.wantSelf(')')
		}
		return tn

	// ConstCharacter: CHARACTER [VARYING] [(n)], CHAR [(n)], VARCHAR [(n)]
	case p.isKeyword("character"):
		p.next()
		var name string
		if p.gotKeyword("varying") {
			name = "varchar"
		} else {
			name = "bpchar"
		}
		tn := &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", name}}
		if p.tok == Token('(') {
			p.next()
			tn.Typmods = p.parseExprList()
			p.wantSelf(')')
		}
		return tn
	case p.isKeyword("char"):
		p.next()
		tn := &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", "bpchar"}}
		if p.tok == Token('(') {
			p.next()
			tn.Typmods = p.parseExprList()
			p.wantSelf(')')
		}
		return tn
	case p.isKeyword("varchar"):
		p.next()
		tn := &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", "varchar"}}
		if p.tok == Token('(') {
			p.next()
			tn.Typmods = p.parseExprList()
			p.wantSelf(')')
		}
		return tn

	// ConstDatetime: TIMESTAMP [(p)] [WITH/WITHOUT TIME ZONE], TIME [(p)] [WITH/WITHOUT TIME ZONE], DATE
	case p.isKeyword("timestamp"):
		p.next()
		tn := &TypeName{baseNode: baseNode{pos}}
		if p.tok == Token('(') {
			p.next()
			tn.Typmods = p.parseExprList()
			p.wantSelf(')')
		}
		if p.gotKeyword("with") {
			p.wantKeyword("time")
			p.wantKeyword("zone")
			tn.Names = []string{"pg_catalog", "timestamptz"}
		} else if p.gotKeyword("without") {
			p.wantKeyword("time")
			p.wantKeyword("zone")
			tn.Names = []string{"pg_catalog", "timestamp"}
		} else {
			tn.Names = []string{"pg_catalog", "timestamp"}
		}
		return tn
	case p.isKeyword("time"):
		p.next()
		tn := &TypeName{baseNode: baseNode{pos}}
		if p.tok == Token('(') {
			p.next()
			tn.Typmods = p.parseExprList()
			p.wantSelf(')')
		}
		if p.gotKeyword("with") {
			p.wantKeyword("time")
			p.wantKeyword("zone")
			tn.Names = []string{"pg_catalog", "timetz"}
		} else if p.gotKeyword("without") {
			p.wantKeyword("time")
			p.wantKeyword("zone")
			tn.Names = []string{"pg_catalog", "time"}
		} else {
			tn.Names = []string{"pg_catalog", "time"}
		}
		return tn
	// NOTE: DATE is not a PG keyword — it's a generic type name (identifier).
	// It's handled by the func_name Sconst path in parseColumnRefOrFunc.

	// JsonType
	case p.isKeyword("json"):
		p.next()
		return &TypeName{baseNode: baseNode{pos}, Names: []string{"pg_catalog", "json"}}
	}

	return nil
}
