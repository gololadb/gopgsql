package pgscan

import (
	"strings"
	"testing"
)

func parseOne(t *testing.T, sql string) Stmt {
	t.Helper()
	stmts, err := Parse(strings.NewReader(sql), nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(stmts) == 0 {
		t.Fatal("no statements parsed")
	}
	return stmts[0].Stmt
}

func parseOK(t *testing.T, sql string) {
	t.Helper()
	_, err := Parse(strings.NewReader(sql), nil)
	if err != nil {
		t.Fatalf("parse error for %q: %v", sql, err)
	}
}

// --- Expression tests ---

func TestExprIntLiteral(t *testing.T) {
	s := parseOne(t, "SELECT 42")
	sel := s.(*SelectStmt)
	c := sel.TargetList[0].Val.(*A_Const)
	if c.Val.Type != ValInt || c.Val.Ival != 42 {
		t.Errorf("expected int 42, got %+v", c.Val)
	}
}

func TestExprStringLiteral(t *testing.T) {
	s := parseOne(t, "SELECT 'hello'")
	sel := s.(*SelectStmt)
	c := sel.TargetList[0].Val.(*A_Const)
	if c.Val.Type != ValStr || c.Val.Str != "hello" {
		t.Errorf("expected string 'hello', got %+v", c.Val)
	}
}

func TestExprBinaryOp(t *testing.T) {
	s := parseOne(t, "SELECT 1 + 2")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Kind != AEXPR_OP || e.Name[0] != "+" {
		t.Errorf("expected + op, got %+v", e)
	}
}

func TestExprPrecedence(t *testing.T) {
	s := parseOne(t, "SELECT 1 + 2 * 3")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Name[0] != "+" {
		t.Errorf("top-level should be +, got %s", e.Name[0])
	}
	rhs := e.Rexpr.(*A_Expr)
	if rhs.Name[0] != "*" {
		t.Errorf("rhs should be *, got %s", rhs.Name[0])
	}
}

func TestExprUnaryMinus(t *testing.T) {
	s := parseOne(t, "SELECT -1")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Name[0] != "-" || e.Lexpr != nil {
		t.Errorf("expected unary minus, got %+v", e)
	}
}

func TestExprParens(t *testing.T) {
	s := parseOne(t, "SELECT (1 + 2) * 3")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Name[0] != "*" {
		t.Errorf("top-level should be *, got %s", e.Name[0])
	}
}

func TestExprBoolOps(t *testing.T) {
	s := parseOne(t, "SELECT true AND false OR true")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*BoolExpr)
	if e.Op != OR_EXPR {
		t.Errorf("top-level should be OR, got %v", e.Op)
	}
}

func TestExprNot(t *testing.T) {
	s := parseOne(t, "SELECT NOT true")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*BoolExpr)
	if e.Op != NOT_EXPR {
		t.Errorf("expected NOT, got %v", e.Op)
	}
}

func TestExprIsNull(t *testing.T) {
	s := parseOne(t, "SELECT x IS NULL")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*NullTest)
	if e.NullTestType != IS_NULL {
		t.Errorf("expected IS_NULL, got %v", e.NullTestType)
	}
}

func TestExprIsNotNull(t *testing.T) {
	s := parseOne(t, "SELECT x IS NOT NULL")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*NullTest)
	if e.NullTestType != IS_NOT_NULL {
		t.Errorf("expected IS_NOT_NULL, got %v", e.NullTestType)
	}
}

func TestExprBetween(t *testing.T) {
	s := parseOne(t, "SELECT x BETWEEN 1 AND 10")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Kind != AEXPR_BETWEEN {
		t.Errorf("expected BETWEEN, got %v", e.Kind)
	}
}

func TestExprIn(t *testing.T) {
	s := parseOne(t, "SELECT x IN (1, 2, 3)")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Kind != AEXPR_IN {
		t.Errorf("expected IN, got %v", e.Kind)
	}
}

func TestExprLike(t *testing.T) {
	s := parseOne(t, "SELECT x LIKE '%foo%'")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Kind != AEXPR_LIKE {
		t.Errorf("expected LIKE, got %v", e.Kind)
	}
}

func TestExprCast(t *testing.T) {
	s := parseOne(t, "SELECT x::integer")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*TypeCast)
	if e.TypeName.Names[1] != "int4" {
		t.Errorf("expected int4, got %v", e.TypeName.Names)
	}
}

func TestExprCastFunc(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS text)")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*TypeCast)
	if e.TypeName.Names[1] != "text" {
		t.Errorf("expected text, got %v", e.TypeName.Names)
	}
}

func TestExprCase(t *testing.T) {
	s := parseOne(t, "SELECT CASE WHEN x > 0 THEN 'pos' ELSE 'neg' END")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*CaseExpr)
	if len(e.Args) != 1 {
		t.Errorf("expected 1 WHEN, got %d", len(e.Args))
	}
	if e.Defresult == nil {
		t.Error("expected ELSE clause")
	}
}

func TestExprCoalesce(t *testing.T) {
	s := parseOne(t, "SELECT COALESCE(a, b, c)")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*CoalesceExpr)
	if len(e.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(e.Args))
	}
}

