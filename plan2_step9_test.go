package pgscan

import "testing"

// ---------------------------------------------------------------------------
// COMMENT ON
// ---------------------------------------------------------------------------

func TestCommentOn(t *testing.T) {
	tests := []struct {
		sql     string
		desc    string
		objType ObjectType
	}{
		{"COMMENT ON TABLE t IS 'a table'", "table", OBJECT_TABLE},
		{"COMMENT ON COLUMN t.c IS 'a column'", "column", OBJECT_COLUMN},
		{"COMMENT ON INDEX idx IS 'an index'", "index", OBJECT_INDEX},
		{"COMMENT ON VIEW v IS 'a view'", "view", OBJECT_VIEW},
		{"COMMENT ON SEQUENCE s IS 'a seq'", "sequence", OBJECT_SEQUENCE},
		{"COMMENT ON TYPE mytype IS 'a type'", "type", OBJECT_TYPE},
		{"COMMENT ON SCHEMA public IS 'public schema'", "schema", OBJECT_SCHEMA},
		{"COMMENT ON FUNCTION myfunc IS 'a func'", "function", OBJECT_FUNCTION},
		{"COMMENT ON PROCEDURE myproc IS 'a proc'", "procedure", OBJECT_PROCEDURE},
		{"COMMENT ON EXTENSION ext IS 'an ext'", "extension", OBJECT_EXTENSION},
		{"COMMENT ON TRIGGER trg ON t IS 'a trigger'", "trigger", OBJECT_TRIGGER},
		{"COMMENT ON RULE rl ON t IS 'a rule'", "rule", OBJECT_RULE},
		{"COMMENT ON DOMAIN d IS 'a domain'", "domain", OBJECT_DOMAIN},
		{"COMMENT ON DATABASE mydb IS 'a db'", "database", OBJECT_DATABASE},
		{"COMMENT ON ROLE myrole IS 'a role'", "role", OBJECT_ROLE},
		{"COMMENT ON TABLESPACE ts IS 'a ts'", "tablespace", OBJECT_TABLESPACE},
		{"COMMENT ON POLICY pol ON t IS 'a policy'", "policy", OBJECT_POLICY},
		{"COMMENT ON PUBLICATION pub IS 'a pub'", "publication", OBJECT_PUBLICATION},
		{"COMMENT ON SUBSCRIPTION sub IS 'a sub'", "subscription", OBJECT_SUBSCRIPTION},
		{"COMMENT ON AGGREGATE myagg IS 'an agg'", "aggregate", OBJECT_AGGREGATE},
		{"COMMENT ON OPERATOR myop IS 'an op'", "operator", OBJECT_OPERATOR},
		{"COMMENT ON COLLATION mycoll IS 'a coll'", "collation", OBJECT_COLLATION},
		{"COMMENT ON CONVERSION myconv IS 'a conv'", "conversion", OBJECT_CONVERSION},
		{"COMMENT ON LANGUAGE plpgsql IS 'a lang'", "language", OBJECT_LANGUAGE},
		{"COMMENT ON EVENT TRIGGER evt IS 'an evt'", "event trigger", OBJECT_EVENT_TRIGGER},
		{"COMMENT ON ACCESS METHOD btree IS 'btree'", "access method", OBJECT_ACCESS_METHOD},
		{"COMMENT ON FOREIGN TABLE ft IS 'a ft'", "foreign table", OBJECT_FOREIGN_TABLE},
		{"COMMENT ON FOREIGN DATA WRAPPER myfdw IS 'a fdw'", "fdw", OBJECT_FDW},
		{"COMMENT ON SERVER mysrv IS 'a server'", "server", OBJECT_FOREIGN_SERVER},
		{"COMMENT ON MATERIALIZED VIEW mv IS 'a mv'", "matview", OBJECT_MATVIEW},
		{"COMMENT ON TEXT SEARCH PARSER myparser IS 'a parser'", "ts parser", OBJECT_TSPARSER},
		{"COMMENT ON TEXT SEARCH DICTIONARY mydict IS 'a dict'", "ts dictionary", OBJECT_TSDICTIONARY},
		{"COMMENT ON TEXT SEARCH TEMPLATE mytmpl IS 'a tmpl'", "ts template", OBJECT_TSTEMPLATE},
		{"COMMENT ON TEXT SEARCH CONFIGURATION myconf IS 'a conf'", "ts configuration", OBJECT_TSCONFIGURATION},
		{"COMMENT ON STATISTICS mystat IS 'a stat'", "statistics", OBJECT_STATISTICS},
		{"COMMENT ON CONSTRAINT mycon ON t IS 'a constraint'", "constraint", OBJECT_CONSTRAINT},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cs, ok := stmt.(*CommentStmt)
			if !ok {
				t.Fatalf("expected *CommentStmt, got %T", stmt)
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
	cs, ok := stmt.(*CommentStmt)
	if !ok {
		t.Fatalf("expected *CommentStmt, got %T", stmt)
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
			sl, ok := stmt.(*SecLabelStmt)
			if !ok {
				t.Fatalf("expected *SecLabelStmt, got %T", stmt)
			}
			if len(sl.Object) == 0 {
				t.Fatal("expected non-empty Object")
			}
		})
	}
}

