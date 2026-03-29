package parser

// parseSelectStmt parses a SELECT statement (including UNION/INTERSECT/EXCEPT,
// VALUES, and parenthesized selects).
func (p *Parser) parseSelectStmt() Stmt {
	return p.parseSelectNoParens()
}

// parseSelectNoParens parses select_no_parens from gram.y.
func (p *Parser) parseSelectNoParens() Stmt {
	sel := p.parseSimpleSelect()

	// Set operations: UNION / INTERSECT / EXCEPT
	for p.isAnyKeyword("union", "intersect", "except") {
		sel = p.parseSetOp(sel)
	}

	// ORDER BY
	if p.isKeyword("order") {
		if s, ok := sel.(*SelectStmt); ok {
			p.next()
			p.wantKeyword("by")
			s.SortClause = p.parseSortClause()
		}
	}

	// LIMIT / OFFSET (in either order)
	p.parseLimitOffset(sel)

	// FOR UPDATE/SHARE
	p.parseForLocking(sel)

	return sel
}

// parseSimpleSelect parses simple_select from gram.y.
func (p *Parser) parseSimpleSelect() Stmt {
	pos := p.pos

	// Parenthesized select
	if p.tok == Token('(') {
		p.next()
		sel := p.parseSelectNoParens()
		p.wantSelf(')')
		return sel
	}

	// VALUES
	if p.isKeyword("values") {
		return p.parseValuesStmt()
	}

	// TABLE tablename (shorthand for SELECT * FROM tablename)
	if p.isKeyword("table") {
		return p.parseTableStmt()
	}

	// SELECT
	p.wantKeyword("select")
	s := &SelectStmt{baseStmt: baseStmt{baseNode{pos}}}

	// DISTINCT / ALL
	if p.gotKeyword("distinct") {
		if p.gotKeyword("on") {
			p.wantSelf('(')
			s.DistinctClause = p.parseExprList()
			p.wantSelf(')')
		} else {
			s.DistinctClause = []Expr{} // empty = plain DISTINCT
		}
	} else {
		p.gotKeyword("all") // optional, ignored
	}

	// Target list
	if p.tok != Token('*') && !p.isKeyword("from") && !p.isKeyword("into") &&
		!p.isKeyword("where") && !p.isKeyword("group") && !p.isKeyword("having") &&
		!p.isKeyword("window") && !p.isKeyword("order") && !p.isKeyword("limit") &&
		!p.isKeyword("offset") && !p.isKeyword("fetch") && !p.isKeyword("for") &&
		!p.isKeyword("union") && !p.isKeyword("intersect") && !p.isKeyword("except") &&
		p.tok != Token(';') && p.tok != Token(')') && p.tok != EOF {
		s.TargetList = p.parseTargetList()
	} else if p.tok == Token('*') {
		s.TargetList = p.parseTargetList()
	}

	// INTO
	if p.gotKeyword("into") {
		s.IntoClause = &IntoClause{
			baseNode: baseNode{p.pos},
			Rel:      p.parseRangeVar(),
		}
	}

	// FROM
	if p.gotKeyword("from") {
		s.FromClause = p.parseFromClause()
	}

	// WHERE
	if p.gotKeyword("where") {
		s.WhereClause = p.parseExpr()
	}

	// GROUP BY [ALL | DISTINCT] group_by_list
	if p.isKeyword("group") {
		p.next()
		p.wantKeyword("by")
		if p.gotKeyword("distinct") {
			s.GroupDistinct = true
		} else {
			p.gotKeyword("all") // optional
		}
		s.GroupClause = p.parseGroupByList()
	}

	// HAVING
	if p.gotKeyword("having") {
		s.HavingClause = p.parseExpr()
	}

	// WINDOW
	if p.isKeyword("window") {
		p.next()
		s.WindowClause = p.parseWindowClause()
	}

	return s
}

// parseSetOp parses UNION/INTERSECT/EXCEPT.
func (p *Parser) parseSetOp(left Stmt) Stmt {
	pos := p.pos
	var op SetOperation
	switch p.lit {
	case "union":
		op = SETOP_UNION
	case "intersect":
		op = SETOP_INTERSECT
	case "except":
		op = SETOP_EXCEPT
	}
	p.next()

	all := p.gotKeyword("all")
	if !all {
		p.gotKeyword("distinct") // optional
	}

	right := p.parseSimpleSelect()

	lsel, _ := left.(*SelectStmt)
	rsel, _ := right.(*SelectStmt)

	return &SelectStmt{
		baseStmt: baseStmt{baseNode{pos}},
		Op:       op,
		All:      all,
		Larg:     lsel,
		Rarg:     rsel,
	}
}

