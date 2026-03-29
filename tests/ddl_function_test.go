package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestCreateFunctionBasic(t *testing.T) {
	sql := `CREATE FUNCTION add(a integer, b integer) RETURNS integer
		LANGUAGE sql IMMUTABLE AS 'SELECT a + b'`
	s := parseOne(t, sql)
	cf, ok := s.(*parser.CreateFunctionStmt)
	if !ok {
		t.Fatalf("expected parser.CreateFunctionStmt, got %T", s)
	}
	if cf.Funcname[0] != "add" {
		t.Errorf("expected add, got %v", cf.Funcname)
	}
	if len(cf.Parameters) != 2 {
		t.Fatalf("expected 2 params, got %d", len(cf.Parameters))
	}
	if cf.Parameters[0].Name != "a" {
		t.Errorf("expected param name a, got %s", cf.Parameters[0].Name)
	}
	if cf.ReturnType == nil {
		t.Error("expected return type")
	}
}


func TestCreateOrReplaceFunction(t *testing.T) {
	sql := `CREATE OR REPLACE FUNCTION inc(x integer) RETURNS integer
		LANGUAGE sql AS 'SELECT x + 1'`
	s := parseOne(t, sql)
	cf := s.(*parser.CreateFunctionStmt)
	if !cf.Replace {
		t.Error("expected Replace=true")
	}
}


func TestCreateFunctionPlpgsql(t *testing.T) {
	sql := `CREATE FUNCTION greet(name text) RETURNS text
		LANGUAGE plpgsql AS $$
		BEGIN
			RETURN 'Hello, ' || name;
		END;
		$$`
	s := parseOne(t, sql)
	cf := s.(*parser.CreateFunctionStmt)
	if cf.Funcname[0] != "greet" {
		t.Errorf("expected greet, got %v", cf.Funcname)
	}
	// Check that body was captured
	hasBody := false
	for _, opt := range cf.Options {
		if opt.Defname == "as" {
			hasBody = true
		}
	}
	if !hasBody {
		t.Error("expected AS body option")
	}
}


func TestCreateFunctionOptions(t *testing.T) {
	sql := `CREATE FUNCTION f() RETURNS void
		LANGUAGE sql IMMUTABLE STRICT SECURITY DEFINER PARALLEL SAFE COST 100`
	s := parseOne(t, sql)
	cf := s.(*parser.CreateFunctionStmt)
	optNames := make(map[string]bool)
	for _, opt := range cf.Options {
		optNames[opt.Defname] = true
	}
	for _, expected := range []string{"language", "immutable", "strict", "security_definer", "parallel", "cost"} {
		if !optNames[expected] {
			t.Errorf("missing option %s", expected)
		}
	}
}


func TestCreateFunctionNoParams(t *testing.T) {
	sql := `CREATE FUNCTION now_utc() RETURNS timestamp LANGUAGE sql AS 'SELECT now() AT TIME ZONE ''UTC'''`
	parseOK(t, sql)
}


func TestCreateFunctionDefaultParam(t *testing.T) {
	sql := `CREATE FUNCTION f(x integer DEFAULT 0) RETURNS integer LANGUAGE sql AS 'SELECT x'`
	s := parseOne(t, sql)
	cf := s.(*parser.CreateFunctionStmt)
	if cf.Parameters[0].DefExpr == nil {
		t.Error("expected default expression")
	}
}


func TestCreateProcedure(t *testing.T) {
	sql := `CREATE PROCEDURE do_something(x integer)
		LANGUAGE sql AS 'INSERT INTO t VALUES (x)'`
	s := parseOne(t, sql)
	cf := s.(*parser.CreateFunctionStmt)
	if !cf.IsProcedure {
		t.Error("expected IsProcedure=true")
	}
}


func TestCreateFunctionReturnsTable(t *testing.T) {
	sql := `CREATE FUNCTION get_users() RETURNS TABLE (id integer, name text)
		LANGUAGE sql AS 'SELECT id, name FROM users'`
	s := parseOne(t, sql)
	cf := s.(*parser.CreateFunctionStmt)
	// RETURNS TABLE adds OUT params
	outCount := 0
	for _, p := range cf.Parameters {
		if p.Mode == parser.FUNC_PARAM_OUT {
			outCount++
		}
	}
	if outCount != 2 {
		t.Errorf("expected 2 OUT params from RETURNS TABLE, got %d", outCount)
	}
}


func TestDoStmt(t *testing.T) {
	sql := `DO $$ BEGIN RAISE NOTICE 'hello'; END $$`
	s := parseOne(t, sql)
	ds, ok := s.(*parser.DoStmt)
	if !ok {
		t.Fatalf("expected parser.DoStmt, got %T", s)
	}
	if len(ds.Args) < 1 {
		t.Fatal("expected at least 1 arg")
	}
	if ds.Args[0].Defname != "as" {
		t.Errorf("expected as, got %s", ds.Args[0].Defname)
	}
}


func TestDoStmtWithLanguage(t *testing.T) {
	sql := `DO $$ BEGIN NULL; END $$ LANGUAGE plpgsql`
	s := parseOne(t, sql)
	ds := s.(*parser.DoStmt)
	hasLang := false
	for _, a := range ds.Args {
		if a.Defname == "language" {
			hasLang = true
		}
	}
	if !hasLang {
		t.Error("expected language arg")
	}
}


func TestCallStmt(t *testing.T) {
	sql := `CALL do_something(1, 'hello')`
	s := parseOne(t, sql)
	cs, ok := s.(*parser.CallStmt)
	if !ok {
		t.Fatalf("expected parser.CallStmt, got %T", s)
	}
	if cs.FuncCall.Funcname[0] != "do_something" {
		t.Errorf("expected do_something, got %v", cs.FuncCall.Funcname)
	}
	if len(cs.FuncCall.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(cs.FuncCall.Args))
	}
}

