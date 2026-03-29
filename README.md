<div align="center">

  [![Build with Ona](https://ona.com/build-with-ona.svg)](https://app.ona.com/#https://github.com/jespino/gopgsql)

# gopgsql

A PostgreSQL SQL parser in pure Go. No generated code, no grammar files, no flex/bison — just a recursive-descent scanner and parser modeled after the Go compiler's architecture.

</div>

Produces an AST equivalent to PostgreSQL's internal parse tree (`parsenodes.h`).

## Usage

```go
package main

import (
	"fmt"
	"strings"

	"github.com/jespino/gopgsql/parser"
)

func main() {
	sql := `SELECT u.name, count(*) FROM users u JOIN orders o ON o.user_id = u.id WHERE u.active GROUP BY u.name`

	stmts, err := parser.Parse(strings.NewReader(sql), nil)
	if err != nil {
		panic(err)
	}

	for _, raw := range stmts {
		switch stmt := raw.Stmt.(type) {
		case *parser.SelectStmt:
			fmt.Printf("SELECT with %d target columns\n", len(stmt.TargetList))
			fmt.Printf("FROM %d sources\n", len(stmt.FromClause))
			if stmt.WhereClause != nil {
				fmt.Println("Has WHERE clause")
			}
			if len(stmt.GroupClause) > 0 {
				fmt.Println("Has GROUP BY clause")
			}
		}
	}
}
```

## API

The public API is a single function:

```go
func parser.Parse(src io.Reader, errh func(pos int, msg string)) ([]*parser.RawStmt, error)
```

- `src` — SQL source text
- `errh` — optional error handler called for each parse error (nil to collect only the first error)
- Returns a slice of `RawStmt`, each wrapping a `Stmt` node

All AST node types are exported from the `parser` package. Statements implement `parser.Stmt`, expressions implement `parser.Expr`, and all nodes implement `parser.Node`.

## What's supported

### Statements (~100 types)

| Category | Statements |
|---|---|
| **DML** | SELECT, INSERT, UPDATE, DELETE, MERGE |
| **DDL — Tables** | CREATE TABLE, CREATE TABLE AS, ALTER TABLE (~40 sub-commands), DROP TABLE, TRUNCATE |
| **DDL — Indexes** | CREATE INDEX, CREATE UNIQUE INDEX |
| **DDL — Views** | CREATE VIEW, CREATE MATERIALIZED VIEW, REFRESH MATERIALIZED VIEW |
| **DDL — Functions** | CREATE FUNCTION/PROCEDURE, DO, CALL |
| **DDL — Types** | CREATE TYPE (enum, composite, range, shell), CREATE DOMAIN |
| **DDL — Sequences** | CREATE SEQUENCE, ALTER SEQUENCE |
| **DDL — Other** | CREATE/ALTER/DROP for SCHEMA, EXTENSION, TRIGGER, RULE, POLICY, PUBLICATION, SUBSCRIPTION, EVENT TRIGGER, STATISTICS |
| **DDL — Define** | CREATE AGGREGATE, OPERATOR, OPERATOR CLASS/FAMILY, TEXT SEARCH (parser/dictionary/template/configuration), COLLATION, CAST, TRANSFORM, ACCESS METHOD, LANGUAGE, CONVERSION |
| **DDL — FDW** | CREATE FOREIGN DATA WRAPPER, SERVER, FOREIGN TABLE, USER MAPPING, IMPORT FOREIGN SCHEMA |
| **DDL — Database** | CREATE/ALTER/DROP DATABASE, CREATE TABLESPACE |
| **Privileges** | GRANT/REVOKE (privileges and roles), ALTER DEFAULT PRIVILEGES |
| **Transactions** | BEGIN, COMMIT, ROLLBACK, SAVEPOINT, RELEASE, SET TRANSACTION |
| **Session** | SET, RESET, SHOW, LISTEN, NOTIFY, UNLISTEN, DISCARD |
| **Cursors** | DECLARE CURSOR, FETCH, MOVE, CLOSE |
| **Utility** | EXPLAIN, COPY, VACUUM, ANALYZE, LOCK, PREPARE, EXECUTE, DEALLOCATE, COMMENT ON, SECURITY LABEL, REINDEX, CHECKPOINT, LOAD, SET CONSTRAINTS |

### Expressions

Full operator precedence parsing including:

- Arithmetic, comparison, logical, string operators
- IS [NOT] NULL/TRUE/FALSE/UNKNOWN/DISTINCT FROM/JSON/DOCUMENT/NORMALIZED
- BETWEEN, IN, LIKE, ILIKE, SIMILAR TO
- CASE, COALESCE, NULLIF, GREATEST, LEAST
- Type casts (`::` and CAST), array subscripts, field selection
- Subqueries (scalar, EXISTS, ANY/ALL/SOME)
- Window functions with full frame specification
- JSON: JSON_OBJECT, JSON_ARRAY, JSON_QUERY, JSON_VALUE, JSON_EXISTS, JSON_SCALAR, JSON_SERIALIZE, JSON_OBJECTAGG, JSON_ARRAYAGG, JSON_TABLE
- XML: XMLELEMENT, XMLFOREST, XMLPARSE, XMLPI, XMLROOT, XMLSERIALIZE, XMLEXISTS, XMLTABLE
- SQL functions: EXTRACT, POSITION, SUBSTRING, OVERLAY, TRIM, NORMALIZE, COLLATION FOR
- Named arguments (`name := expr`, `name => expr`), VARIADIC
- WITHIN GROUP, FILTER, RESPECT/IGNORE NULLS
- OVERLAPS, TABLESAMPLE, MERGE_ACTION()

## Project structure

```
gopgsql/
├── scanner/          # Lexical scanner (tokens, keywords, string/number/operator scanning)
├── parser/           # Recursive-descent parser and AST node definitions
└── tests/            # Integration tests organized by SQL feature
```

### Test organization

Tests are in `tests/` and organized by feature — each file covers a specific area of SQL syntax:

```
tests/
├── expr_test.go           # Expressions, operators, precedence
├── select_test.go         # SELECT, joins, subqueries, CTEs, set operations
├── insert_test.go         # INSERT, ON CONFLICT
├── update_delete_test.go  # UPDATE, DELETE
├── merge_test.go          # MERGE
├── ddl_table_test.go      # CREATE TABLE, constraints
├── ddl_index_test.go      # CREATE INDEX
├── ddl_view_test.go       # CREATE VIEW
├── ddl_function_test.go   # CREATE FUNCTION/PROCEDURE, DO, CALL
├── ddl_type_test.go       # CREATE TYPE, CREATE DOMAIN
├── ddl_define_test.go     # CREATE AGGREGATE, OPERATOR, TEXT SEARCH, CAST, ...
├── alter_table_test.go    # ALTER TABLE sub-commands
├── alter_misc_test.go     # ALTER ROLE, DOMAIN, TYPE, FUNCTION, ...
├── drop_test.go           # DROP, TRUNCATE
├── grant_test.go          # GRANT, REVOKE, CREATE ROLE
├── cursor_test.go         # DECLARE, FETCH, MOVE, CLOSE
├── transaction_test.go    # BEGIN, COMMIT, ROLLBACK, SET, SHOW, LISTEN, ...
├── json_xml_test.go       # JSON/XML constructors, functions, table sources
├── comment_test.go        # COMMENT ON, SECURITY LABEL
├── utility_test.go        # EXPLAIN, COPY, VACUUM, REINDEX, PREPARE, ...
└── ...                    # (32 test files, 948 tests total)
```

## Running tests

```bash
go test ./...
```

## Design

The parser follows the Go compiler's scanner/parser architecture:

- **Scanner** (`scanner/`) reads UTF-8 source one rune at a time, producing tokens. Handles PostgreSQL-specific lexical elements: dollar-quoted strings, Unicode escapes (`U&'...'`), bit-string literals, operator classification, and the full keyword table.
- **Parser** (`parser/`) is a recursive-descent parser using precedence climbing for expressions. No parser generator — each grammar production is a Go function. The AST node types mirror PostgreSQL's `parsenodes.h` naming conventions.

### Why not flex/bison?

- Full control over error recovery and error messages
- No build-time dependencies on parser generators
- Easier to debug and extend
- Matches the approach used by the Go compiler itself

## Limitations

- Error recovery is minimal — the parser stops at the first error
- Some rarely-used syntax forms may not be covered (e.g., EXCLUDE constraint bodies are stubbed)
- No semantic analysis — the parser produces a syntactic AST only
- Position tracking uses a synthetic encoding, not raw byte offsets

## License

MIT
