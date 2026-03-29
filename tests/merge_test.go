package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestMergeBasic(t *testing.T) {
	sql := `MERGE INTO target t
		USING source s ON t.id = s.id
		WHEN MATCHED THEN UPDATE SET val = s.val
		WHEN NOT MATCHED THEN INSERT (id, val) VALUES (s.id, s.val)`
	s := parseOne(t, sql)
	m, ok := s.(*parser.MergeStmt)
	if !ok {
		t.Fatalf("expected parser.MergeStmt, got %T", s)
	}
	if m.Relation.Relname != "target" {
		t.Errorf("expected target, got %s", m.Relation.Relname)
	}
	if m.Relation.Alias == nil || m.Relation.Alias.Aliasname != "t" {
		t.Error("expected alias t")
	}
	if len(m.WhenClauses) != 2 {
		t.Fatalf("expected 2 WHEN clauses, got %d", len(m.WhenClauses))
	}

	// First: WHEN MATCHED THEN UPDATE
	wc0 := m.WhenClauses[0]
	if wc0.MatchKind != parser.MERGE_WHEN_MATCHED {
		t.Errorf("expected MATCHED, got %d", wc0.MatchKind)
	}
	if wc0.CommandType != parser.MERGE_CMD_UPDATE {
		t.Errorf("expected UPDATE, got %d", wc0.CommandType)
	}
	if len(wc0.TargetList) != 1 {
		t.Errorf("expected 1 SET clause, got %d", len(wc0.TargetList))
	}

	// Second: WHEN NOT MATCHED THEN INSERT
	wc1 := m.WhenClauses[1]
	if wc1.MatchKind != parser.MERGE_WHEN_NOT_MATCHED_BY_TARGET {
		t.Errorf("expected NOT_MATCHED_BY_TARGET, got %d", wc1.MatchKind)
	}
	if wc1.CommandType != parser.MERGE_CMD_INSERT {
		t.Errorf("expected INSERT, got %d", wc1.CommandType)
	}
	if len(wc1.Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(wc1.Values))
	}
}


func TestMergeDelete(t *testing.T) {
	sql := `MERGE INTO t USING s ON t.id = s.id
		WHEN MATCHED THEN DELETE`
	s := parseOne(t, sql)
	m := s.(*parser.MergeStmt)
	if m.WhenClauses[0].CommandType != parser.MERGE_CMD_DELETE {
		t.Error("expected DELETE action")
	}
}


func TestMergeDoNothing(t *testing.T) {
	sql := `MERGE INTO t USING s ON t.id = s.id
		WHEN MATCHED THEN DO NOTHING`
	s := parseOne(t, sql)
	m := s.(*parser.MergeStmt)
	if m.WhenClauses[0].CommandType != parser.MERGE_CMD_NOTHING {
		t.Error("expected DO NOTHING action")
	}
}


func TestMergeWithCondition(t *testing.T) {
	sql := `MERGE INTO t USING s ON t.id = s.id
		WHEN MATCHED AND t.val <> s.val THEN UPDATE SET val = s.val
		WHEN MATCHED THEN DO NOTHING`
	s := parseOne(t, sql)
	m := s.(*parser.MergeStmt)
	if len(m.WhenClauses) != 2 {
		t.Fatalf("expected 2 clauses, got %d", len(m.WhenClauses))
	}
	if m.WhenClauses[0].Condition == nil {
		t.Error("expected condition on first WHEN clause")
	}
	if m.WhenClauses[1].Condition != nil {
		t.Error("expected no condition on second WHEN clause")
	}
}


func TestMergeNotMatchedBySource(t *testing.T) {
	sql := `MERGE INTO t USING s ON t.id = s.id
		WHEN NOT MATCHED BY SOURCE THEN DELETE`
	s := parseOne(t, sql)
	m := s.(*parser.MergeStmt)
	if m.WhenClauses[0].MatchKind != parser.MERGE_WHEN_NOT_MATCHED_BY_SOURCE {
		t.Errorf("expected NOT_MATCHED_BY_SOURCE, got %d", m.WhenClauses[0].MatchKind)
	}
}


func TestMergeNotMatchedByTarget(t *testing.T) {
	sql := `MERGE INTO t USING s ON t.id = s.id
		WHEN NOT MATCHED BY TARGET THEN INSERT (id) VALUES (s.id)`
	s := parseOne(t, sql)
	m := s.(*parser.MergeStmt)
	if m.WhenClauses[0].MatchKind != parser.MERGE_WHEN_NOT_MATCHED_BY_TARGET {
		t.Errorf("expected NOT_MATCHED_BY_TARGET, got %d", m.WhenClauses[0].MatchKind)
	}
}


func TestMergeWithCTE(t *testing.T) {
	sql := `WITH src AS (SELECT * FROM source)
		MERGE INTO target t USING src s ON t.id = s.id
		WHEN MATCHED THEN UPDATE SET val = s.val`
	s := parseOne(t, sql)
	m := s.(*parser.MergeStmt)
	if m.WithClause == nil {
		t.Error("expected parser.WithClause")
	}
	if len(m.WithClause.CTEs) != 1 {
		t.Errorf("expected 1 CTE, got %d", len(m.WithClause.CTEs))
	}
}


func TestMergeSubquerySource(t *testing.T) {
	sql := `MERGE INTO t
		USING (SELECT * FROM s WHERE active) AS src ON t.id = src.id
		WHEN MATCHED THEN UPDATE SET val = src.val`
	parseOK(t, sql)
}


func TestMergeMultipleActions(t *testing.T) {
	sql := `MERGE INTO inventory i
		USING new_data n ON i.product_id = n.product_id
		WHEN MATCHED AND n.quantity = 0 THEN DELETE
		WHEN MATCHED THEN UPDATE SET quantity = n.quantity, updated_at = now()
		WHEN NOT MATCHED THEN INSERT (product_id, quantity) VALUES (n.product_id, n.quantity)`
	s := parseOne(t, sql)
	m := s.(*parser.MergeStmt)
	if len(m.WhenClauses) != 3 {
		t.Fatalf("expected 3 WHEN clauses, got %d", len(m.WhenClauses))
	}
	if m.WhenClauses[0].CommandType != parser.MERGE_CMD_DELETE {
		t.Error("expected first clause to be DELETE")
	}
	if m.WhenClauses[1].CommandType != parser.MERGE_CMD_UPDATE {
		t.Error("expected second clause to be UPDATE")
	}
	if m.WhenClauses[2].CommandType != parser.MERGE_CMD_INSERT {
		t.Error("expected third clause to be INSERT")
	}
}


func TestMergeInsertNoColumns(t *testing.T) {
	sql := `MERGE INTO t USING s ON t.id = s.id
		WHEN NOT MATCHED THEN INSERT VALUES (s.id, s.val)`
	s := parseOne(t, sql)
	m := s.(*parser.MergeStmt)
	wc := m.WhenClauses[0]
	if len(wc.TargetList) != 0 {
		t.Errorf("expected no column list, got %d", len(wc.TargetList))
	}
	if len(wc.Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(wc.Values))
	}
}

