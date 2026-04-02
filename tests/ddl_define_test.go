package tests

import (
	"testing"

	"github.com/gololadb/gopgsql/parser"
)

func TestCreateAggregate(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{
			"CREATE AGGREGATE myagg (integer) (sfunc = int4pl, stype = integer, initcond = '0')",
			"new style",
		},
		{
			"CREATE AGGREGATE myagg (sfunc = int4pl, stype = integer, initcond = '0')",
			"old style",
		},
		{
			"CREATE OR REPLACE AGGREGATE myagg (integer) (sfunc = int4pl, stype = integer)",
			"or replace",
		},
		{
			"CREATE AGGREGATE myschema.myagg (integer, integer) (sfunc = myfunc, stype = integer)",
			"qualified name multiple args",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ds, ok := stmt.(*parser.DefineStmt)
			if !ok {
				t.Fatalf("expected *parser.DefineStmt, got %T", stmt)
			}
			if ds.Kind != parser.OBJECT_AGGREGATE {
				t.Fatalf("expected parser.OBJECT_AGGREGATE, got %d", ds.Kind)
			}
			if len(ds.Defnames) == 0 {
				t.Fatal("expected non-empty Defnames")
			}
			if len(ds.Definition) == 0 {
				t.Fatal("expected non-empty Definition")
			}
		})
	}
}


func TestCreateAggregateOldStyle(t *testing.T) {
	stmt := parseOne(t, "CREATE AGGREGATE myagg (sfunc = int4pl, stype = integer)")
	ds := stmt.(*parser.DefineStmt)
	if !ds.OldStyle {
		t.Fatal("expected OldStyle=true")
	}
}

// ---------------------------------------------------------------------------
// CREATE OPERATOR
// ---------------------------------------------------------------------------


