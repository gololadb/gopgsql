package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestCreateIndex(t *testing.T) {
	s := parseOne(t, "CREATE INDEX idx_name ON t (col1, col2)")
	idx, ok := s.(*parser.IndexStmt)
	if !ok {
		t.Fatalf("expected parser.IndexStmt, got %T", s)
	}
	if idx.Idxname != "idx_name" {
		t.Errorf("expected idx_name, got %s", idx.Idxname)
	}
	if idx.Relation.Relname != "t" {
		t.Errorf("expected t, got %s", idx.Relation.Relname)
	}
	if len(idx.IndexParams) != 2 {
		t.Fatalf("expected 2 params, got %d", len(idx.IndexParams))
	}
	if idx.IndexParams[0].Name != "col1" {
		t.Errorf("expected col1, got %s", idx.IndexParams[0].Name)
	}
}


func TestCreateUniqueIndex(t *testing.T) {
	s := parseOne(t, "CREATE UNIQUE INDEX idx ON t (col)")
	idx := s.(*parser.IndexStmt)
	if !idx.Unique {
		t.Error("expected Unique=true")
	}
}


func TestCreateIndexConcurrently(t *testing.T) {
	s := parseOne(t, "CREATE INDEX CONCURRENTLY idx ON t (col)")
	idx := s.(*parser.IndexStmt)
	if !idx.Concurrent {
		t.Error("expected Concurrent=true")
	}
}


func TestCreateIndexIfNotExists(t *testing.T) {
	s := parseOne(t, "CREATE INDEX IF NOT EXISTS idx ON t (col)")
	idx := s.(*parser.IndexStmt)
	if !idx.IfNotExists {
		t.Error("expected IfNotExists=true")
	}
}


func TestCreateIndexUsing(t *testing.T) {
	s := parseOne(t, "CREATE INDEX idx ON t USING gin (col)")
	idx := s.(*parser.IndexStmt)
	if idx.AccessMethod != "gin" {
		t.Errorf("expected gin, got %s", idx.AccessMethod)
	}
}


func TestCreateIndexWhere(t *testing.T) {
	s := parseOne(t, "CREATE INDEX idx ON t (col) WHERE active = true")
	idx := s.(*parser.IndexStmt)
	if idx.WhereClause == nil {
		t.Error("expected WHERE clause")
	}
}


func TestCreateIndexDesc(t *testing.T) {
	s := parseOne(t, "CREATE INDEX idx ON t (col DESC NULLS LAST)")
	idx := s.(*parser.IndexStmt)
	if idx.IndexParams[0].Ordering != parser.SORTBY_DESC {
		t.Error("expected DESC")
	}
	if idx.IndexParams[0].NullsOrder != parser.SORTBY_NULLS_LAST {
		t.Error("expected NULLS LAST")
	}
}


func TestCreateIndexNoName(t *testing.T) {
	s := parseOne(t, "CREATE INDEX ON t (col)")
	idx := s.(*parser.IndexStmt)
	if idx.Idxname != "" {
		t.Errorf("expected empty name, got %s", idx.Idxname)
	}
}

