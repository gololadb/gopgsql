package pgscan

import "testing"

// ---------------------------------------------------------------------------
// Step 6: GROUP BY extensions
// ---------------------------------------------------------------------------

func TestGroupByRollup(t *testing.T) {
	s := parseOne(t, "SELECT a, b, sum(c) FROM t GROUP BY ROLLUP(a, b)")
	sel := s.(*SelectStmt)
	if len(sel.GroupClause) != 1 {
		t.Fatalf("expected 1 group item, got %d", len(sel.GroupClause))
	}
	gs, ok := sel.GroupClause[0].(*GroupingSet)
	if !ok {
		t.Fatalf("expected GroupingSet, got %T", sel.GroupClause[0])
	}
	if gs.Kind != GROUPING_SET_ROLLUP {
		t.Errorf("expected ROLLUP, got %d", gs.Kind)
	}
	if len(gs.Content) != 2 {
		t.Errorf("expected 2 items, got %d", len(gs.Content))
	}
}

func TestGroupByCube(t *testing.T) {
	s := parseOne(t, "SELECT a, b, sum(c) FROM t GROUP BY CUBE(a, b)")
	sel := s.(*SelectStmt)
	gs := sel.GroupClause[0].(*GroupingSet)
	if gs.Kind != GROUPING_SET_CUBE {
		t.Errorf("expected CUBE, got %d", gs.Kind)
	}
	if len(gs.Content) != 2 {
		t.Errorf("expected 2 items, got %d", len(gs.Content))
	}
}

func TestGroupByGroupingSets(t *testing.T) {
	s := parseOne(t, "SELECT a, b, sum(c) FROM t GROUP BY GROUPING SETS ((a, b), (a), ())")
	sel := s.(*SelectStmt)
	gs := sel.GroupClause[0].(*GroupingSet)
	if gs.Kind != GROUPING_SET_SETS {
		t.Errorf("expected GROUPING_SET_SETS, got %d", gs.Kind)
	}
	if len(gs.Content) != 3 {
		t.Fatalf("expected 3 items in GROUPING SETS, got %d", len(gs.Content))
	}
	// Third item should be empty grouping set
	empty, ok := gs.Content[2].(*GroupingSet)
	if !ok {
		t.Fatalf("expected GroupingSet for (), got %T", gs.Content[2])
	}
	if empty.Kind != GROUPING_SET_EMPTY {
		t.Errorf("expected GROUPING_SET_EMPTY, got %d", empty.Kind)
	}
}

func TestGroupByDistinct(t *testing.T) {
	s := parseOne(t, "SELECT a, sum(b) FROM t GROUP BY DISTINCT a")
	sel := s.(*SelectStmt)
	if !sel.GroupDistinct {
		t.Error("expected GroupDistinct=true")
	}
}

func TestGroupByMixed(t *testing.T) {
	s := parseOne(t, "SELECT a, b, c, sum(d) FROM t GROUP BY a, ROLLUP(b, c)")
	sel := s.(*SelectStmt)
	if len(sel.GroupClause) != 2 {
		t.Fatalf("expected 2 group items, got %d", len(sel.GroupClause))
	}
	// First is a plain column ref
	if _, ok := sel.GroupClause[0].(*ColumnRef); !ok {
		t.Errorf("expected ColumnRef for first item, got %T", sel.GroupClause[0])
	}
	// Second is ROLLUP
	gs := sel.GroupClause[1].(*GroupingSet)
	if gs.Kind != GROUPING_SET_ROLLUP {
		t.Errorf("expected ROLLUP, got %d", gs.Kind)
	}
}

func TestGroupByEmptyGroupingSet(t *testing.T) {
	s := parseOne(t, "SELECT sum(a) FROM t GROUP BY ()")
	sel := s.(*SelectStmt)
	if len(sel.GroupClause) != 1 {
		t.Fatalf("expected 1 group item, got %d", len(sel.GroupClause))
	}
	gs := sel.GroupClause[0].(*GroupingSet)
	if gs.Kind != GROUPING_SET_EMPTY {
		t.Errorf("expected GROUPING_SET_EMPTY, got %d", gs.Kind)
	}
}

func TestGroupByNestedGroupingSets(t *testing.T) {
	parseOK(t, "SELECT a, b, sum(c) FROM t GROUP BY GROUPING SETS (ROLLUP(a, b), CUBE(a, b))")
}

func TestGroupByParenthesizedExpr(t *testing.T) {
	// (a + b) in GROUP BY should be a parenthesized expression, not empty grouping set
	s := parseOne(t, "SELECT a + b, sum(c) FROM t GROUP BY (a + b)")
	sel := s.(*SelectStmt)
	if len(sel.GroupClause) != 1 {
		t.Fatalf("expected 1 group item, got %d", len(sel.GroupClause))
	}
	// Should be an A_Expr, not a GroupingSet
	if _, ok := sel.GroupClause[0].(*A_Expr); !ok {
		t.Errorf("expected A_Expr for (a+b), got %T", sel.GroupClause[0])
	}
}
