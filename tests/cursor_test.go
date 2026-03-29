package tests

import (
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestDeclareCursor(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
		opts int
	}{
		{"DECLARE c CURSOR FOR SELECT 1", "basic", 0},
		{"DECLARE c BINARY CURSOR FOR SELECT 1", "binary", parser.CURSOR_OPT_BINARY},
		{"DECLARE c SCROLL CURSOR FOR SELECT 1", "scroll", parser.CURSOR_OPT_SCROLL},
		{"DECLARE c NO SCROLL CURSOR FOR SELECT 1", "no scroll", parser.CURSOR_OPT_NO_SCROLL},
		{"DECLARE c INSENSITIVE CURSOR FOR SELECT 1", "insensitive", parser.CURSOR_OPT_INSENSITIVE},
		{"DECLARE c CURSOR WITH HOLD FOR SELECT 1", "with hold", parser.CURSOR_OPT_HOLD},
		{"DECLARE c CURSOR WITHOUT HOLD FOR SELECT 1", "without hold", 0},
		{"DECLARE c BINARY SCROLL CURSOR WITH HOLD FOR SELECT 1", "all options",
			parser.CURSOR_OPT_BINARY | parser.CURSOR_OPT_SCROLL | parser.CURSOR_OPT_HOLD},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			dc, ok := stmt.(*parser.DeclareCursorStmt)
			if !ok {
				t.Fatalf("expected *parser.DeclareCursorStmt, got %T", stmt)
			}
			if dc.Portalname != "c" {
				t.Fatalf("expected portalname 'c', got %q", dc.Portalname)
			}
			if dc.Options != tt.opts {
				t.Fatalf("expected options %d, got %d", tt.opts, dc.Options)
			}
			if dc.Query == nil {
				t.Fatal("expected non-nil Query")
			}
		})
	}
}


func TestFetchStmt(t *testing.T) {
	tests := []struct {
		sql     string
		desc    string
		dir     parser.FetchDirection
		howMany int64
		portal  string
	}{
		{"FETCH c", "default", parser.FETCH_FORWARD, 1, "c"},
		{"FETCH NEXT FROM c", "next", parser.FETCH_FORWARD, 1, "c"},
		{"FETCH PRIOR FROM c", "prior", parser.FETCH_BACKWARD, 1, "c"},
		{"FETCH FIRST FROM c", "first", parser.FETCH_ABSOLUTE, 1, "c"},
		{"FETCH LAST FROM c", "last", parser.FETCH_ABSOLUTE, -1, "c"},
		{"FETCH ABSOLUTE 5 FROM c", "absolute", parser.FETCH_ABSOLUTE, 5, "c"},
		{"FETCH ABSOLUTE -3 FROM c", "absolute neg", parser.FETCH_ABSOLUTE, -3, "c"},
		{"FETCH RELATIVE 2 FROM c", "relative", parser.FETCH_RELATIVE, 2, "c"},
		{"FETCH 10 FROM c", "count", parser.FETCH_FORWARD, 10, "c"},
		{"FETCH ALL FROM c", "all", parser.FETCH_FORWARD, 0, "c"},
		{"FETCH FORWARD FROM c", "forward", parser.FETCH_FORWARD, 1, "c"},
		{"FETCH FORWARD 5 FROM c", "forward n", parser.FETCH_FORWARD, 5, "c"},
		{"FETCH FORWARD ALL FROM c", "forward all", parser.FETCH_FORWARD, 0, "c"},
		{"FETCH BACKWARD FROM c", "backward", parser.FETCH_BACKWARD, 1, "c"},
		{"FETCH BACKWARD 3 FROM c", "backward n", parser.FETCH_BACKWARD, 3, "c"},
		{"FETCH BACKWARD ALL FROM c", "backward all", parser.FETCH_BACKWARD, 0, "c"},
		{"FETCH NEXT IN c", "in keyword", parser.FETCH_FORWARD, 1, "c"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			fs, ok := stmt.(*parser.FetchStmt)
			if !ok {
				t.Fatalf("expected *parser.FetchStmt, got %T", stmt)
			}
			if fs.IsMove {
				t.Fatal("expected IsMove=false")
			}
			if fs.Direction != tt.dir {
				t.Fatalf("expected direction %d, got %d", tt.dir, fs.Direction)
			}
			if fs.HowMany != tt.howMany {
				t.Fatalf("expected howMany %d, got %d", tt.howMany, fs.HowMany)
			}
			if fs.Portalname != tt.portal {
				t.Fatalf("expected portal %q, got %q", tt.portal, fs.Portalname)
			}
		})
	}
}


func TestMoveStmt(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
	}{
		{"MOVE c", "basic"},
		{"MOVE NEXT FROM c", "next"},
		{"MOVE FORWARD 10 FROM c", "forward n"},
		{"MOVE ALL FROM c", "all"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			fs, ok := stmt.(*parser.FetchStmt)
			if !ok {
				t.Fatalf("expected *parser.FetchStmt, got %T", stmt)
			}
			if !fs.IsMove {
				t.Fatal("expected IsMove=true")
			}
		})
	}
}


func TestClosePortal(t *testing.T) {
	tests := []struct {
		sql    string
		desc   string
		portal string
	}{
		{"CLOSE c", "basic", "c"},
		{"CLOSE ALL", "all", ""},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			cs, ok := stmt.(*parser.ClosePortalStmt)
			if !ok {
				t.Fatalf("expected *parser.ClosePortalStmt, got %T", stmt)
			}
			if cs.Portalname != tt.portal {
				t.Fatalf("expected portal %q, got %q", tt.portal, cs.Portalname)
			}
		})
	}
}

