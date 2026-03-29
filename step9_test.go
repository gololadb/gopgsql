package pgscan

import "testing"

// ---------------------------------------------------------------------------
// Step 9: CREATE INDEX / ALTER TABLE / DROP / TRUNCATE
// ---------------------------------------------------------------------------

func TestCreateIndex(t *testing.T) {
	s := parseOne(t, "CREATE INDEX idx_name ON t (col1, col2)")
	idx, ok := s.(*IndexStmt)
	if !ok {
		t.Fatalf("expected IndexStmt, got %T", s)
	}
	if idx.Idxname != "idx_name" {
		t.Errorf("expected idx_name, got %s", idx.Idxname)
	}
	if idx.Relation.Relname != "t" {
		t.Errorf("expected t, got %s", idx.Relation.Relname)
	}
	if len(idx.IndexParams) != 2 {
		t.Fatalf("expected 2 params, got %d", len(idx.IndexParams))
	}
	if idx.IndexParams[0].Name != "col1" {
		t.Errorf("expected col1, got %s", idx.IndexParams[0].Name)
	}
}

func TestCreateUniqueIndex(t *testing.T) {
	s := parseOne(t, "CREATE UNIQUE INDEX idx ON t (col)")
	idx := s.(*IndexStmt)
	if !idx.Unique {
		t.Error("expected Unique=true")
	}
}

func TestCreateIndexConcurrently(t *testing.T) {
	s := parseOne(t, "CREATE INDEX CONCURRENTLY idx ON t (col)")
	idx := s.(*IndexStmt)
	if !idx.Concurrent {
		t.Error("expected Concurrent=true")
	}
}

func TestCreateIndexIfNotExists(t *testing.T) {
	s := parseOne(t, "CREATE INDEX IF NOT EXISTS idx ON t (col)")
	idx := s.(*IndexStmt)
	if !idx.IfNotExists {
		t.Error("expected IfNotExists=true")
	}
}

func TestCreateIndexUsing(t *testing.T) {
	s := parseOne(t, "CREATE INDEX idx ON t USING gin (col)")
	idx := s.(*IndexStmt)
	if idx.AccessMethod != "gin" {
		t.Errorf("expected gin, got %s", idx.AccessMethod)
	}
}

func TestCreateIndexWhere(t *testing.T) {
	s := parseOne(t, "CREATE INDEX idx ON t (col) WHERE active = true")
	idx := s.(*IndexStmt)
	if idx.WhereClause == nil {
		t.Error("expected WHERE clause")
	}
}

func TestCreateIndexDesc(t *testing.T) {
	s := parseOne(t, "CREATE INDEX idx ON t (col DESC NULLS LAST)")
	idx := s.(*IndexStmt)
	if idx.IndexParams[0].Ordering != SORTBY_DESC {
		t.Error("expected DESC")
	}
	if idx.IndexParams[0].NullsOrder != SORTBY_NULLS_LAST {
		t.Error("expected NULLS LAST")
	}
}

func TestCreateIndexNoName(t *testing.T) {
	s := parseOne(t, "CREATE INDEX ON t (col)")
	idx := s.(*IndexStmt)
	if idx.Idxname != "" {
		t.Errorf("expected empty name, got %s", idx.Idxname)
	}
}

func TestAlterTableAddColumn(t *testing.T) {
	s := parseOne(t, "ALTER TABLE t ADD COLUMN name text NOT NULL")
	alt, ok := s.(*AlterTableStmt)
	if !ok {
		t.Fatalf("expected AlterTableStmt, got %T", s)
	}
	if len(alt.Cmds) != 1 {
		t.Fatalf("expected 1 cmd, got %d", len(alt.Cmds))
	}
	cmd := alt.Cmds[0]
	if cmd.Subtype != AT_AddColumn {
		t.Errorf("expected AT_AddColumn, got %d", cmd.Subtype)
	}
	cd := cmd.Def.(*ColumnDef)
	if cd.Colname != "name" {
		t.Errorf("expected name, got %s", cd.Colname)
	}
}

func TestAlterTableDropColumn(t *testing.T) {
	s := parseOne(t, "ALTER TABLE t DROP COLUMN old_col CASCADE")
	alt := s.(*AlterTableStmt)
	cmd := alt.Cmds[0]
	if cmd.Subtype != AT_DropColumn {
		t.Errorf("expected AT_DropColumn, got %d", cmd.Subtype)
	}
	if cmd.Name != "old_col" {
		t.Errorf("expected old_col, got %s", cmd.Name)
	}
	if cmd.Behavior != DROP_CASCADE {
		t.Error("expected CASCADE")
	}
}

