package parser

import (
	"fmt"
	"strings"
)

func deparseConst(b *strings.Builder, c *A_Const) {
	switch c.Val.Type {
	case ValInt:
		fmt.Fprintf(b, "%d", c.Val.Ival)
	case ValFloat:
		b.WriteString(c.Val.Str)
	case ValStr:
		b.WriteString("'")
		b.WriteString(strings.ReplaceAll(c.Val.Str, "'", "''"))
		b.WriteString("'")
	case ValBitStr:
		b.WriteString("B'")
		b.WriteString(c.Val.Str)
		b.WriteString("'")
	case ValNull:
		b.WriteString("NULL")
	case ValBool:
		if c.Val.Bool {
			b.WriteString("TRUE")
		} else {
			b.WriteString("FALSE")
		}
	}
}

func deparseColumnRef(b *strings.Builder, ref *ColumnRef) {
	for i, f := range ref.Fields {
		if i > 0 {
			b.WriteString(".")
		}
		switch n := f.(type) {
		case *String:
			b.WriteString(n.Str)
		case *A_Star:
			b.WriteString("*")
		}
	}
}

func deparseAExpr(b *strings.Builder, e *A_Expr) {
	opName := ""
	if len(e.Name) > 0 {
		opName = e.Name[len(e.Name)-1]
	}

	switch e.Kind {
	case AEXPR_OP:
		if e.Lexpr == nil {
			// Prefix operator.
			b.WriteString(opName)
			deparseExpr(b, e.Rexpr)
			return
		}
		deparseExpr(b, e.Lexpr)
		fmt.Fprintf(b, " %s ", opName)
		deparseExpr(b, e.Rexpr)

	case AEXPR_OP_ANY:
		deparseExpr(b, e.Lexpr)
		fmt.Fprintf(b, " %s ANY(", opName)
		deparseExpr(b, e.Rexpr)
		b.WriteString(")")

	case AEXPR_OP_ALL:
		deparseExpr(b, e.Lexpr)
		fmt.Fprintf(b, " %s ALL(", opName)
		deparseExpr(b, e.Rexpr)
		b.WriteString(")")

	case AEXPR_DISTINCT:
		deparseExpr(b, e.Lexpr)
		b.WriteString(" IS DISTINCT FROM ")
		deparseExpr(b, e.Rexpr)

	case AEXPR_NOT_DISTINCT:
		deparseExpr(b, e.Lexpr)
		b.WriteString(" IS NOT DISTINCT FROM ")
		deparseExpr(b, e.Rexpr)

	case AEXPR_IN:
		deparseExpr(b, e.Lexpr)
		if opName == "<>" {
			b.WriteString(" NOT IN (")
		} else {
			b.WriteString(" IN (")
		}
		switch list := e.Rexpr.(type) {
		case *A_ArrayExpr:
			deparseExprList(b, list.Elements)
		case *ExprList:
			deparseExprList(b, list.Items)
		default:
			deparseExpr(b, e.Rexpr)
		}
		b.WriteString(")")

	case AEXPR_LIKE:
		deparseExpr(b, e.Lexpr)
		if opName == "!~~" {
			b.WriteString(" NOT LIKE ")
		} else {
			b.WriteString(" LIKE ")
		}
		deparseExpr(b, e.Rexpr)

	case AEXPR_ILIKE:
		deparseExpr(b, e.Lexpr)
		if opName == "!~~*" {
			b.WriteString(" NOT ILIKE ")
		} else {
			b.WriteString(" ILIKE ")
		}
		deparseExpr(b, e.Rexpr)

	case AEXPR_SIMILAR:
		deparseExpr(b, e.Lexpr)
		if opName == "!~" {
			b.WriteString(" NOT SIMILAR TO ")
		} else {
			b.WriteString(" SIMILAR TO ")
		}
		deparseExpr(b, e.Rexpr)

	case AEXPR_BETWEEN:
		deparseExpr(b, e.Lexpr)
		b.WriteString(" BETWEEN ")
		deparseBetweenBounds(b, e.Rexpr)

	case AEXPR_NOT_BETWEEN:
		deparseExpr(b, e.Lexpr)
		b.WriteString(" NOT BETWEEN ")
		deparseBetweenBounds(b, e.Rexpr)

	case AEXPR_BETWEEN_SYM:
		deparseExpr(b, e.Lexpr)
		b.WriteString(" BETWEEN SYMMETRIC ")
		deparseBetweenBounds(b, e.Rexpr)

	case AEXPR_NOT_BETWEEN_SYM:
		deparseExpr(b, e.Lexpr)
		b.WriteString(" NOT BETWEEN SYMMETRIC ")
		deparseBetweenBounds(b, e.Rexpr)

	default:
		deparseExpr(b, e.Lexpr)
		fmt.Fprintf(b, " %s ", opName)
		deparseExpr(b, e.Rexpr)
	}
}

