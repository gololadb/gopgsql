package tests

import (
	"strings"
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func parseOne(t *testing.T, sql string) parser.Stmt {
	t.Helper()
	stmts, err := parser.Parse(strings.NewReader(sql), nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(stmts) == 0 {
		t.Fatal("no statements parsed")
	}
	return stmts[0].Stmt
}

func parseOK(t *testing.T, sql string) {
	t.Helper()
	_, err := parser.Parse(strings.NewReader(sql), nil)
	if err != nil {
		t.Fatalf("parse error for %q: %v", sql, err)
	}
}

func parseMulti(t *testing.T, sql string) []*parser.RawStmt {
	t.Helper()
	stmts, err := parser.Parse(strings.NewReader(sql), nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return stmts
}
