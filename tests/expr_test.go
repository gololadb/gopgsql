package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestNamedArgColonEquals(t *testing.T) {
	s := parseOne(t, "SELECT my_func(a := 1, b := 2)")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	fc, ok := rt.Val.(*parser.FuncCall)
	if !ok {
		t.Fatalf("expected *parser.FuncCall, got %T", rt.Val)
	}
	if len(fc.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(fc.Args))
	}
	na, ok := fc.Args[0].(*parser.NamedArgExpr)
	if !ok {
		t.Fatalf("expected *parser.NamedArgExpr, got %T", fc.Args[0])
	}
	if na.Name != "a" {
		t.Fatalf("expected name 'a', got %q", na.Name)
	}
}


func TestNamedArgEqualsGreater(t *testing.T) {
	s := parseOne(t, "SELECT my_func(a => 1, b => 'hello')")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	fc, ok := rt.Val.(*parser.FuncCall)
	if !ok {
		t.Fatalf("expected *parser.FuncCall, got %T", rt.Val)
	}
	if len(fc.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(fc.Args))
	}
	na, ok := fc.Args[1].(*parser.NamedArgExpr)
	if !ok {
		t.Fatalf("expected *parser.NamedArgExpr for arg 1, got %T", fc.Args[1])
	}
	if na.Name != "b" {
		t.Fatalf("expected name 'b', got %q", na.Name)
	}
}


func TestNamedArgMixed(t *testing.T) {
	// Positional followed by named
	s := parseOne(t, "SELECT my_func(1, name => 'test')")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	fc := rt.Val.(*parser.FuncCall)
	if len(fc.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(fc.Args))
	}
	// First arg is positional (parser.A_Const)
	if _, ok := fc.Args[0].(*parser.A_Const); !ok {
		t.Fatalf("expected *parser.A_Const for arg 0, got %T", fc.Args[0])
	}
	// Second arg is named
	if _, ok := fc.Args[1].(*parser.NamedArgExpr); !ok {
		t.Fatalf("expected *parser.NamedArgExpr for arg 1, got %T", fc.Args[1])
	}
}


func TestVariadicFunc(t *testing.T) {
	s := parseOne(t, "SELECT my_func(1, VARIADIC arr)")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	fc, ok := rt.Val.(*parser.FuncCall)
	if !ok {
		t.Fatalf("expected *parser.FuncCall, got %T", rt.Val)
	}
	if !fc.FuncVariadic {
		t.Fatal("expected FuncVariadic=true")
	}
	if len(fc.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(fc.Args))
	}
}


func TestVariadicOnly(t *testing.T) {
	s := parseOne(t, "SELECT my_func(VARIADIC ARRAY[1,2,3])")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	fc := rt.Val.(*parser.FuncCall)
	if !fc.FuncVariadic {
		t.Fatal("expected FuncVariadic=true")
	}
	if len(fc.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(fc.Args))
	}
}


func TestWithinGroup(t *testing.T) {
	s := parseOne(t, "SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY salary)")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	fc, ok := rt.Val.(*parser.FuncCall)
	if !ok {
		t.Fatalf("expected *parser.FuncCall, got %T", rt.Val)
	}
	if len(fc.AggWithinGroup) == 0 {
		t.Fatal("expected non-empty AggWithinGroup")
	}
	if fc.AggWithinGroup[0].Node == nil {
		t.Fatal("expected non-nil sort node")
	}
}


func TestWithinGroupMultiple(t *testing.T) {
	s := parseOne(t, "SELECT percentile_disc(0.5) WITHIN GROUP (ORDER BY x, y DESC)")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	fc := rt.Val.(*parser.FuncCall)
	if len(fc.AggWithinGroup) != 2 {
		t.Fatalf("expected 2 sort items, got %d", len(fc.AggWithinGroup))
	}
}


