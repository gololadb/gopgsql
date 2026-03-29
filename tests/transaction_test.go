package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestBegin(t *testing.T) {
	s := parseOne(t, "BEGIN")
	ts := s.(*parser.TransactionStmt)
	if ts.Kind != parser.TRANS_STMT_BEGIN {
		t.Errorf("expected BEGIN, got %d", ts.Kind)
	}
}


func TestBeginWork(t *testing.T) {
	parseOK(t, "BEGIN WORK")
}


func TestBeginIsolation(t *testing.T) {
	s := parseOne(t, "BEGIN ISOLATION LEVEL SERIALIZABLE")
	ts := s.(*parser.TransactionStmt)
	if len(ts.Options) != 1 || ts.Options[0] != "SERIALIZABLE" {
		t.Errorf("expected SERIALIZABLE, got %v", ts.Options)
	}
}


func TestStartTransaction(t *testing.T) {
	s := parseOne(t, "START TRANSACTION READ ONLY")
	ts := s.(*parser.TransactionStmt)
	if ts.Kind != parser.TRANS_STMT_START {
		t.Errorf("expected START, got %d", ts.Kind)
	}
	if len(ts.Options) != 1 || ts.Options[0] != "READ ONLY" {
		t.Errorf("expected READ ONLY, got %v", ts.Options)
	}
}


func TestCommit(t *testing.T) {
	s := parseOne(t, "COMMIT")
	ts := s.(*parser.TransactionStmt)
	if ts.Kind != parser.TRANS_STMT_COMMIT {
		t.Errorf("expected COMMIT, got %d", ts.Kind)
	}
}


func TestEnd(t *testing.T) {
	s := parseOne(t, "END")
	ts := s.(*parser.TransactionStmt)
	if ts.Kind != parser.TRANS_STMT_COMMIT {
		t.Errorf("expected COMMIT (END), got %d", ts.Kind)
	}
}


func TestRollback(t *testing.T) {
	s := parseOne(t, "ROLLBACK")
	ts := s.(*parser.TransactionStmt)
	if ts.Kind != parser.TRANS_STMT_ROLLBACK {
		t.Errorf("expected ROLLBACK, got %d", ts.Kind)
	}
}


func TestAbort(t *testing.T) {
	s := parseOne(t, "ABORT")
	ts := s.(*parser.TransactionStmt)
	if ts.Kind != parser.TRANS_STMT_ROLLBACK {
		t.Errorf("expected ROLLBACK (ABORT), got %d", ts.Kind)
	}
}


func TestSavepoint(t *testing.T) {
	s := parseOne(t, "SAVEPOINT sp1")
	ts := s.(*parser.TransactionStmt)
	if ts.Kind != parser.TRANS_STMT_SAVEPOINT {
		t.Errorf("expected SAVEPOINT, got %d", ts.Kind)
	}
	if ts.Options[0] != "sp1" {
		t.Errorf("expected sp1, got %s", ts.Options[0])
	}
}


func TestReleaseSavepoint(t *testing.T) {
	s := parseOne(t, "RELEASE SAVEPOINT sp1")
	ts := s.(*parser.TransactionStmt)
	if ts.Kind != parser.TRANS_STMT_RELEASE {
		t.Errorf("expected RELEASE, got %d", ts.Kind)
	}
}


func TestRollbackToSavepoint(t *testing.T) {
	s := parseOne(t, "ROLLBACK TO SAVEPOINT sp1")
	ts := s.(*parser.TransactionStmt)
	if ts.Kind != parser.TRANS_STMT_ROLLBACK_TO {
		t.Errorf("expected ROLLBACK_TO, got %d", ts.Kind)
	}
}


func TestSetVariable(t *testing.T) {
	s := parseOne(t, "SET search_path TO public, myschema")
	vs := s.(*parser.VariableSetStmt)
	if vs.Name != "search_path" {
		t.Errorf("expected search_path, got %s", vs.Name)
	}
	if len(vs.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(vs.Args))
	}
}


func TestSetLocal(t *testing.T) {
	s := parseOne(t, "SET LOCAL timezone = 'UTC'")
	vs := s.(*parser.VariableSetStmt)
	if !vs.IsLocal {
		t.Error("expected IsLocal=true")
	}
}


func TestSetVarToDefault(t *testing.T) {
	s := parseOne(t, "SET search_path TO DEFAULT")
	vs := s.(*parser.VariableSetStmt)
	if !vs.IsReset {
		t.Error("expected IsReset=true")
	}
}


func TestResetVariable(t *testing.T) {
	s := parseOne(t, "RESET search_path")
	vs := s.(*parser.VariableSetStmt)
	if !vs.IsReset {
		t.Error("expected IsReset=true")
	}
	if vs.Name != "search_path" {
		t.Errorf("expected search_path, got %s", vs.Name)
	}
}


func TestResetAll(t *testing.T) {
	s := parseOne(t, "RESET ALL")
	vs := s.(*parser.VariableSetStmt)
	if vs.Name != "all" {
		t.Errorf("expected all, got %s", vs.Name)
	}
}


func TestShowVariable(t *testing.T) {
	s := parseOne(t, "SHOW search_path")
	vs := s.(*parser.VariableShowStmt)
	if vs.Name != "search_path" {
		t.Errorf("expected search_path, got %s", vs.Name)
	}
}


func TestShowAll(t *testing.T) {
	s := parseOne(t, "SHOW ALL")
	vs := s.(*parser.VariableShowStmt)
	if vs.Name != "all" {
		t.Errorf("expected all, got %s", vs.Name)
	}
}


func TestListen(t *testing.T) {
	s := parseOne(t, "LISTEN my_channel")
	ls := s.(*parser.ListenStmt)
	if ls.Conditionname != "my_channel" {
		t.Errorf("expected my_channel, got %s", ls.Conditionname)
	}
}


func TestNotify(t *testing.T) {
	s := parseOne(t, "NOTIFY my_channel, 'hello'")
	ns := s.(*parser.NotifyStmt)
	if ns.Conditionname != "my_channel" {
		t.Errorf("expected my_channel, got %s", ns.Conditionname)
	}
	if ns.Payload != "hello" {
		t.Errorf("expected hello, got %s", ns.Payload)
	}
}


func TestUnlisten(t *testing.T) {
	s := parseOne(t, "UNLISTEN my_channel")
	us := s.(*parser.UnlistenStmt)
	if us.Conditionname != "my_channel" {
		t.Errorf("expected my_channel, got %s", us.Conditionname)
	}
}


func TestUnlistenAll(t *testing.T) {
	s := parseOne(t, "UNLISTEN *")
	us := s.(*parser.UnlistenStmt)
	if us.Conditionname != "" {
		t.Errorf("expected empty, got %s", us.Conditionname)
	}
}


func TestSetTransaction(t *testing.T) {
	s := parseOne(t, "SET TRANSACTION ISOLATION LEVEL READ COMMITTED")
	vs := s.(*parser.VariableSetStmt)
	if vs.Name != "transaction" {
		t.Errorf("expected transaction, got %s", vs.Name)
	}
}

