# PostgreSQL Parser — Gap-Filling Plan

After completing the initial 15-step plan, the parser covers ~54 of 127
statement types and ~85% of the expression grammar. This plan addresses the
remaining gaps in 12 incremental steps, ordered by usage frequency.

## Current coverage

**Statements (54/127):** SELECT, INSERT, UPDATE, DELETE, MERGE, CREATE TABLE,
CREATE TABLE AS, CREATE INDEX, CREATE VIEW, CREATE FUNCTION/PROCEDURE,
CREATE TRIGGER, CREATE RULE, CREATE SEQUENCE, CREATE EXTENSION, CREATE POLICY,
CREATE PUBLICATION, CREATE SUBSCRIPTION, CREATE EVENT TRIGGER, CREATE ROLE/USER/GROUP,
CREATE SCHEMA, CREATE DOMAIN, CREATE TYPE (ENUM, composite), ALTER TABLE,
ALTER SEQUENCE, ALTER EXTENSION, DROP, TRUNCATE, EXPLAIN, COPY, BEGIN/COMMIT/
ROLLBACK/SAVEPOINT/RELEASE, SET/RESET/SHOW, LISTEN/NOTIFY/UNLISTEN,
VACUUM/ANALYZE, LOCK, PREPARE/EXECUTE/DEALLOCATE, DISCARD, DO, CALL,
GRANT/REVOKE (privileges and roles).

**Expressions:** All arithmetic, comparison, boolean, IS forms, BETWEEN, IN,
LIKE/ILIKE, SIMILAR TO, CASE, CAST, COALESCE, NULLIF, GREATEST/LEAST, ARRAY,
ROW, EXISTS, subqueries (ANY/ALL/SOME), function calls (DISTINCT, ORDER BY,
FILTER, OVER), COLLATE, AT TIME ZONE, AT LOCAL, subscripts, type casts,
EXTRACT, POSITION, SUBSTRING, OVERLAY, TRIM, TREAT, NORMALIZE, COLLATION FOR,
typed literals, interval fields, window frames, IS JSON, all XML functions
(XMLCONCAT/ELEMENT/FOREST/PARSE/PI/ROOT/SERIALIZE/EXISTS), JSON_OBJECT,
JSON_ARRAY, JSON_QUERY, JSON_VALUE, JSON_EXISTS.

---

## Step 1 — Function call completions

Fill in missing function-call features that affect expression parsing.

| Feature | Description | PG production |
|---|---|---|
| Named arguments | `func(name := expr)` and `func(name => expr)` | `func_arg_expr` |
| VARIADIC in calls | `func(VARIADIC array_expr)` | `func_arg_expr` |
| WITHIN GROUP | `percentile_cont(0.5) WITHIN GROUP (ORDER BY col)` | `within_group_clause` |
| Null treatment | `first_value(x) IGNORE NULLS OVER (...)` | `null_treatment` |
| MERGE_ACTION() | `MERGE_ACTION()` inside MERGE WHEN clauses | `func_expr_common_subexpr` |

**New nodes:** `NamedArgExpr`.
**Modified:** `parseFuncCall` (named args, VARIADIC), `parseFuncSuffix`
(WITHIN GROUP, null treatment), `parsePrimaryExpr` (MERGE_ACTION).

---

## Step 2 — Cursor statements

| Feature | Description | PG production |
|---|---|---|
| DECLARE | `DECLARE name [BINARY] [ASENSITIVE\|INSENSITIVE] [NO] SCROLL CURSOR [WITH HOLD] FOR query` | `DeclareCursorStmt` |
| FETCH / MOVE | `FETCH [direction] [FROM\|IN] cursor` — NEXT, PRIOR, FIRST, LAST, ABSOLUTE n, RELATIVE n, n, ALL, FORWARD, BACKWARD | `FetchStmt` |
| CLOSE | `CLOSE cursor` / `CLOSE ALL` | `ClosePortalStmt` |

**New nodes:** `DeclareCursorStmt`, `FetchStmt`, `ClosePortalStmt`.
**New file:** `parse_cursor.go`.

---

## Step 3 — Database and tablespace DDL

