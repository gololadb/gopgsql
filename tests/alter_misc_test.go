package tests

import (
	"testing"

	"github.com/gololadb/gopgsql/parser"
)

func TestAlterRole(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER ROLE myrole SUPERUSER", "superuser"},
		{"ALTER ROLE myrole NOSUPERUSER CREATEDB", "multiple options"},
		{"ALTER ROLE myrole PASSWORD 'secret'", "password"},
		{"ALTER ROLE myrole CONNECTION LIMIT 10", "connection limit"},
		{"ALTER USER myuser LOGIN", "alter user"},
		{"ALTER ROLE myrole WITH NOLOGIN", "with keyword"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ar, ok := stmt.(*parser.AlterRoleStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterRoleStmt, got %T", stmt)
			}
			if ar.RoleName == "" {
				t.Fatal("expected non-empty RoleName")
			}
		})
	}
}


func TestAlterRoleSet(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER ROLE myrole SET search_path TO public", "set"},
		{"ALTER ROLE myrole RESET search_path", "reset"},
		{"ALTER ROLE myrole RESET ALL", "reset all"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ars, ok := stmt.(*parser.AlterRoleSetStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterRoleSetStmt, got %T", stmt)
			}
			if ars.RoleName != "myrole" {
				t.Fatalf("expected 'myrole', got %q", ars.RoleName)
			}
			if ars.SetStmt == nil {
				t.Fatal("expected non-nil SetStmt")
			}
		})
	}
}


func TestAlterDomainStmt(t *testing.T) {
	tests := []struct {
		sql     string
		desc    string
		subtype byte
	}{
		{"ALTER DOMAIN mydom SET DEFAULT 0", "set default", 'T'},
		{"ALTER DOMAIN mydom DROP DEFAULT", "drop default", 'N'},
		{"ALTER DOMAIN mydom SET NOT NULL", "set not null", 'O'},
		{"ALTER DOMAIN mydom DROP NOT NULL", "drop not null", 'N'},
		{"ALTER DOMAIN mydom ADD CONSTRAINT chk CHECK (VALUE > 0)", "add constraint", 'C'},
		{"ALTER DOMAIN mydom DROP CONSTRAINT chk", "drop constraint", 'X'},
		{"ALTER DOMAIN mydom DROP CONSTRAINT IF EXISTS chk", "drop constraint if exists", 'X'},
		{"ALTER DOMAIN mydom VALIDATE CONSTRAINT chk", "validate constraint", 'V'},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ad, ok := stmt.(*parser.AlterDomainStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterDomainStmt, got %T", stmt)
			}
			if ad.Subtype != tt.subtype {
				t.Fatalf("expected subtype '%c', got '%c'", tt.subtype, ad.Subtype)
			}
		})
	}
}


func TestAlterDomainOwner(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{`ALTER DOMAIN public."bigint" OWNER TO postgres`, "owner to"},
		{"ALTER DOMAIN mydom RENAME TO newdom", "rename to"},
		{"ALTER DOMAIN mydom SET SCHEMA newschema", "set schema"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			switch tt.desc {
			case "owner to":
				ao, ok := stmt.(*parser.AlterOwnerStmt)
				if !ok {
					t.Fatalf("expected *parser.AlterOwnerStmt, got %T", stmt)
				}
				if ao.NewOwner != "postgres" {
					t.Fatalf("expected 'postgres', got %q", ao.NewOwner)
				}
			case "rename to":
				rs, ok := stmt.(*parser.RenameStmt)
				if !ok {
					t.Fatalf("expected *parser.RenameStmt, got %T", stmt)
				}
				if rs.Newname != "newdom" {
					t.Fatalf("expected 'newdom', got %q", rs.Newname)
				}
			case "set schema":
				rs, ok := stmt.(*parser.RenameStmt)
				if !ok {
					t.Fatalf("expected *parser.RenameStmt, got %T", stmt)
				}
				if rs.Newname != "newschema" {
					t.Fatalf("expected 'newschema', got %q", rs.Newname)
				}
			}
		})
	}
}

