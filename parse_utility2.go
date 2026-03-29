package pgscan

// ---------------------------------------------------------------------------
// COMMENT ON
// ---------------------------------------------------------------------------

// parseCommentStmt parses COMMENT ON object_type name IS 'text' | NULL.
func (p *Parser) parseCommentStmt() *CommentStmt {
	p.wantKeyword("comment")
	p.wantKeyword("on")
	pos := p.pos

	cs := &CommentStmt{baseStmt: baseStmt{baseNode{pos}}}
	cs.ObjType, cs.Object = p.parseCommentTarget()

	p.wantKeyword("is")

	if p.gotKeyword("null") {
		cs.IsNull = true
	} else if p.tok == SCONST {
		cs.Comment = p.lit
		p.next()
	}

	return cs
}

// parseCommentTarget parses the object type and name for COMMENT ON.
func (p *Parser) parseCommentTarget() (ObjectType, []string) {
	switch {
	case p.gotKeyword("table"):
		return OBJECT_TABLE, p.parseQualifiedName()
	case p.gotKeyword("index"):
		return OBJECT_INDEX, p.parseQualifiedName()
	case p.gotKeyword("view"):
		return OBJECT_VIEW, p.parseQualifiedName()
	case p.gotKeyword("sequence"):
		return OBJECT_SEQUENCE, p.parseQualifiedName()
	case p.gotKeyword("type"):
		return OBJECT_TYPE, p.parseQualifiedName()
	case p.gotKeyword("schema"):
		return OBJECT_SCHEMA, p.parseQualifiedName()
	case p.gotKeyword("domain"):
		return OBJECT_DOMAIN, p.parseQualifiedName()
	case p.gotKeyword("function"):
		return OBJECT_FUNCTION, p.parseQualifiedName()
	case p.gotKeyword("procedure"):
		return OBJECT_PROCEDURE, p.parseQualifiedName()
	case p.gotKeyword("extension"):
		return OBJECT_EXTENSION, p.parseQualifiedName()
	case p.gotKeyword("trigger"):
		name := p.parseQualifiedName()
		if p.gotKeyword("on") {
			tbl := p.parseQualifiedName()
			name = append(tbl, name...)
		}
		return OBJECT_TRIGGER, name
	case p.gotKeyword("rule"):
		name := p.parseQualifiedName()
		if p.gotKeyword("on") {
			tbl := p.parseQualifiedName()
			name = append(tbl, name...)
		}
		return OBJECT_RULE, name
	case p.gotKeyword("column"):
		return OBJECT_COLUMN, p.parseQualifiedName()
	case p.gotKeyword("constraint"):
		name := p.parseQualifiedName()
		if p.gotKeyword("on") {
			p.gotKeyword("domain") // optional
			tbl := p.parseQualifiedName()
			name = append(tbl, name...)
		}
		return OBJECT_CONSTRAINT, name
	case p.gotKeyword("database"):
		return OBJECT_DATABASE, p.parseQualifiedName()
	case p.gotKeyword("role"):
		return OBJECT_ROLE, p.parseQualifiedName()
	case p.gotKeyword("tablespace"):
		return OBJECT_TABLESPACE, p.parseQualifiedName()
	case p.gotKeyword("policy"):
		name := p.parseQualifiedName()
		if p.gotKeyword("on") {
			tbl := p.parseQualifiedName()
			name = append(tbl, name...)
		}
		return OBJECT_POLICY, name
	case p.gotKeyword("publication"):
		return OBJECT_PUBLICATION, p.parseQualifiedName()
	case p.gotKeyword("subscription"):
		return OBJECT_SUBSCRIPTION, p.parseQualifiedName()
	case p.gotKeyword("aggregate"):
		return OBJECT_AGGREGATE, p.parseQualifiedName()
	case p.gotKeyword("operator"):
		return OBJECT_OPERATOR, p.parseQualifiedName()
	case p.gotKeyword("collation"):
		return OBJECT_COLLATION, p.parseQualifiedName()
	case p.gotKeyword("conversion"):
		return OBJECT_CONVERSION, p.parseQualifiedName()
	case p.gotKeyword("language"):
		return OBJECT_LANGUAGE, p.parseQualifiedName()
	case p.gotKeyword("cast"):
		return OBJECT_CAST, p.parseQualifiedName()
	case p.isKeyword("event"):
		p.next()
		p.wantKeyword("trigger")
		return OBJECT_EVENT_TRIGGER, p.parseQualifiedName()
	case p.isKeyword("access"):
		p.next()
		p.wantKeyword("method")
		return OBJECT_ACCESS_METHOD, p.parseQualifiedName()
	case p.isKeyword("foreign"):
		p.next()
		if p.gotKeyword("table") {
			return OBJECT_FOREIGN_TABLE, p.parseQualifiedName()
		}
		// FOREIGN DATA WRAPPER
		p.wantKeyword("data")
		p.wantKeyword("wrapper")
		return OBJECT_FDW, p.parseQualifiedName()
	case p.isKeyword("server"):
		p.next()
		return OBJECT_FOREIGN_SERVER, p.parseQualifiedName()
	case p.isKeyword("materialized"):
		p.next()
		p.wantKeyword("view")
		return OBJECT_MATVIEW, p.parseQualifiedName()
	case p.isKeyword("text"):
		p.next()
		p.wantKeyword("search")
		switch {
		case p.gotKeyword("parser"):
			return OBJECT_TSPARSER, p.parseQualifiedName()
		case p.gotKeyword("dictionary"):
			return OBJECT_TSDICTIONARY, p.parseQualifiedName()
		case p.gotKeyword("template"):
			return OBJECT_TSTEMPLATE, p.parseQualifiedName()
		case p.gotKeyword("configuration"):
			return OBJECT_TSCONFIGURATION, p.parseQualifiedName()
		default:
			p.syntaxError("expected PARSER, DICTIONARY, TEMPLATE, or CONFIGURATION")
			return OBJECT_TABLE, nil
		}
	case p.isKeyword("statistics"):
		p.next()
		return OBJECT_STATISTICS, p.parseQualifiedName()
	case p.isKeyword("large"):
		p.next()
		p.wantKeyword("object")
		return OBJECT_LARGEOBJECT, p.parseQualifiedName()
	default:
		p.syntaxError("expected object type for COMMENT ON")
		return OBJECT_TABLE, nil
	}
}