func TestExprNullif(t *testing.T) {
	s := parseOne(t, "SELECT NULLIF(a, 0)")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*NullIfExpr)
	if len(e.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(e.Args))
	}
}

func TestExprFuncCall(t *testing.T) {
	s := parseOne(t, "SELECT count(*)")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if !fc.AggStar {
		t.Error("expected AggStar")
	}
	if fc.Funcname[0] != "count" {
		t.Errorf("expected count, got %v", fc.Funcname)
	}
}

func TestExprFuncDistinct(t *testing.T) {
	s := parseOne(t, "SELECT count(DISTINCT x)")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if !fc.AggDistinct {
		t.Error("expected AggDistinct")
	}
}

func TestExprExists(t *testing.T) {
	s := parseOne(t, "SELECT EXISTS (SELECT 1)")
	sel := s.(*SelectStmt)
	sl := sel.TargetList[0].Val.(*SubLink)
	if sl.SubLinkType != EXISTS_SUBLINK {
		t.Errorf("expected EXISTS, got %v", sl.SubLinkType)
	}
}

func TestExprArray(t *testing.T) {
	s := parseOne(t, "SELECT ARRAY[1, 2, 3]")
	sel := s.(*SelectStmt)
	a := sel.TargetList[0].Val.(*A_ArrayExpr)
	if len(a.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(a.Elements))
	}
}

func TestExprParam(t *testing.T) {
	s := parseOne(t, "SELECT $1")
	sel := s.(*SelectStmt)
	pr := sel.TargetList[0].Val.(*ParamRef)
	if pr.Number != 1 {
		t.Errorf("expected $1, got $%d", pr.Number)
	}
}

func TestExprColumnRef(t *testing.T) {
	s := parseOne(t, "SELECT t.col")
	sel := s.(*SelectStmt)
	cr := sel.TargetList[0].Val.(*ColumnRef)
	if len(cr.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(cr.Fields))
	}
}

// --- SELECT tests ---

func TestSelectSimple(t *testing.T) {
	s := parseOne(t, "SELECT 1, 2, 3")
	sel := s.(*SelectStmt)
	if len(sel.TargetList) != 3 {
		t.Errorf("expected 3 targets, got %d", len(sel.TargetList))
	}
}

func TestSelectStar(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t")
	sel := s.(*SelectStmt)
	if len(sel.TargetList) != 1 {
		t.Fatalf("expected 1 target, got %d", len(sel.TargetList))
	}
	cr := sel.TargetList[0].Val.(*ColumnRef)
	if _, ok := cr.Fields[0].(*A_Star); !ok {
		t.Error("expected A_Star")
	}
}

func TestSelectAlias(t *testing.T) {
	s := parseOne(t, "SELECT 1 AS one")
	sel := s.(*SelectStmt)
	if sel.TargetList[0].Name != "one" {
		t.Errorf("expected alias 'one', got %q", sel.TargetList[0].Name)
	}
}

func TestSelectFrom(t *testing.T) {
	s := parseOne(t, "SELECT * FROM users")
	sel := s.(*SelectStmt)
	if len(sel.FromClause) != 1 {
		t.Fatalf("expected 1 from item, got %d", len(sel.FromClause))
	}
	rv := sel.FromClause[0].(*RangeVar)
	if rv.Relname != "users" {
		t.Errorf("expected 'users', got %q", rv.Relname)
	}
}

func TestSelectWhere(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t WHERE x > 0")
	sel := s.(*SelectStmt)
	if sel.WhereClause == nil {
		t.Error("expected WHERE clause")
	}
}

func TestSelectJoin(t *testing.T) {
	s := parseOne(t, "SELECT * FROM a JOIN b ON a.id = b.id")
	sel := s.(*SelectStmt)
	j := sel.FromClause[0].(*JoinExpr)
	if j.Jointype != JOIN_INNER {
		t.Errorf("expected INNER join, got %v", j.Jointype)
	}
	if j.Quals == nil {
		t.Error("expected ON clause")
	}
}

func TestSelectLeftJoin(t *testing.T) {
	s := parseOne(t, "SELECT * FROM a LEFT JOIN b ON a.id = b.id")
	sel := s.(*SelectStmt)
	j := sel.FromClause[0].(*JoinExpr)
	if j.Jointype != JOIN_LEFT {
		t.Errorf("expected LEFT join, got %v", j.Jointype)
	}
}

func TestSelectCrossJoin(t *testing.T) {
	s := parseOne(t, "SELECT * FROM a CROSS JOIN b")
	sel := s.(*SelectStmt)
	j := sel.FromClause[0].(*JoinExpr)
	if j.Jointype != JOIN_CROSS {
		t.Errorf("expected CROSS join, got %v", j.Jointype)
	}
}

func TestSelectNaturalJoin(t *testing.T) {
	s := parseOne(t, "SELECT * FROM a NATURAL JOIN b")
	sel := s.(*SelectStmt)
	j := sel.FromClause[0].(*JoinExpr)
	if !j.IsNatural {
		t.Error("expected NATURAL join")
	}
}

