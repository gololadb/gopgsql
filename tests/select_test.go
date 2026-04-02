package tests

import (
	"testing"

	"github.com/gololadb/gopgsql/parser"
)

func TestSelectSimple(t *testing.T) {
	s := parseOne(t, "SELECT 1, 2, 3")
	sel := s.(*parser.SelectStmt)
	if len(sel.TargetList) != 3 {
		t.Errorf("expected 3 targets, got %d", len(sel.TargetList))
	}
}


func TestSelectStar(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t")
	sel := s.(*parser.SelectStmt)
	if len(sel.TargetList) != 1 {
		t.Fatalf("expected 1 target, got %d", len(sel.TargetList))
	}
	cr := sel.TargetList[0].Val.(*parser.ColumnRef)
	if _, ok := cr.Fields[0].(*parser.A_Star); !ok {
		t.Error("expected parser.A_Star")
	}
}


func TestSelectAlias(t *testing.T) {
	s := parseOne(t, "SELECT 1 AS one")
	sel := s.(*parser.SelectStmt)
	if sel.TargetList[0].Name != "one" {
		t.Errorf("expected alias 'one', got %q", sel.TargetList[0].Name)
	}
}


func TestSelectFrom(t *testing.T) {
	s := parseOne(t, "SELECT * FROM users")
	sel := s.(*parser.SelectStmt)
	if len(sel.FromClause) != 1 {
		t.Fatalf("expected 1 from item, got %d", len(sel.FromClause))
	}
	rv := sel.FromClause[0].(*parser.RangeVar)
	if rv.Relname != "users" {
		t.Errorf("expected 'users', got %q", rv.Relname)
	}
}


func TestSelectWhere(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t WHERE x > 0")
	sel := s.(*parser.SelectStmt)
	if sel.WhereClause == nil {
		t.Error("expected WHERE clause")
	}
}


func TestSelectJoin(t *testing.T) {
	s := parseOne(t, "SELECT * FROM a JOIN b ON a.id = b.id")
	sel := s.(*parser.SelectStmt)
	j := sel.FromClause[0].(*parser.JoinExpr)
	if j.Jointype != parser.JOIN_INNER {
		t.Errorf("expected INNER join, got %v", j.Jointype)
	}
	if j.Quals == nil {
		t.Error("expected ON clause")
	}
}


func TestSelectLeftJoin(t *testing.T) {
	s := parseOne(t, "SELECT * FROM a LEFT JOIN b ON a.id = b.id")
	sel := s.(*parser.SelectStmt)
	j := sel.FromClause[0].(*parser.JoinExpr)
	if j.Jointype != parser.JOIN_LEFT {
		t.Errorf("expected LEFT join, got %v", j.Jointype)
	}
}


func TestSelectCrossJoin(t *testing.T) {
	s := parseOne(t, "SELECT * FROM a CROSS JOIN b")
	sel := s.(*parser.SelectStmt)
	j := sel.FromClause[0].(*parser.JoinExpr)
	if j.Jointype != parser.JOIN_CROSS {
		t.Errorf("expected CROSS join, got %v", j.Jointype)
	}
}


func TestSelectNaturalJoin(t *testing.T) {
	s := parseOne(t, "SELECT * FROM a NATURAL JOIN b")
	sel := s.(*parser.SelectStmt)
	j := sel.FromClause[0].(*parser.JoinExpr)
	if !j.IsNatural {
		t.Error("expected NATURAL join")
	}
}


func TestSelectJoinUsing(t *testing.T) {
	s := parseOne(t, "SELECT * FROM a JOIN b USING (id)")
	sel := s.(*parser.SelectStmt)
	j := sel.FromClause[0].(*parser.JoinExpr)
	if len(j.UsingClause) != 1 || j.UsingClause[0] != "id" {
		t.Errorf("expected USING (id), got %v", j.UsingClause)
	}
}


func TestSelectGroupBy(t *testing.T) {
	s := parseOne(t, "SELECT dept, count(*) FROM emp GROUP BY dept")
	sel := s.(*parser.SelectStmt)
	if len(sel.GroupClause) != 1 {
		t.Errorf("expected 1 group item, got %d", len(sel.GroupClause))
	}
}


