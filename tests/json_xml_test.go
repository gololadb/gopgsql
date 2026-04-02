package tests

import (
	"testing"

	"github.com/gololadb/gopgsql/parser"
)

func TestXmlConcat(t *testing.T) {
	s := parseOne(t, "SELECT XMLCONCAT('<a/>', '<b/>')")
	sel := s.(*parser.SelectStmt)
	if sel == nil || len(sel.TargetList) == 0 {
		t.Fatal("expected SELECT with target")
	}
	rt := sel.TargetList[0]
	xe, ok := rt.Val.(*parser.XmlExpr)
	if !ok {
		t.Fatalf("expected *parser.XmlExpr, got %T", rt.Val)
	}
	if xe.Op != parser.IS_XMLCONCAT {
		t.Fatalf("expected parser.IS_XMLCONCAT, got %d", xe.Op)
	}
	if len(xe.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(xe.Args))
	}
}


func TestXmlElement(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT XMLELEMENT(NAME foo)", "basic"},
		{"SELECT XMLELEMENT(NAME foo, 'content')", "with content"},
		{"SELECT XMLELEMENT(NAME foo, XMLATTRIBUTES(a AS bar), 'content')", "with attributes"},
		{"SELECT XMLELEMENT(NAME foo, XMLATTRIBUTES(1 AS id, 'x' AS class))", "multiple attributes"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s := parseOne(t, tt.sql)
			sel := s.(*parser.SelectStmt)
			rt := sel.TargetList[0]
			xe, ok := rt.Val.(*parser.XmlExpr)
			if !ok {
				t.Fatalf("expected *parser.XmlExpr, got %T", rt.Val)
			}
			if xe.Op != parser.IS_XMLELEMENT {
				t.Fatalf("expected parser.IS_XMLELEMENT, got %d", xe.Op)
			}
		})
	}
}


func TestXmlForest(t *testing.T) {
	s := parseOne(t, "SELECT XMLFOREST(a AS x, b AS y, c)")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	xe, ok := rt.Val.(*parser.XmlExpr)
	if !ok {
		t.Fatalf("expected *parser.XmlExpr, got %T", rt.Val)
	}
	if xe.Op != parser.IS_XMLFOREST {
		t.Fatalf("expected parser.IS_XMLFOREST, got %d", xe.Op)
	}
	if len(xe.NamedArgs) != 3 {
		t.Fatalf("expected 3 named args, got %d", len(xe.NamedArgs))
	}
}


func TestXmlParse(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT XMLPARSE(DOCUMENT '<doc/>')", "document"},
		{"SELECT XMLPARSE(CONTENT '<a/>text')", "content"},
		{"SELECT XMLPARSE(DOCUMENT '<doc/>' STRIP WHITESPACE)", "strip whitespace"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s := parseOne(t, tt.sql)
			sel := s.(*parser.SelectStmt)
			rt := sel.TargetList[0]
			xe, ok := rt.Val.(*parser.XmlExpr)
			if !ok {
				t.Fatalf("expected *parser.XmlExpr, got %T", rt.Val)
			}
			if xe.Op != parser.IS_XMLPARSE {
				t.Fatalf("expected parser.IS_XMLPARSE, got %d", xe.Op)
			}
		})
	}
}


func TestXmlPi(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT XMLPI(NAME php)", "basic"},
		{"SELECT XMLPI(NAME php, 'echo 1')", "with content"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s := parseOne(t, tt.sql)
			sel := s.(*parser.SelectStmt)
			rt := sel.TargetList[0]
			xe, ok := rt.Val.(*parser.XmlExpr)
			if !ok {
				t.Fatalf("expected *parser.XmlExpr, got %T", rt.Val)
			}
			if xe.Op != parser.IS_XMLPI {
				t.Fatalf("expected parser.IS_XMLPI, got %d", xe.Op)
			}
		})
	}
}