func TestAlterTableAlterType(t *testing.T) {
	s := parseOne(t, "ALTER TABLE t ALTER COLUMN col SET DATA TYPE bigint")
	alt := s.(*AlterTableStmt)
	cmd := alt.Cmds[0]
	if cmd.Subtype != AT_AlterColumnType {
		t.Errorf("expected AT_AlterColumnType, got %d", cmd.Subtype)
	}
	tn := cmd.Def.(*TypeName)
	if tn.Names[1] != "int8" {
		t.Errorf("expected int8, got %v", tn.Names)
	}
}

func TestAlterTableSetNotNull(t *testing.T) {
	s := parseOne(t, "ALTER TABLE t ALTER COLUMN col SET NOT NULL")
	alt := s.(*AlterTableStmt)
	if alt.Cmds[0].Subtype != AT_SetNotNull {
		t.Errorf("expected AT_SetNotNull, got %d", alt.Cmds[0].Subtype)
	}
}

func TestAlterTableDropNotNull(t *testing.T) {
	s := parseOne(t, "ALTER TABLE t ALTER COLUMN col DROP NOT NULL")
	alt := s.(*AlterTableStmt)
	if alt.Cmds[0].Subtype != AT_DropNotNull {
		t.Errorf("expected AT_DropNotNull, got %d", alt.Cmds[0].Subtype)
	}
}

func TestAlterTableSetDefault(t *testing.T) {
	s := parseOne(t, "ALTER TABLE t ALTER COLUMN col SET DEFAULT 0")
	alt := s.(*AlterTableStmt)
	if alt.Cmds[0].Subtype != AT_SetDefault {
		t.Errorf("expected AT_SetDefault, got %d", alt.Cmds[0].Subtype)
	}
}

func TestAlterTableDropDefault(t *testing.T) {
	s := parseOne(t, "ALTER TABLE t ALTER COLUMN col DROP DEFAULT")
	alt := s.(*AlterTableStmt)
	if alt.Cmds[0].Subtype != AT_DropDefault {
		t.Errorf("expected AT_DropDefault, got %d", alt.Cmds[0].Subtype)
	}
}

func TestAlterTableAddConstraint(t *testing.T) {
	s := parseOne(t, "ALTER TABLE t ADD CONSTRAINT uq_col UNIQUE (col)")
	alt := s.(*AlterTableStmt)
	cmd := alt.Cmds[0]
	if cmd.Subtype != AT_AddConstraint {
		t.Errorf("expected AT_AddConstraint, got %d", cmd.Subtype)
	}
	c := cmd.Def.(*Constraint)
	if c.Conname != "uq_col" {
		t.Errorf("expected uq_col, got %s", c.Conname)
	}
}

func TestAlterTableDropConstraint(t *testing.T) {
	s := parseOne(t, "ALTER TABLE t DROP CONSTRAINT my_constraint")
	alt := s.(*AlterTableStmt)
	cmd := alt.Cmds[0]
	if cmd.Subtype != AT_DropConstraint {
		t.Errorf("expected AT_DropConstraint, got %d", cmd.Subtype)
	}
	if cmd.Name != "my_constraint" {
		t.Errorf("expected my_constraint, got %s", cmd.Name)
	}
}

func TestAlterTableRenameTo(t *testing.T) {
	s := parseOne(t, "ALTER TABLE old_name RENAME TO new_name")
	alt := s.(*AlterTableStmt)
	cmd := alt.Cmds[0]
	if cmd.Subtype != AT_RenameTable {
		t.Errorf("expected AT_RenameTable, got %d", cmd.Subtype)
	}
	if cmd.Name != "new_name" {
		t.Errorf("expected new_name, got %s", cmd.Name)
	}
}

func TestAlterTableRenameColumn(t *testing.T) {
	s := parseOne(t, "ALTER TABLE t RENAME COLUMN old_col TO new_col")
	alt := s.(*AlterTableStmt)
	cmd := alt.Cmds[0]
	if cmd.Subtype != AT_RenameColumn {
		t.Errorf("expected AT_RenameColumn, got %d", cmd.Subtype)
	}
	if cmd.Name != "old_col" {
		t.Errorf("expected old_col, got %s", cmd.Name)
	}
}

