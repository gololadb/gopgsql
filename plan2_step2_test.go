package pgscan

import "testing"

func TestDeclareCursor(t *testing.T) {
	tests := []struct {
		sql  string
		desc string
		opts int
	}{
		{"DECLARE c CURSOR FOR SELECT 1", "basic", 0},
		{"DECLARE c BINARY CURSOR FOR SELECT 1", "binary", CURSOR_OPT_BINARY},
		{"DECLARE c SCROLL CURSOR FOR SELECT 1", "scroll", CURSOR_OPT_SCROLL},
		{"DECLARE c NO SCROLL CURSOR FOR SELECT 1", "no scroll", CURSOR_OPT_NO_SCROLL},
		{"DECLARE c INSENSITIVE CURSOR FOR SELECT 1", "insensitive", CURSOR_OPT_INSENSITIVE},
		{"DECLARE c CURSOR WITH HOLD FOR SELECT 1", "with hold", CURSOR_OPT_HOLD},
		{"DECLARE c CURSOR WITHOUT HOLD FOR SELECT 1", "without hold", 0},
		{"DECLARE c BINARY SCROLL CURSOR WITH HOLD FOR SELECT 1", "all options",
			CURSOR_OPT_BINARY | CURSOR_OPT_SCROLL | CURSOR_OPT_HOLD},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			dc, ok := stmt.(*DeclareCursorStmt)
			if !ok {
				t.Fatalf("expected *DeclareCursorStmt, got %T", stmt)
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
		dir     FetchDirection
		howMany int64
		portal  string
	}{
		{"FETCH c", "default", FETCH_FORWARD, 1, "c"},
		{"FETCH NEXT FROM c", "next", FETCH_FORWARD, 1, "c"},
		{"FETCH PRIOR FROM c", "prior", FETCH_BACKWARD, 1, "c"},
		{"FETCH FIRST FROM c", "first", FETCH_ABSOLUTE, 1, "c"},
		{"FETCH LAST FROM c", "last", FETCH_ABSOLUTE, -1, "c"},
		{"FETCH ABSOLUTE 5 FROM c", "absolute", FETCH_ABSOLUTE, 5, "c"},
		{"FETCH ABSOLUTE -3 FROM c", "absolute neg", FETCH_ABSOLUTE, -3, "c"},
		{"FETCH RELATIVE 2 FROM c", "relative", FETCH_RELATIVE, 2, "c"},
		{"FETCH 10 FROM c", "count", FETCH_FORWARD, 10, "c"},
		{"FETCH ALL FROM c", "all", FETCH_FORWARD, 0, "c"},
		{"FETCH FORWARD FROM c", "forward", FETCH_FORWARD, 1, "c"},
		{"FETCH FORWARD 5 FROM c", "forward n", FETCH_FORWARD, 5, "c"},
		{"FETCH FORWARD ALL FROM c", "forward all", FETCH_FORWARD, 0, "c"},
		{"FETCH BACKWARD FROM c", "backward", FETCH_BACKWARD, 1, "c"},
		{"FETCH BACKWARD 3 FROM c", "backward n", FETCH_BACKWARD, 3, "c"},
		{"FETCH BACKWARD ALL FROM c", "backward all", FETCH_BACKWARD, 0, "c"},
		{"FETCH NEXT IN c", "in keyword", FETCH_FORWARD, 1, "c"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stmt := parseOne(t, tt.sql)
			fs, ok := stmt.(*FetchStmt)
			if !ok {
				t.Fatalf("expected *FetchStmt, got %T", stmt)
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
			fs, ok := stmt.(*FetchStmt)
			if !ok {
				t.Fatalf("expected *FetchStmt, got %T", stmt)
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
			cs, ok := stmt.(*ClosePortalStmt)
			if !ok {
				t.Fatalf("expected *ClosePortalStmt, got %T", stmt)
			}
			if cs.Portalname != tt.portal {
				t.Fatalf("expected portal %q, got %q", tt.portal, cs.Portalname)
			}
		})
	}
}