func TestXmlRoot(t *testing.T) {
	s := parseOne(t, "SELECT XMLROOT(x, VERSION '1.0')")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	xe, ok := rt.Val.(*parser.XmlExpr)
	if !ok {
		t.Fatalf("expected *parser.XmlExpr, got %T", rt.Val)
	}
	if xe.Op != parser.IS_XMLROOT {
		t.Fatalf("expected parser.IS_XMLROOT, got %d", xe.Op)
	}
}


func TestXmlSerialize(t *testing.T) {
	s := parseOne(t, "SELECT XMLSERIALIZE(CONTENT x AS text)")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	xe, ok := rt.Val.(*parser.XmlExpr)
	if !ok {
		t.Fatalf("expected *parser.XmlExpr, got %T", rt.Val)
	}
	if xe.Op != parser.IS_XMLSERIALIZE {
		t.Fatalf("expected parser.IS_XMLSERIALIZE, got %d", xe.Op)
	}
	if xe.TypeName == nil {
		t.Fatal("expected non-nil parser.TypeName")
	}
}


func TestXmlExists(t *testing.T) {
	s := parseOne(t, "SELECT XMLEXISTS('//foo' PASSING x)")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	xe, ok := rt.Val.(*parser.XmlExpr)
	if !ok {
		t.Fatalf("expected *parser.XmlExpr, got %T", rt.Val)
	}
	if xe.Op != parser.IS_XMLEXISTS {
		t.Fatalf("expected parser.IS_XMLEXISTS, got %d", xe.Op)
	}
}


func TestJsonObject(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT JSON_OBJECT('key1' : 'val1')", "basic"},
		{"SELECT JSON_OBJECT('a' : 1, 'b' : 2)", "multiple pairs"},
		{"SELECT JSON_OBJECT('a' : 1 RETURNING jsonb)", "with returning"},
		{"SELECT JSON_OBJECT()", "empty"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s := parseOne(t, tt.sql)
			sel := s.(*parser.SelectStmt)
			rt := sel.TargetList[0]
			_, ok := rt.Val.(*parser.JsonObjectConstructor)
			if !ok {
				t.Fatalf("expected *parser.JsonObjectConstructor, got %T", rt.Val)
			}
		})
	}
}


func TestJsonArray(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT JSON_ARRAY(1, 2, 3)", "basic"},
		{"SELECT JSON_ARRAY(1, 2 RETURNING jsonb)", "with returning"},
		{"SELECT JSON_ARRAY()", "empty"},
		{"SELECT JSON_ARRAY(1, 'hello', true)", "mixed types"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s := parseOne(t, tt.sql)
			sel := s.(*parser.SelectStmt)
			rt := sel.TargetList[0]
			_, ok := rt.Val.(*parser.JsonArrayConstructor)
			if !ok {
				t.Fatalf("expected *parser.JsonArrayConstructor, got %T", rt.Val)
			}
		})
	}
}


func TestJsonQuery(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT JSON_QUERY(doc, '$.name')", "basic"},
		{"SELECT JSON_QUERY(doc, '$.name' RETURNING text)", "with returning"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s := parseOne(t, tt.sql)
			sel := s.(*parser.SelectStmt)
			rt := sel.TargetList[0]
			jf, ok := rt.Val.(*parser.JsonFuncExpr)
			if !ok {
				t.Fatalf("expected *parser.JsonFuncExpr, got %T", rt.Val)
			}
			if jf.Op != parser.JSON_QUERY_OP {
				t.Fatalf("expected parser.JSON_QUERY_OP, got %d", jf.Op)
			}
		})
	}
}


func TestJsonValue(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT JSON_VALUE(doc, '$.id')", "basic"},
		{"SELECT JSON_VALUE(doc, '$.id' RETURNING integer)", "with returning"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s := parseOne(t, tt.sql)
			sel := s.(*parser.SelectStmt)
			rt := sel.TargetList[0]
			jf, ok := rt.Val.(*parser.JsonFuncExpr)
			if !ok {
				t.Fatalf("expected *parser.JsonFuncExpr, got %T", rt.Val)
			}
			if jf.Op != parser.JSON_VALUE_OP {
				t.Fatalf("expected parser.JSON_VALUE_OP, got %d", jf.Op)
			}
		})
	}
}