// parseValuesStmt parses VALUES (...), (...), ...
func (p *Parser) parseValuesStmt() Stmt {
	pos := p.pos
	p.next() // consume VALUES

	s := &SelectStmt{baseStmt: baseStmt{baseNode{pos}}}
	s.ValuesLists = append(s.ValuesLists, p.parseValuesRow())
	for p.gotSelf(',') {
		s.ValuesLists = append(s.ValuesLists, p.parseValuesRow())
	}
	return s
}

func (p *Parser) parseValuesRow() []Expr {
	p.wantSelf('(')
	exprs := p.parseExprList()
	p.wantSelf(')')
	return exprs
}

// parseTableStmt parses TABLE tablename.
func (p *Parser) parseTableStmt() Stmt {
	pos := p.pos
	p.next() // consume TABLE
	rv := p.parseRangeVar()
	return &SelectStmt{
		baseStmt: baseStmt{baseNode{pos}},
		TargetList: []*ResTarget{{
			baseNode: baseNode{pos},
			Val: &ColumnRef{
				baseExpr: baseExpr{baseNode{pos}},
				Fields:   []Node{&A_Star{baseNode: baseNode{pos}}},
			},
		}},
		FromClause: []Node{rv},
	}
}

// --- Target list ---

func (p *Parser) parseTargetList() []*ResTarget {
	var list []*ResTarget
	list = append(list, p.parseResTarget())
	for p.gotSelf(',') {
		list = append(list, p.parseResTarget())
	}
	return list
}

func (p *Parser) parseResTarget() *ResTarget {
	pos := p.pos

	// Check for * (bare star)
	if p.tok == Token('*') {
		p.next()
		return &ResTarget{
			baseNode: baseNode{pos},
			Val: &ColumnRef{
				baseExpr: baseExpr{baseNode{pos}},
				Fields:   []Node{&A_Star{baseNode: baseNode{pos}}},
			},
		}
	}

	val := p.parseExpr()
	rt := &ResTarget{baseNode: baseNode{pos}, Val: val}

	// Optional AS alias (or bare alias)
	if p.gotKeyword("as") {
		rt.Name = p.colLabel()
	} else if p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
		// Bare alias — but only if it's not a keyword that starts a clause
		if !p.isAnyKeyword("from", "where", "group", "having", "order",
			"limit", "offset", "fetch", "for", "union", "intersect",
			"except", "into", "window", "on", "join", "inner", "left",
			"right", "full", "cross", "natural", "using", "returning") {
			rt.Name = p.colLabel()
		}
	}

	return rt
}

// --- FROM clause ---

// parseGroupByList parses a comma-separated list of group_by_items.
// Each item can be an expression, ROLLUP(...), CUBE(...), GROUPING SETS(...), or ().
func (p *Parser) parseGroupByList() []Expr {
	var items []Expr
	items = append(items, p.parseGroupByItem())
	for p.gotSelf(',') {
		items = append(items, p.parseGroupByItem())
	}
	return items
}

// parseGroupByItem parses a single GROUP BY item.
func (p *Parser) parseGroupByItem() Expr {
	pos := p.pos

	// ROLLUP(expr_list)
	if p.isKeyword("rollup") {
		p.next()
		p.wantSelf('(')
		args := p.parseExprList()
		p.wantSelf(')')
		content := make([]Node, len(args))
		for i, a := range args {
			content[i] = a
		}
		return &GroupingSet{
			baseExpr: baseExpr{baseNode{pos}},
			Kind:     GROUPING_SET_ROLLUP,
			Content:  content,
		}
	}

	// CUBE(expr_list)
	if p.isKeyword("cube") {
		p.next()
		p.wantSelf('(')
		args := p.parseExprList()
		p.wantSelf(')')
		content := make([]Node, len(args))
		for i, a := range args {
			content[i] = a
		}
		return &GroupingSet{
			baseExpr: baseExpr{baseNode{pos}},
			Kind:     GROUPING_SET_CUBE,
			Content:  content,
		}
	}

	// GROUPING SETS(group_by_list)
	if p.isKeyword("grouping") {
		p.next()
		p.wantKeyword("sets")
		p.wantSelf('(')
		items := p.parseGroupByList()
		p.wantSelf(')')
		content := make([]Node, len(items))
		for i, item := range items {
			content[i] = item
		}
		return &GroupingSet{
			baseExpr: baseExpr{baseNode{pos}},
			Kind:     GROUPING_SET_SETS,
			Content:  content,
		}
	}

	// '(' — could be empty grouping set (), parenthesized expression, or
	// composite grouping key (a, b).
	if p.tok == Token('(') {
		p.next()
		if p.gotSelf(')') {
			return &GroupingSet{
				baseExpr: baseExpr{baseNode{pos}},
				Kind:     GROUPING_SET_EMPTY,
			}
		}
		// Parse first expression, then check for comma (composite key) or close.
		first := p.parseExpr()
		if p.gotSelf(',') {
			// Composite grouping key: (a, b, ...) → RowExpr
			args := []Expr{first}
			args = append(args, p.parseExpr())
			for p.gotSelf(',') {
				args = append(args, p.parseExpr())
			}
			p.wantSelf(')')
			return &RowExpr{baseExpr: baseExpr{baseNode{pos}}, Args: args}
		}
		p.wantSelf(')')
		return first
	}

	// Plain expression
	return p.parseExpr()
}