func TestNullTreatmentIgnore(t *testing.T) {
	s := parseOne(t, "SELECT first_value(x) IGNORE NULLS OVER (ORDER BY id)")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	fc, ok := rt.Val.(*parser.FuncCall)
	if !ok {
		t.Fatalf("expected *parser.FuncCall, got %T", rt.Val)
	}
	if fc.NullTreatment != parser.NULL_TREATMENT_IGNORE {
		t.Fatalf("expected parser.NULL_TREATMENT_IGNORE, got %d", fc.NullTreatment)
	}
	if fc.Over == nil {
		t.Fatal("expected non-nil Over")
	}
}


func TestNullTreatmentRespect(t *testing.T) {
	s := parseOne(t, "SELECT last_value(x) RESPECT NULLS OVER (ORDER BY id)")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	fc := rt.Val.(*parser.FuncCall)
	if fc.NullTreatment != parser.NULL_TREATMENT_RESPECT {
		t.Fatalf("expected parser.NULL_TREATMENT_RESPECT, got %d", fc.NullTreatment)
	}
}


func TestFilterWithinGroupCombo(t *testing.T) {
	// WITHIN GROUP + FILTER together
	s := parseOne(t, "SELECT mode() WITHIN GROUP (ORDER BY val) FILTER (WHERE val > 0)")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	fc := rt.Val.(*parser.FuncCall)
	if len(fc.AggWithinGroup) == 0 {
		t.Fatal("expected non-empty AggWithinGroup")
	}
	if fc.AggFilter == nil {
		t.Fatal("expected non-nil AggFilter")
	}
}


func TestMergeAction(t *testing.T) {
	// MERGE_ACTION() is valid inside MERGE WHEN clauses, but we test it as a standalone expression
	s := parseOne(t, "SELECT MERGE_ACTION()")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	_, ok := rt.Val.(*parser.MergeActionExpr)
	if !ok {
		t.Fatalf("expected *parser.MergeActionExpr, got %T", rt.Val)
	}
}


func TestRegularFuncStillWorks(t *testing.T) {
	// Ensure normal function calls still work
	tests := []string{
		"SELECT count(*)",
		"SELECT count(DISTINCT x)",
		"SELECT array_agg(x ORDER BY x)",
		"SELECT sum(x) FILTER (WHERE x > 0)",
		"SELECT row_number() OVER (ORDER BY id)",
		"SELECT my_func()",
		"SELECT my_func(1, 2, 3)",
	}
	for _, sql := range tests {
		t.Run(sql, func(t *testing.T) {
			parseOne(t, sql)
		})
	}
}


func TestExprIntLiteral(t *testing.T) {
	s := parseOne(t, "SELECT 42")
	sel := s.(*parser.SelectStmt)
	c := sel.TargetList[0].Val.(*parser.A_Const)
	if c.Val.Type != parser.ValInt || c.Val.Ival != 42 {
		t.Errorf("expected int 42, got %+v", c.Val)
	}
}


func TestExprStringLiteral(t *testing.T) {
	s := parseOne(t, "SELECT 'hello'")
	sel := s.(*parser.SelectStmt)
	c := sel.TargetList[0].Val.(*parser.A_Const)
	if c.Val.Type != parser.ValStr || c.Val.Str != "hello" {
		t.Errorf("expected string 'hello', got %+v", c.Val)
	}
}


func TestExprBinaryOp(t *testing.T) {
	s := parseOne(t, "SELECT 1 + 2")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Kind != parser.AEXPR_OP || e.Name[0] != "+" {
		t.Errorf("expected + op, got %+v", e)
	}
}


func TestExprPrecedence(t *testing.T) {
	s := parseOne(t, "SELECT 1 + 2 * 3")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Name[0] != "+" {
		t.Errorf("top-level should be +, got %s", e.Name[0])
	}
	rhs := e.Rexpr.(*parser.A_Expr)
	if rhs.Name[0] != "*" {
		t.Errorf("rhs should be *, got %s", rhs.Name[0])
	}
}


func TestExprUnaryMinus(t *testing.T) {
	s := parseOne(t, "SELECT -1")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Name[0] != "-" || e.Lexpr != nil {
		t.Errorf("expected unary minus, got %+v", e)
	}
}


