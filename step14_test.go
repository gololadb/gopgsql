package pgscan

import "testing"

func TestCreateSequence(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"CREATE SEQUENCE myseq", "basic"},
		{"CREATE SEQUENCE IF NOT EXISTS myseq", "if not exists"},
		{"CREATE SEQUENCE myseq INCREMENT BY 2", "increment by"},
		{"CREATE SEQUENCE myseq START WITH 100", "start with"},
		{"CREATE SEQUENCE myseq MINVALUE 1 MAXVALUE 1000", "min max"},
		{"CREATE SEQUENCE myseq NO MINVALUE NO MAXVALUE", "no min no max"},
		{"CREATE SEQUENCE myseq CACHE 10", "cache"},
		{"CREATE SEQUENCE myseq CYCLE", "cycle"},
		{"CREATE SEQUENCE myseq NO CYCLE", "no cycle"},
		{"CREATE SEQUENCE myseq OWNED BY t.id", "owned by"},
		{"CREATE SEQUENCE myseq OWNED BY NONE", "owned by none"},
		{"CREATE SEQUENCE myseq AS bigint", "as type"},
		{"CREATE SEQUENCE myseq INCREMENT 5 START 10 MINVALUE 1 MAXVALUE 100 CACHE 20 CYCLE", "all options"},
		{"CREATE TEMP SEQUENCE myseq", "temp sequence"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			s, ok := stmt.(*CreateSeqStmt)
			if !ok {
				t.Fatalf("expected *CreateSeqStmt, got %T", stmt)
			}
			if len(s.Name) == 0 {
				t.Fatal("expected non-empty Name")
			}
		})
	}
}

func TestAlterSequence(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"ALTER SEQUENCE myseq INCREMENT BY 5", "increment"},
		{"ALTER SEQUENCE myseq RESTART", "restart"},
		{"ALTER SEQUENCE myseq RESTART WITH 1", "restart with"},
		{"ALTER SEQUENCE IF EXISTS myseq MAXVALUE 999", "if exists"},
		{"ALTER SEQUENCE myseq OWNED BY t.col", "owned by"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			s, ok := stmt.(*AlterSeqStmt)
			if !ok {
				t.Fatalf("expected *AlterSeqStmt, got %T", stmt)
			}
			if len(s.Name) == 0 {
				t.Fatal("expected non-empty Name")
			}
		})
	}
}

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
			e, ok := stmt.(*CreateExtensionStmt)
			if !ok {
				t.Fatalf("expected *CreateExtensionStmt, got %T", stmt)
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
			e, ok := stmt.(*AlterExtensionStmt)
			if !ok {
				t.Fatalf("expected *AlterExtensionStmt, got %T", stmt)
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
			p, ok := stmt.(*CreatePolicyStmt)
			if !ok {
				t.Fatalf("expected *CreatePolicyStmt, got %T", stmt)
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
			p, ok := stmt.(*CreatePublicationStmt)
			if !ok {
				t.Fatalf("expected *CreatePublicationStmt, got %T", stmt)
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
			s, ok := stmt.(*CreateSubscriptionStmt)
			if !ok {
				t.Fatalf("expected *CreateSubscriptionStmt, got %T", stmt)
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
			e, ok := stmt.(*CreateEventTrigStmt)
			if !ok {
				t.Fatalf("expected *CreateEventTrigStmt, got %T", stmt)
			}
			if e.Trigname == "" {
				t.Fatal("expected non-empty Trigname")
			}
		})
	}
}
