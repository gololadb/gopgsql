package parser

import (
	"fmt"
	"strings"
)

func deparseSelect(b *strings.Builder, s *SelectStmt) {
	// Set operations (UNION, INTERSECT, EXCEPT).
	if s.Op != SETOP_NONE {
		deparseSelect(b, s.Larg)
		switch s.Op {
		case SETOP_UNION:
			b.WriteString(" UNION ")
		case SETOP_INTERSECT:
			b.WriteString(" INTERSECT ")
		case SETOP_EXCEPT:
			b.WriteString(" EXCEPT ")
		}
		if s.All {
			b.WriteString("ALL ")
		}
		deparseSelect(b, s.Rarg)
		deparseSortLimitLocking(b, s)
		return
	}

	// VALUES clause.
	if len(s.ValuesLists) > 0 {
		b.WriteString("VALUES ")
		for i, row := range s.ValuesLists {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString("(")
			deparseExprList(b, row)
			b.WriteString(")")
		}
		deparseSortLimitLocking(b, s)
		return
	}

	// WITH clause.
	if s.WithClause != nil {
		deparseWithClause(b, s.WithClause)
	}

	b.WriteString("SELECT")

	// DISTINCT.
	if s.DistinctClause != nil {
		if len(s.DistinctClause) > 0 {
			b.WriteString(" DISTINCT ON (")
			deparseExprList(b, s.DistinctClause)
			b.WriteString(")")
		} else {
			b.WriteString(" DISTINCT")
		}
	}

	// Target list.
	if len(s.TargetList) > 0 {
		b.WriteString(" ")
		for i, rt := range s.TargetList {
			if i > 0 {
				b.WriteString(", ")
			}
			deparseResTarget(b, rt)
		}
	}

	// FROM.
	if len(s.FromClause) > 0 {
		b.WriteString(" FROM ")
		for i, from := range s.FromClause {
			if i > 0 {
				b.WriteString(", ")
			}
			deparseNode(b, from)
		}
	}

	// WHERE.
	if s.WhereClause != nil {
		b.WriteString(" WHERE ")
		deparseExpr(b, s.WhereClause)
	}

	// GROUP BY.
	if len(s.GroupClause) > 0 {
		b.WriteString(" GROUP BY ")
		if s.GroupDistinct {
			b.WriteString("DISTINCT ")
		}
		deparseExprList(b, s.GroupClause)
	}

	// HAVING.
	if s.HavingClause != nil {
		b.WriteString(" HAVING ")
		deparseExpr(b, s.HavingClause)
	}

	// WINDOW.
	if len(s.WindowClause) > 0 {
		b.WriteString(" WINDOW ")
		for i, w := range s.WindowClause {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(w.Name)
			b.WriteString(" AS ")
			deparseWindowDef(b, w)
		}
	}

	deparseSortLimitLocking(b, s)
}

func deparseSortLimitLocking(b *strings.Builder, s *SelectStmt) {
	if len(s.SortClause) > 0 {
		b.WriteString(" ORDER BY ")
		deparseSortList(b, s.SortClause)
	}
	if s.LimitCount != nil {
		b.WriteString(" LIMIT ")
		deparseExpr(b, s.LimitCount)
	}
	if s.LimitOffset != nil {
		b.WriteString(" OFFSET ")
		deparseExpr(b, s.LimitOffset)
	}
	for _, lc := range s.LockingClause {
		deparseLockingClause(b, lc)
	}
}

func deparseResTarget(b *strings.Builder, rt *ResTarget) {
	deparseExpr(b, rt.Val)
	if rt.Name != "" {
		b.WriteString(" AS ")
		b.WriteString(rt.Name)
	}
}

func deparseSortList(b *strings.Builder, sorts []*SortBy) {
	for i, sb := range sorts {
		if i > 0 {
			b.WriteString(", ")
		}
		deparseExpr(b, sb.Node)
		switch sb.SortbyDir {
		case SORTBY_ASC:
			b.WriteString(" ASC")
		case SORTBY_DESC:
			b.WriteString(" DESC")
		case SORTBY_USING:
			if len(sb.UseOp) > 0 {
				b.WriteString(" USING ")
				b.WriteString(strings.Join(sb.UseOp, "."))
			}
		}
		switch sb.SortbyNulls {
		case SORTBY_NULLS_FIRST:
			b.WriteString(" NULLS FIRST")
		case SORTBY_NULLS_LAST:
			b.WriteString(" NULLS LAST")
		}
	}
}

