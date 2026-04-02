package tests

import (
	"testing"

	"github.com/gololadb/gopgsql/parser"
)

func TestVacuum(t *testing.T) {
	s := parseOne(t, "VACUUM t")
	vs := s.(*parser.VacuumStmt)
	if !vs.IsVacuum {
		t.Error("expected IsVacuum=true")
	}
	if len(vs.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(vs.Relations))
	}
}


func TestVacuumFull(t *testing.T) {
	s := parseOne(t, "VACUUM FULL t")
	vs := s.(*parser.VacuumStmt)
	if len(vs.Options) != 1 || vs.Options[0].Defname != "full" {
		t.Errorf("expected full option, got %v", vs.Options)
	}
}


func TestAnalyzeStmt(t *testing.T) {
	s := parseOne(t, "ANALYZE t")
	vs := s.(*parser.VacuumStmt)
	if vs.IsVacuum {
		t.Error("expected IsVacuum=false")
	}
}


func TestLockTable(t *testing.T) {
	s := parseOne(t, "LOCK TABLE t IN ACCESS EXCLUSIVE MODE NOWAIT")
	ls := s.(*parser.LockStmt)
	if ls.Mode != "access exclusive" {
		t.Errorf("expected ACCESS EXCLUSIVE, got %s", ls.Mode)
	}
	if !ls.Nowait {
		t.Error("expected Nowait=true")
	}
}


func TestPrepareStmt(t *testing.T) {
	s := parseOne(t, "PREPARE myplan (integer, text) AS SELECT * FROM t WHERE id = $1 AND name = $2")
	ps := s.(*parser.PrepareStmt)
	if ps.Name != "myplan" {
		t.Errorf("expected myplan, got %s", ps.Name)
	}
	if len(ps.Argtypes) != 2 {
		t.Fatalf("expected 2 arg types, got %d", len(ps.Argtypes))
	}
	if ps.Query == nil {
		t.Error("expected query")
	}
}


func TestExecuteStmt(t *testing.T) {
	s := parseOne(t, "EXECUTE myplan (1, 'hello')")
	es := s.(*parser.ExecuteStmt)
	if es.Name != "myplan" {
		t.Errorf("expected myplan, got %s", es.Name)
	}
	if len(es.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(es.Params))
	}
}


func TestDeallocate(t *testing.T) {
	s := parseOne(t, "DEALLOCATE myplan")
	ds := s.(*parser.DeallocateStmt)
	if ds.Name != "myplan" {
		t.Errorf("expected myplan, got %s", ds.Name)
	}
}


func TestDeallocateAll(t *testing.T) {
	s := parseOne(t, "DEALLOCATE ALL")
	ds := s.(*parser.DeallocateStmt)
	if !ds.IsAll {
		t.Error("expected IsAll=true")
	}
}


func TestDiscardAll(t *testing.T) {
	s := parseOne(t, "DISCARD ALL")
	ds := s.(*parser.DiscardStmt)
	if ds.Target != "all" {
		t.Errorf("expected all, got %s", ds.Target)
	}
}


func TestExplainSimple(t *testing.T) {
	s := parseOne(t, "EXPLAIN SELECT * FROM t")
	es, ok := s.(*parser.ExplainStmt)
	if !ok {
		t.Fatalf("expected parser.ExplainStmt, got %T", s)
	}
	if es.Query == nil {
		t.Error("expected query")
	}
	if len(es.Options) != 0 {
		t.Errorf("expected no options, got %d", len(es.Options))
	}
}


func TestExplainAnalyze(t *testing.T) {
	s := parseOne(t, "EXPLAIN ANALYZE SELECT * FROM t")
	es := s.(*parser.ExplainStmt)
	if len(es.Options) != 1 || es.Options[0].Defname != "analyze" {
		t.Errorf("expected analyze option, got %v", es.Options)
	}
}


func TestExplainAnalyzeVerbose(t *testing.T) {
	s := parseOne(t, "EXPLAIN ANALYZE VERBOSE SELECT * FROM t")
	es := s.(*parser.ExplainStmt)
	if len(es.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(es.Options))
	}
}


func TestExplainVerbose(t *testing.T) {
	s := parseOne(t, "EXPLAIN VERBOSE SELECT * FROM t")
	es := s.(*parser.ExplainStmt)
	if len(es.Options) != 1 || es.Options[0].Defname != "verbose" {
		t.Errorf("expected verbose option, got %v", es.Options)
	}
}


func TestExplainParenOptions(t *testing.T) {
	s := parseOne(t, "EXPLAIN (ANALYZE, FORMAT json) SELECT * FROM t")
	es := s.(*parser.ExplainStmt)
	if len(es.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(es.Options))
	}
	if es.Options[0].Defname != "analyze" {
		t.Errorf("expected analyze, got %s", es.Options[0].Defname)
	}
	if es.Options[1].Defname != "format" {
		t.Errorf("expected format, got %s", es.Options[1].Defname)
	}
}


func TestExplainInsert(t *testing.T) {
	parseOK(t, "EXPLAIN INSERT INTO t VALUES (1, 2)")
}


