package tests

import (
	"testing"

	"github.com/gololadb/gopgsql/parser"
)

func TestCommentOn(t *testing.T) {
	tests := []struct {
		sql     string
		desc    string
		objType parser.ObjectType
	}{
		{"COMMENT ON TABLE t IS 'a table'", "table", parser.OBJECT_TABLE},
		{"COMMENT ON COLUMN t.c IS 'a column'", "column", parser.OBJECT_COLUMN},
		{"COMMENT ON INDEX idx IS 'an index'", "index", parser.OBJECT_INDEX},
		{"COMMENT ON VIEW v IS 'a view'", "view", parser.OBJECT_VIEW},
		{"COMMENT ON SEQUENCE s IS 'a seq'", "sequence", parser.OBJECT_SEQUENCE},
		{"COMMENT ON TYPE mytype IS 'a type'", "type", parser.OBJECT_TYPE},
		{"COMMENT ON SCHEMA public IS 'public schema'", "schema", parser.OBJECT_SCHEMA},
		{"COMMENT ON FUNCTION myfunc IS 'a func'", "function", parser.OBJECT_FUNCTION},
		{"COMMENT ON PROCEDURE myproc IS 'a proc'", "procedure", parser.OBJECT_PROCEDURE},
		{"COMMENT ON EXTENSION ext IS 'an ext'", "extension", parser.OBJECT_EXTENSION},
		{"COMMENT ON TRIGGER trg ON t IS 'a trigger'", "trigger", parser.OBJECT_TRIGGER},
		{"COMMENT ON RULE rl ON t IS 'a rule'", "rule", parser.OBJECT_RULE},
		{"COMMENT ON DOMAIN d IS 'a domain'", "domain", parser.OBJECT_DOMAIN},
		{"COMMENT ON DATABASE mydb IS 'a db'", "database", parser.OBJECT_DATABASE},
		{"COMMENT ON ROLE myrole IS 'a role'", "role", parser.OBJECT_ROLE},
		{"COMMENT ON TABLESPACE ts IS 'a ts'", "tablespace", parser.OBJECT_TABLESPACE},
		{"COMMENT ON POLICY pol ON t IS 'a policy'", "policy", parser.OBJECT_POLICY},
		{"COMMENT ON PUBLICATION pub IS 'a pub'", "publication", parser.OBJECT_PUBLICATION},
		{"COMMENT ON SUBSCRIPTION sub IS 'a sub'", "subscription", parser.OBJECT_SUBSCRIPTION},
		{"COMMENT ON AGGREGATE myagg IS 'an agg'", "aggregate", parser.OBJECT_AGGREGATE},
		{"COMMENT ON OPERATOR myop IS 'an op'", "operator", parser.OBJECT_OPERATOR},
		{"COMMENT ON COLLATION mycoll IS 'a coll'", "collation", parser.OBJECT_COLLATION},
		{"COMMENT ON CONVERSION myconv IS 'a conv'", "conversion", parser.OBJECT_CONVERSION},
		{"COMMENT ON LANGUAGE plpgsql IS 'a lang'", "language", parser.OBJECT_LANGUAGE},
		{"COMMENT ON EVENT TRIGGER evt IS 'an evt'", "event trigger", parser.OBJECT_EVENT_TRIGGER},
		{"COMMENT ON ACCESS METHOD btree IS 'btree'", "access method", parser.OBJECT_ACCESS_METHOD},
		{"COMMENT ON FOREIGN TABLE ft IS 'a ft'", "foreign table", parser.OBJECT_FOREIGN_TABLE},
		{"COMMENT ON FOREIGN DATA WRAPPER myfdw IS 'a fdw'", "fdw", parser.OBJECT_FDW},
		{"COMMENT ON SERVER mysrv IS 'a server'", "server", parser.OBJECT_FOREIGN_SERVER},
		{"COMMENT ON MATERIALIZED VIEW mv IS 'a mv'", "matview", parser.OBJECT_MATVIEW},
		{"COMMENT ON TEXT SEARCH PARSER myparser IS 'a parser'", "ts parser", parser.OBJECT_TSPARSER},
		{"COMMENT ON TEXT SEARCH DICTIONARY mydict IS 'a dict'", "ts dictionary", parser.OBJECT_TSDICTIONARY},
		{"COMMENT ON TEXT SEARCH TEMPLATE mytmpl IS 'a tmpl'", "ts template", parser.OBJECT_TSTEMPLATE},
		{"COMMENT ON TEXT SEARCH CONFIGURATION myconf IS 'a conf'", "ts configuration", parser.OBJECT_TSCONFIGURATION},
		{"COMMENT ON STATISTICS mystat IS 'a stat'", "statistics", parser.OBJECT_STATISTICS},
		{"COMMENT ON CONSTRAINT mycon ON t IS 'a constraint'", "constraint", parser.OBJECT_CONSTRAINT},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cs, ok := stmt.(*parser.CommentStmt)
			if !ok {
				t.Fatalf("expected *parser.CommentStmt, got %T", stmt)
			}
			if cs.ObjType != tt.objType {
				t.Fatalf("expected ObjType %d, got %d", tt.objType, cs.ObjType)
			}
			if len(cs.Object) == 0 {
				t.Fatal("expected non-empty Object")
			}
		})
	}
}


func TestCommentOnNull(t *testing.T) {
	stmt := parseOne(t, "COMMENT ON TABLE t IS NULL")
	cs, ok := stmt.(*parser.CommentStmt)
	if !ok {
		t.Fatalf("expected *parser.CommentStmt, got %T", stmt)
	}
	if !cs.IsNull {
		t.Fatal("expected IsNull=true")
	}
}

// ---------------------------------------------------------------------------
// SECURITY LABEL
// ---------------------------------------------------------------------------


func TestSecurityLabel(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"SECURITY LABEL ON TABLE t IS 'secret'", "basic"},
		{"SECURITY LABEL FOR selinux ON TABLE t IS 'secret'", "with provider"},
		{"SECURITY LABEL ON FUNCTION f IS 'label'", "function"},
		{"SECURITY LABEL ON SCHEMA s IS NULL", "null label"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			sl, ok := stmt.(*parser.SecLabelStmt)
			if !ok {
				t.Fatalf("expected *parser.SecLabelStmt, got %T", stmt)
			}
			if len(sl.Object) == 0 {
				t.Fatal("expected non-empty Object")
			}
		})
	}
}


func TestSecurityLabelProvider(t *testing.T) {
	stmt := parseOne(t, "SECURITY LABEL FOR selinux ON TABLE t IS 'label'")
	sl := stmt.(*parser.SecLabelStmt)
	if sl.Provider != "selinux" {
		t.Fatalf("expected Provider=selinux, got %q", sl.Provider)
	}
}

// ---------------------------------------------------------------------------
// CHECKPOINT
// ---------------------------------------------------------------------------

