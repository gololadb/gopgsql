package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestExtract(t *testing.T) {
	s := parseOne(t, "SELECT EXTRACT(YEAR FROM d)")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if fc.Funcname[1] != "extract" {
		t.Errorf("expected extract, got %v", fc.Funcname)
	}
	if len(fc.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(fc.Args))
	}
	field := fc.Args[0].(*parser.A_Const)
	if field.Val.Str != "year" {
		t.Errorf("expected 'year', got %q", field.Val.Str)
	}
}


func TestExtractEpoch(t *testing.T) {
	parseOK(t, "SELECT EXTRACT(EPOCH FROM now())")
}


func TestPosition(t *testing.T) {
	s := parseOne(t, "SELECT POSITION('x' IN 'abcxdef')")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if fc.Funcname[1] != "position" {
		t.Errorf("expected position, got %v", fc.Funcname)
	}
	// Args should be (haystack, needle) — reversed from SQL syntax
	if len(fc.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(fc.Args))
	}
}


func TestSubstringFromFor(t *testing.T) {
	s := parseOne(t, "SELECT SUBSTRING('hello' FROM 2 FOR 3)")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if fc.Funcname[1] != "substring" {
		t.Errorf("expected substring, got %v", fc.Funcname)
	}
	if len(fc.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(fc.Args))
	}
}


func TestSubstringFrom(t *testing.T) {
	s := parseOne(t, "SELECT SUBSTRING('hello' FROM 2)")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if len(fc.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(fc.Args))
	}
}


func TestSubstringPlain(t *testing.T) {
	s := parseOne(t, "SELECT SUBSTRING('hello', 2, 3)")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if fc.Funcname[0] != "substring" {
		t.Errorf("expected plain substring, got %v", fc.Funcname)
	}
	if len(fc.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(fc.Args))
	}
}


func TestOverlay(t *testing.T) {
	s := parseOne(t, "SELECT OVERLAY('hello' PLACING 'XX' FROM 2 FOR 3)")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if fc.Funcname[1] != "overlay" {
		t.Errorf("expected overlay, got %v", fc.Funcname)
	}
	if len(fc.Args) != 4 {
		t.Errorf("expected 4 args, got %d", len(fc.Args))
	}
}


func TestOverlayNoFor(t *testing.T) {
	s := parseOne(t, "SELECT OVERLAY('hello' PLACING 'XX' FROM 2)")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if len(fc.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(fc.Args))
	}
}


func TestTrimBoth(t *testing.T) {
	s := parseOne(t, "SELECT TRIM(BOTH 'x' FROM 'xxxhelloxxx')")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if fc.Funcname[1] != "btrim" {
		t.Errorf("expected btrim, got %v", fc.Funcname)
	}
}


func TestTrimLeading(t *testing.T) {
	s := parseOne(t, "SELECT TRIM(LEADING 'x' FROM 'xxxhello')")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if fc.Funcname[1] != "ltrim" {
		t.Errorf("expected ltrim, got %v", fc.Funcname)
	}
}


func TestTrimTrailing(t *testing.T) {
	s := parseOne(t, "SELECT TRIM(TRAILING 'x' FROM 'helloxxx')")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if fc.Funcname[1] != "rtrim" {
		t.Errorf("expected rtrim, got %v", fc.Funcname)
	}
}


func TestTrimDefault(t *testing.T) {
	s := parseOne(t, "SELECT TRIM('  hello  ')")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if fc.Funcname[1] != "btrim" {
		t.Errorf("expected btrim, got %v", fc.Funcname)
	}
}


func TestTreat(t *testing.T) {
	parseOK(t, "SELECT TREAT(x AS integer)")
}


func TestNormalize(t *testing.T) {
	s := parseOne(t, "SELECT NORMALIZE('hello')")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if fc.Funcname[1] != "normalize" {
		t.Errorf("expected normalize, got %v", fc.Funcname)
	}
}


func TestNormalizeWithForm(t *testing.T) {
	s := parseOne(t, "SELECT NORMALIZE('hello', NFC)")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if len(fc.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(fc.Args))
	}
}


func TestCollationFor(t *testing.T) {
	s := parseOne(t, "SELECT COLLATION FOR ('hello')")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	if fc.Funcname[1] != "pg_collation_for" {
		t.Errorf("expected pg_collation_for, got %v", fc.Funcname)
	}
}

// --- Step 2: SQL value functions ---


func TestCurrentDate(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_DATE")
	sel := s.(*parser.SelectStmt)
	svf := sel.TargetList[0].Val.(*parser.SQLValueFunction)
	if svf.Op != parser.SVFOP_CURRENT_DATE {
		t.Errorf("expected CURRENT_DATE, got %v", svf.Op)
	}
}


