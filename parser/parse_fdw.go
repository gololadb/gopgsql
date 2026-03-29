package parser

// parseFdwOptions parses (name 'value', ...) — FDW-style options without '='.
func (p *Parser) parseFdwOptions() []*DefElem {
	p.wantSelf('(')
	var opts []*DefElem
	for {
		name := p.colLabel() // option names can be reserved keywords like "user"
		var arg Node
		if p.tok == SCONST {
			arg = &String{baseNode: baseNode{p.pos}, Str: p.lit}
			p.next()
		}
		opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: name, Arg: arg})
		if !p.gotSelf(',') {
			break
		}
	}
	p.wantSelf(')')
	return opts
}

// parseCreateFdw parses CREATE FOREIGN DATA WRAPPER name [HANDLER func] [VALIDATOR func] [OPTIONS (opts)].
func (p *Parser) parseCreateFdw() *CreateFdwStmt {
	p.wantKeyword("data")
	p.wantKeyword("wrapper")
	pos := p.pos

	cf := &CreateFdwStmt{baseStmt: baseStmt{baseNode{pos}}}
	cf.Fdwname = p.colId()

	for {
		switch {
		case p.isKeyword("handler"):
			p.next()
			cf.FuncOptions = append(cf.FuncOptions, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "handler",
				Arg:      &String{baseNode: baseNode{p.pos}, Str: p.colId()},
			})
		case p.isKeyword("no"):
			p.next()
			if p.gotKeyword("handler") {
				cf.FuncOptions = append(cf.FuncOptions, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "handler",
				})
			} else if p.gotKeyword("validator") {
				cf.FuncOptions = append(cf.FuncOptions, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "validator",
				})
			}
		case p.isKeyword("validator"):
			p.next()
			cf.FuncOptions = append(cf.FuncOptions, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "validator",
				Arg:      &String{baseNode: baseNode{p.pos}, Str: p.colId()},
			})
		case p.isKeyword("options"):
			p.next()
			cf.Options = p.parseFdwOptions()
		default:
			return cf
		}
	}
}

// parseCreateServer parses CREATE SERVER [IF NOT EXISTS] name [TYPE 'type'] [VERSION 'ver']
//   FOREIGN DATA WRAPPER fdw [OPTIONS (opts)].
func (p *Parser) parseCreateServer() *CreateForeignServerStmt {
	p.wantKeyword("server")
	pos := p.pos

	cs := &CreateForeignServerStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("not")
		p.wantKeyword("exists")
		cs.IfNotExists = true
	}

	cs.Servername = p.colId()

	if p.gotKeyword("type") {
		if p.tok == SCONST {
			cs.ServerType = p.lit
			p.next()
		}
	}

	if p.gotKeyword("version") {
		if p.tok == SCONST {
			cs.Version = p.lit
			p.next()
		} else if p.gotKeyword("null") {
			cs.Version = ""
		}
	}

	p.wantKeyword("foreign")
	p.wantKeyword("data")
	p.wantKeyword("wrapper")
	cs.Fdwname = p.colId()

	if p.isKeyword("options") {
		p.next()
		cs.Options = p.parseFdwOptions()
	}

	return cs
}

// parseCreateForeignTable parses CREATE FOREIGN TABLE [IF NOT EXISTS] name (cols) SERVER name [OPTIONS (opts)].
func (p *Parser) parseCreateForeignTable() *CreateForeignTableStmt {
	p.wantKeyword("table")
	pos := p.pos

	cft := &CreateForeignTableStmt{baseStmt: baseStmt{baseNode{pos}}}
	cft.Base.baseStmt = baseStmt{baseNode{pos}}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("not")
		p.wantKeyword("exists")
		cft.Base.IfNotExists = true
	}

	cft.Base.Relation = p.parseRangeVar()

	// Column definitions
	if p.gotSelf('(') {
		if p.tok != Token(')') {
			cft.Base.TableElts = append(cft.Base.TableElts, p.parseColumnDef())
			for p.gotSelf(',') {
				cft.Base.TableElts = append(cft.Base.TableElts, p.parseColumnDef())
			}
		}
		p.wantSelf(')')
	}

	p.wantKeyword("server")
	cft.Servername = p.colId()

	if p.isKeyword("options") {
		p.next()
		cft.Options = p.parseFdwOptions()
	}

	return cft
}

// parseCreateUserMapping parses CREATE USER MAPPING [IF NOT EXISTS] FOR role SERVER name [OPTIONS (opts)].
func (p *Parser) parseCreateUserMapping() *CreateUserMappingStmt {
	p.wantKeyword("mapping")
	pos := p.pos

	cum := &CreateUserMappingStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("not")
		p.wantKeyword("exists")
		cum.IfNotExists = true
	}

	p.wantKeyword("for")

	// User can be a role name, PUBLIC, CURRENT_USER, CURRENT_ROLE, SESSION_USER, USER
	switch {
	case p.gotKeyword("public"):
		cum.User = "public"
	case p.gotKeyword("current_user"):
		cum.User = "current_user"
	case p.gotKeyword("current_role"):
		cum.User = "current_role"
	case p.gotKeyword("session_user"):
		cum.User = "session_user"
	case p.gotKeyword("user"):
		cum.User = "current_user"
	default:
		cum.User = p.colId()
	}

	p.wantKeyword("server")
	cum.Servername = p.colId()

	if p.isKeyword("options") {
		p.next()
		cum.Options = p.parseFdwOptions()
	}

	return cum
}

// parseImportForeignSchema parses IMPORT FOREIGN SCHEMA remote [LIMIT TO|EXCEPT (tables)] FROM SERVER name INTO local.
func (p *Parser) parseImportForeignSchema() *ImportForeignSchemaStmt {
	p.wantKeyword("import")
	pos := p.pos
	p.wantKeyword("foreign")
	p.wantKeyword("schema")

	ifs := &ImportForeignSchemaStmt{baseStmt: baseStmt{baseNode{pos}}}
	ifs.RemoteSchema = p.colId()

	// Optional LIMIT TO or EXCEPT
	if p.isKeyword("limit") {
		p.next()
		p.wantKeyword("to")
		ifs.ListType = "limit_to"
		p.wantSelf('(')
		ifs.TableList = append(ifs.TableList, p.colId())
		for p.gotSelf(',') {
			ifs.TableList = append(ifs.TableList, p.colId())
		}
		p.wantSelf(')')
	} else if p.isKeyword("except") {
		p.next()
		ifs.ListType = "except"
		p.wantSelf('(')
		ifs.TableList = append(ifs.TableList, p.colId())
		for p.gotSelf(',') {
			ifs.TableList = append(ifs.TableList, p.colId())
		}
		p.wantSelf(')')
	}

	p.wantKeyword("from")
	p.wantKeyword("server")
	ifs.ServerName = p.colId()

	p.wantKeyword("into")
	ifs.LocalSchema = p.colId()

	if p.isKeyword("options") {
		p.next()
		ifs.Options = p.parseFdwOptions()
	}

	return ifs
}