// ---------------------------------------------------------------------------
// SECURITY LABEL
// ---------------------------------------------------------------------------

// parseSecLabelStmt parses SECURITY LABEL [FOR provider] ON object_type name IS 'label' | NULL.
func (p *Parser) parseSecLabelStmt() *SecLabelStmt {
	p.wantKeyword("security")
	p.wantKeyword("label")
	pos := p.pos

	sl := &SecLabelStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.gotKeyword("for") {
		sl.Provider = p.colId()
	}

	p.wantKeyword("on")
	sl.ObjType, sl.Object = p.parseCommentTarget() // same object type syntax

	p.wantKeyword("is")

	if p.gotKeyword("null") {
		sl.IsNull = true
	} else if p.tok == SCONST {
		sl.Label = p.lit
		p.next()
	}

	return sl
}

// ---------------------------------------------------------------------------
// CHECKPOINT
// ---------------------------------------------------------------------------

func (p *Parser) parseCheckpointStmt() *CheckPointStmt {
	p.wantKeyword("checkpoint")
	return &CheckPointStmt{baseStmt: baseStmt{baseNode{p.pos}}}
}

// ---------------------------------------------------------------------------
// LOAD
// ---------------------------------------------------------------------------

func (p *Parser) parseLoadStmt() *LoadStmt {
	p.wantKeyword("load")
	pos := p.pos
	ls := &LoadStmt{baseStmt: baseStmt{baseNode{pos}}}
	if p.tok == SCONST {
		ls.Filename = p.lit
		p.next()
	}
	return ls
}

// ---------------------------------------------------------------------------
// REINDEX
// ---------------------------------------------------------------------------