func TestJsonExistsFunc(t *testing.T) {
	s := parseOne(t, "SELECT JSON_EXISTS(doc, '$.name')")
	sel := s.(*parser.SelectStmt)
	rt := sel.TargetList[0]
	jf, ok := rt.Val.(*parser.JsonFuncExpr)
	if !ok {
		t.Fatalf("expected *parser.JsonFuncExpr, got %T", rt.Val)
	}
	if jf.Op != parser.JSON_EXISTS_OP {
		t.Fatalf("expected parser.JSON_EXISTS_OP, got %d", jf.Op)
	}
}


func TestIsJson(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT x IS JSON", "is json"},
		{"SELECT x IS NOT JSON", "is not json"},
		{"SELECT x IS JSON OBJECT", "is json object"},
		{"SELECT x IS JSON ARRAY", "is json array"},
		{"SELECT x IS JSON SCALAR", "is json scalar"},
		{"SELECT x IS JSON WITH UNIQUE KEYS", "with unique keys"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			parseOne(t, tt.sql) // just verify it parses without error
		})
	}
}


func TestXmlIsDocument(t *testing.T) {
	parseOne(t, "SELECT x IS DOCUMENT")
	parseOne(t, "SELECT x IS NOT DOCUMENT")
}


func TestJsonScalar(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT JSON_SCALAR(42)", "integer"},
		{"SELECT JSON_SCALAR('hello')", "string"},
		{"SELECT JSON_SCALAR(x) FROM t", "column ref"},
		{"SELECT JSON_SCALAR(x RETURNING text) FROM t", "with returning"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			sel := stmt.(*parser.SelectStmt)
			rt := sel.TargetList[0].Val
			js, ok := rt.(*parser.JsonScalarExpr)
			if !ok {
				t.Fatalf("expected *parser.JsonScalarExpr, got %T", rt)
			}
			if js.Expr == nil {
				t.Fatal("expected non-nil Expr")
			}
		})
	}
}


func TestJsonScalarReturning(t *testing.T) {
	stmt := parseOne(t, "SELECT JSON_SCALAR(42 RETURNING jsonb)")
	sel := stmt.(*parser.SelectStmt)
	js := sel.TargetList[0].Val.(*parser.JsonScalarExpr)
	if js.Output == nil {
		t.Fatal("expected non-nil Output")
	}
}

// ---------------------------------------------------------------------------
// JSON_SERIALIZE
// ---------------------------------------------------------------------------


