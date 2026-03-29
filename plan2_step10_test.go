package pgscan

import "testing"

// ---------------------------------------------------------------------------
// JSON_SCALAR
// ---------------------------------------------------------------------------

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
			sel := stmt.(*SelectStmt)
			rt := sel.TargetList[0].Val
			js, ok := rt.(*JsonScalarExpr)
			if !ok {
				t.Fatalf("expected *JsonScalarExpr, got %T", rt)
			}
			if js.Expr == nil {
				t.Fatal("expected non-nil Expr")
			}
		})
	}
}

func TestJsonScalarReturning(t *testing.T) {
	stmt := parseOne(t, "SELECT JSON_SCALAR(42 RETURNING jsonb)")
	sel := stmt.(*SelectStmt)
	js := sel.TargetList[0].Val.(*JsonScalarExpr)
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
			sel := stmt.(*SelectStmt)
			js, ok := sel.TargetList[0].Val.(*JsonSerializeExpr)
			if !ok {
				t.Fatalf("expected *JsonSerializeExpr, got %T", sel.TargetList[0].Val)
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
			sel := stmt.(*SelectStmt)
			agg, ok := sel.TargetList[0].Val.(*JsonObjectAgg)
			if !ok {
				t.Fatalf("expected *JsonObjectAgg, got %T", sel.TargetList[0].Val)
			}
			if agg.Arg == nil {
				t.Fatal("expected non-nil Arg")
			}
		})
	}
}

func TestJsonObjectAggUniqueKeys(t *testing.T) {
	stmt := parseOne(t, "SELECT JSON_OBJECTAGG(k : v WITH UNIQUE KEYS) FROM t")
	sel := stmt.(*SelectStmt)
	agg := sel.TargetList[0].Val.(*JsonObjectAgg)
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
			sel := stmt.(*SelectStmt)
			agg, ok := sel.TargetList[0].Val.(*JsonArrayAgg)
			if !ok {
				t.Fatalf("expected *JsonArrayAgg, got %T", sel.TargetList[0].Val)
			}
			if agg.Arg == nil {
				t.Fatal("expected non-nil Arg")
			}
		})
	}
}

func TestJsonArrayAggOrderBy(t *testing.T) {
	stmt := parseOne(t, "SELECT JSON_ARRAYAGG(v ORDER BY v DESC) FROM t")
	sel := stmt.(*SelectStmt)
	agg := sel.TargetList[0].Val.(*JsonArrayAgg)
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
			sel := stmt.(*SelectStmt)
			if len(sel.FromClause) == 0 {
				t.Fatal("expected non-empty FromClause")
			}
			jt, ok := sel.FromClause[0].(*JsonTable)
			if !ok {
				t.Fatalf("expected *JsonTable, got %T", sel.FromClause[0])
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
	sel := stmt.(*SelectStmt)
	jt := sel.FromClause[0].(*JsonTable)
	if len(jt.Columns) < 2 {
		t.Fatalf("expected at least 2 columns, got %d", len(jt.Columns))
	}
	nested := jt.Columns[1]
	if nested.Coltype != JTC_NESTED {
		t.Fatalf("expected JTC_NESTED, got %d", nested.Coltype)
	}
	if len(nested.Columns) == 0 {
		t.Fatal("expected nested columns")
	}
}

func TestJsonTableLateral(t *testing.T) {
	sql := "SELECT * FROM t, LATERAL JSON_TABLE(t.doc, '$' COLUMNS (a int PATH '$.a')) AS jt"
	stmt := parseOne(t, sql)
	sel := stmt.(*SelectStmt)
	// Find the JsonTable in the from clause
	found := false
	for _, f := range sel.FromClause {
		if jt, ok := f.(*JsonTable); ok {
			if !jt.Lateral {
				t.Fatal("expected Lateral=true")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("expected JsonTable in FromClause")
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
			sel := stmt.(*SelectStmt)
			if len(sel.FromClause) == 0 {
				t.Fatal("expected non-empty FromClause")
			}
			xt, ok := sel.FromClause[0].(*XmlTable)
			if !ok {
				t.Fatalf("expected *XmlTable, got %T", sel.FromClause[0])
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
	sel := stmt.(*SelectStmt)
	xt := sel.FromClause[0].(*XmlTable)
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
	sel := stmt.(*SelectStmt)
	found := false
	for _, f := range sel.FromClause {
		if xt, ok := f.(*XmlTable); ok {
			if !xt.Lateral {
				t.Fatal("expected Lateral=true")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("expected XmlTable in FromClause")
	}
}
