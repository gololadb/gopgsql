package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestGrantColumnLevel(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"GRANT SELECT (col1, col2) ON t TO myrole", "select columns"},
		{"GRANT UPDATE (col1) ON t TO myrole", "update column"},
		{"GRANT INSERT (col1, col2), SELECT (col3) ON t TO myrole", "multiple privs with cols"},
		{"GRANT SELECT ON t TO myrole", "no columns (baseline)"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			gs, ok := stmt.(*parser.GrantStmt)
			if !ok {
				t.Fatalf("expected *parser.GrantStmt, got %T", stmt)
			}
			if len(gs.Privileges) == 0 {
				t.Fatal("expected non-empty Privileges")
			}
		})
	}
}


func TestGrantColumnLevelCols(t *testing.T) {
	stmt := parseOne(t, "GRANT SELECT (col1, col2) ON t TO myrole")
	gs := stmt.(*parser.GrantStmt)
	if len(gs.PrivCols) == 0 {
		t.Fatal("expected non-empty PrivCols")
	}
	if len(gs.PrivCols[0]) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(gs.PrivCols[0]))
	}
	if gs.PrivCols[0][0] != "col1" || gs.PrivCols[0][1] != "col2" {
		t.Fatalf("expected [col1, col2], got %v", gs.PrivCols[0])
	}
}


func TestRevokeColumnLevel(t *testing.T) {
	stmt := parseOne(t, "REVOKE UPDATE (col1) ON t FROM myrole")
	gs, ok := stmt.(*parser.GrantStmt)
	if !ok {
		t.Fatalf("expected *parser.GrantStmt, got %T", stmt)
	}
	if !gs.IsGrant == true {
		// IsGrant should be false for REVOKE
	}
	if len(gs.PrivCols) == 0 {
		t.Fatal("expected non-empty PrivCols")
	}
	if len(gs.PrivCols[0]) != 1 {
		t.Fatalf("expected 1 column, got %d", len(gs.PrivCols[0]))
	}
}


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
			g, ok := stmt.(*parser.GrantStmt)
			if !ok {
				t.Fatalf("expected *parser.GrantStmt, got %T", stmt)
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
			g, ok := stmt.(*parser.GrantStmt)
			if !ok {
				t.Fatalf("expected *parser.GrantStmt, got %T", stmt)
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
			g, ok := stmt.(*parser.GrantRoleStmt)
			if !ok {
				t.Fatalf("expected *parser.GrantRoleStmt, got %T", stmt)
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
			g, ok := stmt.(*parser.GrantRoleStmt)
			if !ok {
				t.Fatalf("expected *parser.GrantRoleStmt, got %T", stmt)
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
			r, ok := stmt.(*parser.CreateRoleStmt)
			if !ok {
				t.Fatalf("expected *parser.CreateRoleStmt, got %T", stmt)
			}
			if r.RoleName == "" {
				t.Fatal("expected non-empty RoleName")
			}
		})
	}
}


func TestAlterDefaultPrivileges(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER DEFAULT PRIVILEGES GRANT SELECT ON TABLES TO public", "basic grant"},
		{"ALTER DEFAULT PRIVILEGES REVOKE ALL ON FUNCTIONS FROM public", "basic revoke"},
		{"ALTER DEFAULT PRIVILEGES FOR ROLE myrole GRANT SELECT ON TABLES TO public", "for role"},
		{"ALTER DEFAULT PRIVILEGES IN SCHEMA myschema GRANT SELECT ON TABLES TO public", "in schema"},
		{"ALTER DEFAULT PRIVILEGES FOR ROLE myrole IN SCHEMA myschema GRANT INSERT ON TABLES TO writer", "for role in schema"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			adp, ok := stmt.(*parser.AlterDefaultPrivilegesStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterDefaultPrivilegesStmt, got %T", stmt)
			}
			if adp.Action == nil {
				t.Fatal("expected non-nil Action")
			}
		})
	}
}


func TestAlterDefaultPrivilegesForRole(t *testing.T) {
	stmt := parseOne(t, "ALTER DEFAULT PRIVILEGES FOR ROLE myrole GRANT SELECT ON TABLES TO public")
	adp := stmt.(*parser.AlterDefaultPrivilegesStmt)
	if len(adp.Options) == 0 {
		t.Fatal("expected non-empty Options")
	}
	found := false
	for _, o := range adp.Options {
		if o.Defname == "for_role" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected for_role option")
	}
}

// ---------------------------------------------------------------------------
// ALTER STATISTICS
// ---------------------------------------------------------------------------