func TestExprParens(t *testing.T) {
	s := parseOne(t, "SELECT (1 + 2) * 3")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Name[0] != "*" {
		t.Errorf("top-level should be *, got %s", e.Name[0])
	}
}


func TestExprBoolOps(t *testing.T) {
	s := parseOne(t, "SELECT true AND false OR true")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.BoolExpr)
	if e.Op != parser.OR_EXPR {
		t.Errorf("top-level should be OR, got %v", e.Op)
	}
}


func TestExprNot(t *testing.T) {
	s := parseOne(t, "SELECT NOT true")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.BoolExpr)
	if e.Op != parser.NOT_EXPR {
		t.Errorf("expected NOT, got %v", e.Op)
	}
}


func TestExprIsNull(t *testing.T) {
	s := parseOne(t, "SELECT x IS NULL")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.NullTest)
	if e.NullTestType != parser.IS_NULL {
		t.Errorf("expected parser.IS_NULL, got %v", e.NullTestType)
	}
}


func TestExprIsNotNull(t *testing.T) {
	s := parseOne(t, "SELECT x IS NOT NULL")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.NullTest)
	if e.NullTestType != parser.IS_NOT_NULL {
		t.Errorf("expected parser.IS_NOT_NULL, got %v", e.NullTestType)
	}
}


func TestExprBetween(t *testing.T) {
	s := parseOne(t, "SELECT x BETWEEN 1 AND 10")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Kind != parser.AEXPR_BETWEEN {
		t.Errorf("expected BETWEEN, got %v", e.Kind)
	}
}


func TestExprIn(t *testing.T) {
	s := parseOne(t, "SELECT x IN (1, 2, 3)")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Kind != parser.AEXPR_IN {
		t.Errorf("expected IN, got %v", e.Kind)
	}
}


func TestExprLike(t *testing.T) {
	s := parseOne(t, "SELECT x LIKE '%foo%'")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Kind != parser.AEXPR_LIKE {
		t.Errorf("expected LIKE, got %v", e.Kind)
	}
}


func TestExprCast(t *testing.T) {
	s := parseOne(t, "SELECT x::integer")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.TypeCast)
	if e.TypeName.Names[1] != "int4" {
		t.Errorf("expected int4, got %v", e.TypeName.Names)
	}
}


func TestExprCastFunc(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS text)")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.TypeCast)
	if e.TypeName.Names[1] != "text" {
		t.Errorf("expected text, got %v", e.TypeName.Names)
	}
}


func TestExprCase(t *testing.T) {
	s := parseOne(t, "SELECT CASE WHEN x > 0 THEN 'pos' ELSE 'neg' END")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.CaseExpr)
	if len(e.Args) != 1 {
		t.Errorf("expected 1 WHEN, got %d", len(e.Args))
	}
	if e.Defresult == nil {
		t.Error("expected ELSE clause")
	}
}


func TestExprCoalesce(t *testing.T) {
	s := parseOne(t, "SELECT COALESCE(a, b, c)")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.CoalesceExpr)
	if len(e.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(e.Args))
	}
}


func TestExprNullif(t *testing.T) {
	s := parseOne(t, "SELECT NULLIF(a, 0)")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.NullIfExpr)
	if len(e.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(e.Args))
	}
}


func TestExprFuncCall(t *testing.T) {
	s := parseOne(t, "SELECT count(*)")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if !fc.AggStar {
		t.Error("expected AggStar")
	}
	if fc.Funcname[0] != "count" {
		t.Errorf("expected count, got %v", fc.Funcname)
	}
}


func TestExprFuncDistinct(t *testing.T) {
	s := parseOne(t, "SELECT count(DISTINCT x)")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if !fc.AggDistinct {
		t.Error("expected AggDistinct")
	}
}


func TestExprExists(t *testing.T) {
	s := parseOne(t, "SELECT EXISTS (SELECT 1)")
	sel := s.(*parser.SelectStmt)
	sl := sel.TargetList[0].Val.(*parser.SubLink)
	if sl.SubLinkType != parser.EXISTS_SUBLINK {
		t.Errorf("expected EXISTS, got %v", sl.SubLinkType)
	}
}