func TestSelectHaving(t *testing.T) {
	s := parseOne(t, "SELECT dept, count(*) FROM emp GROUP BY dept HAVING count(*) > 5")
	sel := s.(*parser.SelectStmt)
	if sel.HavingClause == nil {
		t.Error("expected HAVING clause")
	}
}


func TestSelectOrderBy(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t ORDER BY x ASC, y DESC")
	sel := s.(*parser.SelectStmt)
	if len(sel.SortClause) != 2 {
		t.Fatalf("expected 2 sort items, got %d", len(sel.SortClause))
	}
	if sel.SortClause[0].SortbyDir != parser.SORTBY_ASC {
		t.Error("expected ASC")
	}
	if sel.SortClause[1].SortbyDir != parser.SORTBY_DESC {
		t.Error("expected DESC")
	}
}


func TestSelectLimit(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t LIMIT 10")
	sel := s.(*parser.SelectStmt)
	if sel.LimitCount == nil {
		t.Error("expected LIMIT")
	}
}


func TestSelectOffset(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t LIMIT 10 OFFSET 20")
	sel := s.(*parser.SelectStmt)
	if sel.LimitCount == nil || sel.LimitOffset == nil {
		t.Error("expected LIMIT and OFFSET")
	}
}


func TestSelectDistinct(t *testing.T) {
	s := parseOne(t, "SELECT DISTINCT x FROM t")
	sel := s.(*parser.SelectStmt)
	if sel.DistinctClause == nil {
		t.Error("expected DISTINCT")
	}
}


func TestSelectDistinctOn(t *testing.T) {
	s := parseOne(t, "SELECT DISTINCT ON (x) x, y FROM t")
	sel := s.(*parser.SelectStmt)
	if len(sel.DistinctClause) != 1 {
		t.Errorf("expected 1 DISTINCT ON expr, got %d", len(sel.DistinctClause))
	}
}


func TestSelectUnion(t *testing.T) {
	s := parseOne(t, "SELECT 1 UNION SELECT 2")
	sel := s.(*parser.SelectStmt)
	if sel.Op != parser.SETOP_UNION {
		t.Errorf("expected UNION, got %v", sel.Op)
	}
}


func TestSelectUnionAll(t *testing.T) {
	s := parseOne(t, "SELECT 1 UNION ALL SELECT 2")
	sel := s.(*parser.SelectStmt)
	if sel.Op != parser.SETOP_UNION || !sel.All {
		t.Error("expected UNION ALL")
	}
}


func TestSelectIntersect(t *testing.T) {
	s := parseOne(t, "SELECT 1 INTERSECT SELECT 2")
	sel := s.(*parser.SelectStmt)
	if sel.Op != parser.SETOP_INTERSECT {
		t.Errorf("expected INTERSECT, got %v", sel.Op)
	}
}


func TestSelectExcept(t *testing.T) {
	s := parseOne(t, "SELECT 1 EXCEPT SELECT 2")
	sel := s.(*parser.SelectStmt)
	if sel.Op != parser.SETOP_EXCEPT {
		t.Errorf("expected EXCEPT, got %v", sel.Op)
	}
}


func TestSelectSubquery(t *testing.T) {
	s := parseOne(t, "SELECT * FROM (SELECT 1) AS sub")
	sel := s.(*parser.SelectStmt)
	rs := sel.FromClause[0].(*parser.RangeSubselect)
	if rs.Alias == nil || rs.Alias.Aliasname != "sub" {
		t.Error("expected alias 'sub'")
	}
}


func TestSelectCTE(t *testing.T) {
	s := parseOne(t, "WITH cte AS (SELECT 1) SELECT * FROM cte")
	sel := s.(*parser.SelectStmt)
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
	sel := s.(*parser.SelectStmt)
	if !sel.WithClause.Recursive {
		t.Error("expected RECURSIVE")
	}
}