| Feature | Description | PG production |
|---|---|---|
| CREATE DATABASE | `CREATE DATABASE name [WITH] [OWNER=role] [TEMPLATE=name] [ENCODING=enc] [LOCALE=loc] ...` | `CreatedbStmt` |
| DROP DATABASE | `DROP DATABASE [IF EXISTS] name [WITH (FORCE)]` | `DropdbStmt` |
| ALTER DATABASE | `ALTER DATABASE name [WITH] options` / `ALTER DATABASE name SET config` | `AlterDatabaseStmt`, `AlterDatabaseSetStmt` |
| CREATE TABLESPACE | `CREATE TABLESPACE name [OWNER role] LOCATION 'dir' [WITH (opts)]` | `CreateTableSpaceStmt` |
| ALTER TABLESPACE | `ALTER TABLESPACE name SET/RESET/RENAME/OWNER` | `AlterTblSpcStmt` |
| DROP TABLESPACE | `DROP TABLESPACE [IF EXISTS] name` | via `DropStmt` |

**New nodes:** `CreatedbStmt`, `AlterDatabaseStmt`, `AlterDatabaseSetStmt`,
`CreateTableSpaceStmt`.
**New file:** `parse_database.go`.

---

## Step 4 — ALTER dispatch expansion

Expand the ALTER dispatcher to cover the most common ALTER targets beyond
TABLE, SEQUENCE, and EXTENSION.

| Feature | Description | PG production |
|---|---|---|
| ALTER ROLE/USER | `ALTER ROLE name [WITH] options` / `ALTER ROLE name SET config` | `AlterRoleStmt`, `AlterRoleSetStmt` |
| ALTER DOMAIN | `ALTER DOMAIN name SET DEFAULT/DROP DEFAULT/SET NOT NULL/DROP NOT NULL/ADD CONSTRAINT/DROP CONSTRAINT/RENAME/SET SCHEMA/OWNER TO` | `AlterDomainStmt` |
| ALTER TYPE | `ALTER TYPE name ADD VALUE/RENAME VALUE/ADD ATTRIBUTE/DROP ATTRIBUTE/ALTER ATTRIBUTE/RENAME/SET SCHEMA/OWNER TO` | `AlterEnumStmt`, `AlterCompositeTypeStmt`, `AlterTypeStmt` |
| ALTER FUNCTION | `ALTER FUNCTION name (...) action` — RENAME, SET SCHEMA, OWNER TO, SET config, SECURITY, COST, ROWS, PARALLEL | `AlterFunctionStmt` |
| ALTER POLICY | `ALTER POLICY name ON table [TO roles] [USING (expr)] [WITH CHECK (expr)]` | `AlterPolicyStmt` |
| ALTER PUBLICATION | `ALTER PUBLICATION name ADD/DROP/SET TABLE ...` | `AlterPublicationStmt` |
| ALTER SUBSCRIPTION | `ALTER SUBSCRIPTION name SET/ENABLE/DISABLE/REFRESH/ADD/DROP PUBLICATION` | `AlterSubscriptionStmt` |
| ALTER EVENT TRIGGER | `ALTER EVENT TRIGGER name ENABLE/DISABLE/RENAME/OWNER TO` | `AlterEventTrigStmt` |
| ALTER SYSTEM | `ALTER SYSTEM SET/RESET config` | `AlterSystemStmt` |

**New nodes:** `AlterRoleStmt` (exists in nodes.go, wire to dispatch),
`AlterRoleSetStmt`, `AlterDomainStmt`, `AlterFunctionStmt`,
`AlterPublicationStmt`, `AlterSubscriptionStmt`, `AlterEventTrigStmt`,
`AlterSystemStmt`.
**Modified:** `parseAlterStmt` dispatch in `parse_ddl2.go`.
**New file:** `parse_alter.go`.

---

## Step 5 — ALTER TABLE sub-command completions

The current ALTER TABLE handles ADD/DROP COLUMN, ADD/DROP CONSTRAINT,
ALTER COLUMN (SET NOT NULL, DROP NOT NULL, SET DEFAULT, DROP DEFAULT,
SET DATA TYPE, TYPE), RENAME, SET SCHEMA. Add the remaining sub-commands.

