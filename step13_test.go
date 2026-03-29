package pgscan

import "testing"

func TestGrantPrivileges(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"GRANT SELECT ON TABLE t TO alice", "grant select on table"},
		{"GRANT SELECT, INSERT ON t TO alice, bob", "grant multiple privs to multiple roles"},
		{"GRANT ALL PRIVILEGES ON t TO alice", "grant all privileges"},
		{"GRANT ALL ON t TO PUBLIC", "grant all to public"},
		{"GRANT SELECT ON t TO alice WITH GRANT OPTION", "with grant option"},
		{"GRANT UPDATE ON t TO alice", "grant update"},
		{"GRANT USAGE ON SEQUENCE s1 TO alice", "grant on sequence"},
		{"GRANT EXECUTE ON FUNCTION f1 TO alice", "grant on function"},
		{"GRANT CREATE ON SCHEMA myschema TO alice", "grant on schema"},
		{"GRANT CONNECT ON DATABASE mydb TO alice", "grant on database"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			g, ok := stmt.(*GrantStmt)
			if !ok {
				t.Fatalf("expected *GrantStmt, got %T", stmt)
			}
			if !g.IsGrant {
				t.Fatal("expected IsGrant=true")
			}
		})
	}
}

func TestRevokePrivileges(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"REVOKE SELECT ON t FROM alice", "revoke select"},
		{"REVOKE ALL ON t FROM alice CASCADE", "revoke all cascade"},
		{"REVOKE ALL ON t FROM alice RESTRICT", "revoke all restrict"},
		{"REVOKE GRANT OPTION FOR SELECT ON t FROM alice", "revoke grant option for"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			g, ok := stmt.(*GrantStmt)
			if !ok {
				t.Fatalf("expected *GrantStmt, got %T", stmt)
			}
			if g.IsGrant {
				t.Fatal("expected IsGrant=false")
			}
		})
	}
}

func TestGrantRole(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"GRANT admin TO alice", "grant role"},
		{"GRANT admin, editor TO alice, bob", "grant multiple roles"},
		{"GRANT admin TO alice WITH ADMIN OPTION", "with admin option"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			g, ok := stmt.(*GrantRoleStmt)
			if !ok {
				t.Fatalf("expected *GrantRoleStmt, got %T", stmt)
			}
			if !g.IsGrant {
				t.Fatal("expected IsGrant=true")
			}
		})
	}
}

func TestRevokeRole(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"REVOKE admin FROM alice", "revoke role"},
		{"REVOKE ADMIN OPTION FOR admin FROM alice", "revoke admin option for"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			g, ok := stmt.(*GrantRoleStmt)
			if !ok {
				t.Fatalf("expected *GrantRoleStmt, got %T", stmt)
			}
			if g.IsGrant {
				t.Fatal("expected IsGrant=false")
			}
		})
	}
}

func TestCreateRole(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE ROLE myrole", "basic role"},
		{"CREATE USER myuser", "create user"},
		{"CREATE GROUP mygroup", "create group"},
		{"CREATE ROLE myrole SUPERUSER", "superuser"},
		{"CREATE ROLE myrole NOSUPERUSER CREATEDB LOGIN", "multiple options"},
		{"CREATE ROLE myrole PASSWORD 'secret'", "with password"},
		{"CREATE ROLE myrole VALID UNTIL '2025-12-31'", "valid until"},
		{"CREATE ROLE myrole CONNECTION LIMIT 10", "connection limit"},
		{"CREATE ROLE myrole IN ROLE admin", "in role"},
		{"CREATE ROLE myrole LOGIN REPLICATION BYPASSRLS", "login replication bypassrls"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			r, ok := stmt.(*CreateRoleStmt)
			if !ok {
				t.Fatalf("expected *CreateRoleStmt, got %T", stmt)
			}
			if r.RoleName == "" {
				t.Fatal("expected non-empty RoleName")
			}
		})
	}
}

func TestCreateSchema(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE SCHEMA myschema", "basic schema"},
		{"CREATE SCHEMA IF NOT EXISTS myschema", "if not exists"},
		{"CREATE SCHEMA myschema AUTHORIZATION admin", "with authorization"},
		{"CREATE SCHEMA AUTHORIZATION admin", "authorization only"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			s, ok := stmt.(*CreateSchemaStmt)
			if !ok {
				t.Fatalf("expected *CreateSchemaStmt, got %T", stmt)
			}
			_ = s
		})
	}
}

func TestCreateDomain(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE DOMAIN posint AS integer CHECK (VALUE > 0)", "domain with check"},
		{"CREATE DOMAIN email AS text NOT NULL", "domain not null"},
		{"CREATE DOMAIN mydom AS varchar(100) DEFAULT ''", "domain with default"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			d, ok := stmt.(*CreateDomainStmt)
			if !ok {
				t.Fatalf("expected *CreateDomainStmt, got %T", stmt)
			}
			if d.TypeName == nil {
				t.Fatal("expected non-nil TypeName")
			}
		})
	}
}

func TestCreateEnum(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE TYPE mood AS ENUM ('happy', 'sad', 'angry')", "basic enum"},
		{"CREATE TYPE status AS ENUM ('active', 'inactive')", "two-value enum"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			e, ok := stmt.(*CreateEnumStmt)
			if !ok {
				t.Fatalf("expected *CreateEnumStmt, got %T", stmt)
			}
			if len(e.Vals) == 0 {
				t.Fatal("expected non-empty Vals")
			}
		})
	}
}

func TestCreateCompositeType(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE TYPE address AS (street text, city text, zip integer)", "composite type"},
		{"CREATE TYPE pair AS (first integer, second integer)", "two-column composite"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			c, ok := stmt.(*CompositeTypeStmt)
			if !ok {
				t.Fatalf("expected *CompositeTypeStmt, got %T", stmt)
			}
			if len(c.ColDefs) == 0 {
				t.Fatal("expected non-empty ColDefs")
			}
		})
	}
}