func TestExprArray(t *testing.T) {
	s := parseOne(t, "SELECT ARRAY[1, 2, 3]")
	sel := s.(*parser.SelectStmt)
	a := sel.TargetList[0].Val.(*parser.A_ArrayExpr)
	if len(a.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(a.Elements))
	}
}


func TestExprParam(t *testing.T) {
	s := parseOne(t, "SELECT $1")
	sel := s.(*parser.SelectStmt)
	pr := sel.TargetList[0].Val.(*parser.ParamRef)
	if pr.Number != 1 {
		t.Errorf("expected $1, got $%d", pr.Number)
	}
}


func TestExprColumnRef(t *testing.T) {
	s := parseOne(t, "SELECT t.col")
	sel := s.(*parser.SelectStmt)
	cr := sel.TargetList[0].Val.(*parser.ColumnRef)
	if len(cr.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(cr.Fields))
	}
}

// --- SELECT tests ---


func TestExprIsTrue(t *testing.T) {
	s := parseOne(t, "SELECT x IS TRUE")
	sel := s.(*parser.SelectStmt)
	bt := sel.TargetList[0].Val.(*parser.BooleanTest)
	if bt.BooltestType != parser.IS_TRUE {
		t.Errorf("expected parser.IS_TRUE, got %v", bt.BooltestType)
	}
}


func TestExprIsDistinctFrom(t *testing.T) {
	s := parseOne(t, "SELECT x IS DISTINCT FROM y")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Kind != parser.AEXPR_DISTINCT {
		t.Errorf("expected DISTINCT, got %v", e.Kind)
	}
}


func TestExprNotBetween(t *testing.T) {
	s := parseOne(t, "SELECT x NOT BETWEEN 1 AND 10")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Kind != parser.AEXPR_NOT_BETWEEN {
		t.Errorf("expected NOT_BETWEEN, got %v", e.Kind)
	}
}


func TestExprNotIn(t *testing.T) {
	s := parseOne(t, "SELECT x NOT IN (1, 2)")
	sel := s.(*parser.SelectStmt)
	be := sel.TargetList[0].Val.(*parser.BoolExpr)
	if be.Op != parser.NOT_EXPR {
		t.Errorf("expected NOT wrapping IN, got %v", be.Op)
	}
}


func TestExprGreatest(t *testing.T) {
	s := parseOne(t, "SELECT GREATEST(1, 2, 3)")
	sel := s.(*parser.SelectStmt)
	mm := sel.TargetList[0].Val.(*parser.MinMaxExpr)
	if mm.Op != parser.IS_GREATEST || len(mm.Args) != 3 {
		t.Errorf("expected GREATEST with 3 args, got %v with %d", mm.Op, len(mm.Args))
	}
}


func TestExprRowConstructor(t *testing.T) {
	s := parseOne(t, "SELECT (1, 2, 3)")
	sel := s.(*parser.SelectStmt)
	re := sel.TargetList[0].Val.(*parser.RowExpr)
	if len(re.Args) != 3 {
		t.Errorf("expected 3 row elements, got %d", len(re.Args))
	}
}


func TestExprScalarSubquery(t *testing.T) {
	s := parseOne(t, "SELECT (SELECT 1)")
	sel := s.(*parser.SelectStmt)
	sl := sel.TargetList[0].Val.(*parser.SubLink)
	if sl.SubLinkType != parser.EXPR_SUBLINK {
		t.Errorf("expected parser.EXPR_SUBLINK, got %v", sl.SubLinkType)
	}
}


func TestExprInSubquery(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t WHERE x IN (SELECT y FROM s)")
	sel := s.(*parser.SelectStmt)
	sl := sel.WhereClause.(*parser.SubLink)
	if sl.SubLinkType != parser.ANY_SUBLINK {
		t.Errorf("expected parser.ANY_SUBLINK, got %v", sl.SubLinkType)
	}
}


