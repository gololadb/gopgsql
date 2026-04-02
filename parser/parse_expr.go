package parser

import (
	"strconv"
	"strings"
)

// PostgreSQL operator precedence levels (low to high).
// Mapped from gram.y %left/%right declarations.
const (
	precNone    = 0
	precOr      = 1
	precAnd     = 2
	precNot     = 3  // right-assoc, handled specially
	precIs      = 4  // IS, ISNULL, NOTNULL
	precCmp     = 5  // < > = <= >= <>
	precBetween = 6  // BETWEEN, IN, LIKE, ILIKE, SIMILAR
	precOp      = 7  // Op, user-defined operators
	precAddSub  = 8  // + -
	precMulDiv  = 9  // * / %
	precExp     = 10 // ^
	precAt      = 11 // AT
	precCollate = 12 // COLLATE
	precUnary   = 13 // unary minus
	precSub     = 14 // [] subscript
	precParen   = 15 // ()
	precCast    = 16 // ::
	precDot     = 17 // .
)

// parseExpr is the entry point for expression parsing (equivalent to a_expr).
func (p *Parser) parseExpr() Expr {
	return p.parseExprPrec(precNone)
}

// parseExprPrec implements precedence climbing for SQL expressions.
func (p *Parser) parseExprPrec(minPrec int) Expr {
	left := p.parsePrimaryExpr()
	return p.parseExprSuffix(left, minPrec)
}

// parseExprSuffix handles binary operators, postfix operators, and
// special forms (IS NULL, BETWEEN, IN, LIKE, etc.) using precedence climbing.
func (p *Parser) parseExprSuffix(left Expr, minPrec int) Expr {
	for {
		// Typecast :: (highest binary precedence)
		if p.tok == TYPECAST && minPrec <= precCast {
			p.next()
			tn := p.parseTypeName()
			left = &TypeCast{
				baseExpr: baseExpr{baseNode{left.Pos()}},
				Arg:      left,
				TypeName: tn,
			}
			continue
		}

		// COLLATE
		if p.isKeyword("collate") && minPrec <= precCollate {
			p.next()
			names := p.parseQualifiedName()
			left = &CollateClause{
				baseExpr: baseExpr{baseNode{left.Pos()}},
				Arg:      left,
				Collname: names,
			}
			continue
		}

		// AT TIME ZONE / AT LOCAL
		if p.isKeyword("at") && minPrec <= precAt {
			pos := p.pos
			p.next()
			if p.isKeyword("time") {
				p.next()
				p.wantKeyword("zone")
				right := p.parseExprPrec(precAt + 1)
				left = &A_Expr{
					baseExpr: baseExpr{baseNode{pos}},
					Kind:     AEXPR_OP,
					Name:     []string{"timezone"},
					Lexpr:    left,
					Rexpr:    right,
				}
				continue
			}
			if p.isKeyword("local") {
				p.next()
				left = &A_Expr{
					baseExpr: baseExpr{baseNode{pos}},
					Kind:     AEXPR_OP,
					Name:     []string{"timezone"},
					Lexpr:    left,
					Rexpr:    &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValStr, Str: "local"}},
				}
				continue
			}
			p.syntaxError("expected TIME ZONE or LOCAL after AT")
			continue
		}

		// ^ (exponentiation)
		if p.tok == Token('^') && minPrec <= precExp {
			p.next()
			right := p.parseExprPrec(precExp + 1)
			left = &A_Expr{
				baseExpr: baseExpr{baseNode{left.Pos()}},
				Kind:     AEXPR_OP,
				Name:     []string{"^"},
				Lexpr:    left,
				Rexpr:    right,
			}
			continue
		}

		// * / %
		if minPrec <= precMulDiv {
			if p.tok == Token('*') || p.tok == Token('/') || p.tok == Token('%') {
				op := string(rune(p.tok))
				p.next()
				right := p.parseExprPrec(precMulDiv + 1)
				left = &A_Expr{
					baseExpr: baseExpr{baseNode{left.Pos()}},
					Kind:     AEXPR_OP,
					Name:     []string{op},
					Lexpr:    left,
					Rexpr:    right,
				}
				continue
			}
		}

		// + -
		if minPrec <= precAddSub {
			if p.tok == Token('+') || p.tok == Token('-') {
				op := string(rune(p.tok))
				p.next()
				right := p.parseExprPrec(precAddSub + 1)
				left = &A_Expr{
					baseExpr: baseExpr{baseNode{left.Pos()}},
					Kind:     AEXPR_OP,
					Name:     []string{op},
					Lexpr:    left,
					Rexpr:    right,
				}
				continue
			}
		}

		// | as binary operator (same precedence as Op in PG)
		if p.tok == Token('|') && minPrec <= precOp {
			p.next()
			right := p.parseExprPrec(precOp + 1)
			left = &A_Expr{
				baseExpr: baseExpr{baseNode{left.Pos()}},
				Kind:     AEXPR_OP,
				Name:     []string{"|"},
				Lexpr:    left,
				Rexpr:    right,
			}
			continue
		}

		// OPERATOR(schema.op) qualified operator syntax
		if p.isKeyword("operator") && minPrec <= precOp {
			p.next()
			opName := p.parseQualOp()
			// Check for ANY/ALL/SOME
			if p.isAnyKeyword("any", "all", "some") {
				left = p.parseSubqueryOp(left, strings.Join(opName, "."))
				continue
			}
			right := p.parseExprPrec(precOp + 1)
			left = &A_Expr{
				baseExpr: baseExpr{baseNode{left.Pos()}},
				Kind:     AEXPR_OP,
				Name:     opName,
				Lexpr:    left,
				Rexpr:    right,
			}
			continue
		}

		// User-defined / multi-char operators (Op token, also -> etc.)
		if p.tok == Op && minPrec <= precOp {
			op := p.lit
			p.next()
			// Check for ANY/ALL/SOME
			if p.isAnyKeyword("any", "all", "some") {
				left = p.parseSubqueryOp(left, op)
				continue
			}
			right := p.parseExprPrec(precOp + 1)
			left = &A_Expr{
				baseExpr: baseExpr{baseNode{left.Pos()}},
				Kind:     AEXPR_OP,
				Name:     []string{op},
				Lexpr:    left,
				Rexpr:    right,
			}
			continue
		}
		if p.tok == RIGHT_ARROW && minPrec <= precOp {
			p.next()
			right := p.parseExprPrec(precOp + 1)
			left = &A_Expr{
				baseExpr: baseExpr{baseNode{left.Pos()}},
				Kind:     AEXPR_OP,
				Name:     []string{"->"},
				Lexpr:    left,
				Rexpr:    right,
			}
			continue
		}

		// Comparison operators: < > = <= >= <> !=
		// Also handles: expr op ANY/ALL/SOME (subquery_or_expr)
		if minPrec <= precCmp {
			var op string
			switch p.tok {
			case Token('<'):
				op = "<"
			case Token('>'):
				op = ">"
			case Token('='):
				op = "="
			case LESS_EQUALS:
				op = "<="
			case GREATER_EQUALS:
				op = ">="
			case LESS_GREATER:
				op = "<>"
			case NOT_EQUALS:
				op = "!="
			}
			if op != "" {
				p.next()
				// Check for ANY/ALL/SOME (subquery or array expr)
				if p.isAnyKeyword("any", "all", "some") {
					left = p.parseSubqueryOp(left, op)
					continue
				}
				right := p.parseExprPrec(precCmp + 1)
				left = &A_Expr{
					baseExpr: baseExpr{baseNode{left.Pos()}},
					Kind:     AEXPR_OP,
					Name:     []string{op},
					Lexpr:    left,
					Rexpr:    right,
				}
				continue
			}
		}

		// OVERLAPS: (a, b) OVERLAPS (c, d)
		if p.isKeyword("overlaps") && minPrec <= precCmp {
			p.next()
			right := p.parseExprPrec(precCmp + 1)
			left = &A_Expr{
				baseExpr: baseExpr{baseNode{left.Pos()}},
				Kind:     AEXPR_OP,
				Name:     []string{"overlaps"},
				Lexpr:    left,
				Rexpr:    right,
			}
			continue
		}

		// IS forms (IS NULL, IS NOT NULL, IS TRUE, IS DISTINCT FROM, etc.)
		if p.isKeyword("is") && minPrec <= precIs {
			left = p.parseIsExpr(left)
			continue
		}

		// ISNULL / NOTNULL postfix
		if p.isKeyword("isnull") && minPrec <= precIs {
			p.next()
			left = &NullTest{
				baseExpr:     baseExpr{baseNode{left.Pos()}},
				Arg:          left,
				NullTestType: IS_NULL,
			}
			continue
		}
		if p.isKeyword("notnull") && minPrec <= precIs {
			p.next()
			left = &NullTest{
				baseExpr:     baseExpr{baseNode{left.Pos()}},
				Arg:          left,
				NullTestType: IS_NOT_NULL,
			}
			continue
		}

		// NOT (BETWEEN, IN, LIKE, ILIKE, SIMILAR)
		if p.isKeyword("not") && minPrec <= precBetween {
			// In column-constraint context, NOT NULL is a constraint, not an operator.
			if p.inColDefault {
				break
			}
			pos := p.pos
			p.next()
			if p.isKeyword("between") {
				left = p.parseBetween(left, pos, true)
				continue
			}
			if p.isKeyword("in") {
				left = p.parseIn(left, pos, true)
				continue
			}
			if p.isKeyword("like") {
				left = p.parseLike(left, pos, true, false)
				continue
			}
			if p.isKeyword("ilike") {
				left = p.parseLike(left, pos, true, true)
				continue
			}
			if p.isKeyword("similar") {
				left = p.parseSimilar(left, pos, true)
				continue
			}
			// Standalone NOT at this point is wrong — it's a prefix op
			p.syntaxError("unexpected NOT")
			continue
		}

		// BETWEEN
		if p.isKeyword("between") && minPrec <= precBetween {
			left = p.parseBetween(left, p.pos, false)
			continue
		}

		// IN
		if p.isKeyword("in") && minPrec <= precBetween {
			left = p.parseIn(left, p.pos, false)
			continue
		}

		// LIKE / ILIKE
		if p.isKeyword("like") && minPrec <= precBetween {
			left = p.parseLike(left, p.pos, false, false)
			continue
		}
		if p.isKeyword("ilike") && minPrec <= precBetween {
			left = p.parseLike(left, p.pos, false, true)
			continue
		}

		// SIMILAR TO
		if p.isKeyword("similar") && minPrec <= precBetween {
			left = p.parseSimilar(left, p.pos, false)
			continue
		}

		// AND
		if p.isKeyword("and") && minPrec <= precAnd {
			p.next()
			right := p.parseExprPrec(precAnd + 1)
			left = &BoolExpr{
				baseExpr: baseExpr{baseNode{left.Pos()}},
				Op:       AND_EXPR,
				Args:     []Expr{left, right},
			}
			continue
		}

		// OR
		if p.isKeyword("or") && minPrec <= precOr {
			p.next()
			right := p.parseExprPrec(precOr + 1)
			left = &BoolExpr{
				baseExpr: baseExpr{baseNode{left.Pos()}},
				Op:       OR_EXPR,
				Args:     []Expr{left, right},
			}
			continue
		}

		// Array subscript [...]
		if p.tok == Token('[') && minPrec <= precSub {
			left = p.parseSubscript(left)
			continue
		}

		break
	}
	return left
}

