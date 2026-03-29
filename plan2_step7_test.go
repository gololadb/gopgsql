package pgscan

import "testing"

func TestCreateFdw(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE FOREIGN DATA WRAPPER myfdw", "basic"},
		{"CREATE FOREIGN DATA WRAPPER myfdw HANDLER myhandler", "with handler"},
		{"CREATE FOREIGN DATA WRAPPER myfdw HANDLER myhandler VALIDATOR myvalidator", "handler and validator"},
		{"CREATE FOREIGN DATA WRAPPER myfdw NO HANDLER NO VALIDATOR", "no handler no validator"},
		{"CREATE FOREIGN DATA WRAPPER myfdw OPTIONS (debug 'true')", "with options"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cf, ok := stmt.(*CreateFdwStmt)
			if !ok {
				t.Fatalf("expected *CreateFdwStmt, got %T", stmt)
			}
			if cf.Fdwname != "myfdw" {
				t.Fatalf("expected 'myfdw', got %q", cf.Fdwname)
			}
		})
	}
}

func TestCreateServer(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE SERVER myserver FOREIGN DATA WRAPPER myfdw", "basic"},
		{"CREATE SERVER IF NOT EXISTS myserver FOREIGN DATA WRAPPER myfdw", "if not exists"},
		{"CREATE SERVER myserver TYPE 'oracle' FOREIGN DATA WRAPPER myfdw", "with type"},
		{"CREATE SERVER myserver VERSION '1.0' FOREIGN DATA WRAPPER myfdw", "with version"},
		{"CREATE SERVER myserver FOREIGN DATA WRAPPER myfdw OPTIONS (host 'localhost', port '5432')", "with options"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cs, ok := stmt.(*CreateForeignServerStmt)
			if !ok {
				t.Fatalf("expected *CreateForeignServerStmt, got %T", stmt)
			}
			if cs.Servername != "myserver" {
				t.Fatalf("expected 'myserver', got %q", cs.Servername)
			}
			if cs.Fdwname != "myfdw" {
				t.Fatalf("expected fdw 'myfdw', got %q", cs.Fdwname)
			}
		})
	}
}

func TestCreateForeignTable(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE FOREIGN TABLE ft (id integer, name text) SERVER myserver", "basic"},
		{"CREATE FOREIGN TABLE IF NOT EXISTS ft (id integer) SERVER myserver", "if not exists"},
		{"CREATE FOREIGN TABLE ft (id integer) SERVER myserver OPTIONS (schema_name 'public')", "with options"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cft, ok := stmt.(*CreateForeignTableStmt)
			if !ok {
				t.Fatalf("expected *CreateForeignTableStmt, got %T", stmt)
			}
			if cft.Servername != "myserver" {
				t.Fatalf("expected 'myserver', got %q", cft.Servername)
			}
		})
	}
}

func TestCreateUserMapping(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE USER MAPPING FOR alice SERVER myserver", "basic"},
		{"CREATE USER MAPPING IF NOT EXISTS FOR alice SERVER myserver", "if not exists"},
		{"CREATE USER MAPPING FOR PUBLIC SERVER myserver", "public"},
		{"CREATE USER MAPPING FOR CURRENT_USER SERVER myserver", "current_user"},
		{"CREATE USER MAPPING FOR alice SERVER myserver OPTIONS (user 'remote_alice', password 'secret')", "with options"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cum, ok := stmt.(*CreateUserMappingStmt)
			if !ok {
				t.Fatalf("expected *CreateUserMappingStmt, got %T", stmt)
			}
			if cum.Servername != "myserver" {
				t.Fatalf("expected 'myserver', got %q", cum.Servername)
			}
		})
	}
}

func TestImportForeignSchema(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"IMPORT FOREIGN SCHEMA remote_schema FROM SERVER myserver INTO local_schema", "basic"},
		{"IMPORT FOREIGN SCHEMA remote_schema LIMIT TO (t1, t2) FROM SERVER myserver INTO local_schema", "limit to"},
		{"IMPORT FOREIGN SCHEMA remote_schema EXCEPT (t3) FROM SERVER myserver INTO local_schema", "except"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ifs, ok := stmt.(*ImportForeignSchemaStmt)
			if !ok {
				t.Fatalf("expected *ImportForeignSchemaStmt, got %T", stmt)
			}
			if ifs.ServerName != "myserver" {
				t.Fatalf("expected 'myserver', got %q", ifs.ServerName)
			}
		})
	}
}

func TestCreateUserStillWorks(t *testing.T) {
	// Ensure CREATE USER (role) still works after the USER MAPPING dispatch
	stmt := parseOne(t, "CREATE USER myuser LOGIN PASSWORD 'secret'")
	cr, ok := stmt.(*CreateRoleStmt)
	if !ok {
		t.Fatalf("expected *CreateRoleStmt, got %T", stmt)
	}
	if cr.RoleName != "myuser" {
		t.Fatalf("expected 'myuser', got %q", cr.RoleName)
	}
}
