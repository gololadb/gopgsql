package pgscan

// parseCreateDatabase parses CREATE DATABASE name [WITH] [options].
func (p *Parser) parseCreateDatabase() *CreatedbStmt {
	p.wantKeyword("database")
	pos := p.pos

	cd := &CreatedbStmt{baseStmt: baseStmt{baseNode{pos}}}
	cd.Dbname = p.colId()

	// Optional WITH
	p.gotKeyword("with")

	// Options: key = value or key value or boolean key
	cd.Options = p.parseCreatedbOptions()
	return cd
}

func (p *Parser) parseCreatedbOptions() []*DefElem {
	var opts []*DefElem
	for {
		switch {
		case p.isAnyKeyword("owner", "template", "encoding", "locale", "tablespace",
			"connection", "allow_connections"):
			name := p.lit
			p.next()
			// CONNECTION LIMIT is two words
			if name == "connection" {
				p.wantKeyword("limit")
				name = "connection_limit"
			}
			p.gotSelf('=') // optional =
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  name,
				Arg:      p.parseCreatedbOptValue(),
			})
		case p.isKeyword("is_template"):
			p.next()
			p.gotSelf('=')
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "is_template",
				Arg:      p.parseCreatedbOptValue(),
			})
		// Handle identifiers that are option names but not keywords
		case p.tok == IDENT && isCreatedbOption(p.lit):
			name := p.lit
			p.next()
			p.gotSelf('=')
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  name,
				Arg:      p.parseCreatedbOptValue(),
			})
		default:
			return opts
		}
	}
}

func isCreatedbOption(name string) bool {
	switch name {
	case "owner", "template", "encoding", "locale", "lc_collate", "lc_ctype",
		"icu_locale", "icu_rules", "locale_provider", "collation_version",
		"tablespace", "allow_connections", "connection_limit", "is_template",
		"oid", "strategy", "builtin_locale":
		return true
	}
	return false
}

func (p *Parser) parseCreatedbOptValue() Node {
	switch {
	case p.tok == SCONST:
		s := p.lit
		p.next()
		return &String{baseNode: baseNode{p.pos}, Str: s}
	case p.tok == ICONST:
		return &A_Const{baseExpr: baseExpr{baseNode{p.pos}}, Val: Value{Type: ValInt, Ival: p.parseInt()}}
	case p.isAnyKeyword("true", "false"):
		s := p.lit
		p.next()
		return &String{baseNode: baseNode{p.pos}, Str: s}
	case p.gotKeyword("default"):
		return nil
	default:
		s := p.colId()
		return &String{baseNode: baseNode{p.pos}, Str: s}
	}
}

// parseDropDatabase parses DROP DATABASE [IF EXISTS] name [WITH (FORCE)].
func (p *Parser) parseDropDatabase() *DropdbStmt {
	p.wantKeyword("database")
	pos := p.pos

	dd := &DropdbStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("exists")
		dd.MissingOk = true
	}

	dd.Dbname = p.colId()

	// Optional WITH (FORCE)
	if p.isKeyword("with") {
		p.next()
		dd.Options = p.parseParenDefElemList()
	}

	return dd
}

// parseAlterDatabase parses ALTER DATABASE name ...
func (p *Parser) parseAlterDatabase() Stmt {
	p.wantKeyword("database")
	pos := p.pos
	dbname := p.colId()

	// ALTER DATABASE name SET/RESET config
	if p.isKeyword("set") || p.isKeyword("reset") {
		var setStmt Stmt
		if p.isKeyword("set") {
			setStmt = p.parseSetStmt()
		} else {
			setStmt = p.parseResetStmt()
		}
		return &AlterDatabaseSetStmt{
			baseStmt: baseStmt{baseNode{pos}},
			Dbname:   dbname,
			SetStmt:  setStmt,
		}
	}

	// ALTER DATABASE name WITH options / RENAME / OWNER TO / SET TABLESPACE
	ad := &AlterDatabaseStmt{
		baseStmt: baseStmt{baseNode{pos}},
		Dbname:   dbname,
	}

	p.gotKeyword("with") // optional WITH
	ad.Options = p.parseCreatedbOptions()
	return ad
}

// parseCreateTablespace parses CREATE TABLESPACE name [OWNER role] LOCATION 'dir' [WITH (opts)].
func (p *Parser) parseCreateTablespace() *CreateTableSpaceStmt {
	p.wantKeyword("tablespace")
	pos := p.pos

	ct := &CreateTableSpaceStmt{baseStmt: baseStmt{baseNode{pos}}}
	ct.Tablespacename = p.colId()

	if p.gotKeyword("owner") {
		ct.Owner = p.colId()
	}

	p.wantKeyword("location")
	if p.tok == SCONST {
		ct.Location = p.lit
		p.next()
	}

	if p.isKeyword("with") {
		p.next()
		ct.Options = p.parseParenDefElemList()
	}

	return ct
}