// --- IS expressions ---

func (p *Parser) parseIsExpr(left Expr) Expr {
	p.next() // consume IS
	not := p.gotKeyword("not")

	if p.isKeyword("null") {
		p.next()
		tt := IS_NULL
		if not {
			tt = IS_NOT_NULL
		}
		return &NullTest{
			baseExpr:     baseExpr{baseNode{left.Pos()}},
			Arg:          left,
			NullTestType: tt,
		}
	}

	if p.isKeyword("true") {
		p.next()
		tt := IS_TRUE
		if not {
			tt = IS_NOT_TRUE
		}
		return &BooleanTest{
			baseExpr:     baseExpr{baseNode{left.Pos()}},
			Arg:          left,
			BooltestType: tt,
		}
	}

	if p.isKeyword("false") {
		p.next()
		tt := IS_FALSE
		if not {
			tt = IS_NOT_FALSE
		}
		return &BooleanTest{
			baseExpr:     baseExpr{baseNode{left.Pos()}},
			Arg:          left,
			BooltestType: tt,
		}
	}

	if p.isKeyword("unknown") {
		p.next()
		tt := IS_UNKNOWN
		if not {
			tt = IS_NOT_UNKNOWN
		}
		return &BooleanTest{
			baseExpr:     baseExpr{baseNode{left.Pos()}},
			Arg:          left,
			BooltestType: tt,
		}
	}

	if p.isKeyword("distinct") {
		p.next()
		p.wantKeyword("from")
		right := p.parseExprPrec(precIs + 1)
		kind := AEXPR_DISTINCT
		if not {
			kind = AEXPR_NOT_DISTINCT
		}
		return &A_Expr{
			baseExpr: baseExpr{baseNode{left.Pos()}},
			Kind:     kind,
			Name:     []string{"="},
			Lexpr:    left,
			Rexpr:    right,
		}
	}

	// IS [NOT] DOCUMENT (XML)
	if p.isKeyword("document") {
		p.next()
		opName := "is_document"
		if not {
			opName = "is_not_document"
		}
		return &A_Expr{
			baseExpr: baseExpr{baseNode{left.Pos()}},
			Kind:     AEXPR_OP,
			Name:     []string{opName},
			Lexpr:    left,
		}
	}

	// IS [NOT] [form] NORMALIZED
	if p.isKeyword("normalized") || p.isAnyKeyword("nfc", "nfd", "nfkc", "nfkd") {
		form := "NFC" // default
		if p.isAnyKeyword("nfc", "nfd", "nfkc", "nfkd") {
			form = strings.ToUpper(p.lit)
			p.next()
		}
		p.wantKeyword("normalized")
		opName := "is_normalized"
		if not {
			opName = "is_not_normalized"
		}
		return &A_Expr{
			baseExpr: baseExpr{baseNode{left.Pos()}},
			Kind:     AEXPR_OP,
			Name:     []string{opName},
			Lexpr:    left,
			Rexpr:    &A_Const{baseExpr: baseExpr{baseNode{left.Pos()}}, Val: Value{Type: ValStr, Str: form}},
		}
	}

	// IS [NOT] JSON [VALUE|ARRAY|OBJECT|SCALAR] [WITH|WITHOUT UNIQUE KEYS]
	if p.isKeyword("json") {
		return p.parseIsJson(left, not)
	}

	p.syntaxError("expected NULL, TRUE, FALSE, UNKNOWN, DISTINCT, DOCUMENT, NORMALIZED, or JSON after IS")
	return left
}