func TestJsonSerialize(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT JSON_SERIALIZE('{}')", "basic"},
		{"SELECT JSON_SERIALIZE('{}' RETURNING text)", "with returning"},
		{"SELECT JSON_SERIALIZE(x FORMAT JSON) FROM t", "format json"},
		{"SELECT JSON_SERIALIZE(x FORMAT JSON RETURNING bytea) FROM t", "format and returning"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			sel := stmt.(*parser.SelectStmt)
			js, ok := sel.TargetList[0].Val.(*parser.JsonSerializeExpr)
			if !ok {
				t.Fatalf("expected *parser.JsonSerializeExpr, got %T", sel.TargetList[0].Val)
			}
			if js.Expr == nil {
				t.Fatal("expected non-nil Expr")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// JSON_OBJECTAGG
// ---------------------------------------------------------------------------


func TestJsonObjectAgg(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT JSON_OBJECTAGG(k : v) FROM t", "colon syntax"},
		{"SELECT JSON_OBJECTAGG(k VALUE v) FROM t", "value syntax"},
		{"SELECT JSON_OBJECTAGG(k : v NULL ON NULL) FROM t", "null on null"},
		{"SELECT JSON_OBJECTAGG(k : v ABSENT ON NULL) FROM t", "absent on null"},
		{"SELECT JSON_OBJECTAGG(k : v WITH UNIQUE KEYS) FROM t", "unique keys"},
		{"SELECT JSON_OBJECTAGG(k : v RETURNING jsonb) FROM t", "returning"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			sel := stmt.(*parser.SelectStmt)
			agg, ok := sel.TargetList[0].Val.(*parser.JsonObjectAgg)
			if !ok {
				t.Fatalf("expected *parser.JsonObjectAgg, got %T", sel.TargetList[0].Val)
			}
			if agg.Arg == nil {
				t.Fatal("expected non-nil Arg")
			}
		})
	}
}


func TestJsonObjectAggUniqueKeys(t *testing.T) {
	stmt := parseOne(t, "SELECT JSON_OBJECTAGG(k : v WITH UNIQUE KEYS) FROM t")
	sel := stmt.(*parser.SelectStmt)
	agg := sel.TargetList[0].Val.(*parser.JsonObjectAgg)
	if !agg.UniqueKeys {
		t.Fatal("expected UniqueKeys=true")
	}
}

// ---------------------------------------------------------------------------
// JSON_ARRAYAGG
// ---------------------------------------------------------------------------


func TestJsonArrayAgg(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SELECT JSON_ARRAYAGG(v) FROM t", "basic"},
		{"SELECT JSON_ARRAYAGG(v ORDER BY v) FROM t", "order by"},
		{"SELECT JSON_ARRAYAGG(v NULL ON NULL) FROM t", "null on null"},
		{"SELECT JSON_ARRAYAGG(v ABSENT ON NULL) FROM t", "absent on null"},
		{"SELECT JSON_ARRAYAGG(v RETURNING jsonb) FROM t", "returning"},
		{"SELECT JSON_ARRAYAGG(v ORDER BY v RETURNING jsonb) FROM t", "order by and returning"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			sel := stmt.(*parser.SelectStmt)
			agg, ok := sel.TargetList[0].Val.(*parser.JsonArrayAgg)
			if !ok {
				t.Fatalf("expected *parser.JsonArrayAgg, got %T", sel.TargetList[0].Val)
			}
			if agg.Arg == nil {
				t.Fatal("expected non-nil Arg")
			}
		})
	}
}


func TestJsonArrayAggOrderBy(t *testing.T) {
	stmt := parseOne(t, "SELECT JSON_ARRAYAGG(v ORDER BY v DESC) FROM t")
	sel := stmt.(*parser.SelectStmt)
	agg := sel.TargetList[0].Val.(*parser.JsonArrayAgg)
	if len(agg.Order) == 0 {
		t.Fatal("expected non-empty Order")
	}
}

// ---------------------------------------------------------------------------
// JSON_TABLE
// ---------------------------------------------------------------------------


func TestJsonTable(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{
			"SELECT * FROM JSON_TABLE('{\"a\":1}', '$' COLUMNS (a int PATH '$.a')) AS jt",
			"basic",
		},
		{
			"SELECT * FROM JSON_TABLE('{\"items\":[]}', '$.items[*]' COLUMNS (id int PATH '$.id', name text PATH '$.name')) AS jt",
			"multiple columns",
		},
		{
			"SELECT * FROM JSON_TABLE('{\"a\":1}', '$' COLUMNS (rn FOR ORDINALITY, a int PATH '$.a')) AS jt",
			"for ordinality",
		},
		{
			"SELECT * FROM JSON_TABLE('{\"a\":1}', '$' COLUMNS (a int EXISTS PATH '$.a')) AS jt",
			"exists",
		},
		{
			"SELECT * FROM JSON_TABLE(doc, '$' PASSING x AS y COLUMNS (a int PATH '$.a')) AS jt",
			"passing",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			sel := stmt.(*parser.SelectStmt)
			if len(sel.FromClause) == 0 {
				t.Fatal("expected non-empty FromClause")
			}
			jt, ok := sel.FromClause[0].(*parser.JsonTable)
			if !ok {
				t.Fatalf("expected *parser.JsonTable, got %T", sel.FromClause[0])
			}
			if jt.Expr == nil {
				t.Fatal("expected non-nil Expr")
			}
			if len(jt.Columns) == 0 {
				t.Fatal("expected non-empty Columns")
			}
		})
	}
}


