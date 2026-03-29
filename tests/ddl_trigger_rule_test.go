package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestCreateTrigger(t *testing.T) {
	sql := `CREATE TRIGGER trg_audit
		AFTER INSERT OR UPDATE ON accounts
		FOR EACH ROW
		EXECUTE FUNCTION audit_func()`
	s := parseOne(t, sql)
	ct, ok := s.(*parser.CreateTrigStmt)
	if !ok {
		t.Fatalf("expected parser.CreateTrigStmt, got %T", s)
	}
	if ct.Trigname != "trg_audit" {
		t.Errorf("expected trg_audit, got %s", ct.Trigname)
	}
	if ct.Timing != parser.TRIGGER_TYPE_AFTER {
		t.Errorf("expected AFTER, got %d", ct.Timing)
	}
	if ct.Events&parser.TRIGGER_TYPE_INSERT == 0 {
		t.Error("expected INSERT event")
	}
	if ct.Events&parser.TRIGGER_TYPE_UPDATE == 0 {
		t.Error("expected UPDATE event")
	}
	if !ct.Row {
		t.Error("expected FOR EACH ROW")
	}
	if ct.Funcname[0] != "audit_func" {
		t.Errorf("expected audit_func, got %v", ct.Funcname)
	}
}


func TestCreateTriggerBefore(t *testing.T) {
	sql := `CREATE TRIGGER trg BEFORE DELETE ON t FOR EACH ROW EXECUTE FUNCTION f()`
	s := parseOne(t, sql)
	ct := s.(*parser.CreateTrigStmt)
	if ct.Timing != parser.TRIGGER_TYPE_BEFORE {
		t.Errorf("expected BEFORE, got %d", ct.Timing)
	}
	if ct.Events&parser.TRIGGER_TYPE_DELETE == 0 {
		t.Error("expected DELETE event")
	}
}


func TestCreateTriggerWhen(t *testing.T) {
	sql := `CREATE TRIGGER trg AFTER UPDATE ON t
		FOR EACH ROW WHEN (OLD.val <> NEW.val)
		EXECUTE FUNCTION f()`
	s := parseOne(t, sql)
	ct := s.(*parser.CreateTrigStmt)
	if ct.WhenClause == nil {
		t.Error("expected WHEN clause")
	}
}


func TestCreateTriggerUpdateOf(t *testing.T) {
	sql := `CREATE TRIGGER trg AFTER UPDATE OF col1, col2 ON t
		FOR EACH ROW EXECUTE FUNCTION f()`
	s := parseOne(t, sql)
	ct := s.(*parser.CreateTrigStmt)
	if len(ct.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(ct.Columns))
	}
}


func TestCreateRule(t *testing.T) {
	sql := `CREATE RULE notify_insert AS ON INSERT TO t DO ALSO NOTIFY t_changes`
	s := parseOne(t, sql)
	rs, ok := s.(*parser.RuleStmt)
	if !ok {
		t.Fatalf("expected parser.RuleStmt, got %T", s)
	}
	if rs.Rulename != "notify_insert" {
		t.Errorf("expected notify_insert, got %s", rs.Rulename)
	}
	if rs.Event != parser.CMD_INSERT {
		t.Errorf("expected INSERT, got %d", rs.Event)
	}
	if rs.Instead {
		t.Error("expected Instead=false (ALSO)")
	}
}


func TestCreateRuleDoNothing(t *testing.T) {
	sql := `CREATE RULE ignore_delete AS ON DELETE TO t DO INSTEAD NOTHING`
	s := parseOne(t, sql)
	rs := s.(*parser.RuleStmt)
	if !rs.Instead {
		t.Error("expected Instead=true")
	}
	if len(rs.Actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(rs.Actions))
	}
}

