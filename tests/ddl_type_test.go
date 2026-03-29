package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestCreateTypeShell(t *testing.T) {
	stmt := parseOne(t, "CREATE TYPE mytype")
	ds, ok := stmt.(*parser.DefineStmt)
	if !ok {
		t.Fatalf("expected *parser.DefineStmt, got %T", stmt)
	}
	if ds.Kind != parser.OBJECT_TYPE {
		t.Fatalf("expected parser.OBJECT_TYPE, got %d", ds.Kind)
	}
	if len(ds.Defnames) == 0 {
		t.Fatal("expected non-empty Defnames")
	}
	if len(ds.Definition) != 0 {
		t.Fatal("expected empty Definition for shell type")
	}
}


func TestCreateTypeRange(t *testing.T) {
	stmt := parseOne(t, "CREATE TYPE floatrange AS RANGE (subtype = float8, subtype_diff = float8mi)")
	ds, ok := stmt.(*parser.DefineStmt)
	if !ok {
		t.Fatalf("expected *parser.DefineStmt, got %T", stmt)
	}
	if ds.Kind != parser.OBJECT_TYPE {
		t.Fatalf("expected parser.OBJECT_TYPE, got %d", ds.Kind)
	}
	if len(ds.Definition) == 0 {
		t.Fatal("expected non-empty Definition for range type")
	}
}

// ---------------------------------------------------------------------------
// CREATE TEXT SEARCH
// ---------------------------------------------------------------------------


func TestCreateEnum(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE TYPE mood AS ENUM ('happy', 'sad', 'angry')", "basic enum"},
		{"CREATE TYPE status AS ENUM ('active', 'inactive')", "two-value enum"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			e, ok := stmt.(*parser.CreateEnumStmt)
			if !ok {
				t.Fatalf("expected *parser.CreateEnumStmt, got %T", stmt)
			}
			if len(e.Vals) == 0 {
				t.Fatal("expected non-empty Vals")
			}
		})
	}
}


func TestCreateCompositeType(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE TYPE address AS (street text, city text, zip integer)", "composite type"},
		{"CREATE TYPE pair AS (first integer, second integer)", "two-column composite"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			c, ok := stmt.(*parser.CompositeTypeStmt)
			if !ok {
				t.Fatalf("expected *parser.CompositeTypeStmt, got %T", stmt)
			}
			if len(c.ColDefs) == 0 {
				t.Fatal("expected non-empty ColDefs")
			}
		})
	}
}

