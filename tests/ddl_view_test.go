package tests

import (
	"testing"

	"github.com/gololadb/gopgsql/parser"
)

func TestCreateView(t *testing.T) {
	s := parseOne(t, "CREATE VIEW v AS SELECT * FROM t")
	vs, ok := s.(*parser.ViewStmt)
	if !ok {
		t.Fatalf("expected parser.ViewStmt, got %T", s)
	}
	if vs.View.Relname != "v" {
		t.Errorf("expected v, got %s", vs.View.Relname)
	}
	if vs.Query == nil {
		t.Error("expected query")
	}
}


func TestCreateOrReplaceView(t *testing.T) {
	s := parseOne(t, "CREATE OR REPLACE VIEW v AS SELECT 1")
	vs := s.(*parser.ViewStmt)
	if !vs.Replace {
		t.Error("expected Replace=true")
	}
}


func TestCreateViewWithColumns(t *testing.T) {
	s := parseOne(t, "CREATE VIEW v (a, b, c) AS SELECT 1, 2, 3")
	vs := s.(*parser.ViewStmt)
	if len(vs.Aliases) != 3 {
		t.Fatalf("expected 3 aliases, got %d", len(vs.Aliases))
	}
	if vs.Aliases[0] != "a" || vs.Aliases[2] != "c" {
		t.Errorf("expected [a,b,c], got %v", vs.Aliases)
	}
}


func TestCreateViewCheckOption(t *testing.T) {
	s := parseOne(t, "CREATE VIEW v AS SELECT * FROM t WITH CHECK OPTION")
	vs := s.(*parser.ViewStmt)
	if vs.WithCheckOption != parser.CASCADED_CHECK_OPTION {
		t.Errorf("expected parser.CASCADED_CHECK_OPTION, got %d", vs.WithCheckOption)
	}
}


func TestCreateViewLocalCheckOption(t *testing.T) {
	s := parseOne(t, "CREATE VIEW v AS SELECT * FROM t WITH LOCAL CHECK OPTION")
	vs := s.(*parser.ViewStmt)
	if vs.WithCheckOption != parser.LOCAL_CHECK_OPTION {
		t.Errorf("expected parser.LOCAL_CHECK_OPTION, got %d", vs.WithCheckOption)
	}
}


func TestCreateTempView(t *testing.T) {
	s := parseOne(t, "CREATE TEMP VIEW v AS SELECT 1")
	vs := s.(*parser.ViewStmt)
	if vs.Persistence != parser.RELPERSISTENCE_TEMP {
		t.Errorf("expected TEMP, got %d", vs.Persistence)
	}
}