func deparseWindowDef(b *strings.Builder, w *WindowDef) {
	if w.Refname != "" && len(w.PartitionClause) == 0 && len(w.OrderClause) == 0 {
		b.WriteString(w.Refname)
		return
	}
	b.WriteString("(")
	if w.Refname != "" {
		b.WriteString(w.Refname)
		b.WriteString(" ")
	}
	if len(w.PartitionClause) > 0 {
		b.WriteString("PARTITION BY ")
		deparseExprList(b, w.PartitionClause)
	}
	if len(w.OrderClause) > 0 {
		if len(w.PartitionClause) > 0 {
			b.WriteString(" ")
		}
		b.WriteString("ORDER BY ")
		deparseSortList(b, w.OrderClause)
	}
	b.WriteString(")")
}

func deparseLockingClause(b *strings.Builder, lc *LockingClause) {
	switch lc.Strength {
	case LCS_FORUPDATE:
		b.WriteString(" FOR UPDATE")
	case LCS_FORNOKEYUPDATE:
		b.WriteString(" FOR NO KEY UPDATE")
	case LCS_FORSHARE:
		b.WriteString(" FOR SHARE")
	case LCS_FORKEYSHARE:
		b.WriteString(" FOR KEY SHARE")
	}
	if len(lc.LockedRels) > 0 {
		b.WriteString(" OF ")
		for i, rv := range lc.LockedRels {
			if i > 0 {
				b.WriteString(", ")
			}
			deparseNode(b, rv)
		}
	}
}

func deparseWithClause(b *strings.Builder, w *WithClause) {
	b.WriteString("WITH ")
	if w.Recursive {
		b.WriteString("RECURSIVE ")
	}
	for i, cte := range w.CTEs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(cte.Ctename)
		if len(cte.Aliascolnames) > 0 {
			b.WriteString("(")
			b.WriteString(strings.Join(cte.Aliascolnames, ", "))
			b.WriteString(")")
		}
		b.WriteString(" AS ")
		switch cte.CTEMaterialized {
		case CTEMaterializeAlways:
			b.WriteString("MATERIALIZED ")
		case CTEMaterializeNever:
			b.WriteString("NOT MATERIALIZED ")
		}
		b.WriteString("(")
		deparseNode(b, cte.Ctequery)
		b.WriteString(")")
	}
	b.WriteString(" ")
}

func deparseRangeVar(b *strings.Builder, rv *RangeVar) {
	if rv.Schemaname != "" {
		b.WriteString(rv.Schemaname)
		b.WriteString(".")
	}
	b.WriteString(rv.Relname)
	if rv.Alias != nil && rv.Alias.Aliasname != "" {
		b.WriteString(" ")
		b.WriteString(rv.Alias.Aliasname)
		if len(rv.Alias.Colnames) > 0 {
			b.WriteString("(")
			b.WriteString(strings.Join(rv.Alias.Colnames, ", "))
			b.WriteString(")")
		}
	}
}

func deparseJoinExpr(b *strings.Builder, j *JoinExpr) {
	deparseNode(b, j.Larg)
	if j.IsNatural {
		b.WriteString(" NATURAL")
	}
	switch j.Jointype {
	case JOIN_INNER:
		if j.Quals == nil && len(j.UsingClause) == 0 && !j.IsNatural {
			b.WriteString(" CROSS JOIN ")
		} else {
			b.WriteString(" JOIN ")
		}
	case JOIN_LEFT:
		b.WriteString(" LEFT JOIN ")
	case JOIN_FULL:
		b.WriteString(" FULL JOIN ")
	case JOIN_RIGHT:
		b.WriteString(" RIGHT JOIN ")
	case JOIN_CROSS:
		b.WriteString(" CROSS JOIN ")
	}
	deparseNode(b, j.Rarg)
	if j.Quals != nil {
		b.WriteString(" ON ")
		deparseExpr(b, j.Quals)
	}
	if len(j.UsingClause) > 0 {
		b.WriteString(" USING (")
		b.WriteString(strings.Join(j.UsingClause, ", "))
		b.WriteString(")")
	}
	if j.Alias != nil && j.Alias.Aliasname != "" {
		b.WriteString(" ")
		b.WriteString(j.Alias.Aliasname)
	}
}