func (p *Parser) parseFromClause() []Node {
	var list []Node
	list = append(list, p.parseTableRef())
	for p.gotSelf(',') {
		list = append(list, p.parseTableRef())
	}
	return list
}

// parseTableRef parses a single table reference (with joins).
func (p *Parser) parseTableRef() Node {
	left := p.parseTablePrimary()
	return p.parseJoinSuffix(left)
}

// parseJoinSuffix handles JOIN chains.
func (p *Parser) parseJoinSuffix(left Node) Node {
	for {
		var joinType JoinType
		natural := false

		if p.gotKeyword("natural") {
			natural = true
		}

		switch {
		case p.isKeyword("cross"):
			p.next()
			p.wantKeyword("join")
			joinType = JOIN_CROSS
		case p.isKeyword("join"), p.isKeyword("inner"):
			if p.gotKeyword("inner") {
				p.wantKeyword("join")
			} else {
				p.next() // consume JOIN
			}
			joinType = JOIN_INNER
		case p.isKeyword("left"):
			p.next()
			p.gotKeyword("outer") // optional
			p.wantKeyword("join")
			joinType = JOIN_LEFT
		case p.isKeyword("right"):
			p.next()
			p.gotKeyword("outer")
			p.wantKeyword("join")
			joinType = JOIN_RIGHT
		case p.isKeyword("full"):
			p.next()
			p.gotKeyword("outer")
			p.wantKeyword("join")
			joinType = JOIN_FULL
		default:
			if natural {
				// NATURAL without a join keyword — error
				p.syntaxError("expected JOIN after NATURAL")
			}
			return left
		}

		right := p.parseTablePrimary()

		join := &JoinExpr{
			baseNode:  baseNode{left.Pos()},
			Jointype:  joinType,
			IsNatural: natural,
			Larg:      left,
			Rarg:      right,
		}

		// Join condition
		if !natural && joinType != JOIN_CROSS {
			if p.gotKeyword("on") {
				join.Quals = p.parseExpr()
			} else if p.gotKeyword("using") {
				p.wantSelf('(')
				join.UsingClause = p.parseNameList()
				p.wantSelf(')')
			} else if joinType != JOIN_CROSS {
				// Implicit cross join if no ON/USING
			}
		}

		left = join
	}
}