func TestExplainUpdate(t *testing.T) {
	parseOK(t, "EXPLAIN ANALYZE UPDATE t SET a = 1")
}


func TestCopyFromStdin(t *testing.T) {
	s := parseOne(t, "COPY t FROM STDIN")
	cs, ok := s.(*parser.CopyStmt)
	if !ok {
		t.Fatalf("expected parser.CopyStmt, got %T", s)
	}
	if !cs.IsFrom {
		t.Error("expected IsFrom=true")
	}
	if cs.Relation.Relname != "t" {
		t.Errorf("expected t, got %s", cs.Relation.Relname)
	}
}


func TestCopyToStdout(t *testing.T) {
	s := parseOne(t, "COPY t TO STDOUT")
	cs := s.(*parser.CopyStmt)
	if cs.IsFrom {
		t.Error("expected IsFrom=false")
	}
}


func TestCopyFromFile(t *testing.T) {
	s := parseOne(t, "COPY t FROM '/tmp/data.csv'")
	cs := s.(*parser.CopyStmt)
	if cs.Filename != "/tmp/data.csv" {
		t.Errorf("expected /tmp/data.csv, got %s", cs.Filename)
	}
}


func TestCopyWithOptions(t *testing.T) {
	s := parseOne(t, "COPY t FROM STDIN WITH (FORMAT csv, HEADER true, DELIMITER ',')")
	cs := s.(*parser.CopyStmt)
	if len(cs.Options) != 3 {
		t.Fatalf("expected 3 options, got %d", len(cs.Options))
	}
	if cs.Options[0].Defname != "format" {
		t.Errorf("expected format, got %s", cs.Options[0].Defname)
	}
}


func TestCopyColumns(t *testing.T) {
	s := parseOne(t, "COPY t (a, b, c) FROM STDIN")
	cs := s.(*parser.CopyStmt)
	if len(cs.Attlist) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(cs.Attlist))
	}
}


func TestCopyQueryTo(t *testing.T) {
	s := parseOne(t, "COPY (SELECT * FROM t WHERE active) TO STDOUT")
	cs := s.(*parser.CopyStmt)
	if cs.Query == nil {
		t.Error("expected query")
	}
	if cs.IsFrom {
		t.Error("expected IsFrom=false for COPY query TO")
	}
}


func TestCopyProgram(t *testing.T) {
	s := parseOne(t, "COPY t TO PROGRAM 'gzip > /tmp/data.gz'")
	cs := s.(*parser.CopyStmt)
	if !cs.IsProgram {
		t.Error("expected IsProgram=true")
	}
	if cs.Filename != "gzip > /tmp/data.gz" {
		t.Errorf("expected program string, got %s", cs.Filename)
	}
}


func TestCheckpoint(t *testing.T) {
	stmt := parseOne(t, "CHECKPOINT")
	_, ok := stmt.(*parser.CheckPointStmt)
	if !ok {
		t.Fatalf("expected *parser.CheckPointStmt, got %T", stmt)
	}
}

// ---------------------------------------------------------------------------
// LOAD
// ---------------------------------------------------------------------------


func TestLoad(t *testing.T) {
	stmt := parseOne(t, "LOAD 'mylib.so'")
	ls, ok := stmt.(*parser.LoadStmt)
	if !ok {
		t.Fatalf("expected *parser.LoadStmt, got %T", stmt)
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
		kind parser.ReindexObjectType
	}{
		{"REINDEX INDEX myidx", "index", parser.REINDEX_OBJECT_INDEX},
		{"REINDEX TABLE t", "table", parser.REINDEX_OBJECT_TABLE},
		{"REINDEX SCHEMA myschema", "schema", parser.REINDEX_OBJECT_SCHEMA},
		{"REINDEX DATABASE mydb", "database", parser.REINDEX_OBJECT_DATABASE},
		{"REINDEX SYSTEM mydb", "system", parser.REINDEX_OBJECT_SYSTEM},
		{"REINDEX (VERBOSE) TABLE t", "with options", parser.REINDEX_OBJECT_TABLE},
		{"REINDEX TABLE CONCURRENTLY t", "concurrently", parser.REINDEX_OBJECT_TABLE},
		{"REINDEX (VERBOSE) INDEX CONCURRENTLY myidx", "options and concurrently", parser.REINDEX_OBJECT_INDEX},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			rs, ok := stmt.(*parser.ReindexStmt)
			if !ok {
				t.Fatalf("expected *parser.ReindexStmt, got %T", stmt)
			}
			if rs.Kind != tt.kind {
				t.Fatalf("expected Kind %d, got %d", tt.kind, rs.Kind)
			}
		})
	}
}


func TestReindexConcurrently(t *testing.T) {
	stmt := parseOne(t, "REINDEX TABLE CONCURRENTLY t")
	rs := stmt.(*parser.ReindexStmt)
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
			cs, ok := stmt.(*parser.ConstraintsSetStmt)
			if !ok {
				t.Fatalf("expected *parser.ConstraintsSetStmt, got %T", stmt)
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