// --- BETWEEN ---

func (p *Parser) parseBetween(left Expr, pos int, negated bool) Expr {
	p.next() // consume BETWEEN
	symmetric := p.gotKeyword("symmetric")

	low := p.parseBExpr()
	p.wantKeyword("and")
	high := p.parseExprPrec(precBetween + 1)

	var kind A_Expr_Kind
	switch {
	case !negated && !symmetric:
		kind = AEXPR_BETWEEN
	case negated && !symmetric:
		kind = AEXPR_NOT_BETWEEN
	case !negated && symmetric:
		kind = AEXPR_BETWEEN_SYM
	case negated && symmetric:
		kind = AEXPR_NOT_BETWEEN_SYM
	}

	return &A_Expr{
		baseExpr: baseExpr{baseNode{pos}},
		Kind:     kind,
		Name:     []string{"BETWEEN"},
		Lexpr:    left,
		Rexpr:    &ExprList{baseExpr: baseExpr{baseNode{pos}}, Items: []Expr{low, high}},
	}
}

// parseBExpr parses a restricted expression (b_expr in gram.y) that doesn't
// allow AND/OR/NOT/IS/BETWEEN/IN/LIKE at the top level. Used in BETWEEN bounds.
func (p *Parser) parseBExpr() Expr {
	return p.parseBExprPrec(precNone)
}

func (p *Parser) parseBExprPrec(minPrec int) Expr {
	left := p.parsePrimaryExpr()
	for {
		// Typecast
		if p.tok == TYPECAST && minPrec <= precCast {
			p.next()
			tn := p.parseTypeName()
			left = &TypeCast{
				baseExpr: baseExpr{baseNode{left.Pos()}},
				Arg:      left,
				TypeName: tn,
			}
			continue
		}
		// ^
		if p.tok == Token('^') && minPrec <= precExp {
			p.next()
			right := p.parseBExprPrec(precExp + 1)
			left = &A_Expr{baseExpr: baseExpr{baseNode{left.Pos()}}, Kind: AEXPR_OP, Name: []string{"^"}, Lexpr: left, Rexpr: right}
			continue
		}
		// * / %
		if minPrec <= precMulDiv {
			if p.tok == Token('*') || p.tok == Token('/') || p.tok == Token('%') {
				op := string(rune(p.tok))
				p.next()
				right := p.parseBExprPrec(precMulDiv + 1)
				left = &A_Expr{baseExpr: baseExpr{baseNode{left.Pos()}}, Kind: AEXPR_OP, Name: []string{op}, Lexpr: left, Rexpr: right}
				continue
			}
		}
		// + -
		if minPrec <= precAddSub {
			if p.tok == Token('+') || p.tok == Token('-') {
				op := string(rune(p.tok))
				p.next()
				right := p.parseBExprPrec(precAddSub + 1)
				left = &A_Expr{baseExpr: baseExpr{baseNode{left.Pos()}}, Kind: AEXPR_OP, Name: []string{op}, Lexpr: left, Rexpr: right}
				continue
			}
		}
		// Op
		if p.tok == Op && minPrec <= precOp {
			op := p.lit
			p.next()
			right := p.parseBExprPrec(precOp + 1)
			left = &A_Expr{baseExpr: baseExpr{baseNode{left.Pos()}}, Kind: AEXPR_OP, Name: []string{op}, Lexpr: left, Rexpr: right}
			continue
		}
		// Comparison
		if minPrec <= precCmp {
			var op string
			switch p.tok {
			case Token('<'):
				op = "<"
			case Token('>'):
				op = ">"
			case Token('='):
				op = "="
			case LESS_EQUALS:
				op = "<="
			case GREATER_EQUALS:
				op = ">="
			case LESS_GREATER:
				op = "<>"
			case NOT_EQUALS:
				op = "!="
			}
			if op != "" {
				p.next()
				right := p.parseBExprPrec(precCmp + 1)
				left = &A_Expr{baseExpr: baseExpr{baseNode{left.Pos()}}, Kind: AEXPR_OP, Name: []string{op}, Lexpr: left, Rexpr: right}
				continue
			}
		}
		break
	}
	return left
}

// --- IN ---

func (p *Parser) parseIn(left Expr, pos int, negated bool) Expr {
	p.next() // consume IN
	p.wantSelf('(')

	kind := AEXPR_IN
	if negated {
		kind = AEXPR_IN // we negate via wrapping in NOT later
	}

	// Check for subquery
	if p.isKeyword("select") || p.isKeyword("values") || p.isKeyword("with") || p.tok == Token('(') {
		// Could be a subquery
		sub := p.parseSelectStmt()
		p.wantSelf(')')
		linkType := ANY_SUBLINK
		if negated {
			linkType = ALL_SUBLINK
		}
		return &SubLink{
			baseExpr:    baseExpr{baseNode{pos}},
			SubLinkType: linkType,
			Testexpr:    left,
			OperName:    []string{"="},
			Subselect:   sub,
		}
	}

	// Expression list
	exprs := p.parseExprList()
	p.wantSelf(')')

	rhs := &ExprList{baseExpr: baseExpr{baseNode{pos}}, Items: exprs}
	result := Expr(&A_Expr{
		baseExpr: baseExpr{baseNode{pos}},
		Kind:     kind,
		Name:     []string{"="},
		Lexpr:    left,
		Rexpr:    rhs,
	})
	if negated {
		result = &BoolExpr{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       NOT_EXPR,
			Args:     []Expr{result},
		}
	}
	return result
}

// --- LIKE / ILIKE ---

func (p *Parser) parseLike(left Expr, pos int, negated, icase bool) Expr {
	p.next() // consume LIKE/ILIKE
	right := p.parseExprPrec(precBetween + 1)

	kind := AEXPR_LIKE
	if icase {
		kind = AEXPR_ILIKE
	}
	opName := "~~"
	if icase {
		opName = "~~*"
	}
	if negated {
		opName = "!~~"
		if icase {
			opName = "!~~*"
		}
	}

	// Optional ESCAPE clause
	if p.gotKeyword("escape") {
		esc := p.parseExprPrec(precBetween + 1)
		// With ESCAPE, wrap pattern and escape in a list
		return &A_Expr{
			baseExpr: baseExpr{baseNode{pos}},
			Kind:     kind,
			Name:     []string{opName},
			Lexpr:    left,
			Rexpr:    &ExprList{baseExpr: baseExpr{baseNode{pos}}, Items: []Expr{right, esc}},
		}
	}

	return &A_Expr{
		baseExpr: baseExpr{baseNode{pos}},
		Kind:     kind,
		Name:     []string{opName},
		Lexpr:    left,
		Rexpr:    right,
	}
}

