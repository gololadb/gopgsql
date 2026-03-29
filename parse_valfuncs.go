package pgscan

// trySQLValueFunction checks if the current token is a SQL value function
// keyword and returns the corresponding node, or nil if not.
func (p *Parser) trySQLValueFunction(pos int) Expr {
	if p.tok != KEYWORD {
		return nil
	}

	switch p.lit {
	case "current_date":
		p.next()
		return &SQLValueFunction{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       SVFOP_CURRENT_DATE,
			Typmod:   -1,
		}

	case "current_time":
		p.next()
		if p.tok == Token('(') {
			p.next()
			typmod := int32(p.parseInt())
			p.wantSelf(')')
			return &SQLValueFunction{
				baseExpr: baseExpr{baseNode{pos}},
				Op:       SVFOP_CURRENT_TIME_N,
				Typmod:   typmod,
			}
		}
		return &SQLValueFunction{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       SVFOP_CURRENT_TIME,
			Typmod:   -1,
		}

	case "current_timestamp":
		p.next()
		if p.tok == Token('(') {
			p.next()
			typmod := int32(p.parseInt())
			p.wantSelf(')')
			return &SQLValueFunction{
				baseExpr: baseExpr{baseNode{pos}},
				Op:       SVFOP_CURRENT_TIMESTAMP_N,
				Typmod:   typmod,
			}
		}
		return &SQLValueFunction{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       SVFOP_CURRENT_TIMESTAMP,
			Typmod:   -1,
		}

	case "localtime":
		p.next()
		if p.tok == Token('(') {
			p.next()
			typmod := int32(p.parseInt())
			p.wantSelf(')')
			return &SQLValueFunction{
				baseExpr: baseExpr{baseNode{pos}},
				Op:       SVFOP_LOCALTIME_N,
				Typmod:   typmod,
			}
		}
		return &SQLValueFunction{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       SVFOP_LOCALTIME,
			Typmod:   -1,
		}

	case "localtimestamp":
		p.next()
		if p.tok == Token('(') {
			p.next()
			typmod := int32(p.parseInt())
			p.wantSelf(')')
			return &SQLValueFunction{
				baseExpr: baseExpr{baseNode{pos}},
				Op:       SVFOP_LOCALTIMESTAMP_N,
				Typmod:   typmod,
			}
		}
		return &SQLValueFunction{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       SVFOP_LOCALTIMESTAMP,
			Typmod:   -1,
		}

	case "current_role":
		p.next()
		return &SQLValueFunction{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       SVFOP_CURRENT_ROLE,
			Typmod:   -1,
		}

	case "current_user":
		p.next()
		return &SQLValueFunction{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       SVFOP_CURRENT_USER,
			Typmod:   -1,
		}

	case "session_user":
		p.next()
		return &SQLValueFunction{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       SVFOP_SESSION_USER,
			Typmod:   -1,
		}

	case "user":
		p.next()
		return &SQLValueFunction{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       SVFOP_USER,
			Typmod:   -1,
		}

	case "current_catalog":
		p.next()
		return &SQLValueFunction{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       SVFOP_CURRENT_CATALOG,
			Typmod:   -1,
		}

	case "current_schema":
		p.next()
		// CURRENT_SCHEMA can also be called as CURRENT_SCHEMA()
		if p.tok == Token('(') {
			p.next()
			p.wantSelf(')')
		}
		return &SQLValueFunction{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       SVFOP_CURRENT_SCHEMA,
			Typmod:   -1,
		}

	case "system_user":
		// SYSTEM_USER is implemented as a function call in PG
		p.next()
		return sysFuncCall(pos, "system_user", nil)
	}

	return nil
}