func TestAlterSchema(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER SCHEMA public OWNER TO postgres", "owner to"},
		{"ALTER SCHEMA myschema RENAME TO newschema", "rename to"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			switch tt.desc {
			case "owner to":
				ao, ok := stmt.(*parser.AlterOwnerStmt)
				if !ok {
					t.Fatalf("expected *parser.AlterOwnerStmt, got %T", stmt)
				}
				if ao.NewOwner != "postgres" {
					t.Fatalf("expected 'postgres', got %q", ao.NewOwner)
				}
			case "rename to":
				rs, ok := stmt.(*parser.RenameStmt)
				if !ok {
					t.Fatalf("expected *parser.RenameStmt, got %T", stmt)
				}
				if rs.Newname != "newschema" {
					t.Fatalf("expected 'newschema', got %q", rs.Newname)
				}
			}
		})
	}
}

func TestAlterAggregate(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER AGGREGATE public.group_concat(text) OWNER TO postgres", "owner to"},
		{"ALTER AGGREGATE myagg(integer) RENAME TO newagg", "rename to"},
		{"ALTER AGGREGATE myagg(integer) SET SCHEMA newschema", "set schema"},
		{"ALTER AGGREGATE myagg(*) OWNER TO alice", "star arg owner"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			switch s := stmt.(type) {
			case *parser.AlterOwnerStmt:
				if s.NewOwner == "" {
					t.Fatal("expected non-empty NewOwner")
				}
			case *parser.RenameStmt:
				if s.Newname == "" {
					t.Fatal("expected non-empty Newname")
				}
			default:
				t.Fatalf("unexpected type %T", stmt)
			}
		})
	}
}

func TestAlterTypeOwner(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER TYPE public.mpaa_rating OWNER TO postgres", "owner to"},
		{"ALTER TYPE mytype SET SCHEMA newschema", "set schema"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			switch tt.desc {
			case "owner to":
				ao, ok := stmt.(*parser.AlterOwnerStmt)
				if !ok {
					t.Fatalf("expected *parser.AlterOwnerStmt, got %T", stmt)
				}
				if ao.NewOwner != "postgres" {
					t.Fatalf("expected 'postgres', got %q", ao.NewOwner)
				}
			case "set schema":
				rs, ok := stmt.(*parser.RenameStmt)
				if !ok {
					t.Fatalf("expected *parser.RenameStmt, got %T", stmt)
				}
				if rs.Newname != "newschema" {
					t.Fatalf("expected 'newschema', got %q", rs.Newname)
				}
			}
		})
	}
}

func TestAlterFunctionNamedParams(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER FUNCTION public.film_in_stock(p_film_id integer, p_store_id integer) OWNER TO postgres", "named params"},
		{"ALTER FUNCTION public.inventory_held_by_customer(p_inventory_id integer) OWNER TO postgres", "single named param"},
		{"ALTER FUNCTION myfunc(IN p_id integer, OUT result text) OWNER TO alice", "with modes"},
		{"ALTER FUNCTION myfunc(integer, text) OWNER TO alice", "bare types"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			af, ok := stmt.(*parser.AlterFunctionStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterFunctionStmt, got %T", stmt)
			}
			if af.Func == nil {
				t.Fatal("expected non-nil Func")
			}
		})
	}
}

func TestAlterEnumType(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER TYPE mood ADD VALUE 'excited'", "add value"},
		{"ALTER TYPE mood ADD VALUE IF NOT EXISTS 'excited'", "add value if not exists"},
		{"ALTER TYPE mood ADD VALUE 'excited' BEFORE 'happy'", "add value before"},
		{"ALTER TYPE mood ADD VALUE 'excited' AFTER 'sad'", "add value after"},
		{"ALTER TYPE mood RENAME VALUE 'sad' TO 'melancholy'", "rename value"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ae, ok := stmt.(*parser.AlterEnumStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterEnumStmt, got %T", stmt)
			}
			if ae.NewVal == "" {
				t.Fatal("expected non-empty NewVal")
			}
		})
	}
}


