package tests

import (
	"strings"
	"testing"

	"github.com/gololadb/gopgsql/parser"
)

// deparseRoundTrip parses SQL, deparses it, then parses again to verify
// the deparsed SQL is valid. Returns the deparsed string.
func deparseRoundTrip(t *testing.T, sql string) string {
	t.Helper()
	stmts, err := parser.Parse(strings.NewReader(sql), nil)
	if err != nil {
		t.Fatalf("initial parse error: %v", err)
	}
	if len(stmts) == 0 {
		t.Fatal("no statements parsed")
	}
	deparsed := parser.Deparse(stmts[0])
	// Verify the deparsed SQL is parseable.
	_, err = parser.Parse(strings.NewReader(deparsed), nil)
	if err != nil {
		t.Fatalf("re-parse of deparsed SQL failed: %v\nDeparsed: %s", err, deparsed)
	}
	return deparsed
}

// --- SELECT ---

func TestDeparseSimpleSelect(t *testing.T) {
	sql := "SELECT 1"
	d := deparseRoundTrip(t, sql)
	if !strings.Contains(d, "SELECT 1") {
		t.Errorf("unexpected: %s", d)
	}
}

func TestDeparseSelectColumns(t *testing.T) {
	sql := "SELECT a, b, c FROM t"
	d := deparseRoundTrip(t, sql)
	if !strings.Contains(d, "a") || !strings.Contains(d, "FROM t") {
		t.Errorf("unexpected: %s", d)
	}
}

func TestDeparseSelectWhere(t *testing.T) {
	deparseRoundTrip(t, "SELECT * FROM t WHERE x = 1 AND y > 2")
}

func TestDeparseSelectJoin(t *testing.T) {
	deparseRoundTrip(t, "SELECT a.id, b.name FROM a JOIN b ON a.id = b.a_id")
}

func TestDeparseSelectLeftJoin(t *testing.T) {
	deparseRoundTrip(t, "SELECT * FROM a LEFT JOIN b ON a.id = b.a_id")
}

func TestDeparseSelectOrderBy(t *testing.T) {
	deparseRoundTrip(t, "SELECT * FROM t ORDER BY a DESC, b ASC NULLS FIRST")
}

func TestDeparseSelectLimit(t *testing.T) {
	deparseRoundTrip(t, "SELECT * FROM t LIMIT 10 OFFSET 5")
}

func TestDeparseSelectDistinct(t *testing.T) {
	deparseRoundTrip(t, "SELECT DISTINCT a, b FROM t")
}

func TestDeparseSelectGroupBy(t *testing.T) {
	deparseRoundTrip(t, "SELECT dept, count(*) FROM emp GROUP BY dept HAVING count(*) > 5")
}

func TestDeparseSelectSubquery(t *testing.T) {
	deparseRoundTrip(t, "SELECT * FROM (SELECT 1 AS x) sub")
}

func TestDeparseSelectUnion(t *testing.T) {
	deparseRoundTrip(t, "SELECT 1 UNION ALL SELECT 2")
}

func TestDeparseSelectCTE(t *testing.T) {
	deparseRoundTrip(t, "WITH cte AS (SELECT 1 AS x) SELECT * FROM cte")
}

func TestDeparseSelectExists(t *testing.T) {
	deparseRoundTrip(t, "SELECT * FROM t WHERE EXISTS (SELECT 1 FROM u WHERE u.id = t.id)")
}

func TestDeparseSelectIn(t *testing.T) {
	deparseRoundTrip(t, "SELECT * FROM t WHERE id IN (1, 2, 3)")
}

func TestDeparseSelectCase(t *testing.T) {
	deparseRoundTrip(t, "SELECT CASE WHEN x > 0 THEN 'pos' WHEN x < 0 THEN 'neg' ELSE 'zero' END FROM t")
}

func TestDeparseSelectCoalesce(t *testing.T) {
	deparseRoundTrip(t, "SELECT COALESCE(a, b, 0) FROM t")
}

func TestDeparseSelectBooleanLiterals(t *testing.T) {
	d := deparseRoundTrip(t, "SELECT TRUE, FALSE")
	if !strings.Contains(d, "TRUE") || !strings.Contains(d, "FALSE") {
		t.Errorf("expected TRUE and FALSE in: %s", d)
	}
}