func TestAtLocal(t *testing.T) {
	s := parseOne(t, "SELECT ts AT LOCAL")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Name[0] != "timezone" {
		t.Errorf("expected timezone op, got %v", e.Name)
	}
	rhs := e.Rexpr.(*parser.A_Const)
	if rhs.Val.Str != "local" {
		t.Errorf("expected 'local', got %q", rhs.Val.Str)
	}
}


func TestPipeOperator(t *testing.T) {
	s := parseOne(t, "SELECT 1 | 2")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Name[0] != "|" {
		t.Errorf("expected | op, got %v", e.Name)
	}
}


func TestLikeEscape(t *testing.T) {
	s := parseOne(t, "SELECT x LIKE '%a%' ESCAPE '\\'")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Kind != parser.AEXPR_LIKE {
		t.Errorf("expected LIKE, got %v", e.Kind)
	}
	// rhs should be parser.ExprList with pattern and escape
	el := e.Rexpr.(*parser.ExprList)
	if len(el.Items) != 2 {
		t.Errorf("expected 2 items (pattern + escape), got %d", len(el.Items))
	}
}


func TestIlikeEscape(t *testing.T) {
	parseOK(t, "SELECT x ILIKE '%a%' ESCAPE '\\'")
}


func TestSimilarToEscape(t *testing.T) {
	parseOK(t, "SELECT x SIMILAR TO '%a%' ESCAPE '\\'")
}


func TestIsDocument(t *testing.T) {
	s := parseOne(t, "SELECT x IS DOCUMENT")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Name[0] != "is_document" {
		t.Errorf("expected is_document, got %v", e.Name)
	}
}


func TestIsNotDocument(t *testing.T) {
	s := parseOne(t, "SELECT x IS NOT DOCUMENT")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Name[0] != "is_not_document" {
		t.Errorf("expected is_not_document, got %v", e.Name)
	}
}


func TestIsNormalized(t *testing.T) {
	s := parseOne(t, "SELECT x IS NORMALIZED")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Name[0] != "is_normalized" {
		t.Errorf("expected is_normalized, got %v", e.Name)
	}
}


func TestIsNFCNormalized(t *testing.T) {
	s := parseOne(t, "SELECT x IS NFC NORMALIZED")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	rhs := e.Rexpr.(*parser.A_Const)
	if rhs.Val.Str != "NFC" {
		t.Errorf("expected NFC, got %q", rhs.Val.Str)
	}
}


func TestIsNotNFKDNormalized(t *testing.T) {
	s := parseOne(t, "SELECT x IS NOT NFKD NORMALIZED")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Name[0] != "is_not_normalized" {
		t.Errorf("expected is_not_normalized, got %v", e.Name)
	}
}


func TestOpAnySubquery(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t WHERE x = ANY (SELECT y FROM s)")
	sel := s.(*parser.SelectStmt)
	sl := sel.WhereClause.(*parser.SubLink)
	if sl.SubLinkType != parser.ANY_SUBLINK {
		t.Errorf("expected parser.ANY_SUBLINK, got %v", sl.SubLinkType)
	}
	if sl.OperName[0] != "=" {
		t.Errorf("expected = operator, got %v", sl.OperName)
	}
}


func TestOpAllSubquery(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t WHERE x > ALL (SELECT y FROM s)")
	sel := s.(*parser.SelectStmt)
	sl := sel.WhereClause.(*parser.SubLink)
	if sl.SubLinkType != parser.ALL_SUBLINK {
		t.Errorf("expected parser.ALL_SUBLINK, got %v", sl.SubLinkType)
	}
}


func TestOpAnyArray(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t WHERE x = ANY (ARRAY[1,2,3])")
	sel := s.(*parser.SelectStmt)
	e := sel.WhereClause.(*parser.A_Expr)
	if e.Kind != parser.AEXPR_OP_ANY {
		t.Errorf("expected parser.AEXPR_OP_ANY, got %v", e.Kind)
	}
}