func TestSecurityLabelProvider(t *testing.T) {
	stmt := parseOne(t, "SECURITY LABEL FOR selinux ON TABLE t IS 'label'")
	sl := stmt.(*SecLabelStmt)
	if sl.Provider != "selinux" {
		t.Fatalf("expected Provider=selinux, got %q", sl.Provider)
	}
}

// ---------------------------------------------------------------------------
// CHECKPOINT
// ---------------------------------------------------------------------------

func TestCheckpoint(t *testing.T) {
	stmt := parseOne(t, "CHECKPOINT")
	_, ok := stmt.(*CheckPointStmt)
	if !ok {
		t.Fatalf("expected *CheckPointStmt, got %T", stmt)
	}
}

// ---------------------------------------------------------------------------
// LOAD
// ---------------------------------------------------------------------------

func TestLoad(t *testing.T) {
	stmt := parseOne(t, "LOAD 'mylib.so'")
	ls, ok := stmt.(*LoadStmt)
	if !ok {
		t.Fatalf("expected *LoadStmt, got %T", stmt)
	}
	if ls.Filename != "mylib.so" {
		t.Fatalf("expected Filename=mylib.so, got %q", ls.Filename)
	}
}

// ---------------------------------------------------------------------------
// REINDEX
// ---------------------------------------------------------------------------

func TestReindex(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
		kind ReindexObjectType
	}{
		{"REINDEX INDEX myidx", "index", REINDEX_OBJECT_INDEX},
		{"REINDEX TABLE t", "table", REINDEX_OBJECT_TABLE},
		{"REINDEX SCHEMA myschema", "schema", REINDEX_OBJECT_SCHEMA},
		{"REINDEX DATABASE mydb", "database", REINDEX_OBJECT_DATABASE},
		{"REINDEX SYSTEM mydb", "system", REINDEX_OBJECT_SYSTEM},
		{"REINDEX (VERBOSE) TABLE t", "with options", REINDEX_OBJECT_TABLE},
		{"REINDEX TABLE CONCURRENTLY t", "concurrently", REINDEX_OBJECT_TABLE},
		{"REINDEX (VERBOSE) INDEX CONCURRENTLY myidx", "options and concurrently", REINDEX_OBJECT_INDEX},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			rs, ok := stmt.(*ReindexStmt)
			if !ok {
				t.Fatalf("expected *ReindexStmt, got %T", stmt)
			}
			if rs.Kind != tt.kind {
				t.Fatalf("expected Kind %d, got %d", tt.kind, rs.Kind)
			}
		})
	}
}

func TestReindexConcurrently(t *testing.T) {
	stmt := parseOne(t, "REINDEX TABLE CONCURRENTLY t")
	rs := stmt.(*ReindexStmt)
	if !rs.Concurrent {
		t.Fatal("expected Concurrent=true")
	}
}

// ---------------------------------------------------------------------------
// SET CONSTRAINTS
// ---------------------------------------------------------------------------

