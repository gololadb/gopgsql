package parser

// parseDropRole parses DROP ROLE/USER/GROUP [IF EXISTS] name, ...
func (p *Parser) parseDropRole() *DropRoleStmt {
	p.next() // consume ROLE, USER, or GROUP
	pos := p.pos

	dr := &DropRoleStmt{baseStmt: baseStmt{baseNode{pos}}}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("exists")
		dr.MissingOk = true
	}

	dr.Roles = append(dr.Roles, p.colId())
	for p.gotSelf(',') {
		dr.Roles = append(dr.Roles, p.colId())
	}

	return dr
}

// parseDropOwned parses DROP OWNED BY role, ... [CASCADE|RESTRICT].
func (p *Parser) parseDropOwned() *DropOwnedStmt {
	p.wantKeyword("owned")
	pos := p.pos
	p.wantKeyword("by")

	do := &DropOwnedStmt{baseStmt: baseStmt{baseNode{pos}}}
	do.Roles = append(do.Roles, p.colId())
	for p.gotSelf(',') {
		do.Roles = append(do.Roles, p.colId())
	}

	if p.gotKeyword("cascade") {
		do.Behavior = DROP_CASCADE
	} else {
		p.gotKeyword("restrict")
	}

	return do
}

// parseReassignOwned parses REASSIGN OWNED BY role, ... TO newrole.
func (p *Parser) parseReassignOwned() *ReassignOwnedStmt {
	p.wantKeyword("reassign")
	pos := p.pos
	p.wantKeyword("owned")
	p.wantKeyword("by")

	ro := &ReassignOwnedStmt{baseStmt: baseStmt{baseNode{pos}}}
	ro.Roles = append(ro.Roles, p.colId())
	for p.gotSelf(',') {
		ro.Roles = append(ro.Roles, p.colId())
	}

	p.wantKeyword("to")
	ro.NewRole = p.colId()

	return ro
}

// parseDropFunction parses DROP FUNCTION/PROCEDURE/AGGREGATE [IF EXISTS] name (args) [CASCADE|RESTRICT].
func (p *Parser) parseDropFunction() *RemoveFuncStmt {
	var objType ObjectType
	switch {
	case p.gotKeyword("function"):
		objType = OBJECT_FUNCTION
	case p.gotKeyword("procedure"):
		objType = OBJECT_PROCEDURE
	case p.gotKeyword("aggregate"):
		objType = OBJECT_AGGREGATE
	}
	pos := p.pos

	rf := &RemoveFuncStmt{baseStmt: baseStmt{baseNode{pos}}, ObjType: objType}

	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("exists")
		rf.MissingOk = true
	}

	rf.Funcname = p.parseQualifiedName()

	// Optional argument types
	if p.gotSelf('(') {
		if p.tok != Token(')') {
			rf.Funcargs = append(rf.Funcargs, p.parseTypeName())
			for p.gotSelf(',') {
				rf.Funcargs = append(rf.Funcargs, p.parseTypeName())
			}
		}
		p.wantSelf(')')
	}

	if p.gotKeyword("cascade") {
		rf.Behavior = DROP_CASCADE
	} else {
		p.gotKeyword("restrict")
	}

	return rf
}