// --- SIMILAR TO ---

func (p *Parser) parseSimilar(left Expr, pos int, negated bool) Expr {
	p.next() // consume SIMILAR
	p.wantKeyword("to")
	right := p.parseExprPrec(precBetween + 1)

	kind := AEXPR_SIMILAR

	// Optional ESCAPE clause
	if p.gotKeyword("escape") {
		esc := p.parseExprPrec(precBetween + 1)
		return &A_Expr{
			baseExpr: baseExpr{baseNode{pos}},
			Kind:     kind,
			Name:     []string{"~"},
			Lexpr:    left,
			Rexpr:    &ExprList{baseExpr: baseExpr{baseNode{pos}}, Items: []Expr{right, esc}},
		}
	}

	return &A_Expr{
		baseExpr: baseExpr{baseNode{pos}},
		Kind:     kind,
		Name:     []string{"~"},
		Lexpr:    left,
		Rexpr:    right,
	}
}

// --- Subscript ---

func (p *Parser) parseSubscript(left Expr) Expr {
	p.next() // consume [
	var idx *A_Indices
	if p.tok == Token(':') {
		// [:upper]
		p.next()
		upper := p.parseExpr()
		idx = &A_Indices{IsSlice: true, Uidx: upper}
	} else {
		lower := p.parseExpr()
		if p.gotSelf(':') {
			// [lower:upper]
			var upper Expr
			if p.tok != Token(']') {
				upper = p.parseExpr()
			}
			idx = &A_Indices{IsSlice: true, Lidx: lower, Uidx: upper}
		} else {
			idx = &A_Indices{Uidx: lower}
		}
	}
	p.wantSelf(']')
	return &A_Indirection{
		baseExpr:    baseExpr{baseNode{left.Pos()}},
		Arg:         left,
		Indirection: []Node{idx},
	}
}

// parseOptionalIndirection checks for .field or [subscript] after an expression
// and wraps it in A_Indirection if found.
func (p *Parser) parseOptionalIndirection(expr Expr) Expr {
	var indirs []Node
	for {
		if p.gotSelf('.') {
			// .field
			name := p.colLabel()
			indirs = append(indirs, &String{baseNode: baseNode{p.pos}, Str: name})
		} else if p.tok == Token('[') {
			p.next()
			var idx Node
			if p.tok == Token(':') {
				// [:upper]
				p.next()
				var upper Expr
				if p.tok != Token(']') {
					upper = p.parseExpr()
				}
				idx = &A_Indices{IsSlice: true, Uidx: upper}
			} else {
				lower := p.parseExpr()
				if p.gotSelf(':') {
					var upper Expr
					if p.tok != Token(']') {
						upper = p.parseExpr()
					}
					idx = &A_Indices{IsSlice: true, Lidx: lower, Uidx: upper}
				} else {
					idx = &A_Indices{Uidx: lower}
				}
			}
			p.wantSelf(']')
			indirs = append(indirs, idx)
		} else {
			break
		}
	}
	if len(indirs) == 0 {
		return expr
	}
	return &A_Indirection{
		baseExpr:    baseExpr{baseNode{expr.Pos()}},
		Arg:         expr,
		Indirection: indirs,
	}
}

// --- Primary expressions (c_expr) ---