| Feature | Description | PG constant |
|---|---|---|
| SET STATISTICS | `ALTER TABLE t ALTER COLUMN c SET STATISTICS n` | `AT_SetStatistics` |
| SET STORAGE | `ALTER TABLE t ALTER COLUMN c SET STORAGE type` | `AT_SetStorage` |
| SET COMPRESSION | `ALTER TABLE t ALTER COLUMN c SET COMPRESSION method` | `AT_SetCompression` |
| ADD GENERATED | `ALTER TABLE t ALTER COLUMN c ADD GENERATED {ALWAYS\|BY DEFAULT} AS IDENTITY [(seq_opts)]` | `AT_AddIdentity` |
| DROP IDENTITY | `ALTER TABLE t ALTER COLUMN c DROP IDENTITY [IF EXISTS]` | `AT_DropIdentity` |
| SET EXPRESSION | `ALTER TABLE t ALTER COLUMN c SET EXPRESSION AS (expr)` | `AT_SetExpression` |
| DROP EXPRESSION | `ALTER TABLE t ALTER COLUMN c DROP EXPRESSION [IF EXISTS]` | `AT_DropExpression` |
| ALTER CONSTRAINT | `ALTER TABLE t ALTER CONSTRAINT name [DEFERRABLE\|NOT DEFERRABLE] [INITIALLY ...]` | `AT_AlterConstraint` |
| VALIDATE CONSTRAINT | `ALTER TABLE t VALIDATE CONSTRAINT name` | `AT_ValidateConstraint` |
| CLUSTER ON | `ALTER TABLE t CLUSTER ON index` | `AT_ClusterOn` |
| SET WITHOUT CLUSTER | `ALTER TABLE t SET WITHOUT CLUSTER` | `AT_DropCluster` |
| SET LOGGED/UNLOGGED | `ALTER TABLE t SET LOGGED` / `SET UNLOGGED` | `AT_SetLogged`, `AT_SetUnLogged` |
| SET ACCESS METHOD | `ALTER TABLE t SET ACCESS METHOD name` | `AT_SetAccessMethod` |
| SET TABLESPACE | `ALTER TABLE t SET TABLESPACE name` | `AT_SetTableSpace` |
| SET/RESET reloptions | `ALTER TABLE t SET (opt=val, ...)` / `RESET (opt, ...)` | `AT_SetRelOptions`, `AT_ResetRelOptions` |
| ENABLE/DISABLE TRIGGER | `ALTER TABLE t ENABLE\|DISABLE [ALWAYS\|REPLICA] TRIGGER name\|ALL\|USER` | `AT_EnableTrig*`, `AT_DisableTrig*` |
| ENABLE/DISABLE RULE | `ALTER TABLE t ENABLE\|DISABLE [ALWAYS\|REPLICA] RULE name` | `AT_EnableRule*`, `AT_DisableRule*` |
| INHERIT / NO INHERIT | `ALTER TABLE t INHERIT parent` / `NO INHERIT parent` | `AT_AddInherit`, `AT_DropInherit` |
| OF type / NOT OF | `ALTER TABLE t OF type` / `NOT OF` | `AT_AddOf`, `AT_DropOf` |
| OWNER TO | `ALTER TABLE t OWNER TO role` | `AT_ChangeOwner` |
| REPLICA IDENTITY | `ALTER TABLE t REPLICA IDENTITY DEFAULT\|USING INDEX name\|FULL\|NOTHING` | `AT_ReplicaIdentity` |
| ROW LEVEL SECURITY | `ALTER TABLE t ENABLE\|DISABLE\|FORCE\|NO FORCE ROW LEVEL SECURITY` | `AT_EnableRowSecurity` etc. |

**Modified:** `parseAlterTableCmd` in `parse_ddl2.go`.
**New AT_* constants** in `nodes.go`.

---

## Step 6 — DROP completions and RENAME

