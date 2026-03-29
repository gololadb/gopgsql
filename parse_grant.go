package pgscan

// parseGrantStmt parses GRANT privileges ON object TO roles.
func (p *Parser) parseGrantStmt() Stmt {
	p.wantKeyword("grant")
	pos := p.pos

	privs, privCols := p.parsePrivilegeListWithCols()

	if p.gotKeyword("on") {
		// Privilege grant: GRANT privs ON type objects TO roles
		gs := &GrantStmt{baseStmt: baseStmt{baseNode{pos}}, IsGrant: true, Privileges: privs, PrivCols: privCols}
		gs.TargetType, gs.Objects = p.parseGrantTarget()
		p.wantKeyword("to")
		gs.Grantees = p.parseRoleList()
		if p.isKeyword("with") {
			p.next()
			p.wantKeyword("grant")
			p.wantKeyword("option")
			gs.GrantOption = true
		}
		return gs
	}

	// Role grant: GRANT roles TO roles
	p.wantKeyword("to")
	grs := &GrantRoleStmt{
		baseStmt:     baseStmt{baseNode{pos}},
		IsGrant:      true,
		GrantedRoles: privs,
		Grantees:     p.parseRoleList(),
	}
	if p.isKeyword("with") {
		p.next()
		p.wantKeyword("admin")
		p.wantKeyword("option")
		grs.AdminOption = true
	}
	return grs
}

// parseRevokeStmt parses REVOKE privileges ON object FROM roles.
func (p *Parser) parseRevokeStmt() Stmt {
	p.wantKeyword("revoke")
	pos := p.pos

	// Check for GRANT OPTION FOR (GRANT is reserved, so unambiguous)
	grantOptionFor := false
	if p.isKeyword("grant") {
		p.next()
		p.wantKeyword("option")
		p.wantKeyword("for")
		grantOptionFor = true
	}

	privs, privCols := p.parsePrivilegeListWithCols()

	// After parsePrivilegeListWithCols, check for ADMIN OPTION FOR pattern:
	// REVOKE admin OPTION FOR role FROM role
	adminOptionFor := false
	if !grantOptionFor && p.isKeyword("option") && len(privs) == 1 && privs[0] == "admin" {
		p.next() // consume OPTION
		p.wantKeyword("for")
		adminOptionFor = true
		privs, privCols = p.parsePrivilegeListWithCols()
	}

	if p.gotKeyword("on") {
		gs := &GrantStmt{baseStmt: baseStmt{baseNode{pos}}, IsGrant: false, Privileges: privs, PrivCols: privCols}
		gs.TargetType, gs.Objects = p.parseGrantTarget()
		p.wantKeyword("from")
		gs.Grantees = p.parseRoleList()
		if p.gotKeyword("cascade") {
			gs.GrantOption = true
		} else {
			p.gotKeyword("restrict")
		}
		_ = grantOptionFor
		return gs
	}

	// Role revoke
	p.wantKeyword("from")
	return &GrantRoleStmt{
		baseStmt:     baseStmt{baseNode{pos}},
		IsGrant:      false,
		GrantedRoles: privs,
		Grantees:     p.parseRoleList(),
		AdminOption:  adminOptionFor,
	}
}

// parsePrivilegeList parses ALL [PRIVILEGES] or a comma-separated list of privilege names.
func (p *Parser) parsePrivilegeList() []string {
	var privs []string
	if p.gotKeyword("all") {
		p.gotKeyword("privileges") // optional
		return []string{"ALL"}
	}
	privs = append(privs, p.parsePrivilege())
	for p.gotSelf(',') {
		privs = append(privs, p.parsePrivilege())
	}
	return privs
}

func (p *Parser) parsePrivilege() string {
	name := p.colLabel()
	// Column-level privilege: SELECT (col1, col2)
	// The column list is consumed here but stored separately via parsePrivilegeListWithCols
	return name
}

