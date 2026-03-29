package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestDropRole(t *testing.T) {
	tests := []struct {
		sql       string
		desc      string
		missing   bool
		roleCount int
	}{
		{"DROP ROLE myrole", "basic", false, 1},
		{"DROP ROLE IF EXISTS myrole", "if exists", true, 1},
		{"DROP ROLE r1, r2, r3", "multiple", false, 3},
		{"DROP USER myuser", "drop user", false, 1},
		{"DROP GROUP mygroup", "drop group", false, 1},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			dr, ok := stmt.(*parser.DropRoleStmt)
			if !ok {
				t.Fatalf("expected *parser.DropRoleStmt, got %T", stmt)
			}
			if dr.MissingOk != tt.missing {
				t.Fatalf("expected MissingOk=%v, got %v", tt.missing, dr.MissingOk)
			}
			if len(dr.Roles) != tt.roleCount {
				t.Fatalf("expected %d roles, got %d", tt.roleCount, len(dr.Roles))
			}
		})
	}
}


func TestDropOwned(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"DROP OWNED BY alice", "basic"},
		{"DROP OWNED BY alice, bob", "multiple"},
		{"DROP OWNED BY alice CASCADE", "cascade"},
		{"DROP OWNED BY alice RESTRICT", "restrict"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			do, ok := stmt.(*parser.DropOwnedStmt)
			if !ok {
				t.Fatalf("expected *parser.DropOwnedStmt, got %T", stmt)
			}
			if len(do.Roles) == 0 {
				t.Fatal("expected non-empty Roles")
			}
		})
	}
}


func TestReassignOwned(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"REASSIGN OWNED BY alice TO bob", "basic"},
		{"REASSIGN OWNED BY alice, charlie TO bob", "multiple"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ro, ok := stmt.(*parser.ReassignOwnedStmt)
			if !ok {
				t.Fatalf("expected *parser.ReassignOwnedStmt, got %T", stmt)
			}
			if ro.NewRole == "" {
				t.Fatal("expected non-empty NewRole")
			}
		})
	}
}


func TestDropFunction(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
		obj  parser.ObjectType
	}{
		{"DROP FUNCTION myfunc", "basic", parser.OBJECT_FUNCTION},
		{"DROP FUNCTION myfunc()", "empty args", parser.OBJECT_FUNCTION},
		{"DROP FUNCTION myfunc(integer, text)", "with args", parser.OBJECT_FUNCTION},
		{"DROP FUNCTION IF EXISTS myfunc(integer)", "if exists", parser.OBJECT_FUNCTION},
		{"DROP FUNCTION myfunc(integer) CASCADE", "cascade", parser.OBJECT_FUNCTION},
		{"DROP PROCEDURE myproc(integer)", "procedure", parser.OBJECT_PROCEDURE},
		{"DROP AGGREGATE myagg(integer)", "aggregate", parser.OBJECT_AGGREGATE},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			rf, ok := stmt.(*parser.RemoveFuncStmt)
			if !ok {
				t.Fatalf("expected *parser.RemoveFuncStmt, got %T", stmt)
			}
			if rf.ObjType != tt.obj {
				t.Fatalf("expected ObjType %d, got %d", tt.obj, rf.ObjType)
			}
		})
	}
}


func TestDropStmtStillWorks(t *testing.T) {
	// Ensure existing DROP TABLE/INDEX/VIEW/etc. still work
	tests := []string{
		"DROP TABLE t",
		"DROP TABLE IF EXISTS t CASCADE",
		"DROP INDEX myidx",
		"DROP VIEW myview",
		"DROP SCHEMA myschema CASCADE",
		"DROP SEQUENCE myseq",
		"DROP TYPE mytype",
		"DROP EXTENSION hstore",
		"DROP TRIGGER mytrig ON t",
	}
	for _, sql := range tests {
		t.Run(sql, func(t *testing.T) {
			parseOne(t, sql)
		})
	}
}