func TestSelectForUpdate(t *testing.T) {
	s := parseOne(t, "SELECT * FROM t FOR UPDATE")
	sel := s.(*parser.SelectStmt)
	if len(sel.LockingClause) != 1 {
		t.Fatalf("expected 1 locking clause, got %d", len(sel.LockingClause))
	}
	if sel.LockingClause[0].Strength != parser.LCS_FORUPDATE {
		t.Error("expected FOR UPDATE")
	}
}


func TestSelectValues(t *testing.T) {
	s := parseOne(t, "VALUES (1, 'a'), (2, 'b')")
	sel := s.(*parser.SelectStmt)
	if len(sel.ValuesLists) != 2 {
		t.Errorf("expected 2 value rows, got %d", len(sel.ValuesLists))
	}
}


func TestSelectSchemaQualified(t *testing.T) {
	s := parseOne(t, "SELECT * FROM public.users")
	sel := s.(*parser.SelectStmt)
	rv := sel.FromClause[0].(*parser.RangeVar)
	if rv.Schemaname != "public" || rv.Relname != "users" {
		t.Errorf("expected public.users, got %s.%s", rv.Schemaname, rv.Relname)
	}
}

// --- INSERT tests ---


func TestSelectTableAlias(t *testing.T) {
	s := parseOne(t, "SELECT u.id FROM users u")
	sel := s.(*parser.SelectStmt)
	rv := sel.FromClause[0].(*parser.RangeVar)
	if rv.Alias == nil || rv.Alias.Aliasname != "u" {
		t.Error("expected alias 'u'")
	}
}


func TestSelectMultipleJoins(t *testing.T) {
	sql := "SELECT * FROM a JOIN b ON a.id = b.a_id LEFT JOIN c ON b.id = c.b_id"
	s := parseOne(t, sql)
	sel := s.(*parser.SelectStmt)
	// Should be: LEFT JOIN(JOIN(a, b), c)
	j := sel.FromClause[0].(*parser.JoinExpr)
	if j.Jointype != parser.JOIN_LEFT {
		t.Errorf("outer join should be LEFT, got %v", j.Jointype)
	}
	inner := j.Larg.(*parser.JoinExpr)
	if inner.Jointype != parser.JOIN_INNER {
		t.Errorf("inner join should be INNER, got %v", inner.Jointype)
	}
}

// --- Step 1: SQL syntax functions ---


func TestTablesample(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT * FROM t TABLESAMPLE bernoulli(10)", "bernoulli"},
		{"SELECT * FROM t TABLESAMPLE system(50)", "system"},
		{"SELECT * FROM t TABLESAMPLE bernoulli(10) REPEATABLE (42)", "repeatable"},
		{"SELECT * FROM myschema.t TABLESAMPLE system(25)", "qualified table"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			sel := stmt.(*parser.SelectStmt)
			if len(sel.FromClause) == 0 {
				t.Fatal("expected non-empty FromClause")
			}
			ts, ok := sel.FromClause[0].(*parser.RangeTableSample)
			if !ok {
				t.Fatalf("expected *parser.RangeTableSample, got %T", sel.FromClause[0])
			}
			if ts.Method == "" {
				t.Fatal("expected non-empty Method")
			}
			if len(ts.Args) == 0 {
				t.Fatal("expected non-empty Args")
			}
		})
	}
}


func TestTablesampleRepeatable(t *testing.T) {
	stmt := parseOne(t, "SELECT * FROM t TABLESAMPLE bernoulli(10) REPEATABLE (42)")
	sel := stmt.(*parser.SelectStmt)
	ts := sel.FromClause[0].(*parser.RangeTableSample)
	if ts.Repeatable == nil {
		t.Fatal("expected non-nil Repeatable")
	}
}


func TestTablesampleMethod(t *testing.T) {
	stmt := parseOne(t, "SELECT * FROM t TABLESAMPLE system(50)")
	sel := stmt.(*parser.SelectStmt)
	ts := sel.FromClause[0].(*parser.RangeTableSample)
	if ts.Method != "system" {
		t.Fatalf("expected Method=system, got %q", ts.Method)
	}
}

// ---------------------------------------------------------------------------
// Indirection on subqueries
// ---------------------------------------------------------------------------

