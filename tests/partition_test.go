package tests

import (
	"testing"

	"github.com/gololadb/gopgsql/parser"
)

func TestCreateTablePartitionOf_Range(t *testing.T) {
	stmt := parseOne(t, `CREATE TABLE payment_p2022_01 PARTITION OF payment FOR VALUES FROM ('2022-01-01') TO ('2022-02-01')`)
	cs := stmt.(*parser.CreateStmt)
	if cs.Relation.Relname != "payment_p2022_01" {
		t.Fatalf("expected child=payment_p2022_01, got %s", cs.Relation.Relname)
	}
	if cs.PartitionOf == nil || cs.PartitionOf.Relname != "payment" {
		t.Fatal("expected PartitionOf=payment")
	}
	if cs.PartBound == nil {
		t.Fatal("expected PartBound")
	}
	if cs.PartBound.Strategy != "range" {
		t.Fatalf("expected strategy=range, got %s", cs.PartBound.Strategy)
	}
	if len(cs.PartBound.LowerBound) != 1 || len(cs.PartBound.UpperBound) != 1 {
		t.Fatalf("expected 1 bound value each, got lower=%d upper=%d", len(cs.PartBound.LowerBound), len(cs.PartBound.UpperBound))
	}
}

func TestCreateTablePartitionOf_List(t *testing.T) {
	stmt := parseOne(t, `CREATE TABLE sales_east PARTITION OF sales FOR VALUES IN ('east', 'northeast')`)
	cs := stmt.(*parser.CreateStmt)
	if cs.PartitionOf == nil || cs.PartitionOf.Relname != "sales" {
		t.Fatal("expected PartitionOf=sales")
	}
	if cs.PartBound.Strategy != "list" {
		t.Fatalf("expected strategy=list, got %s", cs.PartBound.Strategy)
	}
	if len(cs.PartBound.ListValues) != 2 {
		t.Fatalf("expected 2 list values, got %d", len(cs.PartBound.ListValues))
	}
}

func TestCreateTablePartitionOf_Default(t *testing.T) {
	stmt := parseOne(t, `CREATE TABLE sales_other PARTITION OF sales DEFAULT`)
	cs := stmt.(*parser.CreateStmt)
	if cs.PartitionOf == nil || cs.PartitionOf.Relname != "sales" {
		t.Fatal("expected PartitionOf=sales")
	}
	if !cs.PartBound.IsDefault {
		t.Fatal("expected IsDefault=true")
	}
}

func TestCreateTablePartitionOf_SchemaQualified(t *testing.T) {
	stmt := parseOne(t, `CREATE TABLE public.payment_p2022_01 PARTITION OF public.payment FOR VALUES FROM ('2022-01-01 00:00:00+00') TO ('2022-02-01 00:00:00+00')`)
	cs := stmt.(*parser.CreateStmt)
	if cs.Relation.Schemaname != "public" || cs.Relation.Relname != "payment_p2022_01" {
		t.Fatalf("unexpected child: %s.%s", cs.Relation.Schemaname, cs.Relation.Relname)
	}
	if cs.PartitionOf.Schemaname != "public" || cs.PartitionOf.Relname != "payment" {
		t.Fatalf("unexpected parent: %s.%s", cs.PartitionOf.Schemaname, cs.PartitionOf.Relname)
	}
}

func TestCreateTablePartitionOf_IfNotExists(t *testing.T) {
	stmt := parseOne(t, `CREATE TABLE IF NOT EXISTS part_child PARTITION OF parent FOR VALUES IN (1, 2)`)
	cs := stmt.(*parser.CreateStmt)
	if !cs.IfNotExists {
		t.Fatal("expected IfNotExists=true")
	}
	if cs.PartitionOf == nil {
		t.Fatal("expected PartitionOf")
	}
}

func TestAlterTableAttachPartition_Range(t *testing.T) {
	stmt := parseOne(t, `ALTER TABLE payment ATTACH PARTITION payment_p2022_01 FOR VALUES FROM ('2022-01-01') TO ('2022-02-01')`)
	at := stmt.(*parser.AlterTableStmt)
	if len(at.Cmds) != 1 {
		t.Fatalf("expected 1 cmd, got %d", len(at.Cmds))
	}
	if at.Cmds[0].Subtype != parser.AT_AttachPartition {
		t.Fatalf("expected AT_AttachPartition, got %d", at.Cmds[0].Subtype)
	}
	pc := at.Cmds[0].Def.(*parser.PartitionCmd)
	if pc.Name.Relname != "payment_p2022_01" {
		t.Fatalf("expected child=payment_p2022_01, got %s", pc.Name.Relname)
	}
	if pc.Bound.Strategy != "range" {
		t.Fatalf("expected strategy=range, got %s", pc.Bound.Strategy)
	}
}

