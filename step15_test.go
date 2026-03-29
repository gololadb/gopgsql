package pgscan

import "testing"

func TestXmlConcat(t *testing.T) {
	s := parseOne(t, "SELECT XMLCONCAT('<a/>', '<b/>')")
	sel := s.(*SelectStmt)
	if sel == nil || len(sel.TargetList) == 0 {
		t.Fatal("expected SELECT with target")
	}
	rt := sel.TargetList[0]
	xe, ok := rt.Val.(*XmlExpr)
	if !ok {
		t.Fatalf("expected *XmlExpr, got %T", rt.Val)
	}
	if xe.Op != IS_XMLCONCAT {
		t.Fatalf("expected IS_XMLCONCAT, got %d", xe.Op)
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
			sel := s.(*SelectStmt)
			rt := sel.TargetList[0]
			xe, ok := rt.Val.(*XmlExpr)
			if !ok {
				t.Fatalf("expected *XmlExpr, got %T", rt.Val)
			}
			if xe.Op != IS_XMLELEMENT {
				t.Fatalf("expected IS_XMLELEMENT, got %d", xe.Op)
			}
		})
	}
}

func TestXmlForest(t *testing.T) {
	s := parseOne(t, "SELECT XMLFOREST(a AS x, b AS y, c)")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	xe, ok := rt.Val.(*XmlExpr)
	if !ok {
		t.Fatalf("expected *XmlExpr, got %T", rt.Val)
	}
	if xe.Op != IS_XMLFOREST {
		t.Fatalf("expected IS_XMLFOREST, got %d", xe.Op)
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
			sel := s.(*SelectStmt)
			rt := sel.TargetList[0]
			xe, ok := rt.Val.(*XmlExpr)
			if !ok {
				t.Fatalf("expected *XmlExpr, got %T", rt.Val)
			}
			if xe.Op != IS_XMLPARSE {
				t.Fatalf("expected IS_XMLPARSE, got %d", xe.Op)
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
			sel := s.(*SelectStmt)
			rt := sel.TargetList[0]
			xe, ok := rt.Val.(*XmlExpr)
			if !ok {
				t.Fatalf("expected *XmlExpr, got %T", rt.Val)
			}
			if xe.Op != IS_XMLPI {
				t.Fatalf("expected IS_XMLPI, got %d", xe.Op)
			}
		})
	}
}

func TestXmlRoot(t *testing.T) {
	s := parseOne(t, "SELECT XMLROOT(x, VERSION '1.0')")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	xe, ok := rt.Val.(*XmlExpr)
	if !ok {
		t.Fatalf("expected *XmlExpr, got %T", rt.Val)
	}
	if xe.Op != IS_XMLROOT {
		t.Fatalf("expected IS_XMLROOT, got %d", xe.Op)
	}
}

func TestXmlSerialize(t *testing.T) {
	s := parseOne(t, "SELECT XMLSERIALIZE(CONTENT x AS text)")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	xe, ok := rt.Val.(*XmlExpr)
	if !ok {
		t.Fatalf("expected *XmlExpr, got %T", rt.Val)
	}
	if xe.Op != IS_XMLSERIALIZE {
		t.Fatalf("expected IS_XMLSERIALIZE, got %d", xe.Op)
	}
	if xe.TypeName == nil {
		t.Fatal("expected non-nil TypeName")
	}
}

func TestXmlExists(t *testing.T) {
	s := parseOne(t, "SELECT XMLEXISTS('//foo' PASSING x)")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	xe, ok := rt.Val.(*XmlExpr)
	if !ok {
		t.Fatalf("expected *XmlExpr, got %T", rt.Val)
	}
	if xe.Op != IS_XMLEXISTS {
		t.Fatalf("expected IS_XMLEXISTS, got %d", xe.Op)
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
			sel := s.(*SelectStmt)
			rt := sel.TargetList[0]
			_, ok := rt.Val.(*JsonObjectConstructor)
			if !ok {
				t.Fatalf("expected *JsonObjectConstructor, got %T", rt.Val)
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
			sel := s.(*SelectStmt)
			rt := sel.TargetList[0]
			_, ok := rt.Val.(*JsonArrayConstructor)
			if !ok {
				t.Fatalf("expected *JsonArrayConstructor, got %T", rt.Val)
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
			sel := s.(*SelectStmt)
			rt := sel.TargetList[0]
			jf, ok := rt.Val.(*JsonFuncExpr)
			if !ok {
				t.Fatalf("expected *JsonFuncExpr, got %T", rt.Val)
			}
			if jf.Op != JSON_QUERY_OP {
				t.Fatalf("expected JSON_QUERY_OP, got %d", jf.Op)
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
			sel := s.(*SelectStmt)
			rt := sel.TargetList[0]
			jf, ok := rt.Val.(*JsonFuncExpr)
			if !ok {
				t.Fatalf("expected *JsonFuncExpr, got %T", rt.Val)
			}
			if jf.Op != JSON_VALUE_OP {
				t.Fatalf("expected JSON_VALUE_OP, got %d", jf.Op)
			}
		})
	}
}

func TestJsonExistsFunc(t *testing.T) {
	s := parseOne(t, "SELECT JSON_EXISTS(doc, '$.name')")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	jf, ok := rt.Val.(*JsonFuncExpr)
	if !ok {
		t.Fatalf("expected *JsonFuncExpr, got %T", rt.Val)
	}
	if jf.Op != JSON_EXISTS_OP {
		t.Fatalf("expected JSON_EXISTS_OP, got %d", jf.Op)
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