func TestAlterTableIfExists(t *testing.T) {
	s := parseOne(t, "ALTER TABLE IF EXISTS t ADD COLUMN col integer")
	alt := s.(*AlterTableStmt)
	if !alt.MissingOk {
		t.Error("expected MissingOk=true")
	}
}

func TestAlterTableMultipleCmds(t *testing.T) {
	s := parseOne(t, "ALTER TABLE t ADD COLUMN a integer, ADD COLUMN b text")
	alt := s.(*AlterTableStmt)
	if len(alt.Cmds) != 2 {
		t.Fatalf("expected 2 cmds, got %d", len(alt.Cmds))
	}
}

func TestDropTable(t *testing.T) {
	s := parseOne(t, "DROP TABLE t")
	ds, ok := s.(*DropStmt)
	if !ok {
		t.Fatalf("expected DropStmt, got %T", s)
	}
	if ds.RemoveType != OBJECT_TABLE {
		t.Errorf("expected OBJECT_TABLE, got %d", ds.RemoveType)
	}
	if len(ds.Objects) != 1 || ds.Objects[0][0] != "t" {
		t.Errorf("expected [t], got %v", ds.Objects)
	}
}

func TestDropTableIfExists(t *testing.T) {
	s := parseOne(t, "DROP TABLE IF EXISTS t CASCADE")
	ds := s.(*DropStmt)
	if !ds.MissingOk {
		t.Error("expected MissingOk=true")
	}
	if ds.Behavior != DROP_CASCADE {
		t.Error("expected CASCADE")
	}
}

func TestDropMultipleTables(t *testing.T) {
	s := parseOne(t, "DROP TABLE t1, t2, t3")
	ds := s.(*DropStmt)
	if len(ds.Objects) != 3 {
		t.Fatalf("expected 3 objects, got %d", len(ds.Objects))
	}
}

func TestDropIndex(t *testing.T) {
	s := parseOne(t, "DROP INDEX idx_name")
	ds := s.(*DropStmt)
	if ds.RemoveType != OBJECT_INDEX {
		t.Errorf("expected OBJECT_INDEX, got %d", ds.RemoveType)
	}
}

func TestDropIndexConcurrently(t *testing.T) {
	s := parseOne(t, "DROP INDEX CONCURRENTLY idx_name")
	ds := s.(*DropStmt)
	if !ds.Concurrent {
		t.Error("expected Concurrent=true")
	}
}

func TestDropView(t *testing.T) {
	s := parseOne(t, "DROP VIEW IF EXISTS my_view")
	ds := s.(*DropStmt)
	if ds.RemoveType != OBJECT_VIEW {
		t.Errorf("expected OBJECT_VIEW, got %d", ds.RemoveType)
	}
}

func TestDropSchema(t *testing.T) {
	s := parseOne(t, "DROP SCHEMA myschema CASCADE")
	ds := s.(*DropStmt)
	if ds.RemoveType != OBJECT_SCHEMA {
		t.Errorf("expected OBJECT_SCHEMA, got %d", ds.RemoveType)
	}
	if ds.Behavior != DROP_CASCADE {
		t.Error("expected CASCADE")
	}
}

func TestTruncateBasic(t *testing.T) {
	s := parseOne(t, "TRUNCATE t")
	ts, ok := s.(*TruncateStmt)
	if !ok {
		t.Fatalf("expected TruncateStmt, got %T", s)
	}
	if len(ts.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(ts.Relations))
	}
	if ts.Relations[0].Relname != "t" {
		t.Errorf("expected t, got %s", ts.Relations[0].Relname)
	}
}

func TestTruncateTable(t *testing.T) {
	parseOK(t, "TRUNCATE TABLE t")
}

func TestTruncateMultiple(t *testing.T) {
	s := parseOne(t, "TRUNCATE t1, t2, t3 RESTART IDENTITY CASCADE")
	ts := s.(*TruncateStmt)
	if len(ts.Relations) != 3 {
		t.Fatalf("expected 3 relations, got %d", len(ts.Relations))
	}
	if !ts.RestartSeqs {
		t.Error("expected RestartSeqs=true")
	}
	if ts.Behavior != DROP_CASCADE {
		t.Error("expected CASCADE")
	}
}

func TestTruncateContinueIdentity(t *testing.T) {
	s := parseOne(t, "TRUNCATE t CONTINUE IDENTITY")
	ts := s.(*TruncateStmt)
	if ts.RestartSeqs {
		t.Error("expected RestartSeqs=false")
	}
}