func TestSetConstraints(t *testing.T) {
	tests := []struct {
		sql      string
		desc     string
		deferred bool
		all      bool
	}{
		{"SET CONSTRAINTS ALL DEFERRED", "all deferred", true, true},
		{"SET CONSTRAINTS ALL IMMEDIATE", "all immediate", false, true},
		{"SET CONSTRAINTS mycon DEFERRED", "named deferred", true, false},
		{"SET CONSTRAINTS mycon1, mycon2 IMMEDIATE", "multiple immediate", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cs, ok := stmt.(*ConstraintsSetStmt)
			if !ok {
				t.Fatalf("expected *ConstraintsSetStmt, got %T", stmt)
			}
			if cs.Deferred != tt.deferred {
				t.Fatalf("expected Deferred=%v, got %v", tt.deferred, cs.Deferred)
			}
			if tt.all && cs.Constraints != nil {
				t.Fatal("expected nil Constraints for ALL")
			}
			if !tt.all && len(cs.Constraints) == 0 {
				t.Fatal("expected non-empty Constraints")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ALTER DEFAULT PRIVILEGES
// ---------------------------------------------------------------------------

func TestAlterDefaultPrivileges(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER DEFAULT PRIVILEGES GRANT SELECT ON TABLES TO public", "basic grant"},
		{"ALTER DEFAULT PRIVILEGES REVOKE ALL ON FUNCTIONS FROM public", "basic revoke"},
		{"ALTER DEFAULT PRIVILEGES FOR ROLE myrole GRANT SELECT ON TABLES TO public", "for role"},
		{"ALTER DEFAULT PRIVILEGES IN SCHEMA myschema GRANT SELECT ON TABLES TO public", "in schema"},
		{"ALTER DEFAULT PRIVILEGES FOR ROLE myrole IN SCHEMA myschema GRANT INSERT ON TABLES TO writer", "for role in schema"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			adp, ok := stmt.(*AlterDefaultPrivilegesStmt)
			if !ok {
				t.Fatalf("expected *AlterDefaultPrivilegesStmt, got %T", stmt)
			}
			if adp.Action == nil {
				t.Fatal("expected non-nil Action")
			}
		})
	}
}

func TestAlterDefaultPrivilegesForRole(t *testing.T) {
	stmt := parseOne(t, "ALTER DEFAULT PRIVILEGES FOR ROLE myrole GRANT SELECT ON TABLES TO public")
	adp := stmt.(*AlterDefaultPrivilegesStmt)
	if len(adp.Options) == 0 {
		t.Fatal("expected non-empty Options")
	}
	found := false
	for _, o := range adp.Options {
		if o.Defname == "for_role" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected for_role option")
	}
}

// ---------------------------------------------------------------------------
// ALTER STATISTICS
// ---------------------------------------------------------------------------

func TestAlterStatistics(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER STATISTICS mystat SET STATISTICS 100", "set statistics"},
		{"ALTER STATISTICS mystat RENAME TO newstat", "rename"},
		{"ALTER STATISTICS mystat SET SCHEMA newschema", "set schema"},
		{"ALTER STATISTICS mystat OWNER TO newowner", "owner to"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_ = parseOne(t, tt.sql) // just verify it parses
		})
	}
}

func TestAlterStatisticsSetValue(t *testing.T) {
	stmt := parseOne(t, "ALTER STATISTICS mystat SET STATISTICS 200")
	as, ok := stmt.(*AlterStatsStmt)
	if !ok {
		t.Fatalf("expected *AlterStatsStmt, got %T", stmt)
	}
	if as.Stxstattarget != 200 {
		t.Fatalf("expected Stxstattarget=200, got %d", as.Stxstattarget)
	}
}

func TestAlterStatisticsOwner(t *testing.T) {
	stmt := parseOne(t, "ALTER STATISTICS mystat OWNER TO newowner")
	ao, ok := stmt.(*AlterOwnerStmt)
	if !ok {
		t.Fatalf("expected *AlterOwnerStmt, got %T", stmt)
	}
	if ao.NewOwner != "newowner" {
		t.Fatalf("expected NewOwner=newowner, got %q", ao.NewOwner)
	}
}