func TestOpSome(t *testing.T) {
	parseOK(t, "SELECT * FROM t WHERE x = SOME (ARRAY[1,2,3])")
}


func TestQualifiedOperator(t *testing.T) {
	s := parseOne(t, "SELECT a OPERATOR(pg_catalog.=) b")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if len(e.Name) != 2 || e.Name[0] != "pg_catalog" || e.Name[1] != "=" {
		t.Errorf("expected [pg_catalog, =], got %v", e.Name)
	}
}


func TestPrefixOp(t *testing.T) {
	s := parseOne(t, "SELECT ~x")
	sel := s.(*parser.SelectStmt)
	e := sel.TargetList[0].Val.(*parser.A_Expr)
	if e.Name[0] != "~" || e.Lexpr != nil {
		t.Errorf("expected prefix ~, got %+v", e)
	}
}




func TestOverlaps(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT (DATE '2001-02-16', DATE '2001-12-21') OVERLAPS (DATE '2001-10-30', DATE '2002-10-30')", "date ranges"},
		{"SELECT (a, b) OVERLAPS (c, d) FROM t", "column refs"},
		{"SELECT (ts1, ts2) OVERLAPS (ts3, interval '1 day') FROM t", "with interval"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			sel := stmt.(*parser.SelectStmt)
			expr := sel.TargetList[0].Val
			ae, ok := expr.(*parser.A_Expr)
			if !ok {
				t.Fatalf("expected *parser.A_Expr, got %T", expr)
			}
			if ae.Name[0] != "overlaps" {
				t.Fatalf("expected operator 'overlaps', got %q", ae.Name[0])
			}
			if ae.Lexpr == nil || ae.Rexpr == nil {
				t.Fatal("expected non-nil Lexpr and Rexpr")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TABLESAMPLE
// ---------------------------------------------------------------------------


func TestSubqueryIndirectionField(t *testing.T) {
	stmt := parseOne(t, "SELECT (SELECT r FROM t LIMIT 1).field")
	sel := stmt.(*parser.SelectStmt)
	expr := sel.TargetList[0].Val
	ind, ok := expr.(*parser.A_Indirection)
	if !ok {
		t.Fatalf("expected *parser.A_Indirection, got %T", expr)
	}
	if _, ok := ind.Arg.(*parser.SubLink); !ok {
		t.Fatalf("expected parser.SubLink as Arg, got %T", ind.Arg)
	}
	if len(ind.Indirection) == 0 {
		t.Fatal("expected non-empty Indirection")
	}
}


func TestSubqueryIndirectionSubscript(t *testing.T) {
	stmt := parseOne(t, "SELECT (SELECT arr FROM t LIMIT 1)[1]")
	sel := stmt.(*parser.SelectStmt)
	expr := sel.TargetList[0].Val
	ind, ok := expr.(*parser.A_Indirection)
	if !ok {
		t.Fatalf("expected *parser.A_Indirection, got %T", expr)
	}
	if len(ind.Indirection) == 0 {
		t.Fatal("expected non-empty Indirection")
	}
	idx, ok := ind.Indirection[0].(*parser.A_Indices)
	if !ok {
		t.Fatalf("expected *parser.A_Indices, got %T", ind.Indirection[0])
	}
	if idx.Uidx == nil {
		t.Fatal("expected non-nil Uidx")
	}
}


func TestSubqueryIndirectionChained(t *testing.T) {
	stmt := parseOne(t, "SELECT (SELECT r FROM t LIMIT 1).arr[1]")
	sel := stmt.(*parser.SelectStmt)
	expr := sel.TargetList[0].Val
	ind, ok := expr.(*parser.A_Indirection)
	if !ok {
		t.Fatalf("expected *parser.A_Indirection, got %T", expr)
	}
	if len(ind.Indirection) < 2 {
		t.Fatalf("expected at least 2 indirections, got %d", len(ind.Indirection))
	}
}

// ---------------------------------------------------------------------------
// Column-level GRANT
// ---------------------------------------------------------------------------