// parseTablePrimary parses a base table reference.
func (p *Parser) parseTablePrimary() Node {
	pos := p.pos

	// Subquery in FROM: (SELECT ...) [AS] alias
	if p.tok == Token('(') {
		p.next()
		if p.isKeyword("select") || p.isKeyword("values") || p.isKeyword("with") || p.isKeyword("table") {
			sub := p.parseSelectStmt()
			p.wantSelf(')')
			rs := &RangeSubselect{
				baseNode: baseNode{pos},
				Subquery: sub,
			}
			if p.gotKeyword("as") || p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
				p.gotKeyword("as") // optional
				rs.Alias = p.parseAlias()
			}
			return rs
		}
		// Parenthesized table ref
		inner := p.parseTableRef()
		p.wantSelf(')')
		return inner
	}

	// LATERAL
	lateral := p.gotKeyword("lateral")

	// JSON_TABLE(...) [AS alias]
	if p.isKeyword("json_table") {
		p.next()
		jt := p.parseJsonTable()
		jt.Lateral = lateral
		if p.gotKeyword("as") || p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
			p.gotKeyword("as")
			jt.Alias = p.parseAlias()
		}
		return jt
	}

	// XMLTABLE(...) [AS alias]
	if p.isKeyword("xmltable") {
		p.next()
		xt := p.parseXmlTable()
		xt.Lateral = lateral
		if p.gotKeyword("as") || p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
			p.gotKeyword("as")
			xt.Alias = p.parseAlias()
		}
		return xt
	}

	if lateral && p.tok == Token('(') {
		p.next()
		sub := p.parseSelectStmt()
		p.wantSelf(')')
		rs := &RangeSubselect{
			baseNode: baseNode{pos},
			Lateral:  true,
			Subquery: sub,
		}
		if p.gotKeyword("as") || p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
			p.gotKeyword("as")
			rs.Alias = p.parseAlias()
		}
		return rs
	}

	// Table name
	rv := p.parseRangeVar()
	if lateral {
		// LATERAL function_call — for simplicity treat as range var
	}

	// TABLESAMPLE method(args) [REPEATABLE (seed)]
	if p.isKeyword("tablesample") {
		ts := p.parseTableSample(rv)
		// Alias goes on the tablesample node — but we return it as a Node
		// For simplicity, parse alias onto the underlying RangeVar
		if p.gotKeyword("as") {
			rv.Alias = p.parseAlias()
		} else if p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
			if !p.isFromAliasStop() {
				rv.Alias = p.parseAlias()
			}
		}
		return ts
	}

	// Optional alias
	if p.gotKeyword("as") {
		rv.Alias = p.parseAlias()
	} else if p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
		if !p.isFromAliasStop() {
			rv.Alias = p.parseAlias()
		}
	}

	return rv
}

// parseRangeVar parses [schema.]tablename.
func (p *Parser) parseRangeVar() *RangeVar {
	pos := p.pos
	rv := &RangeVar{baseNode: baseNode{pos}, Inh: true}

	name := p.colId()
	if p.gotSelf('.') {
		rv.Schemaname = name
		rv.Relname = p.colId()
	} else {
		rv.Relname = name
	}
	return rv
}

// isFromAliasStop returns true if the current token is a keyword that should
// NOT be consumed as an alias in a FROM clause.
func (p *Parser) isFromAliasStop() bool {
	return p.isAnyKeyword("where", "group", "having", "order", "limit",
		"offset", "fetch", "for", "union", "intersect", "except",
		"on", "join", "inner", "left", "right", "full", "cross",
		"natural", "using", "returning", "into", "window", "set",
		"tablesample")
}

// parseTableSample parses TABLESAMPLE method(args) [REPEATABLE (seed)].
func (p *Parser) parseTableSample(rel Node) *RangeTableSample {
	p.wantKeyword("tablesample")
	pos := p.pos

	ts := &RangeTableSample{baseNode: baseNode{pos}, Relation: rel}
	ts.Method = p.colId()

	p.wantSelf('(')
	ts.Args = append(ts.Args, p.parseExpr())
	for p.gotSelf(',') {
		ts.Args = append(ts.Args, p.parseExpr())
	}
	p.wantSelf(')')

	if p.gotKeyword("repeatable") {
		p.wantSelf('(')
		ts.Repeatable = p.parseExpr()
		p.wantSelf(')')
	}

	return ts
}

// parseAlias parses alias_name [(col1, col2, ...)].
func (p *Parser) parseAlias() *Alias {
	a := &Alias{baseNode: baseNode{p.pos}}
	a.Aliasname = p.colId()
	if p.gotSelf('(') {
		a.Colnames = p.parseNameList()
		p.wantSelf(')')
	}
	return a
}

// --- ORDER BY ---

func (p *Parser) parseSortClause() []*SortBy {
	var list []*SortBy
	list = append(list, p.parseSortBy())
	for p.gotSelf(',') {
		list = append(list, p.parseSortBy())
	}
	return list
}

func (p *Parser) parseSortBy() *SortBy {
	sb := &SortBy{baseNode: baseNode{p.pos}}
	sb.Node = p.parseExpr()

	if p.gotKeyword("asc") {
		sb.SortbyDir = SORTBY_ASC
	} else if p.gotKeyword("desc") {
		sb.SortbyDir = SORTBY_DESC
	} else if p.isKeyword("using") {
		p.next()
		sb.SortbyDir = SORTBY_USING
		sb.UseOp = p.parseQualifiedName()
	}

	if p.isKeyword("nulls") {
		p.next()
		if p.gotKeyword("first") {
			sb.SortbyNulls = SORTBY_NULLS_FIRST
		} else if p.gotKeyword("last") {
			sb.SortbyNulls = SORTBY_NULLS_LAST
		} else {
			p.syntaxError("expected FIRST or LAST after NULLS")
		}
	}

	return sb
}

