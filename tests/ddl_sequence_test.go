package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestCreateSequence(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE SEQUENCE myseq", "basic"},
		{"CREATE SEQUENCE IF NOT EXISTS myseq", "if not exists"},
		{"CREATE SEQUENCE myseq INCREMENT BY 2", "increment by"},
		{"CREATE SEQUENCE myseq START WITH 100", "start with"},
		{"CREATE SEQUENCE myseq MINVALUE 1 MAXVALUE 1000", "min max"},
		{"CREATE SEQUENCE myseq NO MINVALUE NO MAXVALUE", "no min no max"},
		{"CREATE SEQUENCE myseq CACHE 10", "cache"},
		{"CREATE SEQUENCE myseq CYCLE", "cycle"},
		{"CREATE SEQUENCE myseq NO CYCLE", "no cycle"},
		{"CREATE SEQUENCE myseq OWNED BY t.id", "owned by"},
		{"CREATE SEQUENCE myseq OWNED BY NONE", "owned by none"},
		{"CREATE SEQUENCE myseq AS bigint", "as type"},
		{"CREATE SEQUENCE myseq INCREMENT 5 START 10 MINVALUE 1 MAXVALUE 100 CACHE 20 CYCLE", "all options"},
		{"CREATE TEMP SEQUENCE myseq", "temp sequence"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			s, ok := stmt.(*parser.CreateSeqStmt)
			if !ok {
				t.Fatalf("expected *parser.CreateSeqStmt, got %T", stmt)
			}
			if len(s.Name) == 0 {
				t.Fatal("expected non-empty Name")
			}
		})
	}
}


func TestAlterSequence(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER SEQUENCE myseq INCREMENT BY 5", "increment"},
		{"ALTER SEQUENCE myseq RESTART", "restart"},
		{"ALTER SEQUENCE myseq RESTART WITH 1", "restart with"},
		{"ALTER SEQUENCE IF EXISTS myseq MAXVALUE 999", "if exists"},
		{"ALTER SEQUENCE myseq OWNED BY t.col", "owned by"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			s, ok := stmt.(*parser.AlterSeqStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterSeqStmt, got %T", stmt)
			}
			if len(s.Name) == 0 {
				t.Fatal("expected non-empty Name")
			}
		})
	}
}

