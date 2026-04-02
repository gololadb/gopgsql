package tests

import (
	"testing"

	"github.com/gololadb/gopgsql/parser"
)

func TestWindowFrameRowsUnbounded(t *testing.T) {
	s := parseOne(t, "SELECT sum(x) OVER (ORDER BY id ROWS UNBOUNDED PRECEDING) FROM t")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	w := fc.Over
	if w.FrameOptions&parser.FRAMEOPTION_ROWS == 0 {
		t.Error("expected ROWS mode")
	}
	if w.FrameOptions&parser.FRAMEOPTION_NONDEFAULT == 0 {
		t.Error("expected NONDEFAULT flag")
	}
	if w.FrameOptions&parser.FRAMEOPTION_START_UNBOUNDED_PRECEDING == 0 {
		t.Error("expected START_UNBOUNDED_PRECEDING")
	}
	if w.FrameOptions&parser.FRAMEOPTION_END_CURRENT_ROW == 0 {
		t.Error("expected END_CURRENT_ROW (implicit)")
	}
}


func TestWindowFrameRangeCurrentRow(t *testing.T) {
	s := parseOne(t, "SELECT sum(x) OVER (ORDER BY id RANGE CURRENT ROW) FROM t")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	w := fc.Over
	if w.FrameOptions&parser.FRAMEOPTION_RANGE == 0 {
		t.Error("expected RANGE mode")
	}
	if w.FrameOptions&parser.FRAMEOPTION_START_CURRENT_ROW == 0 {
		t.Error("expected START_CURRENT_ROW")
	}
}


func TestWindowFrameRowsBetween(t *testing.T) {
	s := parseOne(t, "SELECT sum(x) OVER (ORDER BY id ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING) FROM t")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	w := fc.Over
	if w.FrameOptions&parser.FRAMEOPTION_ROWS == 0 {
		t.Error("expected ROWS mode")
	}
	if w.FrameOptions&parser.FRAMEOPTION_BETWEEN == 0 {
		t.Error("expected BETWEEN flag")
	}
	if w.FrameOptions&parser.FRAMEOPTION_START_OFFSET_PRECEDING == 0 {
		t.Error("expected START_OFFSET_PRECEDING")
	}
	if w.FrameOptions&parser.FRAMEOPTION_END_OFFSET_FOLLOWING == 0 {
		t.Error("expected END_OFFSET_FOLLOWING")
	}
	if w.StartOffset == nil || w.EndOffset == nil {
		t.Error("expected non-nil start and end offsets")
	}
}


func TestWindowFrameGroupsBetweenUnbounded(t *testing.T) {
	s := parseOne(t, "SELECT sum(x) OVER (ORDER BY id GROUPS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) FROM t")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	w := fc.Over
	if w.FrameOptions&parser.FRAMEOPTION_GROUPS == 0 {
		t.Error("expected GROUPS mode")
	}
	if w.FrameOptions&parser.FRAMEOPTION_BETWEEN == 0 {
		t.Error("expected BETWEEN flag")
	}
	if w.FrameOptions&parser.FRAMEOPTION_START_UNBOUNDED_PRECEDING == 0 {
		t.Error("expected START_UNBOUNDED_PRECEDING")
	}
	if w.FrameOptions&parser.FRAMEOPTION_END_UNBOUNDED_FOLLOWING == 0 {
		t.Error("expected END_UNBOUNDED_FOLLOWING")
	}
}


func TestWindowFrameExcludeCurrentRow(t *testing.T) {
	s := parseOne(t, "SELECT sum(x) OVER (ORDER BY id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW EXCLUDE CURRENT ROW) FROM t")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	w := fc.Over
	if w.FrameOptions&parser.FRAMEOPTION_EXCLUDE_CURRENT_ROW == 0 {
		t.Error("expected EXCLUDE_CURRENT_ROW")
	}
}


