package pgscan

import "testing"

func TestNamedArgColonEquals(t *testing.T) {
	s := parseOne(t, "SELECT my_func(a := 1, b := 2)")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	fc, ok := rt.Val.(*FuncCall)
	if !ok {
		t.Fatalf("expected *FuncCall, got %T", rt.Val)
	}
	if len(fc.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(fc.Args))
	}
	na, ok := fc.Args[0].(*NamedArgExpr)
	if !ok {
		t.Fatalf("expected *NamedArgExpr, got %T", fc.Args[0])
	}
	if na.Name != "a" {
		t.Fatalf("expected name 'a', got %q", na.Name)
	}
}

func TestNamedArgEqualsGreater(t *testing.T) {
	s := parseOne(t, "SELECT my_func(a => 1, b => 'hello')")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	fc, ok := rt.Val.(*FuncCall)
	if !ok {
		t.Fatalf("expected *FuncCall, got %T", rt.Val)
	}
	if len(fc.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(fc.Args))
	}
	na, ok := fc.Args[1].(*NamedArgExpr)
	if !ok {
		t.Fatalf("expected *NamedArgExpr for arg 1, got %T", fc.Args[1])
	}
	if na.Name != "b" {
		t.Fatalf("expected name 'b', got %q", na.Name)
	}
}

func TestNamedArgMixed(t *testing.T) {
	// Positional followed by named
	s := parseOne(t, "SELECT my_func(1, name => 'test')")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	fc := rt.Val.(*FuncCall)
	if len(fc.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(fc.Args))
	}
	// First arg is positional (A_Const)
	if _, ok := fc.Args[0].(*A_Const); !ok {
		t.Fatalf("expected *A_Const for arg 0, got %T", fc.Args[0])
	}
	// Second arg is named
	if _, ok := fc.Args[1].(*NamedArgExpr); !ok {
		t.Fatalf("expected *NamedArgExpr for arg 1, got %T", fc.Args[1])
	}
}

func TestVariadicFunc(t *testing.T) {
	s := parseOne(t, "SELECT my_func(1, VARIADIC arr)")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	fc, ok := rt.Val.(*FuncCall)
	if !ok {
		t.Fatalf("expected *FuncCall, got %T", rt.Val)
	}
	if !fc.FuncVariadic {
		t.Fatal("expected FuncVariadic=true")
	}
	if len(fc.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(fc.Args))
	}
}

func TestVariadicOnly(t *testing.T) {
	s := parseOne(t, "SELECT my_func(VARIADIC ARRAY[1,2,3])")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	fc := rt.Val.(*FuncCall)
	if !fc.FuncVariadic {
		t.Fatal("expected FuncVariadic=true")
	}
	if len(fc.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(fc.Args))
	}
}

func TestWithinGroup(t *testing.T) {
	s := parseOne(t, "SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY salary)")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	fc, ok := rt.Val.(*FuncCall)
	if !ok {
		t.Fatalf("expected *FuncCall, got %T", rt.Val)
	}
	if len(fc.AggWithinGroup) == 0 {
		t.Fatal("expected non-empty AggWithinGroup")
	}
	if fc.AggWithinGroup[0].Node == nil {
		t.Fatal("expected non-nil sort node")
	}
}

func TestWithinGroupMultiple(t *testing.T) {
	s := parseOne(t, "SELECT percentile_disc(0.5) WITHIN GROUP (ORDER BY x, y DESC)")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	fc := rt.Val.(*FuncCall)
	if len(fc.AggWithinGroup) != 2 {
		t.Fatalf("expected 2 sort items, got %d", len(fc.AggWithinGroup))
	}
}

func TestNullTreatmentIgnore(t *testing.T) {
	s := parseOne(t, "SELECT first_value(x) IGNORE NULLS OVER (ORDER BY id)")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	fc, ok := rt.Val.(*FuncCall)
	if !ok {
		t.Fatalf("expected *FuncCall, got %T", rt.Val)
	}
	if fc.NullTreatment != NULL_TREATMENT_IGNORE {
		t.Fatalf("expected NULL_TREATMENT_IGNORE, got %d", fc.NullTreatment)
	}
	if fc.Over == nil {
		t.Fatal("expected non-nil Over")
	}
}

func TestNullTreatmentRespect(t *testing.T) {
	s := parseOne(t, "SELECT last_value(x) RESPECT NULLS OVER (ORDER BY id)")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	fc := rt.Val.(*FuncCall)
	if fc.NullTreatment != NULL_TREATMENT_RESPECT {
		t.Fatalf("expected NULL_TREATMENT_RESPECT, got %d", fc.NullTreatment)
	}
}

func TestFilterWithinGroupCombo(t *testing.T) {
	// WITHIN GROUP + FILTER together
	s := parseOne(t, "SELECT mode() WITHIN GROUP (ORDER BY val) FILTER (WHERE val > 0)")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	fc := rt.Val.(*FuncCall)
	if len(fc.AggWithinGroup) == 0 {
		t.Fatal("expected non-empty AggWithinGroup")
	}
	if fc.AggFilter == nil {
		t.Fatal("expected non-nil AggFilter")
	}
}

func TestMergeAction(t *testing.T) {
	// MERGE_ACTION() is valid inside MERGE WHEN clauses, but we test it as a standalone expression
	s := parseOne(t, "SELECT MERGE_ACTION()")
	sel := s.(*SelectStmt)
	rt := sel.TargetList[0]
	_, ok := rt.Val.(*MergeActionExpr)
	if !ok {
		t.Fatalf("expected *MergeActionExpr, got %T", rt.Val)
	}
}

func TestRegularFuncStillWorks(t *testing.T) {
	// Ensure normal function calls still work
	tests := []string{
		"SELECT count(*)",
		"SELECT count(DISTINCT x)",
		"SELECT array_agg(x ORDER BY x)",
		"SELECT sum(x) FILTER (WHERE x > 0)",
		"SELECT row_number() OVER (ORDER BY id)",
		"SELECT my_func()",
		"SELECT my_func(1, 2, 3)",
	}
	for _, sql := range tests {
		t.Run(sql, func(t *testing.T) {
			parseOne(t, sql)
		})
	}
}