func deparseRangeSubselect(b *strings.Builder, rs *RangeSubselect) {
	if rs.Lateral {
		b.WriteString("LATERAL ")
	}
	b.WriteString("(")
	deparseNode(b, rs.Subquery)
	b.WriteString(")")
	if rs.Alias != nil && rs.Alias.Aliasname != "" {
		b.WriteString(" ")
		b.WriteString(rs.Alias.Aliasname)
		if len(rs.Alias.Colnames) > 0 {
			b.WriteString("(")
			b.WriteString(strings.Join(rs.Alias.Colnames, ", "))
			b.WriteString(")")
		}
	}
}

func deparseRangeFunction(b *strings.Builder, rf *RangeFunction) {
	if rf.Lateral {
		b.WriteString("LATERAL ")
	}
	for i, fn := range rf.Functions {
		if i > 0 {
			b.WriteString(", ")
		}
		deparseNode(b, fn)
	}
	if rf.Ordinality {
		b.WriteString(" WITH ORDINALITY")
	}
	if rf.Alias != nil && rf.Alias.Aliasname != "" {
		b.WriteString(" AS ")
		b.WriteString(rf.Alias.Aliasname)
		if len(rf.Alias.Colnames) > 0 {
			b.WriteString("(")
			b.WriteString(strings.Join(rf.Alias.Colnames, ", "))
			b.WriteString(")")
		}
	}
}

// --- DML statements ---

func deparseInsert(b *strings.Builder, s *InsertStmt) {
	if s.WithClause != nil {
		deparseWithClause(b, s.WithClause)
	}
	b.WriteString("INSERT INTO ")
	deparseRangeVar(b, s.Relation)
	if len(s.Cols) > 0 {
		b.WriteString(" (")
		for i, rt := range s.Cols {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(rt.Name)
		}
		b.WriteString(")")
	}
	if s.SelectStmt != nil {
		b.WriteString(" ")
		deparseNode(b, s.SelectStmt)
	} else {
		b.WriteString(" DEFAULT VALUES")
	}
	if s.OnConflict != nil {
		deparseOnConflict(b, s.OnConflict)
	}
	if len(s.ReturningList) > 0 {
		b.WriteString(" RETURNING ")
		for i, rt := range s.ReturningList {
			if i > 0 {
				b.WriteString(", ")
			}
			deparseResTarget(b, rt)
		}
	}
}

func deparseOnConflict(b *strings.Builder, oc *OnConflictClause) {
	b.WriteString(" ON CONFLICT")
	if oc.Infer != nil {
		if oc.Infer.Conname != "" {
			b.WriteString(" ON CONSTRAINT ")
			b.WriteString(oc.Infer.Conname)
		} else if len(oc.Infer.IndexElems) > 0 {
			b.WriteString(" (")
			for i, elem := range oc.Infer.IndexElems {
				if i > 0 {
					b.WriteString(", ")
				}
				deparseNode(b, elem)
			}
			b.WriteString(")")
		}
	}
	switch oc.Action {
	case ONCONFLICT_NOTHING:
		b.WriteString(" DO NOTHING")
	case ONCONFLICT_UPDATE:
		b.WriteString(" DO UPDATE SET ")
		for i, rt := range oc.TargetList {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(rt.Name)
			b.WriteString(" = ")
			deparseExpr(b, rt.Val)
		}
		if oc.WhereClause != nil {
			b.WriteString(" WHERE ")
			deparseExpr(b, oc.WhereClause)
		}
	}
}