func deparseBoolExpr(b *strings.Builder, e *BoolExpr) {
	switch e.Op {
	case AND_EXPR:
		for i, arg := range e.Args {
			if i > 0 {
				b.WriteString(" AND ")
			}
			needParen := false
			if be, ok := arg.(*BoolExpr); ok && be.Op == OR_EXPR {
				needParen = true
			}
			if needParen {
				b.WriteString("(")
			}
			deparseExpr(b, arg)
			if needParen {
				b.WriteString(")")
			}
		}
	case OR_EXPR:
		for i, arg := range e.Args {
			if i > 0 {
				b.WriteString(" OR ")
			}
			needParen := false
			if be, ok := arg.(*BoolExpr); ok && be.Op == AND_EXPR {
				needParen = true
			}
			if needParen {
				b.WriteString("(")
			}
			deparseExpr(b, arg)
			if needParen {
				b.WriteString(")")
			}
		}
	case NOT_EXPR:
		b.WriteString("NOT ")
		if len(e.Args) > 0 {
			needParen := false
			if _, ok := e.Args[0].(*BoolExpr); ok {
				needParen = true
			}
			if needParen {
				b.WriteString("(")
			}
			deparseExpr(b, e.Args[0])
			if needParen {
				b.WriteString(")")
			}
		}
	}
}

func deparseTypeCast(b *strings.Builder, e *TypeCast) {
	deparseExpr(b, e.Arg)
	b.WriteString("::")
	deparseTypeName(b, e.TypeName)
}

func deparseTypeName(b *strings.Builder, tn *TypeName) {
	if tn == nil {
		return
	}
	if tn.Setof {
		b.WriteString("SETOF ")
	}
	// Map pg_catalog names back to SQL keywords.
	name := typeNameToSQL(tn.Names)
	b.WriteString(name)
	if len(tn.Typmods) > 0 && name != "interval" {
		b.WriteString("(")
		deparseExprList(b, tn.Typmods)
		b.WriteString(")")
	}
	for range tn.ArrayBounds {
		b.WriteString("[]")
	}
}

func typeNameToSQL(names []string) string {
	if len(names) == 2 && names[0] == "pg_catalog" {
		switch names[1] {
		case "int4":
			return "integer"
		case "int2":
			return "smallint"
		case "int8":
			return "bigint"
		case "float4":
			return "real"
		case "float8":
			return "double precision"
		case "numeric":
			return "numeric"
		case "bool":
			return "boolean"
		case "text":
			return "text"
		case "varchar":
			return "varchar"
		case "bpchar":
			return "char"
		case "timestamp":
			return "timestamp"
		case "timestamptz":
			return "timestamp with time zone"
		case "time":
			return "time"
		case "timetz":
			return "time with time zone"
		case "interval":
			return "interval"
		case "json":
			return "json"
		case "jsonb":
			return "jsonb"
		case "uuid":
			return "uuid"
		case "xml":
			return "xml"
		case "bytea":
			return "bytea"
		case "bit":
			return "bit"
		case "varbit":
			return "bit varying"
		default:
			return names[1]
		}
	}
	return strings.Join(names, ".")
}

func deparseFuncCall(b *strings.Builder, f *FuncCall) {
	b.WriteString(strings.Join(f.Funcname, "."))
	b.WriteString("(")
	if f.AggStar {
		b.WriteString("*")
	} else {
		if f.AggDistinct {
			b.WriteString("DISTINCT ")
		}
		if f.FuncVariadic && len(f.Args) > 0 {
			for i, arg := range f.Args {
				if i > 0 {
					b.WriteString(", ")
				}
				if i == len(f.Args)-1 {
					b.WriteString("VARIADIC ")
				}
				deparseExpr(b, arg)
			}
		} else {
			deparseExprList(b, f.Args)
		}
		if len(f.AggOrder) > 0 {
			b.WriteString(" ORDER BY ")
			deparseSortList(b, f.AggOrder)
		}
	}
	b.WriteString(")")
	if f.AggFilter != nil {
		b.WriteString(" FILTER (WHERE ")
		deparseExpr(b, f.AggFilter)
		b.WriteString(")")
	}
	if len(f.AggWithinGroup) > 0 {
		b.WriteString(" WITHIN GROUP (ORDER BY ")
		deparseSortList(b, f.AggWithinGroup)
		b.WriteString(")")
	}
	if f.Over != nil {
		b.WriteString(" OVER ")
		deparseWindowDef(b, f.Over)
	}
}

func deparseSubLink(b *strings.Builder, s *SubLink) {
	switch s.SubLinkType {
	case EXISTS_SUBLINK:
		b.WriteString("EXISTS (")
		deparseNode(b, s.Subselect)
		b.WriteString(")")
	case ANY_SUBLINK:
		deparseExpr(b, s.Testexpr)
		op := "="
		if len(s.OperName) > 0 {
			op = s.OperName[len(s.OperName)-1]
		}
		fmt.Fprintf(b, " %s ANY (", op)
		deparseNode(b, s.Subselect)
		b.WriteString(")")
	case ALL_SUBLINK:
		deparseExpr(b, s.Testexpr)
		op := "="
		if len(s.OperName) > 0 {
			op = s.OperName[len(s.OperName)-1]
		}
		fmt.Fprintf(b, " %s ALL (", op)
		deparseNode(b, s.Subselect)
		b.WriteString(")")
	case EXPR_SUBLINK:
		b.WriteString("(")
		deparseNode(b, s.Subselect)
		b.WriteString(")")
	case ARRAY_SUBLINK:
		b.WriteString("ARRAY(")
		deparseNode(b, s.Subselect)
		b.WriteString(")")
	default:
		b.WriteString("(")
		deparseNode(b, s.Subselect)
		b.WriteString(")")
	}
}