func TestSelectJoinUsing(t *testing.T) {
	s := parseOne(t, "SELECT * FROM a JOIN b USING (id)")
	sel := s.(*SelectStmt)
	j := sel.FromClause[0].(*JoinExpr)
	if len(j.UsingClause) != 1 || j.UsingClause[0] != "id" {
		t.Errorf("expected USING (id), got %v", j.UsingClause)
	}
}

func TestSelectGroupBy(t *testing.T) {
	s := parseOne(t, "SELECT dept, count(*) FROM emp GROUP BY dept")
	sel := s.(*SelectStmt)
	if len(sel.GroupClause) != 1 {
		t.Errorf("expected 1 group item, got %d", len(sel.GroupClause))
	}
}

func TestSelectHaving(t *testing.T) {
	s := parseOne(t, "SELECT dept, count(*) FROM emp GROUP BY dept HAVING count(*) > 5")
	sel := s.(*SelectStmt)
	if sel.HavingClause == nil {
		t.Error("expected HAVING clause")
	}
}

func TestSelectOrderBy(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t ORDER BY x ASC, y DESC")
	sel := s.(*SelectStmt)
	if len(sel.SortClause) != 2 {
		t.Fatalf("expected 2 sort items, got %d", len(sel.SortClause))
	}
	if sel.SortClause[0].SortbyDir != SORTBY_ASC {
		t.Error("expected ASC")
	}
	if sel.SortClause[1].SortbyDir != SORTBY_DESC {
		t.Error("expected DESC")
	}
}

func TestSelectLimit(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t LIMIT 10")
	sel := s.(*SelectStmt)
	if sel.LimitCount == nil {
		t.Error("expected LIMIT")
	}
}

func TestSelectOffset(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t LIMIT 10 OFFSET 20")
	sel := s.(*SelectStmt)
	if sel.LimitCount == nil || sel.LimitOffset == nil {
		t.Error("expected LIMIT and OFFSET")
	}
}

func TestSelectDistinct(t *testing.T) {
	s := parseOne(t, "SELECT DISTINCT x FROM t")
	sel := s.(*SelectStmt)
	if sel.DistinctClause == nil {
		t.Error("expected DISTINCT")
	}
}

func TestSelectDistinctOn(t *testing.T) {
	s := parseOne(t, "SELECT DISTINCT ON (x) x, y FROM t")
	sel := s.(*SelectStmt)
	if len(sel.DistinctClause) != 1 {
		t.Errorf("expected 1 DISTINCT ON expr, got %d", len(sel.DistinctClause))
	}
}

func TestSelectUnion(t *testing.T) {
	s := parseOne(t, "SELECT 1 UNION SELECT 2")
	sel := s.(*SelectStmt)
	if sel.Op != SETOP_UNION {
		t.Errorf("expected UNION, got %v", sel.Op)
	}
}

func TestSelectUnionAll(t *testing.T) {
	s := parseOne(t, "SELECT 1 UNION ALL SELECT 2")
	sel := s.(*SelectStmt)
	if sel.Op != SETOP_UNION || !sel.All {
		t.Error("expected UNION ALL")
	}
}

func TestSelectIntersect(t *testing.T) {
	s := parseOne(t, "SELECT 1 INTERSECT SELECT 2")
	sel := s.(*SelectStmt)
	if sel.Op != SETOP_INTERSECT {
		t.Errorf("expected INTERSECT, got %v", sel.Op)
	}
}

func TestSelectExcept(t *testing.T) {
	s := parseOne(t, "SELECT 1 EXCEPT SELECT 2")
	sel := s.(*SelectStmt)
	if sel.Op != SETOP_EXCEPT {
		t.Errorf("expected EXCEPT, got %v", sel.Op)
	}
}

func TestSelectSubquery(t *testing.T) {
	s := parseOne(t, "SELECT * FROM (SELECT 1) AS sub")
	sel := s.(*SelectStmt)
	rs := sel.FromClause[0].(*RangeSubselect)
	if rs.Alias == nil || rs.Alias.Aliasname != "sub" {
		t.Error("expected alias 'sub'")
	}
}

func TestSelectCTE(t *testing.T) {
	s := parseOne(t, "WITH cte AS (SELECT 1) SELECT * FROM cte")
	sel := s.(*SelectStmt)
	if sel.WithClause == nil {
		t.Fatal("expected WITH clause")
	}
	if len(sel.WithClause.CTEs) != 1 {
		t.Errorf("expected 1 CTE, got %d", len(sel.WithClause.CTEs))
	}
	if sel.WithClause.CTEs[0].Ctename != "cte" {
		t.Errorf("expected CTE name 'cte', got %q", sel.WithClause.CTEs[0].Ctename)
	}
}

func TestSelectRecursiveCTE(t *testing.T) {
	sql := `WITH RECURSIVE t(n) AS (
		VALUES (1)
		UNION ALL
		SELECT n+1 FROM t WHERE n < 100
	) SELECT * FROM t`
	s := parseOne(t, sql)
	sel := s.(*SelectStmt)
	if !sel.WithClause.Recursive {
		t.Error("expected RECURSIVE")
	}
}