func TestCurrentTimestamp(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_TIMESTAMP")
	sel := s.(*parser.SelectStmt)
	svf := sel.TargetList[0].Val.(*parser.SQLValueFunction)
	if svf.Op != parser.SVFOP_CURRENT_TIMESTAMP {
		t.Errorf("expected CURRENT_TIMESTAMP, got %v", svf.Op)
	}
}


func TestCurrentTimestampPrecision(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_TIMESTAMP(3)")
	sel := s.(*parser.SelectStmt)
	svf := sel.TargetList[0].Val.(*parser.SQLValueFunction)
	if svf.Op != parser.SVFOP_CURRENT_TIMESTAMP_N || svf.Typmod != 3 {
		t.Errorf("expected CURRENT_TIMESTAMP_N(3), got op=%v typmod=%d", svf.Op, svf.Typmod)
	}
}


func TestCurrentTime(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_TIME")
	sel := s.(*parser.SelectStmt)
	svf := sel.TargetList[0].Val.(*parser.SQLValueFunction)
	if svf.Op != parser.SVFOP_CURRENT_TIME {
		t.Errorf("expected CURRENT_TIME, got %v", svf.Op)
	}
}


func TestLocaltime(t *testing.T) {
	s := parseOne(t, "SELECT LOCALTIME")
	sel := s.(*parser.SelectStmt)
	svf := sel.TargetList[0].Val.(*parser.SQLValueFunction)
	if svf.Op != parser.SVFOP_LOCALTIME {
		t.Errorf("expected LOCALTIME, got %v", svf.Op)
	}
}


func TestLocaltimestamp(t *testing.T) {
	s := parseOne(t, "SELECT LOCALTIMESTAMP(6)")
	sel := s.(*parser.SelectStmt)
	svf := sel.TargetList[0].Val.(*parser.SQLValueFunction)
	if svf.Op != parser.SVFOP_LOCALTIMESTAMP_N || svf.Typmod != 6 {
		t.Errorf("expected LOCALTIMESTAMP_N(6), got op=%v typmod=%d", svf.Op, svf.Typmod)
	}
}


func TestCurrentUser(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_USER")
	sel := s.(*parser.SelectStmt)
	svf := sel.TargetList[0].Val.(*parser.SQLValueFunction)
	if svf.Op != parser.SVFOP_CURRENT_USER {
		t.Errorf("expected CURRENT_USER, got %v", svf.Op)
	}
}


func TestSessionUser(t *testing.T) {
	s := parseOne(t, "SELECT SESSION_USER")
	sel := s.(*parser.SelectStmt)
	svf := sel.TargetList[0].Val.(*parser.SQLValueFunction)
	if svf.Op != parser.SVFOP_SESSION_USER {
		t.Errorf("expected SESSION_USER, got %v", svf.Op)
	}
}


func TestCurrentRole(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_ROLE")
	sel := s.(*parser.SelectStmt)
	svf := sel.TargetList[0].Val.(*parser.SQLValueFunction)
	if svf.Op != parser.SVFOP_CURRENT_ROLE {
		t.Errorf("expected CURRENT_ROLE, got %v", svf.Op)
	}
}


func TestCurrentCatalog(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_CATALOG")
	sel := s.(*parser.SelectStmt)
	svf := sel.TargetList[0].Val.(*parser.SQLValueFunction)
	if svf.Op != parser.SVFOP_CURRENT_CATALOG {
		t.Errorf("expected CURRENT_CATALOG, got %v", svf.Op)
	}
}


func TestCurrentSchema(t *testing.T) {
	s := parseOne(t, "SELECT CURRENT_SCHEMA")
	sel := s.(*parser.SelectStmt)
	svf := sel.TargetList[0].Val.(*parser.SQLValueFunction)
	if svf.Op != parser.SVFOP_CURRENT_SCHEMA {
		t.Errorf("expected CURRENT_SCHEMA, got %v", svf.Op)
	}
}


func TestCurrentSchemaParens(t *testing.T) {
	parseOK(t, "SELECT CURRENT_SCHEMA()")
}


func TestGroupingFunc(t *testing.T) {
	s := parseOne(t, "SELECT GROUPING(a, b)")
	sel := s.(*parser.SelectStmt)
	gf := sel.TargetList[0].Val.(*parser.GroupingFunc)
	if len(gf.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(gf.Args))
	}
}


func TestSetToDefault(t *testing.T) {
	s := parseOne(t, "INSERT INTO t (a) VALUES (DEFAULT)")
	ins := s.(*parser.InsertStmt)
	sel := ins.SelectStmt.(*parser.SelectStmt)
	val := sel.ValuesLists[0][0]
	if _, ok := val.(*parser.SetToDefault); !ok {
		t.Errorf("expected parser.SetToDefault, got %T", val)
	}
}


func TestValFuncsInExpr(t *testing.T) {
	parseOK(t, "SELECT * FROM t WHERE created_at > CURRENT_TIMESTAMP - 1")
}

// --- Step 3: Operator forms, ESCAPE, IS DOCUMENT/NORMALIZED, AT LOCAL, | ---