// --- LIMIT / OFFSET ---

func (p *Parser) parseLimitOffset(sel Stmt) {
	s, ok := sel.(*SelectStmt)
	if !ok {
		return
	}

	for p.isAnyKeyword("limit", "offset", "fetch") {
		if p.gotKeyword("limit") {
			if p.isKeyword("all") {
				p.next()
				// LIMIT ALL = no limit
			} else {
				s.LimitCount = p.parseExpr()
			}
		} else if p.gotKeyword("offset") {
			s.LimitOffset = p.parseExpr()
			// Optional ROW/ROWS after offset value
			p.gotKeyword("row")
			p.gotKeyword("rows")
		} else if p.isKeyword("fetch") {
			// FETCH FIRST/NEXT n ROW/ROWS ONLY
			p.next()
			p.gotKeyword("first")
			p.gotKeyword("next")
			if p.tok == ICONST || p.tok == FCONST {
				s.LimitCount = p.parseExpr()
			} else {
				// FETCH FIRST ROW ONLY = LIMIT 1
				s.LimitCount = &A_Const{
					baseExpr: baseExpr{baseNode{p.pos}},
					Val:      Value{Type: ValInt, Ival: 1},
				}
			}
			p.gotKeyword("row")
			p.gotKeyword("rows")
			p.gotKeyword("only")
		}
	}
}

// --- FOR UPDATE/SHARE ---

func (p *Parser) parseForLocking(sel Stmt) {
	s, ok := sel.(*SelectStmt)
	if !ok {
		return
	}

	for p.isKeyword("for") {
		p.next()
		lc := &LockingClause{baseNode: baseNode{p.pos}}

		switch {
		case p.gotKeyword("update"):
			lc.Strength = LCS_FORUPDATE
		case p.gotKeyword("share"):
			lc.Strength = LCS_FORSHARE
		case p.isKeyword("no"):
			p.next()
			p.wantKeyword("key")
			p.wantKeyword("update")
			lc.Strength = LCS_FORNOKEYUPDATE
		case p.isKeyword("key"):
			p.next()
			p.wantKeyword("share")
			lc.Strength = LCS_FORKEYSHARE
		default:
			p.syntaxError("expected UPDATE, SHARE, NO KEY UPDATE, or KEY SHARE after FOR")
		}

		if p.gotKeyword("of") {
			for {
				lc.LockedRels = append(lc.LockedRels, p.parseRangeVar())
				if !p.gotSelf(',') {
					break
				}
			}
		}

		if p.gotKeyword("nowait") {
			lc.WaitPolicy = LockWaitError
		} else if p.isKeyword("skip") {
			p.next()
			p.wantKeyword("locked")
			lc.WaitPolicy = LockWaitSkip
		}

		s.LockingClause = append(s.LockingClause, lc)
	}
}

// --- WINDOW clause ---

func (p *Parser) parseWindowClause() []*WindowDef {
	var list []*WindowDef
	for {
		w := &WindowDef{baseNode: baseNode{p.pos}}
		w.Name = p.colId()
		p.wantKeyword("as")
		spec := p.parseWindowSpec()
		spec.Name = w.Name
		list = append(list, spec)
		if !p.gotSelf(',') {
			break
		}
	}
	return list
}

// --- WITH clause ---

func (p *Parser) parseWithClause() *WithClause {
	pos := p.pos
	p.next() // consume WITH
	w := &WithClause{baseNode: baseNode{pos}}

	if p.gotKeyword("recursive") {
		w.Recursive = true
	}

	w.CTEs = append(w.CTEs, p.parseCTE())
	for p.gotSelf(',') {
		w.CTEs = append(w.CTEs, p.parseCTE())
	}
	return w
}

func (p *Parser) parseCTE() *CommonTableExpr {
	cte := &CommonTableExpr{baseNode: baseNode{p.pos}}
	cte.Ctename = p.colId()

	// Optional column name list
	if p.gotSelf('(') {
		cte.Aliascolnames = p.parseNameList()
		p.wantSelf(')')
	}

	p.wantKeyword("as")

	// Optional materialization hint
	if p.gotKeyword("materialized") {
		cte.CTEMaterialized = CTEMaterializeAlways
	} else if p.isKeyword("not") {
		p.next()
		p.wantKeyword("materialized")
		cte.CTEMaterialized = CTEMaterializeNever
	}

	p.wantSelf('(')
	cte.Ctequery = p.parseSelectStmt()
	p.wantSelf(')')

	return cte
}