func TestSelectForUpdate(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t FOR UPDATE")
	sel := s.(*SelectStmt)
	if len(sel.LockingClause) != 1 {
		t.Fatalf("expected 1 locking clause, got %d", len(sel.LockingClause))
	}
	if sel.LockingClause[0].Strength != LCS_FORUPDATE {
		t.Error("expected FOR UPDATE")
	}
}

func TestSelectValues(t *testing.T) {
	s := parseOne(t, "VALUES (1, 'a'), (2, 'b')")
	sel := s.(*SelectStmt)
	if len(sel.ValuesLists) != 2 {
		t.Errorf("expected 2 value rows, got %d", len(sel.ValuesLists))
	}
}

func TestSelectSchemaQualified(t *testing.T) {
	s := parseOne(t, "SELECT * FROM public.users")
	sel := s.(*SelectStmt)
	rv := sel.FromClause[0].(*RangeVar)
	if rv.Schemaname != "public" || rv.Relname != "users" {
		t.Errorf("expected public.users, got %s.%s", rv.Schemaname, rv.Relname)
	}
}

// --- INSERT tests ---

func TestInsertValues(t *testing.T) {
	s := parseOne(t, "INSERT INTO t (a, b) VALUES (1, 2)")
	ins := s.(*InsertStmt)
	if ins.Relation.Relname != "t" {
		t.Errorf("expected table 't', got %q", ins.Relation.Relname)
	}
	if len(ins.Cols) != 2 {
		t.Errorf("expected 2 columns, got %d", len(ins.Cols))
	}
}

func TestInsertSelect(t *testing.T) {
	s := parseOne(t, "INSERT INTO t SELECT * FROM s")
	ins := s.(*InsertStmt)
	if ins.SelectStmt == nil {
		t.Error("expected SELECT source")
	}
}

func TestInsertReturning(t *testing.T) {
	s := parseOne(t, "INSERT INTO t (a) VALUES (1) RETURNING *")
	ins := s.(*InsertStmt)
	if len(ins.ReturningList) != 1 {
		t.Errorf("expected 1 returning item, got %d", len(ins.ReturningList))
	}
}

func TestInsertOnConflictDoNothing(t *testing.T) {
	s := parseOne(t, "INSERT INTO t (a) VALUES (1) ON CONFLICT DO NOTHING")
	ins := s.(*InsertStmt)
	if ins.OnConflict == nil {
		t.Fatal("expected ON CONFLICT")
	}
	if ins.OnConflict.Action != ONCONFLICT_NOTHING {
		t.Error("expected DO NOTHING")
	}
}

func TestInsertOnConflictDoUpdate(t *testing.T) {
	s := parseOne(t, "INSERT INTO t (a, b) VALUES (1, 2) ON CONFLICT (a) DO UPDATE SET b = EXCLUDED.b")
	ins := s.(*InsertStmt)
	if ins.OnConflict == nil {
		t.Fatal("expected ON CONFLICT")
	}
	if ins.OnConflict.Action != ONCONFLICT_UPDATE {
		t.Error("expected DO UPDATE")
	}
	if len(ins.OnConflict.TargetList) != 1 {
		t.Errorf("expected 1 SET clause, got %d", len(ins.OnConflict.TargetList))
	}
}

func TestInsertWithCTE(t *testing.T) {
	sql := "WITH src AS (SELECT 1 AS a) INSERT INTO t SELECT * FROM src"
	s := parseOne(t, sql)
	ins := s.(*InsertStmt)
	if ins.WithClause == nil {
		t.Error("expected WITH clause")
	}
}

// --- UPDATE tests ---

func TestUpdateSimple(t *testing.T) {
	s := parseOne(t, "UPDATE t SET a = 1, b = 2 WHERE id = 1")
	upd := s.(*UpdateStmt)
	if upd.Relation.Relname != "t" {
		t.Errorf("expected table 't', got %q", upd.Relation.Relname)
	}
	if len(upd.TargetList) != 2 {
		t.Errorf("expected 2 SET clauses, got %d", len(upd.TargetList))
	}
	if upd.WhereClause == nil {
		t.Error("expected WHERE clause")
	}
}

func TestUpdateFrom(t *testing.T) {
	s := parseOne(t, "UPDATE t SET a = s.a FROM s WHERE t.id = s.id")
	upd := s.(*UpdateStmt)
	if len(upd.FromClause) != 1 {
		t.Errorf("expected 1 FROM item, got %d", len(upd.FromClause))
	}
}

func TestUpdateReturning(t *testing.T) {
	s := parseOne(t, "UPDATE t SET a = 1 RETURNING *")
	upd := s.(*UpdateStmt)
	if len(upd.ReturningList) != 1 {
		t.Errorf("expected 1 returning item, got %d", len(upd.ReturningList))
	}
}

// --- DELETE tests ---