func (p *Parser) parsePrimaryExpr() Expr {
	pos := p.pos

	// Unary prefix operators
	if p.tok == Token('-') {
		p.next()
		arg := p.parseExprPrec(precUnary)
		return &A_Expr{
			baseExpr: baseExpr{baseNode{pos}},
			Kind:     AEXPR_OP,
			Name:     []string{"-"},
			Rexpr:    arg,
		}
	}
	if p.tok == Token('+') {
		p.next()
		return p.parseExprPrec(precUnary)
	}

	// NOT
	if p.isKeyword("not") {
		p.next()
		arg := p.parseExprPrec(precNot)
		return &BoolExpr{
			baseExpr: baseExpr{baseNode{pos}},
			Op:       NOT_EXPR,
			Args:     []Expr{arg},
		}
	}

	// Prefix OPERATOR(schema.op) expr
	if p.isKeyword("operator") {
		p.next()
		opName := p.parseQualOp()
		arg := p.parseExprPrec(precUnary)
		return &A_Expr{
			baseExpr: baseExpr{baseNode{pos}},
			Kind:     AEXPR_OP,
			Name:     opName,
			Rexpr:    arg,
		}
	}

	// Prefix Op (user-defined prefix operator like ~, !!)
	if p.tok == Op {
		op := p.lit
		p.next()
		arg := p.parseExprPrec(precUnary)
		return &A_Expr{
			baseExpr: baseExpr{baseNode{pos}},
			Kind:     AEXPR_OP,
			Name:     []string{op},
			Rexpr:    arg,
		}
	}

	// NULL
	if p.isKeyword("null") {
		p.next()
		return &A_Const{
			baseExpr: baseExpr{baseNode{pos}},
			Val:      Value{Type: ValNull},
		}
	}

	// TRUE / FALSE
	if p.isKeyword("true") {
		p.next()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValBool, Bool: true}}
	}
	if p.isKeyword("false") {
		p.next()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValBool, Bool: false}}
	}

	// Integer literal
	if p.tok == ICONST {
		s := strings.ReplaceAll(p.lit, "_", "")
		val, _ := strconv.ParseInt(s, 0, 64)
		p.next()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: val}}
	}

	// Float literal
	if p.tok == FCONST {
		s := p.lit
		p.next()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValFloat, Str: s}}
	}

	// String literal
	if p.tok == SCONST || p.tok == USCONST {
		s := p.lit
		p.next()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValStr, Str: s}}
	}

	// Bit string literal
	if p.tok == BCONST {
		s := p.lit
		p.next()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValBitStr, Str: s}}
	}

	// Hex string literal
	if p.tok == XCONST {
		s := p.lit
		p.next()
		return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValStr, Str: s}}
	}

	// Positional parameter $N
	if p.tok == PARAM {
		num, _ := strconv.Atoi(p.lit[1:]) // skip '$'
		p.next()
		return &ParamRef{baseExpr: baseExpr{baseNode{pos}}, Number: num}
	}

	// Parenthesized expression, row constructor, or subquery
	if p.tok == Token('(') {
		return p.parseParenExpr()
	}

	// CASE expression
	if p.isKeyword("case") {
		return p.parseCaseExpr()
	}

	// EXISTS subquery
	if p.isKeyword("exists") {
		p.next()
		p.wantSelf('(')
		sub := p.parseSelectStmt()
		p.wantSelf(')')
		return &SubLink{
			baseExpr:    baseExpr{baseNode{pos}},
			SubLinkType: EXISTS_SUBLINK,
			Subselect:   sub,
		}
	}

	// ARRAY
	if p.isKeyword("array") {
		return p.parseArrayExpr()
	}

	// ROW
	if p.isKeyword("row") {
		p.next()
		p.wantSelf('(')
		var args []Expr
		if p.tok != Token(')') {
			args = p.parseExprList()
		}
		p.wantSelf(')')
		return &RowExpr{baseExpr: baseExpr{baseNode{pos}}, Args: args}
	}

	// COALESCE
	if p.isKeyword("coalesce") {
		p.next()
		p.wantSelf('(')
		args := p.parseExprList()
		p.wantSelf(')')
		return &CoalesceExpr{baseExpr: baseExpr{baseNode{pos}}, Args: args}
	}

	// NULLIF
	if p.isKeyword("nullif") {
		p.next()
		p.wantSelf('(')
		a := p.parseExpr()
		p.wantSelf(',')
		b := p.parseExpr()
		p.wantSelf(')')
		return &NullIfExpr{baseExpr: baseExpr{baseNode{pos}}, Args: []Expr{a, b}}
	}

	// GREATEST / LEAST
	if p.isKeyword("greatest") {
		p.next()
		p.wantSelf('(')
		args := p.parseExprList()
		p.wantSelf(')')
		return &MinMaxExpr{baseExpr: baseExpr{baseNode{pos}}, Op: IS_GREATEST, Args: args}
	}
	if p.isKeyword("least") {
		p.next()
		p.wantSelf('(')
		args := p.parseExprList()
		p.wantSelf(')')
		return &MinMaxExpr{baseExpr: baseExpr{baseNode{pos}}, Op: IS_LEAST, Args: args}
	}

	// CAST(expr AS type)
	if p.isKeyword("cast") {
		p.next()
		p.wantSelf('(')
		arg := p.parseExpr()
		p.wantKeyword("as")
		tn := p.parseTypeName()
		p.wantSelf(')')
		return &TypeCast{
			baseExpr: baseExpr{baseNode{pos}},
			Arg:      arg,
			TypeName: tn,
		}
	}

	// DEFAULT (in INSERT/UPDATE value contexts)
	if p.isKeyword("default") {
		p.next()
		return &SetToDefault{baseExpr: baseExpr{baseNode{pos}}}
	}

	// SQL value functions (parameterless keyword expressions)
	if svf := p.trySQLValueFunction(pos); svf != nil {
		return svf
	}

	// GROUPING(expr, ...)
	if p.isKeyword("grouping") {
		p.next()
		p.wantSelf('(')
		args := p.parseExprList()
		p.wantSelf(')')
		return &GroupingFunc{baseExpr: baseExpr{baseNode{pos}}, Args: args}
	}

	// MERGE_ACTION()
	if p.isKeyword("merge_action") {
		p.next()
		p.wantSelf('(')
		p.wantSelf(')')
		return &MergeActionExpr{baseExpr: baseExpr{baseNode{pos}}}
	}

	// EXTRACT(field FROM expr)
	if p.isKeyword("extract") {
		return p.parseExtract(pos)
	}

	// POSITION(expr IN expr)
	if p.isKeyword("position") {
		return p.parsePosition(pos)
	}

	// SUBSTRING(expr FROM expr FOR expr) or SUBSTRING(expr, expr, expr)
	if p.isKeyword("substring") {
		return p.parseSubstring(pos)
	}

	// OVERLAY(expr PLACING expr FROM expr [FOR expr])
	if p.isKeyword("overlay") {
		return p.parseOverlay(pos)
	}

	// TRIM([LEADING|TRAILING|BOTH] [expr FROM] expr)
	if p.isKeyword("trim") {
		return p.parseTrim(pos)
	}

	// TREAT(expr AS type)
	if p.isKeyword("treat") {
		return p.parseTreat(pos)
	}

	// NORMALIZE(expr [, form])
	if p.isKeyword("normalize") {
		return p.parseNormalize(pos)
	}

	// COLLATION FOR (expr)
	if p.isKeyword("collation") {
		return p.parseCollationFor(pos)
	}

	// XML functions
	if p.isKeyword("xmlconcat") {
		p.next()
		return p.parseXmlConcat()
	}
	if p.isKeyword("xmlelement") {
		p.next()
		return p.parseXmlElement()
	}
	if p.isKeyword("xmlforest") {
		p.next()
		return p.parseXmlForest()
	}
	if p.isKeyword("xmlparse") {
		p.next()
		return p.parseXmlParse()
	}
	if p.isKeyword("xmlpi") {
		p.next()
		return p.parseXmlPi()
	}
	if p.isKeyword("xmlroot") {
		p.next()
		return p.parseXmlRoot()
	}
	if p.isKeyword("xmlserialize") {
		p.next()
		return p.parseXmlSerialize()
	}
	if p.isKeyword("xmlexists") {
		p.next()
		return p.parseXmlExists()
	}

	// JSON constructor functions
	if p.isKeyword("json_object") {
		p.next()
		return p.parseJsonObject()
	}
	if p.isKeyword("json_array") {
		p.next()
		return p.parseJsonArray()
	}
	if p.isKeyword("json_query") {
		p.next()
		return p.parseJsonQuery()
	}
	if p.isKeyword("json_value") {
		p.next()
		return p.parseJsonValue()
	}
	if p.isKeyword("json_exists") {
		p.next()
		return p.parseJsonExistsFunc()
	}
	if p.isKeyword("json_scalar") {
		p.next()
		return p.parseJsonScalar()
	}
	if p.isKeyword("json_serialize") {
		p.next()
		return p.parseJsonSerialize()
	}
	if p.isKeyword("json_objectagg") {
		p.next()
		return p.parseJsonObjectAgg()
	}
	if p.isKeyword("json_arrayagg") {
		p.next()
		return p.parseJsonArrayAgg()
	}

	// INTERVAL 'string' [field qualifier] or INTERVAL (p) 'string'
	if p.isKeyword("interval") {
		return p.parseIntervalLiteral(pos)
	}

	// Typed literal: ConstTypename Sconst (e.g. DATE '2024-01-01', BIT '101')
	if tn := p.tryConstTypename(); tn != nil {
		if p.tok == SCONST {
			s := p.lit
			p.next()
			return &TypeCast{
				baseExpr: baseExpr{baseNode{pos}},
				Arg:      &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValStr, Str: s}},
				TypeName: tn,
			}
		}
		// Not followed by a string literal — this was a false start.
		// This shouldn't happen in practice since tryConstTypename only
		// matches type keywords that can't be column names.
		p.syntaxError("expected string literal after type name")
	}

	// Identifier — could be column ref, function call, or type name
	if p.tok == IDENT || p.tok == KEYWORD || p.tok == UIDENT {
		return p.parseColumnRefOrFunc()
	}

	p.syntaxError("expected expression")
	p.next()
	return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValNull}}
}

