package pgscan

import "testing"

// ---------------------------------------------------------------------------
// Step 8: CREATE TABLE / CREATE TABLE AS
// ---------------------------------------------------------------------------

func TestCreateTableBasic(t *testing.T) {
	sql := `CREATE TABLE users (
		id integer PRIMARY KEY,
		name varchar(100) NOT NULL,
		email text UNIQUE,
		age integer CHECK (age > 0),
		created_at timestamp DEFAULT now()
	)`
	s := parseOne(t, sql)
	cs, ok := s.(*CreateStmt)
	if !ok {
		t.Fatalf("expected CreateStmt, got %T", s)
	}
	if cs.Relation.Relname != "users" {
		t.Errorf("expected users, got %s", cs.Relation.Relname)
	}
	if len(cs.TableElts) != 5 {
		t.Fatalf("expected 5 elements, got %d", len(cs.TableElts))
	}

	// Check first column: id integer PRIMARY KEY
	col0 := cs.TableElts[0].(*ColumnDef)
	if col0.Colname != "id" {
		t.Errorf("expected id, got %s", col0.Colname)
	}
	hasPK := false
	for _, c := range col0.Constraints {
		if c.Contype == CONSTR_PRIMARY {
			hasPK = true
		}
	}
	if !hasPK {
		t.Error("expected PRIMARY KEY constraint on id")
	}

	// Check second column: name varchar(100) NOT NULL
	col1 := cs.TableElts[1].(*ColumnDef)
	if col1.Colname != "name" {
		t.Errorf("expected name, got %s", col1.Colname)
	}
	hasNotNull := false
	for _, c := range col1.Constraints {
		if c.Contype == CONSTR_NOTNULL {
			hasNotNull = true
		}
	}
	if !hasNotNull {
		t.Error("expected NOT NULL constraint on name")
	}

	// Check fifth column: created_at timestamp DEFAULT now()
	col4 := cs.TableElts[4].(*ColumnDef)
	hasDefault := false
	for _, c := range col4.Constraints {
		if c.Contype == CONSTR_DEFAULT {
			hasDefault = true
		}
	}
	if !hasDefault {
		t.Error("expected DEFAULT constraint on created_at")
	}
}

func TestCreateTableIfNotExists(t *testing.T) {
	sql := `CREATE TABLE IF NOT EXISTS t (id integer)`
	s := parseOne(t, sql)
	cs := s.(*CreateStmt)
	if !cs.IfNotExists {
		t.Error("expected IfNotExists=true")
	}
}

func TestCreateTempTable(t *testing.T) {
	sql := `CREATE TEMP TABLE t (id integer)`
	s := parseOne(t, sql)
	cs := s.(*CreateStmt)
	if cs.Persistence != RELPERSISTENCE_TEMP {
		t.Errorf("expected TEMP, got %d", cs.Persistence)
	}
}

func TestCreateUnloggedTable(t *testing.T) {
	sql := `CREATE UNLOGGED TABLE t (id integer)`
	s := parseOne(t, sql)
	cs := s.(*CreateStmt)
	if cs.Persistence != RELPERSISTENCE_UNLOGGED {
		t.Errorf("expected UNLOGGED, got %d", cs.Persistence)
	}
}

func TestCreateTableTableConstraints(t *testing.T) {
	sql := `CREATE TABLE orders (
		id integer,
		user_id integer,
		product_id integer,
		PRIMARY KEY (id),
		UNIQUE (user_id, product_id),
		CHECK (id > 0),
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	)`
	s := parseOne(t, sql)
	cs := s.(*CreateStmt)
	// 3 columns + 4 constraints = 7 elements
	if len(cs.TableElts) != 7 {
		t.Fatalf("expected 7 elements, got %d", len(cs.TableElts))
	}

	// Check PRIMARY KEY constraint
	pk := cs.TableElts[3].(*Constraint)
	if pk.Contype != CONSTR_PRIMARY {
		t.Errorf("expected PRIMARY KEY, got %d", pk.Contype)
	}
	if len(pk.Keys) != 1 || pk.Keys[0] != "id" {
		t.Errorf("expected keys [id], got %v", pk.Keys)
	}

	// Check UNIQUE constraint
	uq := cs.TableElts[4].(*Constraint)
	if uq.Contype != CONSTR_UNIQUE {
		t.Errorf("expected UNIQUE, got %d", uq.Contype)
	}
	if len(uq.Keys) != 2 {
		t.Errorf("expected 2 unique keys, got %d", len(uq.Keys))
	}

	// Check FOREIGN KEY
	fk := cs.TableElts[6].(*Constraint)
	if fk.Contype != CONSTR_FOREIGN {
		t.Errorf("expected FOREIGN KEY, got %d", fk.Contype)
	}
	if fk.PkTable.Relname != "users" {
		t.Errorf("expected references users, got %s", fk.PkTable.Relname)
	}
	if fk.FkDelAction != "CASCADE" {
		t.Errorf("expected ON DELETE CASCADE, got %s", fk.FkDelAction)
	}
}