func TestDeleteSimple(t *testing.T) {
	s := parseOne(t, "DELETE FROM t WHERE id = 1")
	del := s.(*DeleteStmt)
	if del.Relation.Relname != "t" {
		t.Errorf("expected table 't', got %q", del.Relation.Relname)
	}
	if del.WhereClause == nil {
		t.Error("expected WHERE clause")
	}
}

func TestDeleteUsing(t *testing.T) {
	s := parseOne(t, "DELETE FROM t USING s WHERE t.id = s.id")
	del := s.(*DeleteStmt)
	if len(del.UsingClause) != 1 {
		t.Errorf("expected 1 USING item, got %d", len(del.UsingClause))
	}
}

func TestDeleteReturning(t *testing.T) {
	s := parseOne(t, "DELETE FROM t RETURNING *")
	del := s.(*DeleteStmt)
	if len(del.ReturningList) != 1 {
		t.Errorf("expected 1 returning item, got %d", len(del.ReturningList))
	}
}

func TestDeleteWithCTE(t *testing.T) {
	sql := "WITH old AS (SELECT id FROM t WHERE age > 100) DELETE FROM t WHERE id IN (SELECT id FROM old)"
	s := parseOne(t, sql)
	del := s.(*DeleteStmt)
	if del.WithClause == nil {
		t.Error("expected WITH clause")
	}
}

// --- Multi-statement ---

func TestMultiStatement(t *testing.T) {
	sql := "SELECT 1; SELECT 2; SELECT 3"
	stmts, err := Parse(strings.NewReader(sql), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 3 {
		t.Errorf("expected 3 statements, got %d", len(stmts))
	}
}

// --- Complex real-world queries ---

func TestComplexQuery(t *testing.T) {
	sql := `
		WITH regional_sales AS (
			SELECT region, SUM(amount) AS total_sales
			FROM orders
			GROUP BY region
		), top_regions AS (
			SELECT region
			FROM regional_sales
			WHERE total_sales > (SELECT SUM(total_sales) / 10 FROM regional_sales)
		)
		SELECT region, product, SUM(quantity) AS product_units, SUM(amount) AS product_sales
		FROM orders
		WHERE region IN (SELECT region FROM top_regions)
		GROUP BY region, product
		ORDER BY region, product_sales DESC
		LIMIT 100
	`
	parseOK(t, sql)
}

func TestComplexInsert(t *testing.T) {
	sql := `
		INSERT INTO summary (region, total)
		SELECT region, SUM(amount)
		FROM orders
		GROUP BY region
		ON CONFLICT (region) DO UPDATE SET total = EXCLUDED.total
		RETURNING *
	`
	parseOK(t, sql)
}

func TestComplexUpdate(t *testing.T) {
	sql := `
		UPDATE accounts a
		SET balance = a.balance + t.amount
		FROM transactions t
		WHERE a.id = t.account_id AND t.processed = false
		RETURNING a.id, a.balance
	`
	parseOK(t, sql)
}

func TestExprIsTrue(t *testing.T) {
	s := parseOne(t, "SELECT x IS TRUE")
	sel := s.(*SelectStmt)
	bt := sel.TargetList[0].Val.(*BooleanTest)
	if bt.BooltestType != IS_TRUE {
		t.Errorf("expected IS_TRUE, got %v", bt.BooltestType)
	}
}

func TestExprIsDistinctFrom(t *testing.T) {
	s := parseOne(t, "SELECT x IS DISTINCT FROM y")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Kind != AEXPR_DISTINCT {
		t.Errorf("expected DISTINCT, got %v", e.Kind)
	}
}

func TestExprNotBetween(t *testing.T) {
	s := parseOne(t, "SELECT x NOT BETWEEN 1 AND 10")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Kind != AEXPR_NOT_BETWEEN {
		t.Errorf("expected NOT_BETWEEN, got %v", e.Kind)
	}
}

func TestExprNotIn(t *testing.T) {
	s := parseOne(t, "SELECT x NOT IN (1, 2)")
	sel := s.(*SelectStmt)
	be := sel.TargetList[0].Val.(*BoolExpr)
	if be.Op != NOT_EXPR {
		t.Errorf("expected NOT wrapping IN, got %v", be.Op)
	}
}

func TestExprGreatest(t *testing.T) {
	s := parseOne(t, "SELECT GREATEST(1, 2, 3)")
	sel := s.(*SelectStmt)
	mm := sel.TargetList[0].Val.(*MinMaxExpr)
	if mm.Op != IS_GREATEST || len(mm.Args) != 3 {
		t.Errorf("expected GREATEST with 3 args, got %v with %d", mm.Op, len(mm.Args))
	}
}

func TestExprRowConstructor(t *testing.T) {
	s := parseOne(t, "SELECT (1, 2, 3)")
	sel := s.(*SelectStmt)
	re := sel.TargetList[0].Val.(*RowExpr)
	if len(re.Args) != 3 {
		t.Errorf("expected 3 row elements, got %d", len(re.Args))
	}
}

func TestExprScalarSubquery(t *testing.T) {
	s := parseOne(t, "SELECT (SELECT 1)")
	sel := s.(*SelectStmt)
	sl := sel.TargetList[0].Val.(*SubLink)
	if sl.SubLinkType != EXPR_SUBLINK {
		t.Errorf("expected EXPR_SUBLINK, got %v", sl.SubLinkType)
	}
}

func TestExprInSubquery(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t WHERE x IN (SELECT y FROM s)")
	sel := s.(*SelectStmt)
	sl := sel.WhereClause.(*SubLink)
	if sl.SubLinkType != ANY_SUBLINK {
		t.Errorf("expected ANY_SUBLINK, got %v", sl.SubLinkType)
	}
}

func TestSelectTableAlias(t *testing.T) {
	s := parseOne(t, "SELECT u.id FROM users u")
	sel := s.(*SelectStmt)
	rv := sel.FromClause[0].(*RangeVar)
	if rv.Alias == nil || rv.Alias.Aliasname != "u" {
		t.Error("expected alias 'u'")
	}
}

func TestSelectMultipleJoins(t *testing.T) {
	sql := "SELECT * FROM a JOIN b ON a.id = b.a_id LEFT JOIN c ON b.id = c.b_id"
	s := parseOne(t, sql)
	sel := s.(*SelectStmt)
	// Should be: LEFT JOIN(JOIN(a, b), c)
	j := sel.FromClause[0].(*JoinExpr)
	if j.Jointype != JOIN_LEFT {
		t.Errorf("outer join should be LEFT, got %v", j.Jointype)
	}
	inner := j.Larg.(*JoinExpr)
	if inner.Jointype != JOIN_INNER {
		t.Errorf("inner join should be INNER, got %v", inner.Jointype)
	}
}

// --- Step 1: SQL syntax functions ---

func TestExtract(t *testing.T) {
	s := parseOne(t, "SELECT EXTRACT(YEAR FROM d)")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if fc.Funcname[1] != "extract" {
		t.Errorf("expected extract, got %v", fc.Funcname)
	}
	if len(fc.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(fc.Args))
	}
	field := fc.Args[0].(*A_Const)
	if field.Val.Str != "year" {
		t.Errorf("expected 'year', got %q", field.Val.Str)
	}
}