func deparseNullTest(b *strings.Builder, e *NullTest) {
	deparseExpr(b, e.Arg)
	if e.NullTestType == IS_NULL {
		b.WriteString(" IS NULL")
	} else {
		b.WriteString(" IS NOT NULL")
	}
}

func deparseBooleanTest(b *strings.Builder, e *BooleanTest) {
	deparseExpr(b, e.Arg)
	switch e.BooltestType {
	case IS_TRUE:
		b.WriteString(" IS TRUE")
	case IS_NOT_TRUE:
		b.WriteString(" IS NOT TRUE")
	case IS_FALSE:
		b.WriteString(" IS FALSE")
	case IS_NOT_FALSE:
		b.WriteString(" IS NOT FALSE")
	case IS_UNKNOWN:
		b.WriteString(" IS UNKNOWN")
	case IS_NOT_UNKNOWN:
		b.WriteString(" IS NOT UNKNOWN")
	}
}

func deparseCaseExpr(b *strings.Builder, e *CaseExpr) {
	b.WriteString("CASE")
	if e.Arg != nil {
		b.WriteString(" ")
		deparseExpr(b, e.Arg)
	}
	for _, w := range e.Args {
		b.WriteString(" WHEN ")
		deparseExpr(b, w.Expr)
		b.WriteString(" THEN ")
		deparseExpr(b, w.Result)
	}
	if e.Defresult != nil {
		b.WriteString(" ELSE ")
		deparseExpr(b, e.Defresult)
	}
	b.WriteString(" END")
}

func deparseSVF(b *strings.Builder, e *SQLValueFunction) {
	switch e.Op {
	case SVFOP_CURRENT_DATE:
		b.WriteString("CURRENT_DATE")
	case SVFOP_CURRENT_TIME:
		b.WriteString("CURRENT_TIME")
	case SVFOP_CURRENT_TIME_N:
		fmt.Fprintf(b, "CURRENT_TIME(%d)", e.Typmod)
	case SVFOP_CURRENT_TIMESTAMP:
		b.WriteString("CURRENT_TIMESTAMP")
	case SVFOP_CURRENT_TIMESTAMP_N:
		fmt.Fprintf(b, "CURRENT_TIMESTAMP(%d)", e.Typmod)
	case SVFOP_LOCALTIME:
		b.WriteString("LOCALTIME")
	case SVFOP_LOCALTIME_N:
		fmt.Fprintf(b, "LOCALTIME(%d)", e.Typmod)
	case SVFOP_LOCALTIMESTAMP:
		b.WriteString("LOCALTIMESTAMP")
	case SVFOP_LOCALTIMESTAMP_N:
		fmt.Fprintf(b, "LOCALTIMESTAMP(%d)", e.Typmod)
	case SVFOP_CURRENT_ROLE:
		b.WriteString("CURRENT_ROLE")
	case SVFOP_CURRENT_USER:
		b.WriteString("CURRENT_USER")
	case SVFOP_USER:
		b.WriteString("USER")
	case SVFOP_SESSION_USER:
		b.WriteString("SESSION_USER")
	case SVFOP_CURRENT_CATALOG:
		b.WriteString("CURRENT_CATALOG")
	case SVFOP_CURRENT_SCHEMA:
		b.WriteString("CURRENT_SCHEMA")
	}
}

func deparseBetweenBounds(b *strings.Builder, expr Expr) {
	switch list := expr.(type) {
	case *ExprList:
		if len(list.Items) == 2 {
			deparseExpr(b, list.Items[0])
			b.WriteString(" AND ")
			deparseExpr(b, list.Items[1])
			return
		}
	case *A_ArrayExpr:
		if len(list.Elements) == 2 {
			deparseExpr(b, list.Elements[0])
			b.WriteString(" AND ")
			deparseExpr(b, list.Elements[1])
			return
		}
	}
	deparseExpr(b, expr)
}

func deparseIndirection(b *strings.Builder, e *A_Indirection) {
	deparseExpr(b, e.Arg)
	for _, ind := range e.Indirection {
		switch n := ind.(type) {
		case *String:
			b.WriteString(".")
			b.WriteString(n.Str)
		case *A_Indices:
			b.WriteString("[")
			if n.IsSlice {
				if n.Lidx != nil {
					deparseExpr(b, n.Lidx)
				}
				b.WriteString(":")
				if n.Uidx != nil {
					deparseExpr(b, n.Uidx)
				}
			} else {
				deparseExpr(b, n.Uidx)
			}
			b.WriteString("]")
		case *A_Star:
			b.WriteString(".*")
		}
	}
}