| Feature | Description | PG production |
|---|---|---|
| DROP ROLE/USER/GROUP | `DROP ROLE [IF EXISTS] name, ...` | `DropRoleStmt` |
| DROP DATABASE | Already in Step 3 | `DropdbStmt` |
| DROP OWNED | `DROP OWNED BY role, ... [CASCADE\|RESTRICT]` | `DropOwnedStmt` |
| DROP CAST | `DROP CAST [IF EXISTS] (source AS target) [CASCADE\|RESTRICT]` | `DropCastStmt` |
| DROP OPERATOR CLASS | `DROP OPERATOR CLASS [IF EXISTS] name USING method [CASCADE\|RESTRICT]` | `DropOpClassStmt` |
| DROP OPERATOR FAMILY | `DROP OPERATOR FAMILY [IF EXISTS] name USING method [CASCADE\|RESTRICT]` | `DropOpFamilyStmt` |
| DROP SUBSCRIPTION | `DROP SUBSCRIPTION [IF EXISTS] name` | `DropSubscriptionStmt` |
| DROP TRANSFORM | `DROP TRANSFORM [IF EXISTS] FOR type LANGUAGE lang [CASCADE\|RESTRICT]` | `DropTransformStmt` |
| DROP USER MAPPING | `DROP USER MAPPING [IF EXISTS] FOR role SERVER name` | `DropUserMappingStmt` |
| DROP AGGREGATE | `DROP AGGREGATE [IF EXISTS] name (args) [CASCADE\|RESTRICT]` | `RemoveAggrStmt` |
| DROP FUNCTION/PROCEDURE | `DROP FUNCTION [IF EXISTS] name (args) [CASCADE\|RESTRICT]` | `RemoveFuncStmt` |
| DROP OPERATOR | `DROP OPERATOR [IF EXISTS] name (left, right) [CASCADE\|RESTRICT]` | `RemoveOperStmt` |
| RENAME (generic) | `ALTER type name RENAME TO newname` — covers TABLE, INDEX, VIEW, SEQUENCE, COLUMN, CONSTRAINT, SCHEMA, DATABASE, ROLE, etc. | `RenameStmt` |
| REASSIGN OWNED | `REASSIGN OWNED BY role, ... TO newrole` | `ReassignOwnedStmt` |