func deparseUpdate(b *strings.Builder, s *UpdateStmt) {
	if s.WithClause != nil {
		deparseWithClause(b, s.WithClause)
	}
	b.WriteString("UPDATE ")
	deparseRangeVar(b, s.Relation)
	b.WriteString(" SET ")
	for i, rt := range s.TargetList {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(rt.Name)
		b.WriteString(" = ")
		deparseExpr(b, rt.Val)
	}
	if len(s.FromClause) > 0 {
		b.WriteString(" FROM ")
		for i, from := range s.FromClause {
			if i > 0 {
				b.WriteString(", ")
			}
			deparseNode(b, from)
		}
	}
	if s.WhereClause != nil {
		b.WriteString(" WHERE ")
		deparseExpr(b, s.WhereClause)
	}
	if len(s.ReturningList) > 0 {
		b.WriteString(" RETURNING ")
		for i, rt := range s.ReturningList {
			if i > 0 {
				b.WriteString(", ")
			}
			deparseResTarget(b, rt)
		}
	}
}

func deparseDelete(b *strings.Builder, s *DeleteStmt) {
	if s.WithClause != nil {
		deparseWithClause(b, s.WithClause)
	}
	b.WriteString("DELETE FROM ")
	deparseRangeVar(b, s.Relation)
	if len(s.UsingClause) > 0 {
		b.WriteString(" USING ")
		for i, u := range s.UsingClause {
			if i > 0 {
				b.WriteString(", ")
			}
			deparseNode(b, u)
		}
	}
	if s.WhereClause != nil {
		b.WriteString(" WHERE ")
		deparseExpr(b, s.WhereClause)
	}
	if len(s.ReturningList) > 0 {
		b.WriteString(" RETURNING ")
		for i, rt := range s.ReturningList {
			if i > 0 {
				b.WriteString(", ")
			}
			deparseResTarget(b, rt)
		}
	}
}

// --- DDL statements ---

func deparseCreateTable(b *strings.Builder, cs *CreateStmt) {
	b.WriteString("CREATE ")
	switch cs.Persistence {
	case RELPERSISTENCE_TEMP:
		b.WriteString("TEMPORARY ")
	case RELPERSISTENCE_UNLOGGED:
		b.WriteString("UNLOGGED ")
	}
	b.WriteString("TABLE ")
	if cs.IfNotExists {
		b.WriteString("IF NOT EXISTS ")
	}
	deparseRangeVar(b, cs.Relation)
	b.WriteString(" (")
	for i, elt := range cs.TableElts {
		if i > 0 {
			b.WriteString(", ")
		}
		deparseNode(b, elt)
	}
	b.WriteString(")")
	if len(cs.InhRelations) > 0 {
		b.WriteString(" INHERITS (")
		for i, inh := range cs.InhRelations {
			if i > 0 {
				b.WriteString(", ")
			}
			deparseNode(b, inh)
		}
		b.WriteString(")")
	}
	if cs.PartitionSpec != nil {
		deparsePartitionSpec(b, cs.PartitionSpec)
	}
}

func deparsePartitionSpec(b *strings.Builder, ps *PartitionSpec) {
	fmt.Fprintf(b, " PARTITION BY %s (", strings.ToUpper(ps.Strategy))
	for i, pe := range ps.PartParams {
		if i > 0 {
			b.WriteString(", ")
		}
		if pe.Expr != nil {
			b.WriteString("(")
			deparseExpr(b, pe.Expr)
			b.WriteString(")")
		} else {
			b.WriteString(pe.Name)
		}
		if len(pe.Collation) > 0 {
			b.WriteString(" COLLATE ")
			b.WriteString(strings.Join(pe.Collation, "."))
		}
		if len(pe.OpClass) > 0 {
			b.WriteString(" ")
			b.WriteString(strings.Join(pe.OpClass, "."))
		}
	}
	b.WriteString(")")
}

func deparseColumnDef(b *strings.Builder, cd *ColumnDef) {
	b.WriteString(cd.Colname)
	b.WriteString(" ")
	deparseTypeName(b, cd.TypeName)
	for _, c := range cd.Constraints {
		b.WriteString(" ")
		deparseConstraint(b, c)
	}
}