func TestAlterFunction(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER FUNCTION myfunc(integer) RENAME TO newfunc", "rename"},
		{"ALTER FUNCTION myfunc() OWNER TO alice", "owner"},
		{"ALTER FUNCTION myfunc() SET SCHEMA myschema", "set schema"},
		{"ALTER PROCEDURE myproc() SECURITY DEFINER", "procedure security"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			af, ok := stmt.(*parser.AlterFunctionStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterFunctionStmt, got %T", stmt)
			}
			if af.Func == nil {
				t.Fatal("expected non-nil Func")
			}
		})
	}
}


func TestAlterPolicyStmt(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER POLICY mypol ON mytable TO alice", "to role"},
		{"ALTER POLICY mypol ON mytable USING (true)", "using"},
		{"ALTER POLICY mypol ON mytable WITH CHECK (user_id = 1)", "with check"},
		{"ALTER POLICY mypol ON mytable TO alice USING (true) WITH CHECK (true)", "all clauses"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ap, ok := stmt.(*parser.AlterPolicyStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterPolicyStmt, got %T", stmt)
			}
			if ap.PolicyName != "mypol" {
				t.Fatalf("expected 'mypol', got %q", ap.PolicyName)
			}
		})
	}
}


func TestAlterPublication(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER PUBLICATION mypub ADD TABLE t1", "add table"},
		{"ALTER PUBLICATION mypub DROP TABLE t1, t2", "drop tables"},
		{"ALTER PUBLICATION mypub SET TABLE t1", "set table"},
		{"ALTER PUBLICATION mypub SET (publish = 'insert')", "set options"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ap, ok := stmt.(*parser.AlterPublicationStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterPublicationStmt, got %T", stmt)
			}
			if ap.Pubname != "mypub" {
				t.Fatalf("expected 'mypub', got %q", ap.Pubname)
			}
		})
	}
}


func TestAlterSubscription(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER SUBSCRIPTION mysub CONNECTION 'host=newhost'", "connection"},
		{"ALTER SUBSCRIPTION mysub SET PUBLICATION pub1, pub2", "set publication"},
		{"ALTER SUBSCRIPTION mysub SET (slot_name = 'myslot')", "set options"},
		{"ALTER SUBSCRIPTION mysub ENABLE", "enable"},
		{"ALTER SUBSCRIPTION mysub DISABLE", "disable"},
		{"ALTER SUBSCRIPTION mysub REFRESH PUBLICATION", "refresh"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			as, ok := stmt.(*parser.AlterSubscriptionStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterSubscriptionStmt, got %T", stmt)
			}
			if as.Subname != "mysub" {
				t.Fatalf("expected 'mysub', got %q", as.Subname)
			}
		})
	}
}


func TestAlterEventTrigger(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
		flag byte
	}{
		{"ALTER EVENT TRIGGER mytrig ENABLE", "enable", 'O'},
		{"ALTER EVENT TRIGGER mytrig DISABLE", "disable", 'D'},
		{"ALTER EVENT TRIGGER mytrig ENABLE REPLICA", "enable replica", 'R'},
		{"ALTER EVENT TRIGGER mytrig ENABLE ALWAYS", "enable always", 'A'},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ae, ok := stmt.(*parser.AlterEventTrigStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterEventTrigStmt, got %T", stmt)
			}
			if ae.Tgenabled != tt.flag {
				t.Fatalf("expected flag '%c', got '%c'", tt.flag, ae.Tgenabled)
			}
		})
	}
}


func TestAlterSystem(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER SYSTEM SET work_mem TO '256MB'", "set"},
		{"ALTER SYSTEM RESET work_mem", "reset"},
		{"ALTER SYSTEM RESET ALL", "reset all"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			as, ok := stmt.(*parser.AlterSystemStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterSystemStmt, got %T", stmt)
			}
			if as.SetStmt == nil {
				t.Fatal("expected non-nil SetStmt")
			}
		})
	}
}