func TestDeparseSelectTypeCast(t *testing.T) {
	deparseRoundTrip(t, "SELECT x::integer FROM t")
}

func TestDeparseSelectIsNull(t *testing.T) {
	deparseRoundTrip(t, "SELECT * FROM t WHERE x IS NULL AND y IS NOT NULL")
}

func TestDeparseSelectBetween(t *testing.T) {
	deparseRoundTrip(t, "SELECT * FROM t WHERE x BETWEEN 1 AND 10")
}

func TestDeparseSelectLike(t *testing.T) {
	deparseRoundTrip(t, "SELECT * FROM t WHERE name LIKE 'foo%'")
}

func TestDeparseSelectFuncCall(t *testing.T) {
	deparseRoundTrip(t, "SELECT count(*), sum(x), avg(DISTINCT y) FROM t")
}

func TestDeparseSelectWindowFunc(t *testing.T) {
	deparseRoundTrip(t, "SELECT row_number() OVER (PARTITION BY dept ORDER BY salary DESC) FROM emp")
}

func TestDeparseSelectCurrentUser(t *testing.T) {
	d := deparseRoundTrip(t, "SELECT CURRENT_USER")
	if !strings.Contains(d, "CURRENT_USER") {
		t.Errorf("expected CURRENT_USER in: %s", d)
	}
}

// --- INSERT ---

func TestDeparseInsertValues(t *testing.T) {
	deparseRoundTrip(t, "INSERT INTO t (a, b) VALUES (1, 'hello')")
}

func TestDeparseInsertSelect(t *testing.T) {
	deparseRoundTrip(t, "INSERT INTO t SELECT * FROM u")
}

func TestDeparseInsertOnConflict(t *testing.T) {
	deparseRoundTrip(t, "INSERT INTO t (id, val) VALUES (1, 'x') ON CONFLICT (id) DO UPDATE SET val = 'y'")
}

func TestDeparseInsertReturning(t *testing.T) {
	deparseRoundTrip(t, "INSERT INTO t (a) VALUES (1) RETURNING *")
}

// --- UPDATE ---

func TestDeparseUpdate(t *testing.T) {
	deparseRoundTrip(t, "UPDATE t SET a = 1, b = 'hello' WHERE id = 5")
}

func TestDeparseUpdateReturning(t *testing.T) {
	deparseRoundTrip(t, "UPDATE t SET a = 1 RETURNING *")
}

// --- DELETE ---

func TestDeparseDelete(t *testing.T) {
	deparseRoundTrip(t, "DELETE FROM t WHERE id = 5")
}

func TestDeparseDeleteReturning(t *testing.T) {
	deparseRoundTrip(t, "DELETE FROM t WHERE id = 5 RETURNING *")
}

// --- CREATE TABLE ---

func TestDeparseCreateTable(t *testing.T) {
	deparseRoundTrip(t, "CREATE TABLE t (id integer NOT NULL, name text)")
}

func TestDeparseCreateTablePartition(t *testing.T) {
	d := deparseRoundTrip(t, "CREATE TABLE t (id integer, dt date) PARTITION BY RANGE (dt)")
	if !strings.Contains(d, "PARTITION BY RANGE") {
		t.Errorf("expected PARTITION BY RANGE in: %s", d)
	}
}

// --- CREATE VIEW ---

func TestDeparseCreateView(t *testing.T) {
	d := deparseRoundTrip(t, "CREATE VIEW v AS SELECT a, b FROM t WHERE active = TRUE")
	if !strings.Contains(d, "CREATE") && !strings.Contains(d, "VIEW") {
		t.Errorf("unexpected: %s", d)
	}
	// The key test: the deparsed view definition should contain the SELECT.
	if !strings.Contains(d, "SELECT") {
		t.Errorf("expected SELECT in view deparse: %s", d)
	}
}

// --- CREATE INDEX ---

func TestDeparseCreateIndex(t *testing.T) {
	deparseRoundTrip(t, "CREATE INDEX idx ON t (col)")
}

func TestDeparseCreateUniqueIndex(t *testing.T) {
	deparseRoundTrip(t, "CREATE UNIQUE INDEX idx ON t (col)")
}

// --- EXPLAIN ---

func TestDeparseExplain(t *testing.T) {
	deparseRoundTrip(t, "EXPLAIN SELECT * FROM t")
}
