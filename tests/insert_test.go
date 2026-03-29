package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestInsertValues(t *testing.T) {
	s := parseOne(t, "INSERT INTO t (a, b) VALUES (1, 2)")
	ins := s.(*parser.InsertStmt)
	if ins.Relation.Relname != "t" {
		t.Errorf("expected table 't', got %q", ins.Relation.Relname)
	}
	if len(ins.Cols) != 2 {
		t.Errorf("expected 2 columns, got %d", len(ins.Cols))
	}
}


func TestInsertSelect(t *testing.T) {
	s := parseOne(t, "INSERT INTO t SELECT * FROM s")
	ins := s.(*parser.InsertStmt)
	if ins.SelectStmt == nil {
		t.Error("expected SELECT source")
	}
}


func TestInsertReturning(t *testing.T) {
	s := parseOne(t, "INSERT INTO t (a) VALUES (1) RETURNING *")
	ins := s.(*parser.InsertStmt)
	if len(ins.ReturningList) != 1 {
		t.Errorf("expected 1 returning item, got %d", len(ins.ReturningList))
	}
}


func TestInsertOnConflictDoNothing(t *testing.T) {
	s := parseOne(t, "INSERT INTO t (a) VALUES (1) ON CONFLICT DO NOTHING")
	ins := s.(*parser.InsertStmt)
	if ins.OnConflict == nil {
		t.Fatal("expected ON CONFLICT")
	}
	if ins.OnConflict.Action != parser.ONCONFLICT_NOTHING {
		t.Error("expected DO NOTHING")
	}
}


func TestInsertOnConflictDoUpdate(t *testing.T) {
	s := parseOne(t, "INSERT INTO t (a, b) VALUES (1, 2) ON CONFLICT (a) DO UPDATE SET b = EXCLUDED.b")
	ins := s.(*parser.InsertStmt)
	if ins.OnConflict == nil {
		t.Fatal("expected ON CONFLICT")
	}
	if ins.OnConflict.Action != parser.ONCONFLICT_UPDATE {
		t.Error("expected DO UPDATE")
	}
	if len(ins.OnConflict.TargetList) != 1 {
		t.Errorf("expected 1 SET clause, got %d", len(ins.OnConflict.TargetList))
	}
}


func TestInsertWithCTE(t *testing.T) {
	sql := "WITH src AS (SELECT 1 AS a) INSERT INTO t SELECT * FROM src"
	s := parseOne(t, sql)
	ins := s.(*parser.InsertStmt)
	if ins.WithClause == nil {
		t.Error("expected WITH clause")
	}
}

// --- UPDATE tests ---

