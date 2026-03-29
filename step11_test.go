package pgscan

import "testing"

// ---------------------------------------------------------------------------
// Step 11: Transaction control / Session / Utility
// ---------------------------------------------------------------------------

func TestBegin(t *testing.T) {
	s := parseOne(t, "BEGIN")
	ts := s.(*TransactionStmt)
	if ts.Kind != TRANS_STMT_BEGIN {
		t.Errorf("expected BEGIN, got %d", ts.Kind)
	}
}

func TestBeginWork(t *testing.T) {
	parseOK(t, "BEGIN WORK")
}

func TestBeginIsolation(t *testing.T) {
	s := parseOne(t, "BEGIN ISOLATION LEVEL SERIALIZABLE")
	ts := s.(*TransactionStmt)
	if len(ts.Options) != 1 || ts.Options[0] != "SERIALIZABLE" {
		t.Errorf("expected SERIALIZABLE, got %v", ts.Options)
	}
}

func TestStartTransaction(t *testing.T) {
	s := parseOne(t, "START TRANSACTION READ ONLY")
	ts := s.(*TransactionStmt)
	if ts.Kind != TRANS_STMT_START {
		t.Errorf("expected START, got %d", ts.Kind)
	}
	if len(ts.Options) != 1 || ts.Options[0] != "READ ONLY" {
		t.Errorf("expected READ ONLY, got %v", ts.Options)
	}
}

func TestCommit(t *testing.T) {
	s := parseOne(t, "COMMIT")
	ts := s.(*TransactionStmt)
	if ts.Kind != TRANS_STMT_COMMIT {
		t.Errorf("expected COMMIT, got %d", ts.Kind)
	}
}

func TestEnd(t *testing.T) {
	s := parseOne(t, "END")
	ts := s.(*TransactionStmt)
	if ts.Kind != TRANS_STMT_COMMIT {
		t.Errorf("expected COMMIT (END), got %d", ts.Kind)
	}
}

func TestRollback(t *testing.T) {
	s := parseOne(t, "ROLLBACK")
	ts := s.(*TransactionStmt)
	if ts.Kind != TRANS_STMT_ROLLBACK {
		t.Errorf("expected ROLLBACK, got %d", ts.Kind)
	}
}

func TestAbort(t *testing.T) {
	s := parseOne(t, "ABORT")
	ts := s.(*TransactionStmt)
	if ts.Kind != TRANS_STMT_ROLLBACK {
		t.Errorf("expected ROLLBACK (ABORT), got %d", ts.Kind)
	}
}

func TestSavepoint(t *testing.T) {
	s := parseOne(t, "SAVEPOINT sp1")
	ts := s.(*TransactionStmt)
	if ts.Kind != TRANS_STMT_SAVEPOINT {
		t.Errorf("expected SAVEPOINT, got %d", ts.Kind)
	}
	if ts.Options[0] != "sp1" {
		t.Errorf("expected sp1, got %s", ts.Options[0])
	}
}

func TestReleaseSavepoint(t *testing.T) {
	s := parseOne(t, "RELEASE SAVEPOINT sp1")
	ts := s.(*TransactionStmt)
	if ts.Kind != TRANS_STMT_RELEASE {
		t.Errorf("expected RELEASE, got %d", ts.Kind)
	}
}

func TestRollbackToSavepoint(t *testing.T) {
	s := parseOne(t, "ROLLBACK TO SAVEPOINT sp1")
	ts := s.(*TransactionStmt)
	if ts.Kind != TRANS_STMT_ROLLBACK_TO {
		t.Errorf("expected ROLLBACK_TO, got %d", ts.Kind)
	}
}

func TestSetVariable(t *testing.T) {
	s := parseOne(t, "SET search_path TO public, myschema")
	vs := s.(*VariableSetStmt)
	if vs.Name != "search_path" {
		t.Errorf("expected search_path, got %s", vs.Name)
	}
	if len(vs.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(vs.Args))
	}
}

func TestSetLocal(t *testing.T) {
	s := parseOne(t, "SET LOCAL timezone = 'UTC'")
	vs := s.(*VariableSetStmt)
	if !vs.IsLocal {
		t.Error("expected IsLocal=true")
	}
}

func TestSetVarToDefault(t *testing.T) {
	s := parseOne(t, "SET search_path TO DEFAULT")
	vs := s.(*VariableSetStmt)
	if !vs.IsReset {
		t.Error("expected IsReset=true")
	}
}

func TestResetVariable(t *testing.T) {
	s := parseOne(t, "RESET search_path")
	vs := s.(*VariableSetStmt)
	if !vs.IsReset {
		t.Error("expected IsReset=true")
	}
	if vs.Name != "search_path" {
		t.Errorf("expected search_path, got %s", vs.Name)
	}
}

