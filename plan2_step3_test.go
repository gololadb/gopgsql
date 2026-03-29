package pgscan

import "testing"

func TestCreateDatabase(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE DATABASE mydb", "basic"},
		{"CREATE DATABASE mydb OWNER alice", "with owner"},
		{"CREATE DATABASE mydb TEMPLATE template0", "with template"},
		{"CREATE DATABASE mydb ENCODING 'UTF8'", "with encoding"},
		{"CREATE DATABASE mydb OWNER alice TEMPLATE template0 ENCODING 'UTF8'", "multiple options"},
		{"CREATE DATABASE mydb CONNECTION LIMIT 10", "connection limit"},
		{"CREATE DATABASE mydb WITH OWNER = alice TEMPLATE = template0", "with equals"},
		{"CREATE DATABASE mydb TABLESPACE pg_default", "tablespace"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cd, ok := stmt.(*CreatedbStmt)
			if !ok {
				t.Fatalf("expected *CreatedbStmt, got %T", stmt)
			}
			if cd.Dbname != "mydb" {
				t.Fatalf("expected dbname 'mydb', got %q", cd.Dbname)
			}
		})
	}
}

func TestDropDatabase(t *testing.T) {
	tests := []struct {
		sql       string
		desc      string
		missing   bool
	}{
		{"DROP DATABASE mydb", "basic", false},
		{"DROP DATABASE IF EXISTS mydb", "if exists", true},
		{"DROP DATABASE mydb WITH (FORCE)", "with force", false},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			dd, ok := stmt.(*DropdbStmt)
			if !ok {
				t.Fatalf("expected *DropdbStmt, got %T", stmt)
			}
			if dd.Dbname != "mydb" {
				t.Fatalf("expected dbname 'mydb', got %q", dd.Dbname)
			}
			if dd.MissingOk != tt.missing {
				t.Fatalf("expected MissingOk=%v, got %v", tt.missing, dd.MissingOk)
			}
		})
	}
}

func TestAlterDatabase(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER DATABASE mydb OWNER alice", "owner"},
		{"ALTER DATABASE mydb CONNECTION LIMIT 100", "connection limit"},
		{"ALTER DATABASE mydb WITH CONNECTION LIMIT 50", "with options"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ad, ok := stmt.(*AlterDatabaseStmt)
			if !ok {
				t.Fatalf("expected *AlterDatabaseStmt, got %T", stmt)
			}
			if ad.Dbname != "mydb" {
				t.Fatalf("expected dbname 'mydb', got %q", ad.Dbname)
			}
		})
	}
}

func TestAlterDatabaseSet(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER DATABASE mydb SET search_path TO public", "set"},
		{"ALTER DATABASE mydb RESET search_path", "reset"},
		{"ALTER DATABASE mydb RESET ALL", "reset all"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ads, ok := stmt.(*AlterDatabaseSetStmt)
			if !ok {
				t.Fatalf("expected *AlterDatabaseSetStmt, got %T", stmt)
			}
			if ads.Dbname != "mydb" {
				t.Fatalf("expected dbname 'mydb', got %q", ads.Dbname)
			}
			if ads.SetStmt == nil {
				t.Fatal("expected non-nil SetStmt")
			}
		})
	}
}

func TestCreateTablespace(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE TABLESPACE myts LOCATION '/data/ts'", "basic"},
		{"CREATE TABLESPACE myts OWNER alice LOCATION '/data/ts'", "with owner"},
		{"CREATE TABLESPACE myts LOCATION '/data/ts' WITH (seq_page_cost = 1.0)", "with options"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			ct, ok := stmt.(*CreateTableSpaceStmt)
			if !ok {
				t.Fatalf("expected *CreateTableSpaceStmt, got %T", stmt)
			}
			if ct.Tablespacename != "myts" {
				t.Fatalf("expected name 'myts', got %q", ct.Tablespacename)
			}
			if ct.Location == "" {
				t.Fatal("expected non-empty Location")
			}
		})
	}
}