// parseParenExpr handles '(' ... ')' which could be:
// - parenthesized expression: (expr)
// - row constructor: (expr, expr, ...)
// - scalar subquery: (SELECT ...)
func (p *Parser) parseParenExpr() Expr {
	pos := p.pos
	p.next() // consume (

	// Subquery
	if p.isKeyword("select") || p.isKeyword("values") || p.isKeyword("with") || p.isKeyword("table") {
		sub := p.parseSelectStmt()
		p.wantSelf(')')
		var result Expr = &SubLink{
			baseExpr:    baseExpr{baseNode{pos}},
			SubLinkType: EXPR_SUBLINK,
			Subselect:   sub,
		}
		// Indirection: (SELECT ...).field or (SELECT ...)[n]
		result = p.parseOptionalIndirection(result)
		return result
	}

	first := p.parseExpr()
	if p.gotSelf(',') {
		// Row constructor
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

// parseCaseExpr parses CASE [expr] WHEN ... THEN ... [ELSE ...] END
func (p *Parser) parseCaseExpr() Expr {
	pos := p.pos
	p.next() // consume CASE

	var arg Expr
	// Simple CASE: CASE expr WHEN ...
	if !p.isKeyword("when") {
		arg = p.parseExpr()
	}

	var whens []*CaseWhen
	for p.gotKeyword("when") {
		cond := p.parseExpr()
		p.wantKeyword("then")
		result := p.parseExpr()
		whens = append(whens, &CaseWhen{
			baseExpr: baseExpr{baseNode{cond.Pos()}},
			Expr:     cond,
			Result:   result,
		})
	}

	var defresult Expr
	if p.gotKeyword("else") {
		defresult = p.parseExpr()
	}

	p.wantKeyword("end")

	return &CaseExpr{
		baseExpr:  baseExpr{baseNode{pos}},
		Arg:       arg,
		Args:      whens,
		Defresult: defresult,
	}
}

// parseArrayExpr parses ARRAY[...] or ARRAY(subquery)
func (p *Parser) parseArrayExpr() Expr {
	pos := p.pos
	p.next() // consume ARRAY

	if p.tok == Token('[') {
		p.next()
		var elems []Expr
		if p.tok != Token(']') {
			elems = p.parseExprList()
		}
		p.wantSelf(']')
		return &A_ArrayExpr{baseExpr: baseExpr{baseNode{pos}}, Elements: elems}
	}

	if p.tok == Token('(') {
		p.next()
		sub := p.parseSelectStmt()
		p.wantSelf(')')
		return &SubLink{
			baseExpr:    baseExpr{baseNode{pos}},
			SubLinkType: ARRAY_SUBLINK,
			Subselect:   sub,
		}
	}

	p.syntaxError("expected '[' or '(' after ARRAY")
	return &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValNull}}
}

// parseColumnRefOrFunc parses an identifier that could be a column reference
// or a function call (possibly qualified).
func (p *Parser) parseColumnRefOrFunc() Expr {
	pos := p.pos
	name := p.colLabel()

	// Collect qualified name parts: name.name.name...
	var parts []string
	parts = append(parts, name)
	for p.tok == Token('.') && p.peekIsIdentOrStar() {
		p.next() // consume .
		if p.tok == Token('*') {
			p.next()
			// name.* — column ref with star
			fields := make([]Node, len(parts)+1)
			for i, s := range parts {
				fields[i] = &String{baseNode: baseNode{pos}, Str: s}
			}
			fields[len(parts)] = &A_Star{baseNode: baseNode{pos}}
			return &ColumnRef{baseExpr: baseExpr{baseNode{pos}}, Fields: fields}
		}
		parts = append(parts, p.colLabel())
	}

	// Function call: name(...)
	if p.tok == Token('(') {
		// Could be func_name '(' func_arg_list ')' Sconst — typed literal with modifiers.
		// e.g. numeric(10,2) '123.45'
		// For now, parse as function call; the func_name(...) Sconst form is rare
		// and can be handled later if needed.
		return p.parseFuncCall(parts, pos)
	}

	// Typed literal: func_name Sconst (e.g. DATE '2024-01-01', mytype 'value')
	if p.tok == SCONST {
		s := p.lit
		p.next()
		tn := &TypeName{
			baseNode: baseNode{pos},
			Names:    parts,
		}
		return &TypeCast{
			baseExpr: baseExpr{baseNode{pos}},
			Arg:      &A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValStr, Str: s}},
			TypeName: tn,
		}
	}

	// Plain column reference
	fields := make([]Node, len(parts))
	for i, s := range parts {
		fields[i] = &String{baseNode: baseNode{pos}, Str: s}
	}
	return &ColumnRef{baseExpr: baseExpr{baseNode{pos}}, Fields: fields}
}

// peekIsIdentOrStar returns true if the token after '.' is an identifier,
// keyword, or '*'. This prevents consuming '.' that belongs to DOT_DOT.
func (p *Parser) peekIsIdentOrStar() bool {
	// We've already seen '.'. Check what follows.
	// We can't truly peek without consuming, but we know that if the next
	// token after '.' is an ident/keyword/star, it's a qualified name.
	// The scanner has already consumed '.', so p.tok is whatever follows.
	// Actually, we haven't consumed '.' yet — we check p.tok == Token('.')
	// in the caller. So we need to look at what would come after '.'.
	// Since we can't peek, we'll just check common cases.
	// For now, always return true and let the parser handle it.
	return true
}

// parseFuncCall parses a function call: name(args...) with optional
// DISTINCT, ORDER BY, FILTER, WITHIN GROUP, null treatment, OVER.
func (p *Parser) parseFuncCall(name []string, pos int) Expr {
	p.next() // consume (

	fc := &FuncCall{
		baseExpr: baseExpr{baseNode{pos}},
		Funcname: name,
	}

	// count(*)
	if p.tok == Token('*') {
		p.next()
		fc.AggStar = true
		p.wantSelf(')')
		return p.parseFuncSuffix(fc)
	}

	// Empty args
	if p.tok == Token(')') {
		p.next()
		return p.parseFuncSuffix(fc)
	}

	// DISTINCT / ALL
	if p.isKeyword("all") {
		p.next()
	} else if p.gotKeyword("distinct") {
		fc.AggDistinct = true
	}

	// VARIADIC as first (or only) keyword before args
	if p.gotKeyword("variadic") {
		fc.FuncVariadic = true
		fc.Args = append(fc.Args, p.parseExpr())
	} else {
		// Argument list — each arg may be named (name := expr or name => expr)
		fc.Args = append(fc.Args, p.parseFuncArgExpr())
		for p.gotSelf(',') {
			if p.gotKeyword("variadic") {
				fc.FuncVariadic = true
				fc.Args = append(fc.Args, p.parseExpr())
				break
			}
			fc.Args = append(fc.Args, p.parseFuncArgExpr())
		}
	}

	// ORDER BY inside aggregate
	if p.isKeyword("order") {
		p.next()
		p.wantKeyword("by")
		fc.AggOrder = p.parseSortClause()
	}

	p.wantSelf(')')

	return p.parseFuncSuffix(fc)
}

// parseFuncArgExpr parses a single function argument, which may be a named
// argument: name := expr or name => expr.
func (p *Parser) parseFuncArgExpr() Expr {
	// Parse the expression. If it turns out to be a simple identifier
	// (ColumnRef with one field) followed by := or =>, rewrite as NamedArgExpr.
	expr := p.parseExpr()

	if p.tok == COLON_EQUALS || p.tok == EQUALS_GREATER {
		// Check if expr is a simple column ref (single name)
		if cr, ok := expr.(*ColumnRef); ok && len(cr.Fields) == 1 {
			if s, ok := cr.Fields[0].(*String); ok {
				p.next() // consume := or =>
				return &NamedArgExpr{
					baseExpr:  baseExpr{baseNode{cr.Pos()}},
					Name:      s.Str,
					Arg:       p.parseExpr(),
					Argnumber: -1,
				}
			}
		}
	}
	return expr
}

