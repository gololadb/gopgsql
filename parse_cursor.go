package pgscan

// parseDeclareCursorStmt parses:
//   DECLARE name [BINARY] [ASENSITIVE|INSENSITIVE] [[NO] SCROLL] CURSOR
//     [WITH HOLD | WITHOUT HOLD] FOR query
func (p *Parser) parseDeclareCursorStmt() *DeclareCursorStmt {
	p.wantKeyword("declare")
	pos := p.pos

	dc := &DeclareCursorStmt{baseStmt: baseStmt{baseNode{pos}}}
	dc.Portalname = p.colId()

	// Optional cursor properties before CURSOR keyword
	for {
		switch {
		case p.gotKeyword("binary"):
			dc.Options |= CURSOR_OPT_BINARY
		case p.isKeyword("insensitive"):
			p.next()
			dc.Options |= CURSOR_OPT_INSENSITIVE
		case p.isKeyword("asensitive"):
			// ASENSITIVE is not a keyword in PG, it's an identifier
			if p.tok == IDENT && p.lit == "asensitive" {
				p.next()
				dc.Options |= CURSOR_OPT_ASENSITIVE
			} else {
				goto done
			}
		case p.gotKeyword("scroll"):
			dc.Options |= CURSOR_OPT_SCROLL
		case p.isKeyword("no"):
			p.next()
			p.wantKeyword("scroll")
			dc.Options |= CURSOR_OPT_NO_SCROLL
		default:
			goto done
		}
	}
done:

	p.wantKeyword("cursor")

	// WITH HOLD / WITHOUT HOLD
	if p.isKeyword("with") {
		p.next()
		p.wantKeyword("hold")
		dc.Options |= CURSOR_OPT_HOLD
	} else if p.isKeyword("without") {
		p.next()
		p.wantKeyword("hold")
		// no flag — WITHOUT HOLD is the default
	}

	p.wantKeyword("for")
	dc.Query = p.parseSelectStmt()

	return dc
}

// parseFetchStmt parses FETCH or MOVE.
//   FETCH [direction] [FROM|IN] cursor_name
//   MOVE [direction] [FROM|IN] cursor_name
func (p *Parser) parseFetchStmt() *FetchStmt {
	pos := p.pos
	isMove := p.isKeyword("move")
	p.next() // consume FETCH or MOVE

	fs := &FetchStmt{
		baseStmt: baseStmt{baseNode{pos}},
		IsMove:   isMove,
		HowMany:  1,
	}

	// Parse direction
	switch {
	case p.gotKeyword("next"):
		fs.Direction = FETCH_FORWARD
		fs.HowMany = 1
	case p.gotKeyword("prior"):
		fs.Direction = FETCH_BACKWARD
		fs.HowMany = 1
	case p.gotKeyword("first"):
		fs.Direction = FETCH_ABSOLUTE
		fs.HowMany = 1
	case p.gotKeyword("last"):
		fs.Direction = FETCH_ABSOLUTE
		fs.HowMany = -1
	case p.gotKeyword("absolute"):
		fs.Direction = FETCH_ABSOLUTE
		fs.HowMany = p.parseSignedInt()
	case p.gotKeyword("relative"):
		fs.Direction = FETCH_RELATIVE
		fs.HowMany = p.parseSignedInt()
	case p.gotKeyword("all"):
		fs.Direction = FETCH_FORWARD
		fs.HowMany = 0 // 0 means ALL
	case p.isKeyword("forward"):
		p.next()
		fs.Direction = FETCH_FORWARD
		if p.gotKeyword("all") {
			fs.HowMany = 0
		} else if p.tok == ICONST || p.tok == Token('+') || p.tok == Token('-') {
			fs.HowMany = p.parseSignedInt()
		} else {
			fs.HowMany = 1
		}
	case p.isKeyword("backward"):
		p.next()
		fs.Direction = FETCH_BACKWARD
		if p.gotKeyword("all") {
			fs.HowMany = 0
		} else if p.tok == ICONST || p.tok == Token('+') || p.tok == Token('-') {
			fs.HowMany = p.parseSignedInt()
		} else {
			fs.HowMany = 1
		}
	case p.tok == ICONST || p.tok == Token('+') || p.tok == Token('-'):
		// Numeric count: FETCH n FROM cursor
		fs.Direction = FETCH_FORWARD
		fs.HowMany = p.parseSignedInt()
	default:
		// No direction — default is NEXT
		fs.Direction = FETCH_FORWARD
		fs.HowMany = 1
	}

	// Optional FROM or IN
	if !p.gotKeyword("from") {
		p.gotKeyword("in")
	}

	fs.Portalname = p.colId()
	return fs
}

// parseSignedInt parses an optional sign followed by an integer.
func (p *Parser) parseSignedInt() int64 {
	neg := false
	if p.tok == Token('-') {
		neg = true
		p.next()
	} else if p.tok == Token('+') {
		p.next()
	}
	val := p.parseInt()
	if neg {
		return -val
	}
	return val
}

// parseClosePortalStmt parses CLOSE cursor_name or CLOSE ALL.
func (p *Parser) parseClosePortalStmt() *ClosePortalStmt {
	p.wantKeyword("close")
	pos := p.pos

	if p.gotKeyword("all") {
		return &ClosePortalStmt{baseStmt: baseStmt{baseNode{pos}}, Portalname: ""}
	}

	return &ClosePortalStmt{
		baseStmt:   baseStmt{baseNode{pos}},
		Portalname: p.colId(),
	}
}
