package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestTypedLiteralDate(t *testing.T) {
	s := parseOne(t, "SELECT DATE '2024-01-15'")
	sel := s.(*parser.SelectStmt)
	tc, ok := sel.TargetList[0].Val.(*parser.TypeCast)
	if !ok {
		t.Fatalf("expected parser.TypeCast, got %T", sel.TargetList[0].Val)
	}
	// DATE is not a PG keyword — it's a generic type name (identifier)
	if tc.TypeName.Names[0] != "date" {
		t.Errorf("expected date type, got %v", tc.TypeName.Names)
	}
	ac := tc.Arg.(*parser.A_Const)
	if ac.Val.Str != "2024-01-15" {
		t.Errorf("expected '2024-01-15', got %q", ac.Val.Str)
	}
}


func TestTypedLiteralTimestamp(t *testing.T) {
	s := parseOne(t, "SELECT TIMESTAMP '2024-01-15 10:30:00'")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "timestamp" {
		t.Errorf("expected timestamp, got %v", tc.TypeName.Names)
	}
}


func TestTypedLiteralTimestampTZ(t *testing.T) {
	s := parseOne(t, "SELECT TIMESTAMP WITH TIME ZONE '2024-01-15 10:30:00+05'")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "timestamptz" {
		t.Errorf("expected timestamptz, got %v", tc.TypeName.Names)
	}
}


func TestTypedLiteralTimeTZ(t *testing.T) {
	s := parseOne(t, "SELECT TIME WITH TIME ZONE '10:30:00+05'")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "timetz" {
		t.Errorf("expected timetz, got %v", tc.TypeName.Names)
	}
}


func TestTypedLiteralBit(t *testing.T) {
	s := parseOne(t, "SELECT BIT '101'")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "bit" {
		t.Errorf("expected bit, got %v", tc.TypeName.Names)
	}
}


func TestTypedLiteralBitVarying(t *testing.T) {
	s := parseOne(t, "SELECT BIT VARYING '101'")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "varbit" {
		t.Errorf("expected varbit, got %v", tc.TypeName.Names)
	}
}


func TestIntervalLiteralSimple(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '1 day'")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "interval" {
		t.Errorf("expected interval, got %v", tc.TypeName.Names)
	}
	ac := tc.Arg.(*parser.A_Const)
	if ac.Val.Str != "1 day" {
		t.Errorf("expected '1 day', got %q", ac.Val.Str)
	}
	if tc.TypeName.Typmods != nil {
		t.Errorf("expected nil typmods, got %v", tc.TypeName.Typmods)
	}
}


func TestIntervalLiteralYear(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '1' YEAR")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if len(tc.TypeName.Typmods) != 1 {
		t.Fatalf("expected 1 typmod, got %d", len(tc.TypeName.Typmods))
	}
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	if mask != parser.IntervalMask(parser.IntervalFieldYear) {
		t.Errorf("expected YEAR mask %d, got %d", parser.IntervalMask(parser.IntervalFieldYear), mask)
	}
}


func TestIntervalLiteralYearToMonth(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '1-6' YEAR TO MONTH")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	expected := parser.IntervalMask(parser.IntervalFieldYear) | parser.IntervalMask(parser.IntervalFieldMonth)
	if mask != expected {
		t.Errorf("expected YEAR|MONTH mask %d, got %d", expected, mask)
	}
}


func TestIntervalLiteralDayToSecond(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '1 12:30:45' DAY TO SECOND")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	expected := parser.IntervalMask(parser.IntervalFieldDay) | parser.IntervalMask(parser.IntervalFieldHour) |
		parser.IntervalMask(parser.IntervalFieldMinute) | parser.IntervalMask(parser.IntervalFieldSecond)
	if mask != expected {
		t.Errorf("expected DAY|HOUR|MINUTE|SECOND mask %d, got %d", expected, mask)
	}
}


func TestIntervalLiteralDayToHour(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '1 12' DAY TO HOUR")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	expected := parser.IntervalMask(parser.IntervalFieldDay) | parser.IntervalMask(parser.IntervalFieldHour)
	if mask != expected {
		t.Errorf("expected DAY|HOUR mask %d, got %d", expected, mask)
	}
}


func TestIntervalLiteralHourToMinute(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '12:30' HOUR TO MINUTE")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	expected := parser.IntervalMask(parser.IntervalFieldHour) | parser.IntervalMask(parser.IntervalFieldMinute)
	if mask != expected {
		t.Errorf("expected HOUR|MINUTE mask %d, got %d", expected, mask)
	}
}