// parseReindexStmt parses REINDEX [(options)] {INDEX|TABLE|SCHEMA|DATABASE|SYSTEM} [CONCURRENTLY] name.
func (p *Parser) parseReindexStmt() *ReindexStmt {
	p.wantKeyword("reindex")
	pos := p.pos

	rs := &ReindexStmt{baseStmt: baseStmt{baseNode{pos}}}

	// Optional (options)
	if p.tok == Token('(') {
		p.next()
		rs.Options = p.parseReindexOptions()
		p.wantSelf(')')
	}

	// Object kind
	switch {
	case p.gotKeyword("index"):
		rs.Kind = REINDEX_OBJECT_INDEX
	case p.gotKeyword("table"):
		rs.Kind = REINDEX_OBJECT_TABLE
	case p.gotKeyword("schema"):
		rs.Kind = REINDEX_OBJECT_SCHEMA
	case p.gotKeyword("database"):
		rs.Kind = REINDEX_OBJECT_DATABASE
	case p.isKeyword("system"):
		p.next()
		rs.Kind = REINDEX_OBJECT_SYSTEM
	default:
		p.syntaxError("expected INDEX, TABLE, SCHEMA, DATABASE, or SYSTEM")
	}

	// Optional CONCURRENTLY
	if p.gotKeyword("concurrently") {
		rs.Concurrent = true
	}

	// Name
	switch rs.Kind {
	case REINDEX_OBJECT_INDEX, REINDEX_OBJECT_TABLE:
		rs.Relation = p.parseRangeVar()
	default:
		rs.Name = p.colId()
	}

	return rs
}

func (p *Parser) parseReindexOptions() []*DefElem {
	var opts []*DefElem
	for {
		pos := p.pos
		name := p.colLabel()
		de := &DefElem{baseNode: baseNode{pos}, Defname: name}
		// Optional boolean value
		if p.tok == IDENT || p.tok == KEYWORD {
			de.Arg = &String{baseNode: baseNode{p.pos}, Str: p.lit}
			p.next()
		}
		opts = append(opts, de)
		if !p.gotSelf(',') {
			break
		}
	}
	return opts
}

// ---------------------------------------------------------------------------
// SET CONSTRAINTS
// ---------------------------------------------------------------------------

// parseSetConstraintsStmt parses SET CONSTRAINTS {ALL|name,...} {DEFERRED|IMMEDIATE}.
// Called when SET has been consumed and CONSTRAINTS is the current token.
func (p *Parser) parseSetConstraintsStmt() *ConstraintsSetStmt {
	p.wantKeyword("constraints")
	pos := p.pos

	cs := &ConstraintsSetStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.gotKeyword("all") {
		cs.Constraints = nil // nil means ALL
	} else {
		cs.Constraints = append(cs.Constraints, p.parseQualifiedName())
		for p.gotSelf(',') {
			cs.Constraints = append(cs.Constraints, p.parseQualifiedName())
		}
	}

	if p.gotKeyword("deferred") {
		cs.Deferred = true
	} else {
		p.wantKeyword("immediate")
		cs.Deferred = false
	}

	return cs
}

// ---------------------------------------------------------------------------
// ALTER DEFAULT PRIVILEGES
// ---------------------------------------------------------------------------

// parseAlterDefaultPrivileges parses ALTER DEFAULT PRIVILEGES [FOR ROLE ...] [IN SCHEMA ...] grant_or_revoke.
// Called after ALTER has been consumed and DEFAULT is the current token.
func (p *Parser) parseAlterDefaultPrivileges() *AlterDefaultPrivilegesStmt {
	p.wantKeyword("default")
	p.wantKeyword("privileges")
	pos := p.pos

	adp := &AlterDefaultPrivilegesStmt{baseStmt: baseStmt{baseNode{pos}}}

	// Optional FOR ROLE/USER and IN SCHEMA clauses (can appear in any order)
	for {
		if p.isKeyword("for") {
			p.next()
			if !p.gotKeyword("role") {
			p.gotKeyword("user")
		}
			roles := p.parseRoleList()
			for _, r := range roles {
				adp.Options = append(adp.Options, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "for_role",
					Arg:      &String{baseNode: baseNode{p.pos}, Str: r},
				})
			}
		} else if p.isKeyword("in") {
			p.next()
			p.wantKeyword("schema")
			schemas := p.parseNameList()
			for _, s := range schemas {
				adp.Options = append(adp.Options, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "in_schema",
					Arg:      &String{baseNode: baseNode{p.pos}, Str: s},
				})
			}
		} else {
			break
		}
	}

	// The action is a GRANT or REVOKE statement
	if p.isKeyword("grant") {
		adp.Action = p.parseGrantStmt()
	} else if p.isKeyword("revoke") {
		adp.Action = p.parseRevokeStmt()
	} else {
		p.syntaxError("expected GRANT or REVOKE")
	}

	return adp
}
