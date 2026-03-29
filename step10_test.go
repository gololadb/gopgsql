package pgscan

import "testing"

// ---------------------------------------------------------------------------
// Step 10: CREATE VIEW / EXPLAIN / COPY
// ---------------------------------------------------------------------------

func TestCreateView(t *testing.T) {
	s := parseOne(t, "CREATE VIEW v AS SELECT * FROM t")
	vs, ok := s.(*ViewStmt)
	if !ok {
		t.Fatalf("expected ViewStmt, got %T", s)
	}
	if vs.View.Relname != "v" {
		t.Errorf("expected v, got %s", vs.View.Relname)
	}
	if vs.Query == nil {
		t.Error("expected query")
	}
}

func TestCreateOrReplaceView(t *testing.T) {
	s := parseOne(t, "CREATE OR REPLACE VIEW v AS SELECT 1")
	vs := s.(*ViewStmt)
	if !vs.Replace {
		t.Error("expected Replace=true")
	}
}

func TestCreateViewWithColumns(t *testing.T) {
	s := parseOne(t, "CREATE VIEW v (a, b, c) AS SELECT 1, 2, 3")
	vs := s.(*ViewStmt)
	if len(vs.Aliases) != 3 {
		t.Fatalf("expected 3 aliases, got %d", len(vs.Aliases))
	}
	if vs.Aliases[0] != "a" || vs.Aliases[2] != "c" {
		t.Errorf("expected [a,b,c], got %v", vs.Aliases)
	}
}

func TestCreateViewCheckOption(t *testing.T) {
	s := parseOne(t, "CREATE VIEW v AS SELECT * FROM t WITH CHECK OPTION")
	vs := s.(*ViewStmt)
	if vs.WithCheckOption != CASCADED_CHECK_OPTION {
		t.Errorf("expected CASCADED_CHECK_OPTION, got %d", vs.WithCheckOption)
	}
}

func TestCreateViewLocalCheckOption(t *testing.T) {
	s := parseOne(t, "CREATE VIEW v AS SELECT * FROM t WITH LOCAL CHECK OPTION")
	vs := s.(*ViewStmt)
	if vs.WithCheckOption != LOCAL_CHECK_OPTION {
		t.Errorf("expected LOCAL_CHECK_OPTION, got %d", vs.WithCheckOption)
	}
}

func TestCreateTempView(t *testing.T) {
	s := parseOne(t, "CREATE TEMP VIEW v AS SELECT 1")
	vs := s.(*ViewStmt)
	if vs.Persistence != RELPERSISTENCE_TEMP {
		t.Errorf("expected TEMP, got %d", vs.Persistence)
	}
}

func TestExplainSimple(t *testing.T) {
	s := parseOne(t, "EXPLAIN SELECT * FROM t")
	es, ok := s.(*ExplainStmt)
	if !ok {
		t.Fatalf("expected ExplainStmt, got %T", s)
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
	es := s.(*ExplainStmt)
	if len(es.Options) != 1 || es.Options[0].Defname != "analyze" {
		t.Errorf("expected analyze option, got %v", es.Options)
	}
}

func TestExplainAnalyzeVerbose(t *testing.T) {
	s := parseOne(t, "EXPLAIN ANALYZE VERBOSE SELECT * FROM t")
	es := s.(*ExplainStmt)
	if len(es.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(es.Options))
	}
}

func TestExplainVerbose(t *testing.T) {
	s := parseOne(t, "EXPLAIN VERBOSE SELECT * FROM t")
	es := s.(*ExplainStmt)
	if len(es.Options) != 1 || es.Options[0].Defname != "verbose" {
		t.Errorf("expected verbose option, got %v", es.Options)
	}
}

func TestExplainParenOptions(t *testing.T) {
	s := parseOne(t, "EXPLAIN (ANALYZE, FORMAT json) SELECT * FROM t")
	es := s.(*ExplainStmt)
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
	cs, ok := s.(*CopyStmt)
	if !ok {
		t.Fatalf("expected CopyStmt, got %T", s)
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
	cs := s.(*CopyStmt)
	if cs.IsFrom {
		t.Error("expected IsFrom=false")
	}
}

func TestCopyFromFile(t *testing.T) {
	s := parseOne(t, "COPY t FROM '/tmp/data.csv'")
	cs := s.(*CopyStmt)
	if cs.Filename != "/tmp/data.csv" {
		t.Errorf("expected /tmp/data.csv, got %s", cs.Filename)
	}
}

func TestCopyWithOptions(t *testing.T) {
	s := parseOne(t, "COPY t FROM STDIN WITH (FORMAT csv, HEADER true, DELIMITER ',')")
	cs := s.(*CopyStmt)
	if len(cs.Options) != 3 {
		t.Fatalf("expected 3 options, got %d", len(cs.Options))
	}
	if cs.Options[0].Defname != "format" {
		t.Errorf("expected format, got %s", cs.Options[0].Defname)
	}
}

func TestCopyColumns(t *testing.T) {
	s := parseOne(t, "COPY t (a, b, c) FROM STDIN")
	cs := s.(*CopyStmt)
	if len(cs.Attlist) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(cs.Attlist))
	}
}

func TestCopyQueryTo(t *testing.T) {
	s := parseOne(t, "COPY (SELECT * FROM t WHERE active) TO STDOUT")
	cs := s.(*CopyStmt)
	if cs.Query == nil {
		t.Error("expected query")
	}
	if cs.IsFrom {
		t.Error("expected IsFrom=false for COPY query TO")
	}
}

func TestCopyProgram(t *testing.T) {
	s := parseOne(t, "COPY t TO PROGRAM 'gzip > /tmp/data.gz'")
	cs := s.(*CopyStmt)
	if !cs.IsProgram {
		t.Error("expected IsProgram=true")
	}
	if cs.Filename != "gzip > /tmp/data.gz" {
		t.Errorf("expected program string, got %s", cs.Filename)
	}
}
