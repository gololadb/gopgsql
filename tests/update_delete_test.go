package tests

import (
	"testing"

	"github.com/gololadb/gopgsql/parser"
)

func TestUpdateSimple(t *testing.T) {
	s := parseOne(t, "UPDATE t SET a = 1, b = 2 WHERE id = 1")
	upd := s.(*parser.UpdateStmt)
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
	upd := s.(*parser.UpdateStmt)
	if len(upd.FromClause) != 1 {
		t.Errorf("expected 1 FROM item, got %d", len(upd.FromClause))
	}
}


func TestUpdateReturning(t *testing.T) {
	s := parseOne(t, "UPDATE t SET a = 1 RETURNING *")
	upd := s.(*parser.UpdateStmt)
	if len(upd.ReturningList) != 1 {
		t.Errorf("expected 1 returning item, got %d", len(upd.ReturningList))
	}
}

// --- DELETE tests ---


func TestDeleteSimple(t *testing.T) {
	s := parseOne(t, "DELETE FROM t WHERE id = 1")
	del := s.(*parser.DeleteStmt)
	if del.Relation.Relname != "t" {
		t.Errorf("expected table 't', got %q", del.Relation.Relname)
	}
	if del.WhereClause == nil {
		t.Error("expected WHERE clause")
	}
}


func TestDeleteUsing(t *testing.T) {
	s := parseOne(t, "DELETE FROM t USING s WHERE t.id = s.id")
	del := s.(*parser.DeleteStmt)
	if len(del.UsingClause) != 1 {
		t.Errorf("expected 1 USING item, got %d", len(del.UsingClause))
	}
}


func TestDeleteReturning(t *testing.T) {
	s := parseOne(t, "DELETE FROM t RETURNING *")
	del := s.(*parser.DeleteStmt)
	if len(del.ReturningList) != 1 {
		t.Errorf("expected 1 returning item, got %d", len(del.ReturningList))
	}
}


func TestDeleteWithCTE(t *testing.T) {
	sql := "WITH old AS (SELECT id FROM t WHERE age > 100) DELETE FROM t WHERE id IN (SELECT id FROM old)"
	s := parseOne(t, sql)
	del := s.(*parser.DeleteStmt)
	if del.WithClause == nil {
		t.Error("expected WITH clause")
	}
}

// --- Multi-statement ---