// parseFuncSuffix handles post-function-call clauses:
// WITHIN GROUP, FILTER, null treatment, OVER.
func (p *Parser) parseFuncSuffix(fc *FuncCall) Expr {
	// WITHIN GROUP (ORDER BY ...)
	if p.isKeyword("within") {
		p.next()
		p.wantKeyword("group")
		p.wantSelf('(')
		p.wantKeyword("order")
		p.wantKeyword("by")
		fc.AggWithinGroup = p.parseSortClause()
		p.wantSelf(')')
	}

	// FILTER (WHERE ...)
	if p.isKeyword("filter") {
		p.next()
		p.wantSelf('(')
		p.wantKeyword("where")
		fc.AggFilter = p.parseExpr()
		p.wantSelf(')')
	}

	// Null treatment: RESPECT NULLS / IGNORE NULLS
	// Both RESPECT and IGNORE are unreserved keywords.
	if p.isKeyword("respect") {
		p.next()
		p.wantKeyword("nulls")
		fc.NullTreatment = NULL_TREATMENT_RESPECT
	} else if p.isKeyword("ignore") {
		p.next()
		p.wantKeyword("nulls")
		fc.NullTreatment = NULL_TREATMENT_IGNORE
	}

	// OVER clause
	if p.isKeyword("over") {
		p.next()
		fc.Over = p.parseWindowSpec()
	}
	return fc
}

// parseWindowSpec parses a window specification (OVER clause).
func (p *Parser) parseWindowSpec() *WindowDef {
	w := &WindowDef{baseNode: baseNode{p.pos}}

	if p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword) {
		// OVER window_name
		if p.tok != Token('(') {
			w.Refname = p.colId()
			return w
		}
	}

	p.wantSelf('(')

	// Optional existing window name
	if (p.tok == IDENT || (p.tok == KEYWORD && p.kwcat != ReservedKeyword)) &&
		!p.isKeyword("partition") && !p.isKeyword("order") &&
		!p.isKeyword("rows") && !p.isKeyword("range") && !p.isKeyword("groups") {
		w.Refname = p.colId()
	}

	// PARTITION BY
	if p.gotKeyword("partition") {
		p.wantKeyword("by")
		w.PartitionClause = p.parseExprList()
	}

	// ORDER BY
	if p.isKeyword("order") {
		p.next()
		p.wantKeyword("by")
		w.OrderClause = p.parseSortClause()
	}

	// Frame clause: RANGE|ROWS|GROUPS frame_extent [exclusion]
	p.parseFrameClause(w)

	p.wantSelf(')')
	return w
}

// parseFrameClause parses opt_frame_clause into the given WindowDef.
func (p *Parser) parseFrameClause(w *WindowDef) {
	var mode int
	switch {
	case p.gotKeyword("range"):
		mode = FRAMEOPTION_RANGE
	case p.gotKeyword("rows"):
		mode = FRAMEOPTION_ROWS
	case p.gotKeyword("groups"):
		mode = FRAMEOPTION_GROUPS
	default:
		// No frame clause — use defaults
		w.FrameOptions = FRAMEOPTION_DEFAULTS
		return
	}

	w.FrameOptions = FRAMEOPTION_NONDEFAULT | mode

	// frame_extent: BETWEEN bound AND bound | bound (implies end = CURRENT ROW)
	if p.gotKeyword("between") {
		w.FrameOptions |= FRAMEOPTION_BETWEEN
		startOpts, startOff := p.parseFrameBound()
		w.FrameOptions |= startOpts
		w.StartOffset = startOff
		p.wantKeyword("and")
		endOpts, endOff := p.parseFrameBound()
		// Shift START_ flags to END_ flags (they're 1 bit apart)
		w.FrameOptions |= endOpts << 1
		w.EndOffset = endOff
	} else {
		startOpts, startOff := p.parseFrameBound()
		w.FrameOptions |= startOpts
		w.StartOffset = startOff
		w.FrameOptions |= FRAMEOPTION_END_CURRENT_ROW
	}

	// opt_window_exclusion_clause
	if p.gotKeyword("exclude") {
		switch {
		case p.isKeyword("current"):
			p.next()
			p.wantKeyword("row")
			w.FrameOptions |= FRAMEOPTION_EXCLUDE_CURRENT_ROW
		case p.gotKeyword("group"):
			w.FrameOptions |= FRAMEOPTION_EXCLUDE_GROUP
		case p.gotKeyword("ties"):
			w.FrameOptions |= FRAMEOPTION_EXCLUDE_TIES
		case p.gotKeyword("no"):
			p.wantKeyword("others")
			// no exclusion — nothing to set
		default:
			p.syntaxError("expected CURRENT ROW, GROUP, TIES, or NO OTHERS after EXCLUDE")
		}
	}
}

// parseFrameBound parses a single frame bound and returns (START_ flags, offset expr).
// The flags use START_ positions; the caller shifts them for END_ if needed.
func (p *Parser) parseFrameBound() (int, Expr) {
	if p.gotKeyword("unbounded") {
		if p.gotKeyword("preceding") {
			return FRAMEOPTION_START_UNBOUNDED_PRECEDING, nil
		}
		p.wantKeyword("following")
		return FRAMEOPTION_START_UNBOUNDED_FOLLOWING, nil
	}

	if p.isKeyword("current") {
		p.next()
		p.wantKeyword("row")
		return FRAMEOPTION_START_CURRENT_ROW, nil
	}

	// expr PRECEDING | expr FOLLOWING
	expr := p.parseExpr()
	if p.gotKeyword("preceding") {
		return FRAMEOPTION_START_OFFSET_PRECEDING, expr
	}
	p.wantKeyword("following")
	return FRAMEOPTION_START_OFFSET_FOLLOWING, expr
}

// --- Type names ---