func TestWindowFrameExcludeGroup(t *testing.T) {
	s := parseOne(t, "SELECT sum(x) OVER (ORDER BY id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW EXCLUDE GROUP) FROM t")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	w := fc.Over
	if w.FrameOptions&parser.FRAMEOPTION_EXCLUDE_GROUP == 0 {
		t.Error("expected EXCLUDE_GROUP")
	}
}


func TestWindowFrameExcludeTies(t *testing.T) {
	s := parseOne(t, "SELECT sum(x) OVER (ORDER BY id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW EXCLUDE TIES) FROM t")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	w := fc.Over
	if w.FrameOptions&parser.FRAMEOPTION_EXCLUDE_TIES == 0 {
		t.Error("expected EXCLUDE_TIES")
	}
}


func TestWindowFrameExcludeNoOthers(t *testing.T) {
	parseOK(t, "SELECT sum(x) OVER (ORDER BY id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW EXCLUDE NO OTHERS) FROM t")
}


func TestWindowFrameDefault(t *testing.T) {
	s := parseOne(t, "SELECT sum(x) OVER (ORDER BY id) FROM t")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	w := fc.Over
	if w.FrameOptions != parser.FRAMEOPTION_DEFAULTS {
		t.Errorf("expected parser.FRAMEOPTION_DEFAULTS (%d), got %d", parser.FRAMEOPTION_DEFAULTS, w.FrameOptions)
	}
}


func TestWindowFrameRowsOffsetPreceding(t *testing.T) {
	s := parseOne(t, "SELECT sum(x) OVER (ORDER BY id ROWS 3 PRECEDING) FROM t")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	w := fc.Over
	if w.FrameOptions&parser.FRAMEOPTION_START_OFFSET_PRECEDING == 0 {
		t.Error("expected START_OFFSET_PRECEDING")
	}
	ac := w.StartOffset.(*parser.A_Const)
	if ac.Val.Ival != 3 {
		t.Errorf("expected offset 3, got %d", ac.Val.Ival)
	}
}


func TestWindowFrameBetweenCurrentAndOffset(t *testing.T) {
	s := parseOne(t, "SELECT sum(x) OVER (ORDER BY id RANGE BETWEEN CURRENT ROW AND 5 FOLLOWING) FROM t")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	w := fc.Over
	if w.FrameOptions&parser.FRAMEOPTION_RANGE == 0 {
		t.Error("expected RANGE mode")
	}
	if w.FrameOptions&parser.FRAMEOPTION_START_CURRENT_ROW == 0 {
		t.Error("expected START_CURRENT_ROW")
	}
	if w.FrameOptions&parser.FRAMEOPTION_END_OFFSET_FOLLOWING == 0 {
		t.Error("expected END_OFFSET_FOLLOWING")
	}
	ac := w.EndOffset.(*parser.A_Const)
	if ac.Val.Ival != 5 {
		t.Errorf("expected end offset 5, got %d", ac.Val.Ival)
	}
}


func TestWindowFrameGroupsOffsetBoth(t *testing.T) {
	s := parseOne(t, "SELECT sum(x) OVER (ORDER BY id GROUPS BETWEEN 2 PRECEDING AND 2 FOLLOWING EXCLUDE TIES) FROM t")
	sel := s.(*parser.SelectStmt)
	fc := sel.TargetList[0].Val.(*parser.FuncCall)
	w := fc.Over
	if w.FrameOptions&parser.FRAMEOPTION_GROUPS == 0 {
		t.Error("expected GROUPS mode")
	}
	if w.FrameOptions&parser.FRAMEOPTION_BETWEEN == 0 {
		t.Error("expected BETWEEN")
	}
	if w.FrameOptions&parser.FRAMEOPTION_EXCLUDE_TIES == 0 {
		t.Error("expected EXCLUDE_TIES")
	}
	if w.StartOffset == nil || w.EndOffset == nil {
		t.Fatal("expected non-nil offsets")
	}
	if w.StartOffset.(*parser.A_Const).Val.Ival != 2 {
		t.Error("expected start offset 2")
	}
	if w.EndOffset.(*parser.A_Const).Val.Ival != 2 {
		t.Error("expected end offset 2")
	}
}

