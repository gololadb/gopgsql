package pgscan

import "testing"

// ---------------------------------------------------------------------------
// OVERLAPS
// ---------------------------------------------------------------------------

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
			sel := stmt.(*SelectStmt)
			expr := sel.TargetList[0].Val
			ae, ok := expr.(*A_Expr)
			if !ok {
				t.Fatalf("expected *A_Expr, got %T", expr)
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
			sel := stmt.(*SelectStmt)
			if len(sel.FromClause) == 0 {
				t.Fatal("expected non-empty FromClause")
			}
			ts, ok := sel.FromClause[0].(*RangeTableSample)
			if !ok {
				t.Fatalf("expected *RangeTableSample, got %T", sel.FromClause[0])
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
	sel := stmt.(*SelectStmt)
	ts := sel.FromClause[0].(*RangeTableSample)
	if ts.Repeatable == nil {
		t.Fatal("expected non-nil Repeatable")
	}
}

func TestTablesampleMethod(t *testing.T) {
	stmt := parseOne(t, "SELECT * FROM t TABLESAMPLE system(50)")
	sel := stmt.(*SelectStmt)
	ts := sel.FromClause[0].(*RangeTableSample)
	if ts.Method != "system" {
		t.Fatalf("expected Method=system, got %q", ts.Method)
	}
}

// ---------------------------------------------------------------------------
// Indirection on subqueries
// ---------------------------------------------------------------------------

func TestSubqueryIndirectionField(t *testing.T) {
	stmt := parseOne(t, "SELECT (SELECT r FROM t LIMIT 1).field")
	sel := stmt.(*SelectStmt)
	expr := sel.TargetList[0].Val
	ind, ok := expr.(*A_Indirection)
	if !ok {
		t.Fatalf("expected *A_Indirection, got %T", expr)
	}
	if _, ok := ind.Arg.(*SubLink); !ok {
		t.Fatalf("expected SubLink as Arg, got %T", ind.Arg)
	}
	if len(ind.Indirection) == 0 {
		t.Fatal("expected non-empty Indirection")
	}
}

func TestSubqueryIndirectionSubscript(t *testing.T) {
	stmt := parseOne(t, "SELECT (SELECT arr FROM t LIMIT 1)[1]")
	sel := stmt.(*SelectStmt)
	expr := sel.TargetList[0].Val
	ind, ok := expr.(*A_Indirection)
	if !ok {
		t.Fatalf("expected *A_Indirection, got %T", expr)
	}
	if len(ind.Indirection) == 0 {
		t.Fatal("expected non-empty Indirection")
	}
	idx, ok := ind.Indirection[0].(*A_Indices)
	if !ok {
		t.Fatalf("expected *A_Indices, got %T", ind.Indirection[0])
	}
	if idx.Uidx == nil {
		t.Fatal("expected non-nil Uidx")
	}
}

func TestSubqueryIndirectionChained(t *testing.T) {
	stmt := parseOne(t, "SELECT (SELECT r FROM t LIMIT 1).arr[1]")
	sel := stmt.(*SelectStmt)
	expr := sel.TargetList[0].Val
	ind, ok := expr.(*A_Indirection)
	if !ok {
		t.Fatalf("expected *A_Indirection, got %T", expr)
	}
	if len(ind.Indirection) < 2 {
		t.Fatalf("expected at least 2 indirections, got %d", len(ind.Indirection))
	}
}

// ---------------------------------------------------------------------------
// Column-level GRANT
// ---------------------------------------------------------------------------

func TestGrantColumnLevel(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"GRANT SELECT (col1, col2) ON t TO myrole", "select columns"},
		{"GRANT UPDATE (col1) ON t TO myrole", "update column"},
		{"GRANT INSERT (col1, col2), SELECT (col3) ON t TO myrole", "multiple privs with cols"},
		{"GRANT SELECT ON t TO myrole", "no columns (baseline)"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			gs, ok := stmt.(*GrantStmt)
			if !ok {
				t.Fatalf("expected *GrantStmt, got %T", stmt)
			}
			if len(gs.Privileges) == 0 {
				t.Fatal("expected non-empty Privileges")
			}
		})
	}
}

func TestGrantColumnLevelCols(t *testing.T) {
	stmt := parseOne(t, "GRANT SELECT (col1, col2) ON t TO myrole")
	gs := stmt.(*GrantStmt)
	if len(gs.PrivCols) == 0 {
		t.Fatal("expected non-empty PrivCols")
	}
	if len(gs.PrivCols[0]) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(gs.PrivCols[0]))
	}
	if gs.PrivCols[0][0] != "col1" || gs.PrivCols[0][1] != "col2" {
		t.Fatalf("expected [col1, col2], got %v", gs.PrivCols[0])
	}
}

func TestRevokeColumnLevel(t *testing.T) {
	stmt := parseOne(t, "REVOKE UPDATE (col1) ON t FROM myrole")
	gs, ok := stmt.(*GrantStmt)
	if !ok {
		t.Fatalf("expected *GrantStmt, got %T", stmt)
	}
	if !gs.IsGrant == true {
		// IsGrant should be false for REVOKE
	}
	if len(gs.PrivCols) == 0 {
		t.Fatal("expected non-empty PrivCols")
	}
	if len(gs.PrivCols[0]) != 1 {
		t.Fatalf("expected 1 column, got %d", len(gs.PrivCols[0]))
	}
}