// parsePrivilegeListWithCols parses privileges with optional column lists.
// Returns parallel slices: privilege names and their column lists (nil = no columns).
func (p *Parser) parsePrivilegeListWithCols() ([]string, [][]string) {
	if p.gotKeyword("all") {
		p.gotKeyword("privileges") // optional
		// ALL can also have column list
		var cols []string
		if p.tok == Token('(') {
			p.next()
			cols = p.parseNameList()
			p.wantSelf(')')
		}
		return []string{"ALL"}, [][]string{cols}
	}

	var privs []string
	var colLists [][]string

	name := p.colLabel()
	var cols []string
	if p.tok == Token('(') {
		p.next()
		cols = p.parseNameList()
		p.wantSelf(')')
	}
	privs = append(privs, name)
	colLists = append(colLists, cols)

	for p.gotSelf(',') {
		name = p.colLabel()
		cols = nil
		if p.tok == Token('(') {
			p.next()
			cols = p.parseNameList()
			p.wantSelf(')')
		}
		privs = append(privs, name)
		colLists = append(colLists, cols)
	}

	return privs, colLists
}

// parseGrantTarget parses the object type and names after ON.
func (p *Parser) parseGrantTarget() (ObjectType, [][]string) {
	var objType ObjectType
	switch {
	case p.gotKeyword("table"):
		objType = OBJECT_TABLE
	case p.gotKeyword("sequence"):
		objType = OBJECT_SEQUENCE
	case p.gotKeyword("function"):
		objType = OBJECT_FUNCTION
	case p.gotKeyword("procedure"):
		objType = OBJECT_PROCEDURE
	case p.gotKeyword("schema"):
		objType = OBJECT_SCHEMA
	case p.gotKeyword("database"):
		objType = OBJECT_SCHEMA // approximate
	case p.gotKeyword("type"):
		objType = OBJECT_TYPE
	case p.gotKeyword("domain"):
		objType = OBJECT_DOMAIN
	default:
		objType = OBJECT_TABLE // default is TABLE
	}

	var objects [][]string
	objects = append(objects, p.parseQualifiedName())
	for p.gotSelf(',') {
		objects = append(objects, p.parseQualifiedName())
	}
	return objType, objects
}

// parseRoleList parses a comma-separated list of role names.
func (p *Parser) parseRoleList() []string {
	var roles []string
	roles = append(roles, p.colLabel())
	for p.gotSelf(',') {
		roles = append(roles, p.colLabel())
	}
	return roles
}

// parseCreateRole parses CREATE ROLE/USER/GROUP name [options].
func (p *Parser) parseCreateRole() *CreateRoleStmt {
	pos := p.pos
	var stmtType string
	switch {
	case p.gotKeyword("role"):
		stmtType = "ROLE"
	case p.gotKeyword("user"):
		stmtType = "USER"
	case p.gotKeyword("group"):
		stmtType = "GROUP"
	}

	return p.parseCreateRoleBody(pos, stmtType)
}

// parseCreateRoleAfterKeyword is called when the ROLE/USER/GROUP keyword
// has already been consumed (e.g., CREATE USER where USER was consumed
// to check for USER MAPPING).
func (p *Parser) parseCreateRoleAfterKeyword() *CreateRoleStmt {
	return p.parseCreateRoleBody(p.pos, "USER")
}

func (p *Parser) parseCreateRoleBody(pos int, stmtType string) *CreateRoleStmt {
	cs := &CreateRoleStmt{
		baseStmt: baseStmt{baseNode{pos}},
		StmtType: stmtType,
	}
	cs.RoleName = p.colId()

	// Optional WITH
	p.gotKeyword("with")

	// Role options
	cs.Options = p.parseRoleOptions()
	return cs
}

// isRoleOption checks if the current token is a role option identifier.
func (p *Parser) isRoleOption() bool {
	if p.tok != IDENT {
		return false
	}
	switch p.lit {
	case "superuser", "nosuperuser", "createdb", "nocreatedb",
		"createrole", "nocreaterole", "login", "nologin", "replication",
		"noreplication", "bypassrls", "nobypassrls", "inherit", "noinherit":
		return true
	}
	return false
}