func TestAlterTableAttachPartition_List(t *testing.T) {
	stmt := parseOne(t, `ALTER TABLE sales ATTACH PARTITION sales_east FOR VALUES IN ('east')`)
	at := stmt.(*parser.AlterTableStmt)
	pc := at.Cmds[0].Def.(*parser.PartitionCmd)
	if pc.Bound.Strategy != "list" {
		t.Fatalf("expected strategy=list, got %s", pc.Bound.Strategy)
	}
}

func TestAlterTableAttachPartition_Default(t *testing.T) {
	stmt := parseOne(t, `ALTER TABLE logs ATTACH PARTITION logs_other DEFAULT`)
	at := stmt.(*parser.AlterTableStmt)
	pc := at.Cmds[0].Def.(*parser.PartitionCmd)
	if !pc.Bound.IsDefault {
		t.Fatal("expected IsDefault=true")
	}
}

func TestAlterTableAttachPartition_Only(t *testing.T) {
	stmt := parseOne(t, `ALTER TABLE ONLY public.payment ATTACH PARTITION public.payment_p2022_01 FOR VALUES FROM ('2022-01-01 00:00:00+00') TO ('2022-02-01 00:00:00+00')`)
	at := stmt.(*parser.AlterTableStmt)
	if at.Relation.Inh {
		t.Fatal("expected Inh=false for ONLY")
	}
	if at.Cmds[0].Subtype != parser.AT_AttachPartition {
		t.Fatalf("expected AT_AttachPartition, got %d", at.Cmds[0].Subtype)
	}
	pc := at.Cmds[0].Def.(*parser.PartitionCmd)
	if pc.Name.Schemaname != "public" || pc.Name.Relname != "payment_p2022_01" {
		t.Fatalf("unexpected child: %s.%s", pc.Name.Schemaname, pc.Name.Relname)
	}
}

func TestAlterTableDetachPartition(t *testing.T) {
	stmt := parseOne(t, `ALTER TABLE sales DETACH PARTITION sales_east`)
	at := stmt.(*parser.AlterTableStmt)
	if len(at.Cmds) != 1 {
		t.Fatalf("expected 1 cmd, got %d", len(at.Cmds))
	}
	if at.Cmds[0].Subtype != parser.AT_DetachPartition {
		t.Fatalf("expected AT_DetachPartition, got %d", at.Cmds[0].Subtype)
	}
	pc := at.Cmds[0].Def.(*parser.PartitionCmd)
	if pc.Name.Relname != "sales_east" {
		t.Fatalf("expected child=sales_east, got %s", pc.Name.Relname)
	}
}

func TestClusterStmt(t *testing.T) {
	tests := []struct {
		sql   string
		table string
		index string
	}{
		{"CLUSTER", "", ""},
		{"CLUSTER mytable", "mytable", ""},
		{"CLUSTER mytable USING myindex", "mytable", "myindex"},
	}
	for _, tt := range tests {
		t.Run(tt.sql, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cs := stmt.(*parser.ClusterStmt)
			if tt.table == "" {
				if cs.Relation != nil {
					t.Fatalf("expected no relation, got %s", cs.Relation.Relname)
				}
			} else {
				if cs.Relation == nil || cs.Relation.Relname != tt.table {
					t.Fatalf("expected table=%s", tt.table)
				}
			}
			if cs.IndexName != tt.index {
				t.Fatalf("expected index=%q, got %q", tt.index, cs.IndexName)
			}
		})
	}
}

func TestDeparsePartitionOf(t *testing.T) {
	sql := `CREATE TABLE child PARTITION OF parent FOR VALUES FROM ('2024-01-01') TO ('2025-01-01')`
	stmt := parseOne(t, sql)
	got := parser.Deparse(stmt)
	if got != sql {
		t.Fatalf("deparse mismatch:\n  got:  %s\n  want: %s", got, sql)
	}
}

func TestDeparseAttachPartition(t *testing.T) {
	sql := `ALTER TABLE ONLY public.payment ATTACH PARTITION public.payment_p2022_01 FOR VALUES FROM ('2022-01-01') TO ('2022-02-01')`
	// Just verify it parses and deparses without panic
	stmt := parseOne(t, sql)
	_ = parser.Deparse(stmt)
}