func deparseConstraint(b *strings.Builder, c *Constraint) {
	if c.Conname != "" {
		b.WriteString("CONSTRAINT ")
		b.WriteString(c.Conname)
		b.WriteString(" ")
	}
	switch c.Contype {
	case CONSTR_NULL:
		b.WriteString("NULL")
	case CONSTR_NOTNULL:
		b.WriteString("NOT NULL")
	case CONSTR_DEFAULT:
		b.WriteString("DEFAULT ")
		deparseExpr(b, c.RawExpr)
	case CONSTR_CHECK:
		b.WriteString("CHECK (")
		deparseExpr(b, c.RawExpr)
		b.WriteString(")")
	case CONSTR_PRIMARY:
		b.WriteString("PRIMARY KEY")
		if len(c.Keys) > 0 {
			b.WriteString(" (")
			b.WriteString(strings.Join(c.Keys, ", "))
			b.WriteString(")")
		}
	case CONSTR_UNIQUE:
		b.WriteString("UNIQUE")
		if len(c.Keys) > 0 {
			b.WriteString(" (")
			b.WriteString(strings.Join(c.Keys, ", "))
			b.WriteString(")")
		}
	case CONSTR_FOREIGN:
		b.WriteString("REFERENCES ")
		if c.PkTable != nil {
			deparseRangeVar(b, c.PkTable)
		}
		if len(c.PkAttrs) > 0 {
			b.WriteString(" (")
			b.WriteString(strings.Join(c.PkAttrs, ", "))
			b.WriteString(")")
		}
	case CONSTR_IDENTITY:
		b.WriteString("GENERATED ALWAYS AS IDENTITY")
	}
}

func deparseViewStmt(b *strings.Builder, vs *ViewStmt) {
	b.WriteString("CREATE ")
	if vs.Replace {
		b.WriteString("OR REPLACE ")
	}
	b.WriteString("VIEW ")
	deparseRangeVar(b, vs.View)
	if len(vs.Aliases) > 0 {
		b.WriteString(" (")
		b.WriteString(strings.Join(vs.Aliases, ", "))
		b.WriteString(")")
	}
	b.WriteString(" AS ")
	deparseNode(b, vs.Query)
}

func deparseIndexStmt(b *strings.Builder, is *IndexStmt) {
	b.WriteString("CREATE ")
	if is.Unique {
		b.WriteString("UNIQUE ")
	}
	b.WriteString("INDEX ")
	if is.Concurrent {
		b.WriteString("CONCURRENTLY ")
	}
	if is.IfNotExists {
		b.WriteString("IF NOT EXISTS ")
	}
	if is.Idxname != "" {
		b.WriteString(is.Idxname)
		b.WriteString(" ")
	}
	b.WriteString("ON ")
	deparseRangeVar(b, is.Relation)
	if is.AccessMethod != "" {
		b.WriteString(" USING ")
		b.WriteString(is.AccessMethod)
	}
	b.WriteString(" (")
	for i, ie := range is.IndexParams {
		if i > 0 {
			b.WriteString(", ")
		}
		if ie.Name != "" {
			b.WriteString(ie.Name)
		} else if ie.Expr != nil {
			deparseExpr(b, ie.Expr)
		}
		if ie.Ordering == SORTBY_DESC {
			b.WriteString(" DESC")
		}
		if ie.NullsOrder == SORTBY_NULLS_FIRST {
			b.WriteString(" NULLS FIRST")
		} else if ie.NullsOrder == SORTBY_NULLS_LAST {
			b.WriteString(" NULLS LAST")
		}
	}
	b.WriteString(")")
	if is.WhereClause != nil {
		b.WriteString(" WHERE ")
		deparseExpr(b, is.WhereClause)
	}
}

func deparseExplainStmt(b *strings.Builder, es *ExplainStmt) {
	b.WriteString("EXPLAIN ")
	if len(es.Options) > 0 {
		b.WriteString("(")
		for i, opt := range es.Options {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(strings.ToUpper(opt.Defname))
		}
		b.WriteString(") ")
	}
	deparseNode(b, es.Query)
}