func TestIntervalLiteralMinuteToSecondPrec(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '30:45.123' MINUTE TO SECOND(3)")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if len(tc.TypeName.Typmods) != 2 {
		t.Fatalf("expected 2 typmods, got %d", len(tc.TypeName.Typmods))
	}
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	expected := parser.IntervalMask(parser.IntervalFieldMinute) | parser.IntervalMask(parser.IntervalFieldSecond)
	if mask != expected {
		t.Errorf("expected MINUTE|SECOND mask %d, got %d", expected, mask)
	}
	prec := tc.TypeName.Typmods[1].(*parser.A_Const).Val.Ival
	if prec != 3 {
		t.Errorf("expected precision 3, got %d", prec)
	}
}


func TestIntervalLiteralPrecisionPrefix(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL(6) '1 day'")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if len(tc.TypeName.Typmods) != 2 {
		t.Fatalf("expected 2 typmods, got %d", len(tc.TypeName.Typmods))
	}
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	if mask != parser.IntervalFullRange {
		t.Errorf("expected FULL_RANGE %d, got %d", parser.IntervalFullRange, mask)
	}
	prec := tc.TypeName.Typmods[1].(*parser.A_Const).Val.Ival
	if prec != 6 {
		t.Errorf("expected precision 6, got %d", prec)
	}
}


func TestIntervalLiteralSecondOnly(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '45' SECOND")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	if mask != parser.IntervalMask(parser.IntervalFieldSecond) {
		t.Errorf("expected SECOND mask, got %d", mask)
	}
}


func TestIntervalLiteralMonth(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '6' MONTH")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	if mask != parser.IntervalMask(parser.IntervalFieldMonth) {
		t.Errorf("expected MONTH mask, got %d", mask)
	}
}


func TestCastBitVarying(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS BIT VARYING(8))")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "varbit" {
		t.Errorf("expected varbit, got %v", tc.TypeName.Names)
	}
	if len(tc.TypeName.Typmods) != 1 {
		t.Fatalf("expected 1 typmod, got %d", len(tc.TypeName.Typmods))
	}
}


func TestCastIntervalDayToHour(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS INTERVAL DAY TO HOUR)")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "interval" {
		t.Errorf("expected interval, got %v", tc.TypeName.Names)
	}
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	expected := parser.IntervalMask(parser.IntervalFieldDay) | parser.IntervalMask(parser.IntervalFieldHour)
	if mask != expected {
		t.Errorf("expected DAY|HOUR mask %d, got %d", expected, mask)
	}
}


func TestCastIntervalPrecision(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS INTERVAL(3))")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if len(tc.TypeName.Typmods) != 2 {
		t.Fatalf("expected 2 typmods, got %d", len(tc.TypeName.Typmods))
	}
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	if mask != parser.IntervalFullRange {
		t.Errorf("expected FULL_RANGE, got %d", mask)
	}
}


func TestCastTimestampPrecision(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS TIMESTAMP(3) WITH TIME ZONE)")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "timestamptz" {
		t.Errorf("expected timestamptz, got %v", tc.TypeName.Names)
	}
	if len(tc.TypeName.Typmods) != 1 {
		t.Fatalf("expected 1 typmod, got %d", len(tc.TypeName.Typmods))
	}
}


func TestCastTimePrecision(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS TIME(6) WITHOUT TIME ZONE)")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "time" {
		t.Errorf("expected time, got %v", tc.TypeName.Names)
	}
}


func TestTypedLiteralChar(t *testing.T) {
	s := parseOne(t, "SELECT CHAR 'x'")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "bpchar" {
		t.Errorf("expected bpchar, got %v", tc.TypeName.Names)
	}
}


func TestTypedLiteralVarchar(t *testing.T) {
	s := parseOne(t, "SELECT VARCHAR 'hello'")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "varchar" {
		t.Errorf("expected varchar, got %v", tc.TypeName.Names)
	}
}


func TestTypedLiteralJSON(t *testing.T) {
	s := parseOne(t, `SELECT JSON '{"a":1}'`)
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "json" {
		t.Errorf("expected json, got %v", tc.TypeName.Names)
	}
}


func TestArrayBoundsMultiDim(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS integer[3][4])")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if len(tc.TypeName.ArrayBounds) != 2 {
		t.Fatalf("expected 2 array bounds, got %d", len(tc.TypeName.ArrayBounds))
	}
	if tc.TypeName.ArrayBounds[0] != 3 || tc.TypeName.ArrayBounds[1] != 4 {
		t.Errorf("expected [3][4], got %v", tc.TypeName.ArrayBounds)
	}
}