func TestExtractEpoch(t *testing.T) {
	parseOK(t, "SELECT EXTRACT(EPOCH FROM now())")
}

func TestPosition(t *testing.T) {
	s := parseOne(t, "SELECT POSITION('x' IN 'abcxdef')")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if fc.Funcname[1] != "position" {
		t.Errorf("expected position, got %v", fc.Funcname)
	}
	// Args should be (haystack, needle) — reversed from SQL syntax
	if len(fc.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(fc.Args))
	}
}

func TestSubstringFromFor(t *testing.T) {
	s := parseOne(t, "SELECT SUBSTRING('hello' FROM 2 FOR 3)")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if fc.Funcname[1] != "substring" {
		t.Errorf("expected substring, got %v", fc.Funcname)
	}
	if len(fc.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(fc.Args))
	}
}

func TestSubstringFrom(t *testing.T) {
	s := parseOne(t, "SELECT SUBSTRING('hello' FROM 2)")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if len(fc.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(fc.Args))
	}
}

func TestSubstringPlain(t *testing.T) {
	s := parseOne(t, "SELECT SUBSTRING('hello', 2, 3)")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if fc.Funcname[0] != "substring" {
		t.Errorf("expected plain substring, got %v", fc.Funcname)
	}
	if len(fc.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(fc.Args))
	}
}

func TestOverlay(t *testing.T) {
	s := parseOne(t, "SELECT OVERLAY('hello' PLACING 'XX' FROM 2 FOR 3)")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if fc.Funcname[1] != "overlay" {
		t.Errorf("expected overlay, got %v", fc.Funcname)
	}
	if len(fc.Args) != 4 {
		t.Errorf("expected 4 args, got %d", len(fc.Args))
	}
}

func TestOverlayNoFor(t *testing.T) {
	s := parseOne(t, "SELECT OVERLAY('hello' PLACING 'XX' FROM 2)")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if len(fc.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(fc.Args))
	}
}

func TestTrimBoth(t *testing.T) {
	s := parseOne(t, "SELECT TRIM(BOTH 'x' FROM 'xxxhelloxxx')")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if fc.Funcname[1] != "btrim" {
		t.Errorf("expected btrim, got %v", fc.Funcname)
	}
}

func TestTrimLeading(t *testing.T) {
	s := parseOne(t, "SELECT TRIM(LEADING 'x' FROM 'xxxhello')")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if fc.Funcname[1] != "ltrim" {
		t.Errorf("expected ltrim, got %v", fc.Funcname)
	}
}

func TestTrimTrailing(t *testing.T) {
	s := parseOne(t, "SELECT TRIM(TRAILING 'x' FROM 'helloxxx')")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if fc.Funcname[1] != "rtrim" {
		t.Errorf("expected rtrim, got %v", fc.Funcname)
	}
}

func TestTrimDefault(t *testing.T) {
	s := parseOne(t, "SELECT TRIM('  hello  ')")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if fc.Funcname[1] != "btrim" {
		t.Errorf("expected btrim, got %v", fc.Funcname)
	}
}

