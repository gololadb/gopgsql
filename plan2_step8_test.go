package pgscan

import "testing"

func TestCreateMatView(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE MATERIALIZED VIEW mv AS SELECT 1", "basic"},
		{"CREATE MATERIALIZED VIEW IF NOT EXISTS mv AS SELECT 1", "if not exists"},
		{"CREATE MATERIALIZED VIEW mv AS SELECT * FROM t WITH DATA", "with data"},
		{"CREATE MATERIALIZED VIEW mv AS SELECT * FROM t WITH NO DATA", "with no data"},
		{"CREATE MATERIALIZED VIEW mv USING heap AS SELECT 1", "using method"},
		{"CREATE MATERIALIZED VIEW mv TABLESPACE myts AS SELECT 1", "tablespace"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cm, ok := stmt.(*CreateMatViewStmt)
			if !ok {
				t.Fatalf("expected *CreateMatViewStmt, got %T", stmt)
			}
			if cm.Relation == nil {
				t.Fatal("expected non-nil Relation")
			}
			if cm.Query == nil {
				t.Fatal("expected non-nil Query")
			}
		})
	}
}

func TestRefreshMatView(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"REFRESH MATERIALIZED VIEW mv", "basic"},
		{"REFRESH MATERIALIZED VIEW CONCURRENTLY mv", "concurrently"},
		{"REFRESH MATERIALIZED VIEW mv WITH DATA", "with data"},
		{"REFRESH MATERIALIZED VIEW mv WITH NO DATA", "with no data"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			rm, ok := stmt.(*RefreshMatViewStmt)
			if !ok {
				t.Fatalf("expected *RefreshMatViewStmt, got %T", stmt)
			}
			if rm.Relation == nil {
				t.Fatal("expected non-nil Relation")
			}
		})
	}
}

func TestCreateStatistics(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE STATISTICS mystat ON col1, col2 FROM t", "basic"},
		{"CREATE STATISTICS IF NOT EXISTS mystat ON col1, col2 FROM t", "if not exists"},
		{"CREATE STATISTICS mystat (ndistinct) ON col1, col2 FROM t", "with types"},
		{"CREATE STATISTICS mystat (ndistinct, dependencies) ON col1, col2 FROM t", "multiple types"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cs, ok := stmt.(*CreateStatsStmt)
			if !ok {
				t.Fatalf("expected *CreateStatsStmt, got %T", stmt)
			}
			if len(cs.Defnames) == 0 {
				t.Fatal("expected non-empty Defnames")
			}
			if len(cs.Exprs) == 0 {
				t.Fatal("expected non-empty Exprs")
			}
		})
	}
}