// parseTypeName parses a type name (used in casts, column definitions, etc.).
func (p *Parser) parseTypeName() *TypeName {
	pos := p.pos
	tn := &TypeName{baseNode: baseNode{pos}}

	if p.gotKeyword("setof") {
		tn.Setof = true
	}

	// Built-in type keywords
	switch {
	case p.isAnyKeyword("int", "integer"):
		tn.Names = []string{"pg_catalog", "int4"}
		p.next()
	case p.isKeyword("smallint"):
		tn.Names = []string{"pg_catalog", "int2"}
		p.next()
	case p.isKeyword("bigint"):
		tn.Names = []string{"pg_catalog", "int8"}
		p.next()
	case p.isKeyword("real"):
		tn.Names = []string{"pg_catalog", "float4"}
		p.next()
	case p.isAnyKeyword("float", "double"):
		if p.isKeyword("double") {
			p.next()
			p.wantKeyword("precision")
			tn.Names = []string{"pg_catalog", "float8"}
		} else {
			p.next()
			// FLOAT(n): n<=24 → float4, else float8
			if p.tok == Token('(') {
				p.next()
				tn.Typmods = p.parseExprList()
				p.wantSelf(')')
			}
			tn.Names = []string{"pg_catalog", "float8"}
		}
	case p.isKeyword("decimal"), p.isKeyword("numeric"):
		tn.Names = []string{"pg_catalog", "numeric"}
		p.next()
	case p.isKeyword("boolean"), p.isKeyword("bool"):
		tn.Names = []string{"pg_catalog", "bool"}
		p.next()
	case p.isKeyword("text"):
		tn.Names = []string{"pg_catalog", "text"}
		p.next()
	case p.isAnyKeyword("varchar", "character"):
		if p.isKeyword("character") {
			p.next()
			if p.gotKeyword("varying") {
				tn.Names = []string{"pg_catalog", "varchar"}
			} else {
				tn.Names = []string{"pg_catalog", "bpchar"}
			}
		} else {
			tn.Names = []string{"pg_catalog", "varchar"}
			p.next()
		}
	case p.isKeyword("char"):
		tn.Names = []string{"pg_catalog", "bpchar"}
		p.next()

	// BIT [(n)] / BIT VARYING [(n)]
	case p.isKeyword("bit"):
		p.next()
		if p.gotKeyword("varying") {
			tn.Names = []string{"pg_catalog", "varbit"}
		} else {
			tn.Names = []string{"pg_catalog", "bit"}
		}

	case p.isKeyword("timestamp"):
		p.next()
		// optional precision: TIMESTAMP(p)
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
	case p.isKeyword("time"):
		p.next()
		// optional precision: TIME(p)
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
	// NOTE: DATE is not a PG keyword — parsed as generic type via qualified name.

	// INTERVAL [field qualifier] or INTERVAL(p)
	case p.isKeyword("interval"):
		tn.Names = []string{"pg_catalog", "interval"}
		p.next()
		// INTERVAL '...' is handled by the caller (typed literal).
		// Here we handle INTERVAL used as a type in casts etc.
		// opt_interval field qualifiers are parsed by parseIntervalFields.
		// But first check for precision: INTERVAL(p)
		if p.tok == Token('(') {
			p.next()
			prec := p.parseInt()
			p.wantSelf(')')
			tn.Typmods = []Expr{
				&A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: IntervalFullRange}},
				&A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: prec}},
			}
		} else {
			tn.Typmods = p.parseIntervalFields(pos)
		}

	case p.isKeyword("json"):
		tn.Names = []string{"pg_catalog", "json"}
		p.next()
	case p.isKeyword("jsonb"):
		tn.Names = []string{"pg_catalog", "jsonb"}
		p.next()
	case p.isKeyword("uuid"):
		tn.Names = []string{"pg_catalog", "uuid"}
		p.next()
	case p.isKeyword("xml"):
		tn.Names = []string{"pg_catalog", "xml"}
		p.next()
	case p.isKeyword("bytea"):
		tn.Names = []string{"pg_catalog", "bytea"}
		p.next()
	default:
		// User-defined type: qualified name
		tn.Names = p.parseQualifiedName()
	}

	// Type modifiers: (precision, scale) — skip if already consumed above
	if tn.Typmods == nil && p.tok == Token('(') {
		p.next()
		tn.Typmods = p.parseExprList()
		p.wantSelf(')')
	}

	// %TYPE suffix (PL/pgSQL style)
	if p.tok == Token('%') {
		p.next()
		p.wantKeyword("type")
		tn.PctType = true
	}

	// Array suffix: [] or [][]...
	for p.tok == Token('[') {
		p.next()
		bound := -1
		if p.tok == ICONST {
			bound = int(p.parseInt())
		}
		p.wantSelf(']')
		tn.ArrayBounds = append(tn.ArrayBounds, bound)
	}

	// ARRAY keyword suffix
	if p.isKeyword("array") {
		p.next()
		if p.tok == Token('[') {
			p.next()
			bound := -1
			if p.tok == ICONST {
				bound = int(p.parseInt())
			}
			p.wantSelf(']')
			tn.ArrayBounds = append(tn.ArrayBounds, bound)
		} else {
			tn.ArrayBounds = append(tn.ArrayBounds, -1)
		}
	}

	return tn
}

// parseIntervalFields parses opt_interval: YEAR, MONTH, DAY, HOUR, MINUTE, SECOND,
// and compound forms like YEAR TO MONTH, DAY TO SECOND, etc.
// Returns nil if no interval field qualifier follows.
func (p *Parser) parseIntervalFields(pos int) []Expr {
	mkMask := func(mask int64) []Expr {
		return []Expr{&A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: mask}}}
	}

	switch {
	case p.isKeyword("year"):
		p.next()
		if p.gotKeyword("to") {
			p.wantKeyword("month")
			return mkMask(IntervalMask(IntervalFieldYear) | IntervalMask(IntervalFieldMonth))
		}
		return mkMask(IntervalMask(IntervalFieldYear))

	case p.isKeyword("month"):
		p.next()
		return mkMask(IntervalMask(IntervalFieldMonth))

	case p.isKeyword("day"):
		p.next()
		if p.gotKeyword("to") {
			switch {
			case p.gotKeyword("hour"):
				return mkMask(IntervalMask(IntervalFieldDay) | IntervalMask(IntervalFieldHour))
			case p.gotKeyword("minute"):
				return mkMask(IntervalMask(IntervalFieldDay) | IntervalMask(IntervalFieldHour) | IntervalMask(IntervalFieldMinute))
			case p.isKeyword("second"):
				return p.parseIntervalSecond(pos,
					IntervalMask(IntervalFieldDay)|IntervalMask(IntervalFieldHour)|
						IntervalMask(IntervalFieldMinute)|IntervalMask(IntervalFieldSecond))
			default:
				p.syntaxError("expected HOUR, MINUTE, or SECOND")
			}
		}
		return mkMask(IntervalMask(IntervalFieldDay))

	case p.isKeyword("hour"):
		p.next()
		if p.gotKeyword("to") {
			switch {
			case p.gotKeyword("minute"):
				return mkMask(IntervalMask(IntervalFieldHour) | IntervalMask(IntervalFieldMinute))
			case p.isKeyword("second"):
				return p.parseIntervalSecond(pos,
					IntervalMask(IntervalFieldHour)|IntervalMask(IntervalFieldMinute)|IntervalMask(IntervalFieldSecond))
			default:
				p.syntaxError("expected MINUTE or SECOND")
			}
		}
		return mkMask(IntervalMask(IntervalFieldHour))

	case p.isKeyword("minute"):
		p.next()
		if p.gotKeyword("to") {
			return p.parseIntervalSecond(pos,
				IntervalMask(IntervalFieldMinute)|IntervalMask(IntervalFieldSecond))
		}
		return mkMask(IntervalMask(IntervalFieldMinute))

	case p.isKeyword("second"):
		return p.parseIntervalSecond(pos, IntervalMask(IntervalFieldSecond))
	}

	return nil
}

// parseIntervalSecond parses SECOND or SECOND(p) and returns typmods with the given mask.
func (p *Parser) parseIntervalSecond(pos int, mask int64) []Expr {
	p.wantKeyword("second")
	if p.tok == Token('(') {
		p.next()
		prec := p.parseInt()
		p.wantSelf(')')
		return []Expr{
			&A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: mask}},
			&A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: prec}},
		}
	}
	return []Expr{&A_Const{baseExpr: baseExpr{baseNode{pos}}, Val: Value{Type: ValInt, Ival: mask}}}
}
