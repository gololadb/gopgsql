package tests

import (
	"testing"

	"github.com/gololadb/gopgsql/parser"
)

func TestCreateExtension(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE EXTENSION hstore", "basic"},
		{"CREATE EXTENSION IF NOT EXISTS hstore", "if not exists"},
		{"CREATE EXTENSION hstore SCHEMA public", "with schema"},
		{"CREATE EXTENSION hstore VERSION '1.4'", "with version"},
		{"CREATE EXTENSION hstore CASCADE", "cascade"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			e, ok := stmt.(*parser.CreateExtensionStmt)
			if !ok {
				t.Fatalf("expected *parser.CreateExtensionStmt, got %T", stmt)
			}
			if e.Extname == "" {
				t.Fatal("expected non-empty Extname")
			}
		})
	}
}


func TestAlterExtension(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER EXTENSION hstore UPDATE", "update"},
		{"ALTER EXTENSION hstore UPDATE TO '2.0'", "update to version"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			e, ok := stmt.(*parser.AlterExtensionStmt)
			if !ok {
				t.Fatalf("expected *parser.AlterExtensionStmt, got %T", stmt)
			}
			if e.Extname == "" {
				t.Fatal("expected non-empty Extname")
			}
		})
	}
}


func TestCreatePolicy(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE POLICY mypol ON mytable", "basic"},
		{"CREATE POLICY mypol ON mytable FOR SELECT", "for select"},
		{"CREATE POLICY mypol ON mytable FOR ALL TO PUBLIC", "for all to public"},
		{"CREATE POLICY mypol ON mytable USING (user_id = current_user)", "using"},
		{"CREATE POLICY mypol ON mytable FOR INSERT WITH CHECK (user_id = current_user)", "with check"},
		{"CREATE POLICY mypol ON mytable AS RESTRICTIVE FOR SELECT TO myrole USING (true)", "restrictive"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			p, ok := stmt.(*parser.CreatePolicyStmt)
			if !ok {
				t.Fatalf("expected *parser.CreatePolicyStmt, got %T", stmt)
			}
			if p.PolicyName == "" {
				t.Fatal("expected non-empty PolicyName")
			}
		})
	}
}


func TestCreatePublication(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE PUBLICATION mypub FOR ALL TABLES", "all tables"},
		{"CREATE PUBLICATION mypub FOR TABLE t1, t2", "specific tables"},
		{"CREATE PUBLICATION mypub FOR TABLE t1 WITH (publish = 'insert, update')", "with options"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			p, ok := stmt.(*parser.CreatePublicationStmt)
			if !ok {
				t.Fatalf("expected *parser.CreatePublicationStmt, got %T", stmt)
			}
			if p.Pubname == "" {
				t.Fatal("expected non-empty Pubname")
			}
		})
	}
}


func TestCreateSubscription(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE SUBSCRIPTION mysub CONNECTION 'host=localhost' PUBLICATION mypub", "basic"},
		{"CREATE SUBSCRIPTION mysub CONNECTION 'host=localhost' PUBLICATION pub1, pub2", "multi pub"},
		{"CREATE SUBSCRIPTION mysub CONNECTION 'host=localhost' PUBLICATION mypub WITH (enabled = false)", "with options"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			s, ok := stmt.(*parser.CreateSubscriptionStmt)
			if !ok {
				t.Fatalf("expected *parser.CreateSubscriptionStmt, got %T", stmt)
			}
			if s.Subname == "" {
				t.Fatal("expected non-empty Subname")
			}
		})
	}
}


func TestCreateEventTrigger(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE EVENT TRIGGER mytrig ON ddl_command_start EXECUTE FUNCTION myfunc()", "basic"},
		{"CREATE EVENT TRIGGER mytrig ON ddl_command_end WHEN TAG IN ('CREATE TABLE', 'DROP TABLE') EXECUTE FUNCTION myfunc()", "with filter"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			e, ok := stmt.(*parser.CreateEventTrigStmt)
			if !ok {
				t.Fatalf("expected *parser.CreateEventTrigStmt, got %T", stmt)
			}
			if e.Trigname == "" {
				t.Fatal("expected non-empty Trigname")
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
			s, ok := stmt.(*parser.CreateSchemaStmt)
			if !ok {
				t.Fatalf("expected *parser.CreateSchemaStmt, got %T", stmt)
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
			d, ok := stmt.(*parser.CreateDomainStmt)
			if !ok {
				t.Fatalf("expected *parser.CreateDomainStmt, got %T", stmt)
			}
			if d.TypeName == nil {
				t.Fatal("expected non-nil parser.TypeName")
			}
		})
	}
}