// parseRoleOptions parses role options like SUPERUSER, LOGIN, PASSWORD, etc.
func (p *Parser) parseRoleOptions() []*DefElem {
	var opts []*DefElem
	for {
		switch {
		case p.isRoleOption():
			opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: p.lit})
			p.next()
		case p.isKeyword("password"):
			p.next()
			if p.tok == SCONST {
				opts = append(opts, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "password",
					Arg:      &String{baseNode: baseNode{p.pos}, Str: p.lit},
				})
				p.next()
			} else if p.gotKeyword("null") {
				opts = append(opts, &DefElem{baseNode: baseNode{p.pos}, Defname: "password_null"})
			}
		case p.isKeyword("valid"):
			p.next()
			p.wantKeyword("until")
			if p.tok == SCONST {
				opts = append(opts, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "valid_until",
					Arg:      &String{baseNode: baseNode{p.pos}, Str: p.lit},
				})
				p.next()
			}
		case p.isKeyword("connection"):
			p.next()
			p.wantKeyword("limit")
			opts = append(opts, &DefElem{
				baseNode: baseNode{p.pos},
				Defname:  "connection_limit",
				Arg:      &A_Const{baseExpr: baseExpr{baseNode{p.pos}}, Val: Value{Type: ValInt, Ival: p.parseInt()}},
			})
		case p.isKeyword("in"):
			p.next()
			p.wantKeyword("role")
			roles := p.parseRoleList()
			for _, r := range roles {
				opts = append(opts, &DefElem{
					baseNode: baseNode{p.pos},
					Defname:  "in_role",
					Arg:      &String{baseNode: baseNode{p.pos}, Str: r},
				})
			}
		default:
			return opts
		}
	}
}

// parseCreateSchema parses CREATE SCHEMA [IF NOT EXISTS] name [AUTHORIZATION role].
func (p *Parser) parseCreateSchema() *CreateSchemaStmt {
	p.wantKeyword("schema")
	pos := p.pos

	cs := &CreateSchemaStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("not")
		p.wantKeyword("exists")
		cs.IfNotExists = true
	}

	if p.gotKeyword("authorization") {
		cs.AuthRole = p.colId()
	} else {
		cs.Schemaname = p.colId()
		if p.gotKeyword("authorization") {
			cs.AuthRole = p.colId()
		}
	}

	return cs
}

// parseCreateDomain parses CREATE DOMAIN name AS type [constraints].
func (p *Parser) parseCreateDomain() *CreateDomainStmt {
	p.wantKeyword("domain")
	pos := p.pos

	cd := &CreateDomainStmt{baseStmt: baseStmt{baseNode{pos}}}
	cd.Domainname = p.parseQualifiedName()
	p.wantKeyword("as")
	cd.TypeName = p.parseTypeName()

	// Optional constraints
	for {
		c := p.parseColConstraint()
		if c == nil {
			break
		}
		cd.Constraints = append(cd.Constraints, c)
	}

	return cd
}

// parseCreateType parses CREATE TYPE name AS ENUM (...) or CREATE TYPE name AS (col type, ...).
func (p *Parser) parseCreateType() Stmt {
	p.wantKeyword("type")
	pos := p.pos

	typeName := p.parseQualifiedName()

	// Shell type: CREATE TYPE name (no AS keyword)
	if !p.isKeyword("as") {
		return p.parseCreateTypeShell(typeName)
	}
	p.wantKeyword("as")

	// CREATE TYPE name AS RANGE (...)
	if p.isKeyword("range") {
		p.next()
		return p.parseCreateTypeRange(typeName)
	}

	if p.gotKeyword("enum") {
		// CREATE TYPE name AS ENUM (val, ...)
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
		return &CreateEnumStmt{
			baseStmt: baseStmt{baseNode{pos}},
			TypeName: typeName,
			Vals:     vals,
		}
	}

	// CREATE TYPE name AS (col type, ...)
	p.wantSelf('(')
	var cols []*ColumnDef
	for {
		cd := &ColumnDef{baseNode: baseNode{p.pos}}
		cd.Colname = p.colId()
		cd.TypeName = p.parseTypeName()
		cols = append(cols, cd)
		if !p.gotSelf(',') {
			break
		}
	}
	p.wantSelf(')')

	return &CompositeTypeStmt{
		baseStmt: baseStmt{baseNode{pos}},
		TypeName: typeName,
		ColDefs:  cols,
	}
}