func TestCreateTableNamedConstraint(t *testing.T) {
	sql := `CREATE TABLE t (
		id integer,
		CONSTRAINT pk_t PRIMARY KEY (id)
	)`
	s := parseOne(t, sql)
	cs := s.(*CreateStmt)
	c := cs.TableElts[1].(*Constraint)
	if c.Conname != "pk_t" {
		t.Errorf("expected constraint name pk_t, got %s", c.Conname)
	}
}

func TestCreateTableInherits(t *testing.T) {
	sql := `CREATE TABLE child (extra text) INHERITS (parent)`
	s := parseOne(t, sql)
	cs := s.(*CreateStmt)
	if len(cs.InhRelations) != 1 {
		t.Fatalf("expected 1 parent, got %d", len(cs.InhRelations))
	}
	rv := cs.InhRelations[0].(*RangeVar)
	if rv.Relname != "parent" {
		t.Errorf("expected parent, got %s", rv.Relname)
	}
}

func TestCreateTableOnCommitDrop(t *testing.T) {
	sql := `CREATE TEMP TABLE t (id integer) ON COMMIT DROP`
	s := parseOne(t, sql)
	cs := s.(*CreateStmt)
	if cs.OnCommit != ONCOMMIT_DROP {
		t.Errorf("expected ONCOMMIT_DROP, got %d", cs.OnCommit)
	}
}

func TestCreateTableReferences(t *testing.T) {
	sql := `CREATE TABLE t (
		id integer,
		parent_id integer REFERENCES parent(id) ON DELETE SET NULL ON UPDATE CASCADE
	)`
	s := parseOne(t, sql)
	cs := s.(*CreateStmt)
	col := cs.TableElts[1].(*ColumnDef)
	var fk *Constraint
	for _, c := range col.Constraints {
		if c.Contype == CONSTR_FOREIGN {
			fk = c
		}
	}
	if fk == nil {
		t.Fatal("expected REFERENCES constraint")
	}
	if fk.PkTable.Relname != "parent" {
		t.Errorf("expected parent, got %s", fk.PkTable.Relname)
	}
	if fk.FkDelAction != "SET NULL" {
		t.Errorf("expected SET NULL, got %s", fk.FkDelAction)
	}
	if fk.FkUpdAction != "CASCADE" {
		t.Errorf("expected CASCADE, got %s", fk.FkUpdAction)
	}
}

func TestCreateTableGenerated(t *testing.T) {
	sql := `CREATE TABLE t (
		a integer,
		b integer,
		c integer GENERATED ALWAYS AS (a + b) STORED
	)`
	s := parseOne(t, sql)
	cs := s.(*CreateStmt)
	col := cs.TableElts[2].(*ColumnDef)
	var gen *Constraint
	for _, c := range col.Constraints {
		if c.Contype == CONSTR_GENERATED {
			gen = c
		}
	}
	if gen == nil {
		t.Fatal("expected GENERATED constraint")
	}
	if gen.RawExpr == nil {
		t.Error("expected expression in GENERATED constraint")
	}
}

func TestCreateTableAs(t *testing.T) {
	sql := `CREATE TABLE summary AS SELECT dept, count(*) AS cnt FROM employees GROUP BY dept`
	s := parseOne(t, sql)
	ctas, ok := s.(*CreateTableAsStmt)
	if !ok {
		t.Fatalf("expected CreateTableAsStmt, got %T", s)
	}
	if ctas.Into.Rel.Relname != "summary" {
		t.Errorf("expected summary, got %s", ctas.Into.Rel.Relname)
	}
	if !ctas.WithData {
		t.Error("expected WithData=true by default")
	}
}

func TestCreateTableAsWithNoData(t *testing.T) {
	sql := `CREATE TABLE t AS SELECT 1 WITH NO DATA`
	s := parseOne(t, sql)
	ctas := s.(*CreateTableAsStmt)
	if ctas.WithData {
		t.Error("expected WithData=false")
	}
}

func TestCreateTableLike(t *testing.T) {
	sql := `CREATE TABLE new_t (LIKE old_t, extra integer)`
	s := parseOne(t, sql)
	cs := s.(*CreateStmt)
	if len(cs.TableElts) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(cs.TableElts))
	}
	like, ok := cs.TableElts[0].(*TableLikeClause)
	if !ok {
		t.Fatalf("expected TableLikeClause, got %T", cs.TableElts[0])
	}
	if like.Relation.Relname != "old_t" {
		t.Errorf("expected old_t, got %s", like.Relation.Relname)
	}
}

func TestCreateTableQualifiedName(t *testing.T) {
	sql := `CREATE TABLE myschema.mytable (id integer)`
	s := parseOne(t, sql)
	cs := s.(*CreateStmt)
	if cs.Relation.Schemaname != "myschema" {
		t.Errorf("expected schema myschema, got %s", cs.Relation.Schemaname)
	}
	if cs.Relation.Relname != "mytable" {
		t.Errorf("expected table mytable, got %s", cs.Relation.Relname)
	}
}