func TestTreat(t *testing.T) {
	parseOK(t, "SELECT TREAT(x AS integer)")
}

func TestNormalize(t *testing.T) {
	s := parseOne(t, "SELECT NORMALIZE('hello')")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if fc.Funcname[1] != "normalize" {
		t.Errorf("expected normalize, got %v", fc.Funcname)
	}
}

func TestNormalizeWithForm(t *testing.T) {
	s := parseOne(t, "SELECT NORMALIZE('hello', NFC)")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if len(fc.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(fc.Args))
	}
}

func TestCollationFor(t *testing.T) {
	s := parseOne(t, "SELECT COLLATION FOR ('hello')")
	sel := s.(*SelectStmt)
	fc := sel.TargetList[0].Val.(*FuncCall)
	if fc.Funcname[1] != "pg_collation_for" {
		t.Errorf("expected pg_collation_for, got %v", fc.Funcname)
	}
}

// --- Step 2: SQL value functions ---

func TestCurrentDate(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_DATE")
	sel := s.(*SelectStmt)
	svf := sel.TargetList[0].Val.(*SQLValueFunction)
	if svf.Op != SVFOP_CURRENT_DATE {
		t.Errorf("expected CURRENT_DATE, got %v", svf.Op)
	}
}

func TestCurrentTimestamp(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_TIMESTAMP")
	sel := s.(*SelectStmt)
	svf := sel.TargetList[0].Val.(*SQLValueFunction)
	if svf.Op != SVFOP_CURRENT_TIMESTAMP {
		t.Errorf("expected CURRENT_TIMESTAMP, got %v", svf.Op)
	}
}

func TestCurrentTimestampPrecision(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_TIMESTAMP(3)")
	sel := s.(*SelectStmt)
	svf := sel.TargetList[0].Val.(*SQLValueFunction)
	if svf.Op != SVFOP_CURRENT_TIMESTAMP_N || svf.Typmod != 3 {
		t.Errorf("expected CURRENT_TIMESTAMP_N(3), got op=%v typmod=%d", svf.Op, svf.Typmod)
	}
}

func TestCurrentTime(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_TIME")
	sel := s.(*SelectStmt)
	svf := sel.TargetList[0].Val.(*SQLValueFunction)
	if svf.Op != SVFOP_CURRENT_TIME {
		t.Errorf("expected CURRENT_TIME, got %v", svf.Op)
	}
}

func TestLocaltime(t *testing.T) {
	s := parseOne(t, "SELECT LOCALTIME")
	sel := s.(*SelectStmt)
	svf := sel.TargetList[0].Val.(*SQLValueFunction)
	if svf.Op != SVFOP_LOCALTIME {
		t.Errorf("expected LOCALTIME, got %v", svf.Op)
	}
}

func TestLocaltimestamp(t *testing.T) {
	s := parseOne(t, "SELECT LOCALTIMESTAMP(6)")
	sel := s.(*SelectStmt)
	svf := sel.TargetList[0].Val.(*SQLValueFunction)
	if svf.Op != SVFOP_LOCALTIMESTAMP_N || svf.Typmod != 6 {
		t.Errorf("expected LOCALTIMESTAMP_N(6), got op=%v typmod=%d", svf.Op, svf.Typmod)
	}
}

func TestCurrentUser(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_USER")
	sel := s.(*SelectStmt)
	svf := sel.TargetList[0].Val.(*SQLValueFunction)
	if svf.Op != SVFOP_CURRENT_USER {
		t.Errorf("expected CURRENT_USER, got %v", svf.Op)
	}
}

func TestSessionUser(t *testing.T) {
	s := parseOne(t, "SELECT SESSION_USER")
	sel := s.(*SelectStmt)
	svf := sel.TargetList[0].Val.(*SQLValueFunction)
	if svf.Op != SVFOP_SESSION_USER {
		t.Errorf("expected SESSION_USER, got %v", svf.Op)
	}
}

func TestCurrentRole(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_ROLE")
	sel := s.(*SelectStmt)
	svf := sel.TargetList[0].Val.(*SQLValueFunction)
	if svf.Op != SVFOP_CURRENT_ROLE {
		t.Errorf("expected CURRENT_ROLE, got %v", svf.Op)
	}
}

func TestCurrentCatalog(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_CATALOG")
	sel := s.(*SelectStmt)
	svf := sel.TargetList[0].Val.(*SQLValueFunction)
	if svf.Op != SVFOP_CURRENT_CATALOG {
		t.Errorf("expected CURRENT_CATALOG, got %v", svf.Op)
	}
}

func TestCurrentSchema(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_SCHEMA")
	sel := s.(*SelectStmt)
	svf := sel.TargetList[0].Val.(*SQLValueFunction)
	if svf.Op != SVFOP_CURRENT_SCHEMA {
		t.Errorf("expected CURRENT_SCHEMA, got %v", svf.Op)
	}
}

func TestCurrentSchemaParens(t *testing.T) {
	parseOK(t, "SELECT CURRENT_SCHEMA()")
}