func TestJsonTableNested(t *testing.T) {
	sql := "SELECT * FROM JSON_TABLE(doc, '$' COLUMNS (a int PATH '$.a', NESTED '$.items[*]' COLUMNS (b text PATH '$.b'))) AS jt"
	stmt := parseOne(t, sql)
	sel := stmt.(*parser.SelectStmt)
	jt := sel.FromClause[0].(*parser.JsonTable)
	if len(jt.Columns) < 2 {
		t.Fatalf("expected at least 2 columns, got %d", len(jt.Columns))
	}
	nested := jt.Columns[1]
	if nested.Coltype != parser.JTC_NESTED {
		t.Fatalf("expected parser.JTC_NESTED, got %d", nested.Coltype)
	}
	if len(nested.Columns) == 0 {
		t.Fatal("expected nested columns")
	}
}


func TestJsonTableLateral(t *testing.T) {
	sql := "SELECT * FROM t, LATERAL JSON_TABLE(t.doc, '$' COLUMNS (a int PATH '$.a')) AS jt"
	stmt := parseOne(t, sql)
	sel := stmt.(*parser.SelectStmt)
	// Find the parser.JsonTable in the from clause
	found := false
	for _, f := range sel.FromClause {
		if jt, ok := f.(*parser.JsonTable); ok {
			if !jt.Lateral {
				t.Fatal("expected Lateral=true")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("expected parser.JsonTable in FromClause")
	}
}

// ---------------------------------------------------------------------------
// XMLTABLE
// ---------------------------------------------------------------------------


func TestXmlTable(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{
			"SELECT * FROM XMLTABLE('/root/row' PASSING data COLUMNS id int PATH '@id', val text PATH 'val') AS xt",
			"basic",
		},
		{
			"SELECT * FROM XMLTABLE('/root/row' PASSING data COLUMNS rn FOR ORDINALITY, val text PATH 'val') AS xt",
			"for ordinality",
		},
		{
			"SELECT * FROM XMLTABLE('/root/row' PASSING data COLUMNS val text PATH 'val' DEFAULT 'none' NOT NULL) AS xt",
			"default and not null",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			sel := stmt.(*parser.SelectStmt)
			if len(sel.FromClause) == 0 {
				t.Fatal("expected non-empty FromClause")
			}
			xt, ok := sel.FromClause[0].(*parser.XmlTable)
			if !ok {
				t.Fatalf("expected *parser.XmlTable, got %T", sel.FromClause[0])
			}
			if xt.Xmlexpr == nil {
				t.Fatal("expected non-nil Xmlexpr")
			}
			if xt.Docexpr == nil {
				t.Fatal("expected non-nil Docexpr")
			}
			if len(xt.Columns) == 0 {
				t.Fatal("expected non-empty Columns")
			}
		})
	}
}


func TestXmlTableDefaultNotNull(t *testing.T) {
	sql := "SELECT * FROM XMLTABLE('/r' PASSING d COLUMNS val text PATH 'v' DEFAULT 'x' NOT NULL) AS xt"
	stmt := parseOne(t, sql)
	sel := stmt.(*parser.SelectStmt)
	xt := sel.FromClause[0].(*parser.XmlTable)
	col := xt.Columns[0]
	if col.DefExpr == nil {
		t.Fatal("expected non-nil DefExpr")
	}
	if !col.IsNotNull {
		t.Fatal("expected IsNotNull=true")
	}
}


func TestXmlTableLateral(t *testing.T) {
	sql := "SELECT * FROM t, LATERAL XMLTABLE('/r' PASSING t.doc COLUMNS val text PATH 'v') AS xt"
	stmt := parseOne(t, sql)
	sel := stmt.(*parser.SelectStmt)
	found := false
	for _, f := range sel.FromClause {
		if xt, ok := f.(*parser.XmlTable); ok {
			if !xt.Lateral {
				t.Fatal("expected Lateral=true")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("expected parser.XmlTable in FromClause")
	}
}

