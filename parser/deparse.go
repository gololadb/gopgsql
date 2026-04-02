package parser

import (
	"fmt"
	"strings"
)

// Deparse converts a parsed AST node back into SQL text.
func Deparse(node Node) string {
	var b strings.Builder
	deparseNode(&b, node)
	return b.String()
}

// DeparseExpr converts a parsed expression back into SQL text.
func DeparseExpr(expr Expr) string {
	var b strings.Builder
	deparseExpr(&b, expr)
	return b.String()
}

func deparseNode(b *strings.Builder, node Node) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *RawStmt:
		deparseNode(b, n.Stmt)
	case *SelectStmt:
		deparseSelect(b, n)
	case *InsertStmt:
		deparseInsert(b, n)
	case *UpdateStmt:
		deparseUpdate(b, n)
	case *DeleteStmt:
		deparseDelete(b, n)
	case *CreateStmt:
		deparseCreateTable(b, n)
	case *ViewStmt:
		deparseViewStmt(b, n)
	case *IndexStmt:
		deparseIndexStmt(b, n)
	case *ExplainStmt:
		deparseExplainStmt(b, n)
	case *ColumnDef:
		deparseColumnDef(b, n)
	case *RangeVar:
		deparseRangeVar(b, n)
	case *JoinExpr:
		deparseJoinExpr(b, n)
	case *RangeSubselect:
		deparseRangeSubselect(b, n)
	case *RangeFunction:
		deparseRangeFunction(b, n)
	case *String:
		b.WriteString(n.Str)
	case *A_Star:
		b.WriteString("*")
	default:
		// For expression nodes, try deparseExpr.
		if e, ok := node.(Expr); ok {
			deparseExpr(b, e)
		} else {
			b.WriteString(fmt.Sprintf("/* unsupported node %T */", node))
		}
	}
}

func deparseExpr(b *strings.Builder, expr Expr) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *A_Const:
		deparseConst(b, e)
	case *ColumnRef:
		deparseColumnRef(b, e)
	case *A_Expr:
		deparseAExpr(b, e)
	case *BoolExpr:
		deparseBoolExpr(b, e)
	case *TypeCast:
		deparseTypeCast(b, e)
	case *FuncCall:
		deparseFuncCall(b, e)
	case *SubLink:
		deparseSubLink(b, e)
	case *NullTest:
		deparseNullTest(b, e)
	case *BooleanTest:
		deparseBooleanTest(b, e)
	case *CaseExpr:
		deparseCaseExpr(b, e)
	case *CoalesceExpr:
		b.WriteString("COALESCE(")
		deparseExprList(b, e.Args)
		b.WriteString(")")
	case *NullIfExpr:
		b.WriteString("NULLIF(")
		deparseExprList(b, e.Args)
		b.WriteString(")")
	case *MinMaxExpr:
		if e.Op == IS_GREATEST {
			b.WriteString("GREATEST(")
		} else {
			b.WriteString("LEAST(")
		}
		deparseExprList(b, e.Args)
		b.WriteString(")")
	case *A_ArrayExpr:
		b.WriteString("ARRAY[")
		deparseExprList(b, e.Elements)
		b.WriteString("]")
	case *RowExpr:
		b.WriteString("ROW(")
		deparseExprList(b, e.Args)
		b.WriteString(")")
	case *ParamRef:
		fmt.Fprintf(b, "$%d", e.Number)
	case *SQLValueFunction:
		deparseSVF(b, e)
	case *A_Indirection:
		deparseIndirection(b, e)
	case *CollateClause:
		deparseExpr(b, e.Arg)
		b.WriteString(" COLLATE ")
		b.WriteString(strings.Join(e.Collname, "."))
	case *NamedArgExpr:
		b.WriteString(e.Name)
		b.WriteString(" => ")
		deparseExpr(b, e.Arg)
	case *ExprList:
		deparseExprList(b, e.Items)
	case *RangeVar:
		deparseRangeVar(b, e)
	default:
		b.WriteString(fmt.Sprintf("/* unsupported expr %T */", expr))
	}
}

func deparseExprList(b *strings.Builder, exprs []Expr) {
	for i, e := range exprs {
		if i > 0 {
			b.WriteString(", ")
		}
		deparseExpr(b, e)
	}
}