func TestCreateOperator(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{
			"CREATE OPERATOR === (leftarg = integer, rightarg = integer, function = int4eq)",
			"basic",
		},
		{
			"CREATE OPERATOR myschema.=== (leftarg = integer, rightarg = integer, function = int4eq)",
			"schema qualified",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ds, ok := stmt.(*parser.DefineStmt)
			if !ok {
				t.Fatalf("expected *parser.DefineStmt, got %T", stmt)
			}
			if ds.Kind != parser.OBJECT_OPERATOR {
				t.Fatalf("expected parser.OBJECT_OPERATOR, got %d", ds.Kind)
			}
			if len(ds.Definition) == 0 {
				t.Fatal("expected non-empty Definition")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CREATE TYPE (shell and range)
// ---------------------------------------------------------------------------


func TestCreateTextSearch(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
		kind parser.ObjectType
	}{
		{
			"CREATE TEXT SEARCH PARSER myparser (start = prsd_start, gettoken = prsd_nexttoken, end = prsd_end, lextypes = prsd_lextype)",
			"parser",
			parser.OBJECT_TSPARSER,
		},
		{
			"CREATE TEXT SEARCH DICTIONARY mydict (template = simple, stopwords = english)",
			"dictionary",
			parser.OBJECT_TSDICTIONARY,
		},
		{
			"CREATE TEXT SEARCH TEMPLATE mytmpl (init = dsimple_init, lexize = dsimple_lexize)",
			"template",
			parser.OBJECT_TSTEMPLATE,
		},
		{
			"CREATE TEXT SEARCH CONFIGURATION myconf (parser = default)",
			"configuration",
			parser.OBJECT_TSCONFIGURATION,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ds, ok := stmt.(*parser.DefineStmt)
			if !ok {
				t.Fatalf("expected *parser.DefineStmt, got %T", stmt)
			}
			if ds.Kind != tt.kind {
				t.Fatalf("expected kind %d, got %d", tt.kind, ds.Kind)
			}
			if len(ds.Definition) == 0 {
				t.Fatal("expected non-empty Definition")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CREATE COLLATION
// ---------------------------------------------------------------------------


func TestCreateCollation(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE COLLATION mycoll (locale = 'en_US.utf8')", "with locale"},
		{"CREATE COLLATION mycoll FROM pg_catalog.default", "from existing"},
		{"CREATE COLLATION IF NOT EXISTS mycoll (locale = 'en_US.utf8')", "if not exists"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ds, ok := stmt.(*parser.DefineStmt)
			if !ok {
				t.Fatalf("expected *parser.DefineStmt, got %T", stmt)
			}
			if ds.Kind != parser.OBJECT_COLLATION {
				t.Fatalf("expected parser.OBJECT_COLLATION, got %d", ds.Kind)
			}
			if len(ds.Definition) == 0 {
				t.Fatal("expected non-empty Definition")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CREATE CAST
// ---------------------------------------------------------------------------


func TestCreateCast(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE CAST (integer AS text) WITH FUNCTION int4_to_text(integer)", "with function"},
		{"CREATE CAST (integer AS text) WITH FUNCTION int4_to_text(integer) AS IMPLICIT", "as implicit"},
		{"CREATE CAST (integer AS text) WITH FUNCTION int4_to_text(integer) AS ASSIGNMENT", "as assignment"},
		{"CREATE CAST (integer AS text) WITH INOUT", "with inout"},
		{"CREATE CAST (integer AS text) WITHOUT FUNCTION", "without function"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cs, ok := stmt.(*parser.CreateCastStmt)
			if !ok {
				t.Fatalf("expected *parser.CreateCastStmt, got %T", stmt)
			}
			if cs.SourceType == nil || cs.TargetType == nil {
				t.Fatal("expected non-nil SourceType and TargetType")
			}
		})
	}
}


func TestCreateCastImplicit(t *testing.T) {
	stmt := parseOne(t, "CREATE CAST (integer AS text) WITH FUNCTION f(integer) AS IMPLICIT")
	cs := stmt.(*parser.CreateCastStmt)
	if cs.Context != parser.COERCION_IMPLICIT {
		t.Fatalf("expected parser.COERCION_IMPLICIT, got %d", cs.Context)
	}
}


func TestCreateCastInout(t *testing.T) {
	stmt := parseOne(t, "CREATE CAST (integer AS text) WITH INOUT")
	cs := stmt.(*parser.CreateCastStmt)
	if !cs.Inout {
		t.Fatal("expected Inout=true")
	}
}

// ---------------------------------------------------------------------------
// CREATE TRANSFORM
// ---------------------------------------------------------------------------


func TestCreateTransform(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{
			"CREATE TRANSFORM FOR hstore LANGUAGE plpython3u (FROM SQL WITH FUNCTION hstore_to_plpython(internal), TO SQL WITH FUNCTION plpython_to_hstore(internal))",
			"basic",
		},
		{
			"CREATE OR REPLACE TRANSFORM FOR hstore LANGUAGE plpython3u (FROM SQL WITH FUNCTION f(internal), TO SQL WITH FUNCTION g(internal))",
			"or replace",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ct, ok := stmt.(*parser.CreateTransformStmt)
			if !ok {
				t.Fatalf("expected *parser.CreateTransformStmt, got %T", stmt)
			}
			if ct.TypeName == nil {
				t.Fatal("expected non-nil parser.TypeName")
			}
			if ct.Lang == "" {
				t.Fatal("expected non-empty Lang")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CREATE ACCESS METHOD
// ---------------------------------------------------------------------------


func TestCreateAccessMethod(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE ACCESS METHOD myam TYPE INDEX HANDLER myhandler", "index"},
		{"CREATE ACCESS METHOD myam TYPE TABLE HANDLER myhandler", "table"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			am, ok := stmt.(*parser.CreateAmStmt)
			if !ok {
				t.Fatalf("expected *parser.CreateAmStmt, got %T", stmt)
			}
			if am.AmName == "" {
				t.Fatal("expected non-empty AmName")
			}
			if am.AmType == "" {
				t.Fatal("expected non-empty AmType")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CREATE OPERATOR CLASS
// ---------------------------------------------------------------------------


func TestCreateOpClass(t *testing.T) {
	sql := "CREATE OPERATOR CLASS myopc DEFAULT FOR TYPE integer USING btree AS OPERATOR 1 <, OPERATOR 3 =, FUNCTION 1 btint4cmp(integer, integer)"
	stmt := parseOne(t, sql)
	oc, ok := stmt.(*parser.CreateOpClassStmt)
	if !ok {
		t.Fatalf("expected *parser.CreateOpClassStmt, got %T", stmt)
	}
	if !oc.IsDefault {
		t.Fatal("expected IsDefault=true")
	}
	if oc.AmName != "btree" {
		t.Fatalf("expected AmName=btree, got %q", oc.AmName)
	}
	if len(oc.Items) < 3 {
		t.Fatalf("expected at least 3 items, got %d", len(oc.Items))
	}
}

// ---------------------------------------------------------------------------
// CREATE OPERATOR FAMILY
// ---------------------------------------------------------------------------


func TestCreateOpFamily(t *testing.T) {
	stmt := parseOne(t, "CREATE OPERATOR FAMILY myfam USING btree")
	of, ok := stmt.(*parser.CreateOpFamilyStmt)
	if !ok {
		t.Fatalf("expected *parser.CreateOpFamilyStmt, got %T", stmt)
	}
	if of.AmName != "btree" {
		t.Fatalf("expected AmName=btree, got %q", of.AmName)
	}
}

// ---------------------------------------------------------------------------
// CREATE LANGUAGE
// ---------------------------------------------------------------------------


func TestCreateLanguage(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE LANGUAGE plmylang", "basic"},
		{"CREATE TRUSTED LANGUAGE plmylang HANDLER plmylang_call_handler", "trusted with handler"},
		{"CREATE PROCEDURAL LANGUAGE plmylang", "procedural"},
		{"CREATE OR REPLACE LANGUAGE plmylang HANDLER h INLINE i VALIDATOR v", "or replace full"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			pl, ok := stmt.(*parser.CreatePLangStmt)
			if !ok {
				t.Fatalf("expected *parser.CreatePLangStmt, got %T", stmt)
			}
			if pl.PLName == "" {
				t.Fatal("expected non-empty PLName")
			}
		})
	}
}


func TestCreateLanguageTrusted(t *testing.T) {
	stmt := parseOne(t, "CREATE TRUSTED LANGUAGE plmylang")
	pl := stmt.(*parser.CreatePLangStmt)
	if !pl.Trusted {
		t.Fatal("expected Trusted=true")
	}
}

// ---------------------------------------------------------------------------
// CREATE CONVERSION
// ---------------------------------------------------------------------------


func TestCreateConversion(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE CONVERSION myconv FOR 'UTF8' TO 'LATIN1' FROM utf8_to_iso8859_1", "basic"},
		{"CREATE DEFAULT CONVERSION myconv FOR 'UTF8' TO 'LATIN1' FROM utf8_to_iso8859_1", "default"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cc, ok := stmt.(*parser.CreateConversionStmt)
			if !ok {
				t.Fatalf("expected *parser.CreateConversionStmt, got %T", stmt)
			}
			if len(cc.ConvName) == 0 {
				t.Fatal("expected non-empty ConvName")
			}
			if cc.ForEncoding == "" {
				t.Fatal("expected non-empty ForEncoding")
			}
			if cc.ToEncoding == "" {
				t.Fatal("expected non-empty ToEncoding")
			}
		})
	}
}


func TestCreateConversionDefault(t *testing.T) {
	stmt := parseOne(t, "CREATE DEFAULT CONVERSION myconv FOR 'UTF8' TO 'LATIN1' FROM myfunc")
	cc := stmt.(*parser.CreateConversionStmt)
	if !cc.IsDefault {
		t.Fatal("expected IsDefault=true")
	}
}