**New nodes:** `DropRoleStmt`, `DropOwnedStmt`, `ReassignOwnedStmt`.
**Modified:** `parseDropStmt` to handle specialized DROP forms.
**New file:** `parse_drop.go` (for specialized drops that don't fit generic DROP).

---

## Step 7 — Foreign data wrappers and foreign tables

| Feature | Description | PG production |
|---|---|---|
| CREATE FOREIGN DATA WRAPPER | `CREATE FDW name [HANDLER func] [VALIDATOR func] [OPTIONS (opts)]` | `CreateFdwStmt` |
| CREATE SERVER | `CREATE SERVER name [TYPE 'type'] [VERSION 'ver'] FOREIGN DATA WRAPPER fdw [OPTIONS (opts)]` | `CreateForeignServerStmt` |
| CREATE FOREIGN TABLE | `CREATE FOREIGN TABLE name (cols) SERVER name [OPTIONS (opts)]` | `CreateForeignTableStmt` |
| CREATE USER MAPPING | `CREATE USER MAPPING FOR role SERVER name [OPTIONS (opts)]` | `CreateUserMappingStmt` |
| IMPORT FOREIGN SCHEMA | `IMPORT FOREIGN SCHEMA remote [LIMIT TO\|EXCEPT (tables)] FROM SERVER name INTO local` | `ImportForeignSchemaStmt` |
| ALTER FDW | `ALTER FDW name [HANDLER\|VALIDATOR\|OPTIONS\|OWNER\|RENAME]` | `AlterFdwStmt` |
| ALTER SERVER | `ALTER SERVER name [VERSION\|OPTIONS\|OWNER\|RENAME]` | `AlterForeignServerStmt` |
| ALTER USER MAPPING | `ALTER USER MAPPING FOR role SERVER name OPTIONS (opts)` | `AlterUserMappingStmt` |

**New nodes:** `CreateFdwStmt` (exists), `CreateForeignServerStmt` (exists),
`CreateForeignTableStmt` (exists), `CreateUserMappingStmt`,
`ImportForeignSchemaStmt`, `AlterFdwStmt`, `AlterForeignServerStmt`,
`AlterUserMappingStmt`.
**New file:** `parse_fdw.go`.

---

## Step 8 — Materialized views and statistics

| Feature | Description | PG production |
|---|---|---|
| CREATE MATERIALIZED VIEW | `CREATE MATERIALIZED VIEW [IF NOT EXISTS] name [USING method] [(cols)] [TABLESPACE ts] AS query [WITH [NO] DATA]` | `CreateMatViewStmt` |
| REFRESH MATERIALIZED VIEW | `REFRESH MATERIALIZED VIEW [CONCURRENTLY] name [WITH [NO] DATA]` | `RefreshMatViewStmt` |
| CREATE STATISTICS | `CREATE STATISTICS [IF NOT EXISTS] name [(types)] ON exprs FROM table` | `CreateStatsStmt` (node exists) |
| ALTER STATISTICS | `ALTER STATISTICS name SET STATISTICS n / RENAME / SET SCHEMA / OWNER TO` | `AlterStatsStmt` |

**New nodes:** `CreateMatViewStmt`, `RefreshMatViewStmt`.
**Modified:** `parseCreateStmt` dispatch, statement dispatch.

---

## Step 9 — COMMENT, SECURITY LABEL, and utility statements

| Feature | Description | PG production |
|---|---|---|
| COMMENT ON | `COMMENT ON object_type name IS 'text'` / `IS NULL` | `CommentStmt` |
| SECURITY LABEL | `SECURITY LABEL [FOR provider] ON object_type name IS 'label'` | `SecLabelStmt` |
| CHECKPOINT | `CHECKPOINT` | `CheckPointStmt` |
| LOAD | `LOAD 'filename'` | `LoadStmt` |
| REINDEX | `REINDEX [(options)] {INDEX\|TABLE\|SCHEMA\|DATABASE\|SYSTEM} [CONCURRENTLY] name` | `ReindexStmt` |
| SET CONSTRAINTS | `SET CONSTRAINTS {ALL\|name, ...} {DEFERRED\|IMMEDIATE}` | `ConstraintsSetStmt` |
| ALTER DEFAULT PRIVILEGES | `ALTER DEFAULT PRIVILEGES [FOR ROLE role] [IN SCHEMA schema] grant_or_revoke` | `AlterDefaultPrivilegesStmt` |

**New nodes:** `CommentStmt`, `SecLabelStmt`, `CheckPointStmt`, `LoadStmt`,
`ReindexStmt`, `ConstraintsSetStmt`, `AlterDefaultPrivilegesStmt`.
**New file:** `parse_utility2.go`.

---

## Step 10 — JSON/XML table functions and aggregates

| Feature | Description | PG production |
|---|---|---|
| JSON_SCALAR | `JSON_SCALAR(expr)` | `func_expr_common_subexpr` |
| JSON_SERIALIZE | `JSON_SERIALIZE(expr [FORMAT JSON] [RETURNING type])` | `func_expr_common_subexpr` |
| JSON_OBJECTAGG | `JSON_OBJECTAGG(key: value ... [RETURNING type])` — aggregate | `json_aggregate_func` |
| JSON_ARRAYAGG | `JSON_ARRAYAGG(expr [ORDER BY ...] [RETURNING type])` — aggregate | `json_aggregate_func` |
| JSON_TABLE | `JSON_TABLE(expr, path COLUMNS (col type PATH path, ...))` — row source in FROM | `json_table` |
| XMLTABLE | `XMLTABLE(xpath PASSING xml COLUMNS (col type PATH xpath, ...))` — row source in FROM | `xmltable` |

**New nodes:** `JsonTable`, `JsonTableColumn`, `XmlTable`.
**Modified:** `parsePrimaryExpr` (JSON_SCALAR, JSON_SERIALIZE),
`parseFuncCall` or new dispatch (JSON_OBJECTAGG, JSON_ARRAYAGG),
`parseTableRef` (JSON_TABLE, XMLTABLE as FROM-clause sources).

---

## Step 11 — Expression gaps: OVERLAPS, TABLESAMPLE, indirection

| Feature | Description | PG production |
|---|---|---|
| row OVERLAPS row | `(start1, end1) OVERLAPS (start2, end2)` | `a_expr` |
| TABLESAMPLE | `FROM table TABLESAMPLE method(args) [REPEATABLE (seed)]` | `tablesample_clause` |
| Indirection on subqueries | `(SELECT ...).field` / `(SELECT ...)[n]` | `c_expr` with `opt_indirection` |
| UNIQUE subquery | `UNIQUE [NULLS [NOT] DISTINCT] (SELECT ...)` — SQL standard | `a_expr` |
| Column-level GRANT | `GRANT UPDATE (col1, col2) ON t TO role` | `privilege_list` |

**Modified:** `parseExprPrec` (OVERLAPS), `parseTableRef` (TABLESAMPLE),
`parseParenExpr` (indirection on subqueries), `parsePrivilegeList` (column lists).

---

## Step 12 — DefineStmt and remaining CREATE forms

The PG `DefineStmt` production covers several CREATE forms that share a
`(name = value, ...)` definition syntax.

| Feature | Description | PG production |
|---|---|---|
| CREATE AGGREGATE | `CREATE [OR REPLACE] AGGREGATE name (args) (sfunc=..., stype=..., ...)` | `DefineStmt` |
| CREATE OPERATOR | `CREATE OPERATOR name (FUNCTION=func, LEFTARG=type, ...)` | `DefineStmt` |
| CREATE TYPE (shell) | `CREATE TYPE name` (no AS — shell type for C functions) | `DefineStmt` |
| CREATE TYPE (range) | `CREATE TYPE name AS RANGE (SUBTYPE=type, ...)` | `DefineStmt` |
| CREATE TEXT SEARCH PARSER/DICTIONARY/TEMPLATE/CONFIGURATION | `CREATE TEXT SEARCH {PARSER\|DICTIONARY\|TEMPLATE\|CONFIGURATION} name (opts)` | `DefineStmt` |
| CREATE COLLATION | `CREATE COLLATION name (LOCALE=...) / FROM existing` | `DefineStmt` |
| CREATE CAST | `CREATE CAST (source AS target) WITH FUNCTION func / WITHOUT FUNCTION / WITH INOUT [AS IMPLICIT\|ASSIGNMENT]` | `CreateCastStmt` |
| CREATE TRANSFORM | `CREATE TRANSFORM FOR type LANGUAGE lang (FROM SQL WITH FUNCTION f, TO SQL WITH FUNCTION f)` | `CreateTransformStmt` |
| CREATE ACCESS METHOD | `CREATE ACCESS METHOD name TYPE {INDEX\|TABLE} HANDLER func` | `CreateAmStmt` |
| CREATE OPERATOR CLASS | `CREATE OPERATOR CLASS name [DEFAULT] FOR TYPE type USING method [FAMILY family] AS (...)` | `CreateOpClassStmt` |
| CREATE OPERATOR FAMILY | `CREATE OPERATOR FAMILY name USING method` | `CreateOpFamilyStmt` |
| CREATE LANGUAGE | `CREATE [OR REPLACE] [TRUSTED] [PROCEDURAL] LANGUAGE name [HANDLER func] [INLINE func] [VALIDATOR func]` | `CreatePLangStmt` |
| CREATE CONVERSION | `CREATE [DEFAULT] CONVERSION name FOR 'src' TO 'dst' FROM func` | `CreateConversionStmt` |

**New nodes:** `DefineStmt`, `CreateCastStmt` (exists in nodes.go, wire up),
`CreateTransformStmt`, `CreateAmStmt`, `CreateOpClassStmt`,
`CreateOpFamilyStmt`, `CreatePLangStmt`, `CreateConversionStmt`.
**New file:** `parse_define.go`.

---

## Totals

| Metric | Value |
|---|---|
| Steps | 12 |
| New statement types | ~73 |
| New expression forms | ~10 |
| Currently implemented | 54 statements, ~85% of expression grammar |
| After all steps | 127 statements, ~100% of expression grammar |

## Dependencies

```
Step 1  (func call completions)     — no deps
Step 2  (cursors)                   — no deps
Step 3  (database DDL)              — no deps
Step 4  (ALTER dispatch)            — no deps
Step 5  (ALTER TABLE sub-cmds)      — no deps
Step 6  (DROP completions)          — no deps
Step 7  (FDW)                       — Step 6 (DROP forms)
Step 8  (materialized views)        — no deps
Step 9  (COMMENT, utility)          — no deps
Step 10 (JSON/XML tables)           — Step 1 (func call features)
Step 11 (expression gaps)           — no deps
Step 12 (DefineStmt, CREATE forms)  — no deps
```

Steps 1–6 and 8–9, 11–12 are independent and can be done in any order.
Step 7 benefits from Step 6 (shared DROP patterns). Step 10 benefits from
Step 1 (aggregate function infrastructure).