func TestDropTable(t *testing.T) {
	s := parseOne(t, "DROP TABLE t")
	ds, ok := s.(*parser.DropStmt)
	if !ok {
		t.Fatalf("expected parser.DropStmt, got %T", s)
	}
	if ds.RemoveType != parser.OBJECT_TABLE {
		t.Errorf("expected parser.OBJECT_TABLE, got %d", ds.RemoveType)
	}
	if len(ds.Objects) != 1 || ds.Objects[0][0] != "t" {
		t.Errorf("expected [t], got %v", ds.Objects)
	}
}


func TestDropTableIfExists(t *testing.T) {
	s := parseOne(t, "DROP TABLE IF EXISTS t CASCADE")
	ds := s.(*parser.DropStmt)
	if !ds.MissingOk {
		t.Error("expected MissingOk=true")
	}
	if ds.Behavior != parser.DROP_CASCADE {
		t.Error("expected CASCADE")
	}
}


func TestDropMultipleTables(t *testing.T) {
	s := parseOne(t, "DROP TABLE t1, t2, t3")
	ds := s.(*parser.DropStmt)
	if len(ds.Objects) != 3 {
		t.Fatalf("expected 3 objects, got %d", len(ds.Objects))
	}
}


func TestDropIndex(t *testing.T) {
	s := parseOne(t, "DROP INDEX idx_name")
	ds := s.(*parser.DropStmt)
	if ds.RemoveType != parser.OBJECT_INDEX {
		t.Errorf("expected parser.OBJECT_INDEX, got %d", ds.RemoveType)
	}
}


func TestDropIndexConcurrently(t *testing.T) {
	s := parseOne(t, "DROP INDEX CONCURRENTLY idx_name")
	ds := s.(*parser.DropStmt)
	if !ds.Concurrent {
		t.Error("expected Concurrent=true")
	}
}


func TestDropView(t *testing.T) {
	s := parseOne(t, "DROP VIEW IF EXISTS my_view")
	ds := s.(*parser.DropStmt)
	if ds.RemoveType != parser.OBJECT_VIEW {
		t.Errorf("expected parser.OBJECT_VIEW, got %d", ds.RemoveType)
	}
}


func TestDropSchema(t *testing.T) {
	s := parseOne(t, "DROP SCHEMA myschema CASCADE")
	ds := s.(*parser.DropStmt)
	if ds.RemoveType != parser.OBJECT_SCHEMA {
		t.Errorf("expected parser.OBJECT_SCHEMA, got %d", ds.RemoveType)
	}
	if ds.Behavior != parser.DROP_CASCADE {
		t.Error("expected CASCADE")
	}
}


func TestTruncateBasic(t *testing.T) {
	s := parseOne(t, "TRUNCATE t")
	ts, ok := s.(*parser.TruncateStmt)
	if !ok {
		t.Fatalf("expected parser.TruncateStmt, got %T", s)
	}
	if len(ts.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(ts.Relations))
	}
	if ts.Relations[0].Relname != "t" {
		t.Errorf("expected t, got %s", ts.Relations[0].Relname)
	}
}


func TestTruncateTable(t *testing.T) {
	parseOK(t, "TRUNCATE TABLE t")
}


func TestTruncateMultiple(t *testing.T) {
	s := parseOne(t, "TRUNCATE t1, t2, t3 RESTART IDENTITY CASCADE")
	ts := s.(*parser.TruncateStmt)
	if len(ts.Relations) != 3 {
		t.Fatalf("expected 3 relations, got %d", len(ts.Relations))
	}
	if !ts.RestartSeqs {
		t.Error("expected RestartSeqs=true")
	}
	if ts.Behavior != parser.DROP_CASCADE {
		t.Error("expected CASCADE")
	}
}


func TestTruncateContinueIdentity(t *testing.T) {
	s := parseOne(t, "TRUNCATE t CONTINUE IDENTITY")
	ts := s.(*parser.TruncateStmt)
	if ts.RestartSeqs {
		t.Error("expected RestartSeqs=false")
	}
}