func TestArrayKeywordSuffix(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS text ARRAY[5])")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if len(tc.TypeName.ArrayBounds) != 1 || tc.TypeName.ArrayBounds[0] != 5 {
		t.Errorf("expected ARRAY[5], got %v", tc.TypeName.ArrayBounds)
	}
}


func TestIntervalDayToMinute(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '1 12:30' DAY TO MINUTE")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	expected := parser.IntervalMask(parser.IntervalFieldDay) | parser.IntervalMask(parser.IntervalFieldHour) | parser.IntervalMask(parser.IntervalFieldMinute)
	if mask != expected {
		t.Errorf("expected DAY|HOUR|MINUTE mask %d, got %d", expected, mask)
	}
}


func TestIntervalHourToSecond(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '12:30:45' HOUR TO SECOND")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	expected := parser.IntervalMask(parser.IntervalFieldHour) | parser.IntervalMask(parser.IntervalFieldMinute) | parser.IntervalMask(parser.IntervalFieldSecond)
	if mask != expected {
		t.Errorf("expected HOUR|MINUTE|SECOND mask %d, got %d", expected, mask)
	}
}


func TestIntervalHourOnly(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '12' HOUR")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	if mask != parser.IntervalMask(parser.IntervalFieldHour) {
		t.Errorf("expected HOUR mask, got %d", mask)
	}
}


func TestIntervalDayOnly(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '5' DAY")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	if mask != parser.IntervalMask(parser.IntervalFieldDay) {
		t.Errorf("expected DAY mask, got %d", mask)
	}
}


func TestIntervalMinuteOnly(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '30' MINUTE")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	if mask != parser.IntervalMask(parser.IntervalFieldMinute) {
		t.Errorf("expected MINUTE mask, got %d", mask)
	}
}


func TestIntervalSecondWithPrecision(t *testing.T) {
	s := parseOne(t, "SELECT INTERVAL '45.123' SECOND(3)")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if len(tc.TypeName.Typmods) != 2 {
		t.Fatalf("expected 2 typmods, got %d", len(tc.TypeName.Typmods))
	}
	mask := tc.TypeName.Typmods[0].(*parser.A_Const).Val.Ival
	if mask != parser.IntervalMask(parser.IntervalFieldSecond) {
		t.Errorf("expected SECOND mask, got %d", mask)
	}
	prec := tc.TypeName.Typmods[1].(*parser.A_Const).Val.Ival
	if prec != 3 {
		t.Errorf("expected precision 3, got %d", prec)
	}
}


func TestCastFloatPrecision(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS FLOAT(24))")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "float8" {
		t.Errorf("expected float8, got %v", tc.TypeName.Names)
	}
	if len(tc.TypeName.Typmods) != 1 {
		t.Fatalf("expected 1 typmod, got %d", len(tc.TypeName.Typmods))
	}
}


func TestCastNumericPrecisionScale(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS NUMERIC(10, 2))")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "numeric" {
		t.Errorf("expected numeric, got %v", tc.TypeName.Names)
	}
	if len(tc.TypeName.Typmods) != 2 {
		t.Fatalf("expected 2 typmods, got %d", len(tc.TypeName.Typmods))
	}
}


func TestCastCharacterVaryingLength(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS CHARACTER VARYING(100))")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "varchar" {
		t.Errorf("expected varchar, got %v", tc.TypeName.Names)
	}
	if len(tc.TypeName.Typmods) != 1 {
		t.Fatalf("expected 1 typmod, got %d", len(tc.TypeName.Typmods))
	}
}


func TestCastBitLength(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS BIT(8))")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if tc.TypeName.Names[1] != "bit" {
		t.Errorf("expected bit, got %v", tc.TypeName.Names)
	}
	if len(tc.TypeName.Typmods) != 1 {
		t.Fatalf("expected 1 typmod, got %d", len(tc.TypeName.Typmods))
	}
}


func TestSetofType(t *testing.T) {
	s := parseOne(t, "SELECT CAST(x AS SETOF integer)")
	sel := s.(*parser.SelectStmt)
	tc := sel.TargetList[0].Val.(*parser.TypeCast)
	if !tc.TypeName.Setof {
		t.Error("expected Setof=true")
	}
	if tc.TypeName.Names[1] != "int4" {
		t.Errorf("expected int4, got %v", tc.TypeName.Names)
	}
}

