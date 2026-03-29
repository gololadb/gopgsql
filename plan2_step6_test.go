package pgscan

import "testing"

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
			dr, ok := stmt.(*DropRoleStmt)
			if !ok {
				t.Fatalf("expected *DropRoleStmt, got %T", stmt)
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
			do, ok := stmt.(*DropOwnedStmt)
			if !ok {
				t.Fatalf("expected *DropOwnedStmt, got %T", stmt)
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
			ro, ok := stmt.(*ReassignOwnedStmt)
			if !ok {
				t.Fatalf("expected *ReassignOwnedStmt, got %T", stmt)
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
		obj  ObjectType
	}{
		{"DROP FUNCTION myfunc", "basic", OBJECT_FUNCTION},
		{"DROP FUNCTION myfunc()", "empty args", OBJECT_FUNCTION},
		{"DROP FUNCTION myfunc(integer, text)", "with args", OBJECT_FUNCTION},
		{"DROP FUNCTION IF EXISTS myfunc(integer)", "if exists", OBJECT_FUNCTION},
		{"DROP FUNCTION myfunc(integer) CASCADE", "cascade", OBJECT_FUNCTION},
		{"DROP PROCEDURE myproc(integer)", "procedure", OBJECT_PROCEDURE},
		{"DROP AGGREGATE myagg(integer)", "aggregate", OBJECT_AGGREGATE},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			rf, ok := stmt.(*RemoveFuncStmt)
			if !ok {
				t.Fatalf("expected *RemoveFuncStmt, got %T", stmt)
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