func TestGroupingFunc(t *testing.T) {
	s := parseOne(t, "SELECT GROUPING(a, b)")
	sel := s.(*SelectStmt)
	gf := sel.TargetList[0].Val.(*GroupingFunc)
	if len(gf.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(gf.Args))
	}
}

func TestSetToDefault(t *testing.T) {
	s := parseOne(t, "INSERT INTO t (a) VALUES (DEFAULT)")
	ins := s.(*InsertStmt)
	sel := ins.SelectStmt.(*SelectStmt)
	val := sel.ValuesLists[0][0]
	if _, ok := val.(*SetToDefault); !ok {
		t.Errorf("expected SetToDefault, got %T", val)
	}
}

func TestValFuncsInExpr(t *testing.T) {
	parseOK(t, "SELECT * FROM t WHERE created_at > CURRENT_TIMESTAMP - 1")
}

// --- Step 3: Operator forms, ESCAPE, IS DOCUMENT/NORMALIZED, AT LOCAL, | ---

func TestAtLocal(t *testing.T) {
	s := parseOne(t, "SELECT ts AT LOCAL")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Name[0] != "timezone" {
		t.Errorf("expected timezone op, got %v", e.Name)
	}
	rhs := e.Rexpr.(*A_Const)
	if rhs.Val.Str != "local" {
		t.Errorf("expected 'local', got %q", rhs.Val.Str)
	}
}

func TestPipeOperator(t *testing.T) {
	s := parseOne(t, "SELECT 1 | 2")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Name[0] != "|" {
		t.Errorf("expected | op, got %v", e.Name)
	}
}

func TestLikeEscape(t *testing.T) {
	s := parseOne(t, "SELECT x LIKE '%a%' ESCAPE '\\'")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Kind != AEXPR_LIKE {
		t.Errorf("expected LIKE, got %v", e.Kind)
	}
	// rhs should be ExprList with pattern and escape
	el := e.Rexpr.(*ExprList)
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
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Name[0] != "is_document" {
		t.Errorf("expected is_document, got %v", e.Name)
	}
}

func TestIsNotDocument(t *testing.T) {
	s := parseOne(t, "SELECT x IS NOT DOCUMENT")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Name[0] != "is_not_document" {
		t.Errorf("expected is_not_document, got %v", e.Name)
	}
}

func TestIsNormalized(t *testing.T) {
	s := parseOne(t, "SELECT x IS NORMALIZED")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Name[0] != "is_normalized" {
		t.Errorf("expected is_normalized, got %v", e.Name)
	}
}

func TestIsNFCNormalized(t *testing.T) {
	s := parseOne(t, "SELECT x IS NFC NORMALIZED")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	rhs := e.Rexpr.(*A_Const)
	if rhs.Val.Str != "NFC" {
		t.Errorf("expected NFC, got %q", rhs.Val.Str)
	}
}

func TestIsNotNFKDNormalized(t *testing.T) {
	s := parseOne(t, "SELECT x IS NOT NFKD NORMALIZED")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Name[0] != "is_not_normalized" {
		t.Errorf("expected is_not_normalized, got %v", e.Name)
	}
}

func TestOpAnySubquery(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t WHERE x = ANY (SELECT y FROM s)")
	sel := s.(*SelectStmt)
	sl := sel.WhereClause.(*SubLink)
	if sl.SubLinkType != ANY_SUBLINK {
		t.Errorf("expected ANY_SUBLINK, got %v", sl.SubLinkType)
	}
	if sl.OperName[0] != "=" {
		t.Errorf("expected = operator, got %v", sl.OperName)
	}
}

func TestOpAllSubquery(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t WHERE x > ALL (SELECT y FROM s)")
	sel := s.(*SelectStmt)
	sl := sel.WhereClause.(*SubLink)
	if sl.SubLinkType != ALL_SUBLINK {
		t.Errorf("expected ALL_SUBLINK, got %v", sl.SubLinkType)
	}
}

func TestOpAnyArray(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t WHERE x = ANY (ARRAY[1,2,3])")
	sel := s.(*SelectStmt)
	e := sel.WhereClause.(*A_Expr)
	if e.Kind != AEXPR_OP_ANY {
		t.Errorf("expected AEXPR_OP_ANY, got %v", e.Kind)
	}
}

func TestOpSome(t *testing.T) {
	parseOK(t, "SELECT * FROM t WHERE x = SOME (ARRAY[1,2,3])")
}

func TestQualifiedOperator(t *testing.T) {
	s := parseOne(t, "SELECT a OPERATOR(pg_catalog.=) b")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if len(e.Name) != 2 || e.Name[0] != "pg_catalog" || e.Name[1] != "=" {
		t.Errorf("expected [pg_catalog, =], got %v", e.Name)
	}
}

func TestPrefixOp(t *testing.T) {
	s := parseOne(t, "SELECT ~x")
	sel := s.(*SelectStmt)
	e := sel.TargetList[0].Val.(*A_Expr)
	if e.Name[0] != "~" || e.Lexpr != nil {
		t.Errorf("expected prefix ~, got %+v", e)
	}
}