func TestResetAll(t *testing.T) {
	s := parseOne(t, "RESET ALL")
	vs := s.(*VariableSetStmt)
	if vs.Name != "all" {
		t.Errorf("expected all, got %s", vs.Name)
	}
}

func TestShowVariable(t *testing.T) {
	s := parseOne(t, "SHOW search_path")
	vs := s.(*VariableShowStmt)
	if vs.Name != "search_path" {
		t.Errorf("expected search_path, got %s", vs.Name)
	}
}

func TestShowAll(t *testing.T) {
	s := parseOne(t, "SHOW ALL")
	vs := s.(*VariableShowStmt)
	if vs.Name != "all" {
		t.Errorf("expected all, got %s", vs.Name)
	}
}

func TestListen(t *testing.T) {
	s := parseOne(t, "LISTEN my_channel")
	ls := s.(*ListenStmt)
	if ls.Conditionname != "my_channel" {
		t.Errorf("expected my_channel, got %s", ls.Conditionname)
	}
}

func TestNotify(t *testing.T) {
	s := parseOne(t, "NOTIFY my_channel, 'hello'")
	ns := s.(*NotifyStmt)
	if ns.Conditionname != "my_channel" {
		t.Errorf("expected my_channel, got %s", ns.Conditionname)
	}
	if ns.Payload != "hello" {
		t.Errorf("expected hello, got %s", ns.Payload)
	}
}

func TestUnlisten(t *testing.T) {
	s := parseOne(t, "UNLISTEN my_channel")
	us := s.(*UnlistenStmt)
	if us.Conditionname != "my_channel" {
		t.Errorf("expected my_channel, got %s", us.Conditionname)
	}
}

func TestUnlistenAll(t *testing.T) {
	s := parseOne(t, "UNLISTEN *")
	us := s.(*UnlistenStmt)
	if us.Conditionname != "" {
		t.Errorf("expected empty, got %s", us.Conditionname)
	}
}

func TestVacuum(t *testing.T) {
	s := parseOne(t, "VACUUM t")
	vs := s.(*VacuumStmt)
	if !vs.IsVacuum {
		t.Error("expected IsVacuum=true")
	}
	if len(vs.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(vs.Relations))
	}
}

func TestVacuumFull(t *testing.T) {
	s := parseOne(t, "VACUUM FULL t")
	vs := s.(*VacuumStmt)
	if len(vs.Options) != 1 || vs.Options[0].Defname != "full" {
		t.Errorf("expected full option, got %v", vs.Options)
	}
}

func TestAnalyzeStmt(t *testing.T) {
	s := parseOne(t, "ANALYZE t")
	vs := s.(*VacuumStmt)
	if vs.IsVacuum {
		t.Error("expected IsVacuum=false")
	}
}

func TestLockTable(t *testing.T) {
	s := parseOne(t, "LOCK TABLE t IN ACCESS EXCLUSIVE MODE NOWAIT")
	ls := s.(*LockStmt)
	if ls.Mode != "access exclusive" {
		t.Errorf("expected ACCESS EXCLUSIVE, got %s", ls.Mode)
	}
	if !ls.Nowait {
		t.Error("expected Nowait=true")
	}
}

func TestPrepareStmt(t *testing.T) {
	s := parseOne(t, "PREPARE myplan (integer, text) AS SELECT * FROM t WHERE id = $1 AND name = $2")
	ps := s.(*PrepareStmt)
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
	es := s.(*ExecuteStmt)
	if es.Name != "myplan" {
		t.Errorf("expected myplan, got %s", es.Name)
	}
	if len(es.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(es.Params))
	}
}

func TestDeallocate(t *testing.T) {
	s := parseOne(t, "DEALLOCATE myplan")
	ds := s.(*DeallocateStmt)
	if ds.Name != "myplan" {
		t.Errorf("expected myplan, got %s", ds.Name)
	}
}

func TestDeallocateAll(t *testing.T) {
	s := parseOne(t, "DEALLOCATE ALL")
	ds := s.(*DeallocateStmt)
	if !ds.IsAll {
		t.Error("expected IsAll=true")
	}
}

func TestDiscardAll(t *testing.T) {
	s := parseOne(t, "DISCARD ALL")
	ds := s.(*DiscardStmt)
	if ds.Target != "all" {
		t.Errorf("expected all, got %s", ds.Target)
	}
}

func TestSetTransaction(t *testing.T) {
	s := parseOne(t, "SET TRANSACTION ISOLATION LEVEL READ COMMITTED")
	vs := s.(*VariableSetStmt)
	if vs.Name != "transaction" {
		t.Errorf("expected transaction, got %s", vs.Name)
	}
}
