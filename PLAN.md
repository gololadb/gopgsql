# PostgreSQL Parser — Implementation Plan

Gap analysis between the current hand-written parser and the full PostgreSQL
grammar defined in `src/backend/parser/gram.y` (526 productions, 127 statement
types). The current parser covers SELECT/INSERT/UPDATE/DELETE and ~60% of the
expression grammar. The remaining work is split into 15 incremental steps.

## Current state

Implemented:
- **Expressions**: arithmetic, comparison, boolean, user-defined operators,
  IS NULL/TRUE/FALSE/UNKNOWN/DISTINCT FROM, BETWEEN, IN, LIKE/ILIKE,
  SIMILAR TO, CASE, CAST, COALESCE, NULLIF, GREATEST/LEAST, ARRAY,
  ROW, EXISTS, function calls with DISTINCT/ORDER BY/FILTER/OVER,
  column refs, positional parameters, typecasts, COLLATE, AT TIME ZONE,
  subscripts, unary +/-/NOT.
- **Statements**: SELECT (full clause set including joins, CTEs, set ops,
  locking), INSERT (ON CONFLICT, RETURNING), UPDATE (FROM, RETURNING),
  DELETE (USING, RETURNING), VALUES, TABLE, WITH.
- **Type names**: common built-in types, user-defined types, array bounds,
  type modifiers.

---

## Step 1 — SQL syntax functions

Special-syntax function calls that desugar into `FuncCall` nodes.

| Form | Desugars to |
|---|---|
| `EXTRACT(field FROM expr)` | `pg_catalog.extract(field, expr)` |
| `POSITION(expr IN expr)` | `pg_catalog.position(b, a)` |
| `SUBSTRING(expr FROM expr FOR expr)` | `pg_catalog.substring(a, b, c)` |
| `OVERLAY(expr PLACING expr FROM expr FOR expr)` | `pg_catalog.overlay(...)` |
| `TRIM([LEADING\|TRAILING\|BOTH] expr FROM expr)` | `btrim/ltrim/rtrim(...)` |
| `TREAT(expr AS type)` | `typename(expr)` |
| `NORMALIZE(expr [, form])` | `pg_catalog.normalize(...)` |
| `COLLATION FOR (expr)` | `pg_catalog.pg_collation_for(expr)` |

**~15 productions. No new node types.**

Dependencies: none.

---

## Step 2 — SQL value functions and GROUPING

Parameterless keyword expressions that produce values.

| Keyword | Node |
|---|---|
| `CURRENT_DATE` | `SQLValueFunction(SVFOP_CURRENT_DATE)` |
| `CURRENT_TIME[(n)]` | `SQLValueFunction(SVFOP_CURRENT_TIME[_N])` |
| `CURRENT_TIMESTAMP[(n)]` | `SQLValueFunction(SVFOP_CURRENT_TIMESTAMP[_N])` |
| `LOCALTIME[(n)]` | `SQLValueFunction(SVFOP_LOCALTIME[_N])` |
| `LOCALTIMESTAMP[(n)]` | `SQLValueFunction(SVFOP_LOCALTIMESTAMP[_N])` |
| `CURRENT_ROLE` | `SQLValueFunction(SVFOP_CURRENT_ROLE)` |
| `CURRENT_USER` | `SQLValueFunction(SVFOP_CURRENT_USER)` |
| `SESSION_USER` | `SQLValueFunction(SVFOP_SESSION_USER)` |
| `SYSTEM_USER` | `FuncCall(system_user)` |
| `USER` | `SQLValueFunction(SVFOP_USER)` |
| `CURRENT_CATALOG` | `SQLValueFunction(SVFOP_CURRENT_CATALOG)` |
| `CURRENT_SCHEMA` | `SQLValueFunction(SVFOP_CURRENT_SCHEMA)` |
| `GROUPING(expr_list)` | `GroupingFunc` |
| `DEFAULT` (in INSERT/UPDATE) | `SetToDefault` |

**~15 productions. New node types: `SQLValueFunction`, `GroupingFunc`, `SetToDefault`.**

Dependencies: none.

---

## Step 3 — Operator forms and misc expression gaps

| Feature | Description |
|---|---|
| `expr op ANY/ALL (subquery)` | `SubLink` with `ANY_SUBLINK`/`ALL_SUBLINK` |
| `expr op ANY/ALL (expr)` | `ScalarArrayOpExpr` equivalent |
| `OPERATOR(schema.op)` | Qualified operator syntax (`qual_Op`) |
| `IS [NOT] DOCUMENT` | XML document test |
| `IS [NOT] [form] NORMALIZED` | Unicode normalization test |
| `AT LOCAL` | Shorthand for `AT TIME ZONE 'local'` |
| `LIKE ... ESCAPE expr` | Escape clause on LIKE/ILIKE/SIMILAR TO |
| `\|` as binary operator | Currently treated as self token; should be `precOp` |
| Prefix `OPERATOR(schema.op) expr` | Prefix qualified operator |

**~20 productions. New node types: `XmlSerialize` (for IS DOCUMENT context), extend `A_Expr`.**

Dependencies: none.

---

## Step 4 — Complete type name grammar

The current `parseTypeName` handles common cases but is missing:

- `BIT [(n)]` / `BIT VARYING [(n)]`
- `INTERVAL` with field qualifiers (`YEAR`, `MONTH`, `DAY`, `HOUR`, `MINUTE`,
  `SECOND`, `YEAR TO MONTH`, `DAY TO HOUR`, `DAY TO MINUTE`, `DAY TO SECOND`,
  `HOUR TO MINUTE`, `HOUR TO SECOND`, `MINUTE TO SECOND`)
- `GenericType` with qualified names and type modifiers
- `ConstTypename` for typed literal syntax (`typename 'string'`)
- `%TYPE` suffix
- Full `opt_array_bounds` (multi-dimensional `[][]`)

**~30 productions. No new node types (extends `TypeName`).**

Dependencies: none. Required by steps 8, 9, 12, 13.

---

## Step 5 — Window frame clauses

Currently `parseWindowSpec` parses `PARTITION BY` and `ORDER BY` but skips
frame specifications.

| Feature | Description |
|---|---|
| `ROWS\|RANGE\|GROUPS frame_extent` | Frame mode |
| `frame_bound` | `UNBOUNDED PRECEDING/FOLLOWING`, `CURRENT ROW`, `expr PRECEDING/FOLLOWING` |
| `BETWEEN frame_bound AND frame_bound` | Frame extent |
| `EXCLUDE CURRENT ROW\|GROUP\|TIES\|NO OTHERS` | Exclusion clause |
| `WITHIN GROUP (ORDER BY ...)` | Ordered-set aggregate |
| `RESPECT NULLS` / `IGNORE NULLS` | Null treatment |

**~20 productions. No new node types (extends `WindowDef` with frame option constants).**

Dependencies: none.

---

## Step 6 — GROUP BY extensions

| Feature | Node |
|---|---|
| `GROUPING SETS ((a, b), (a), ())` | `GroupingSet(GROUPING_SET_SETS)` |
| `CUBE(a, b)` | `GroupingSet(GROUPING_SET_CUBE)` |
| `ROLLUP(a, b)` | `GroupingSet(GROUPING_SET_ROLLUP)` |
| `()` (empty grouping set) | `GroupingSet(GROUPING_SET_EMPTY)` |

**~10 productions. New node type: `GroupingSet`.**

Dependencies: none.

---

## Step 7 — MERGE statement

```sql
MERGE INTO target USING source ON condition
  WHEN MATCHED [AND condition] THEN UPDATE SET ...
  WHEN MATCHED [AND condition] THEN DELETE
  WHEN NOT MATCHED [BY TARGET] [AND condition] THEN INSERT (cols) VALUES (...)
  WHEN NOT MATCHED BY SOURCE [AND condition] THEN UPDATE SET ...
  WHEN NOT MATCHED BY SOURCE [AND condition] THEN DELETE
  WHEN ... THEN DO NOTHING
  RETURNING ...
```

**~15 productions. New node types: `MergeStmt`, `MergeWhenClause`.**

Dependencies: none.

---

## Step 8 — CREATE TABLE / CREATE TABLE AS

The most complex DDL statement.

| Feature | Description |
|---|---|
| `CREATE [TEMP\|UNLOGGED] TABLE [IF NOT EXISTS] name` | Table creation |
| Column definitions | name, type, DEFAULT, NOT NULL, NULL, CHECK, UNIQUE, PRIMARY KEY, REFERENCES, GENERATED, COLLATE, COMPRESSION, STORAGE |
| Table constraints | CHECK, UNIQUE, PRIMARY KEY, FOREIGN KEY (with match/actions), EXCLUDE |
| `INHERITS (parent, ...)` | Inheritance |
| `PARTITION BY RANGE\|LIST\|HASH (cols)` | Partitioning |
| `USING method` | Table access method |
| `WITH (storage_params)` / `WITHOUT OIDS` | Storage options |
| `ON COMMIT PRESERVE ROWS\|DELETE ROWS\|DROP` | Temp table behavior |
| `TABLESPACE name` | Tablespace |
| `LIKE source [INCLUDING ...]` | Copy structure from another table |
| `CREATE TABLE AS select` | CTAS |
| `SELECT INTO` | Rewritten to CTAS |

**~80 productions. New node types: `CreateStmt`, `Constraint` (full), `ColumnDef` (full),
`PartitionSpec`, `PartitionElem`, `CreateTableAsStmt`, `IntoClause` (full).**

Dependencies: step 4 (complete type names).

---

## Step 9 — CREATE INDEX / ALTER TABLE

| Feature | Description |
|---|---|
| `CREATE [UNIQUE] INDEX [CONCURRENTLY] [IF NOT EXISTS] name ON table` | Index creation |
| `USING method (columns)` | Index method and columns |
| `INCLUDE (columns)` | Covering index |
| `WITH (storage_params)` | Index options |
| `WHERE predicate` | Partial index |
| `ALTER TABLE ... ADD COLUMN` | Add column |
| `ALTER TABLE ... DROP COLUMN` | Drop column |
| `ALTER TABLE ... ALTER COLUMN ... SET/DROP DEFAULT/NOT NULL/TYPE/STATS` | Alter column |
| `ALTER TABLE ... ADD/DROP CONSTRAINT` | Constraint management |
| `ALTER TABLE ... RENAME [COLUMN\|CONSTRAINT\|TO]` | Rename |
| `ALTER TABLE ... SET SCHEMA` | Move to schema |
| `ALTER TABLE ... ENABLE/DISABLE TRIGGER/RULE` | Trigger/rule control |
| `ALTER TABLE ... ATTACH/DETACH PARTITION` | Partition management |
| `ALTER TABLE ... SET/RESET (params)` | Storage params |
| `ALTER TABLE ... OWNER TO` | Ownership |
| `DROP TABLE/INDEX [IF EXISTS] [CASCADE\|RESTRICT]` | Drop objects |
| `TRUNCATE [TABLE] name [CASCADE\|RESTRICT]` | Truncate |

**~60 productions. New node types: `IndexStmt`, `IndexElem`, `AlterTableStmt`,
`AlterTableCmd`, `RenameStmt`, `DropStmt` (generic).**

Dependencies: step 8 (shares ColumnDef, Constraint).

---

## Step 10 — CREATE VIEW / EXPLAIN / COPY

| Feature | Description |
|---|---|
| `CREATE [OR REPLACE] [TEMP] [RECURSIVE] VIEW name [(cols)] AS select [WITH CHECK OPTION]` | View creation |
| `CREATE MATERIALIZED VIEW [IF NOT EXISTS] name AS select [WITH [NO] DATA]` | Materialized view |
| `REFRESH MATERIALIZED VIEW [CONCURRENTLY] name [WITH [NO] DATA]` | Refresh |
| `EXPLAIN [ANALYZE] [VERBOSE] stmt` | Query plan |
| `EXPLAIN (option, ...) stmt` | Extended explain |
| `COPY table [(cols)] FROM/TO 'file'\|STDIN\|STDOUT [WITH (options)]` | Bulk data |

**~40 productions. New node types: `ViewStmt`, `CreateTableAsStmt` (for matview),
`RefreshMatViewStmt`, `ExplainStmt`, `CopyStmt`.**

Dependencies: none.

---

## Step 11 — Transaction control / Session / Utility

| Feature | Description |
|---|---|
| `BEGIN\|START TRANSACTION [isolation, read only]` | Transaction start |
| `COMMIT\|END` | Transaction commit |
| `ROLLBACK [TO SAVEPOINT name]` | Transaction rollback |
| `SAVEPOINT name` / `RELEASE [SAVEPOINT] name` | Savepoints |
| `SET name = value` / `SET TIME ZONE` / `SET SESSION\|LOCAL` | Variable set |
| `RESET name\|ALL` | Variable reset |
| `SHOW name\|ALL` | Variable show |
| `PREPARE name [(types)] AS stmt` | Prepared statement |
| `EXECUTE name [(params)]` | Execute prepared |
| `DEALLOCATE [PREPARE] name\|ALL` | Deallocate |
| `DECLARE cursor [options] FOR select` | Cursor declaration |
| `FETCH [direction] [FROM\|IN] cursor` | Cursor fetch |
| `CLOSE cursor\|ALL` | Cursor close |
| `LISTEN channel` / `NOTIFY channel [, payload]` / `UNLISTEN channel\|*` | Pub/sub |
| `LOCK [TABLE] name [IN mode MODE] [NOWAIT]` | Table lock |
| `DISCARD ALL\|PLANS\|SEQUENCES\|TEMP` | Session cleanup |
| `CHECKPOINT` | WAL checkpoint |
| `LOAD 'filename'` | Load shared library |

**~50 productions. New node types: `TransactionStmt`, `VariableSetStmt`,
`VariableShowStmt`, `PrepareStmt`, `ExecuteStmt`, `DeallocateStmt`,
`DeclareCursorStmt`, `FetchStmt`, `ClosePortalStmt`, `ListenStmt`,
`NotifyStmt`, `UnlistenStmt`, `LockStmt`, `DiscardStmt`, `CheckPointStmt`,
`LoadStmt`.**

Dependencies: none.

---

## Step 12 — CREATE FUNCTION / DO / CALL / CREATE TRIGGER / CREATE RULE

| Feature | Description |
|---|---|
| `CREATE [OR REPLACE] FUNCTION name (args) RETURNS type AS body LANGUAGE lang [options]` | Function creation |
| `CREATE [OR REPLACE] PROCEDURE name (args) AS body LANGUAGE lang` | Procedure creation |
| Function parameters | IN/OUT/INOUT/VARIADIC, name, type, DEFAULT |
| Function options | IMMUTABLE/STABLE/VOLATILE, STRICT, SECURITY DEFINER, COST, ROWS, SET, PARALLEL |
| `ALTER FUNCTION/PROCEDURE ... SET/RESET/RENAME/OWNER` | Alter function |
| `DROP FUNCTION/PROCEDURE [IF EXISTS] name(argtypes) [CASCADE]` | Drop function |
| `DROP AGGREGATE name(argtypes)` | Drop aggregate |
| `DROP OPERATOR op(lefttype, righttype)` | Drop operator |
| `DO $$ body $$ [LANGUAGE lang]` | Anonymous code block |
| `CALL procedure(args)` | Procedure call |
| `CREATE TRIGGER name BEFORE/AFTER/INSTEAD OF event ON table [FOR EACH ROW/STATEMENT] [WHEN (condition)] EXECUTE FUNCTION func()` | Trigger creation |
| `CREATE [OR REPLACE] RULE name AS ON event TO table [WHERE condition] DO [ALSO\|INSTEAD] (actions)` | Rule creation |

**~60 productions. New node types: `CreateFunctionStmt`, `FunctionParameter`,
`AlterFunctionStmt`, `DoStmt`, `CallStmt`, `CreateTrigStmt`, `RuleStmt`,
`RemoveFuncStmt`, `RemoveAggrStmt`, `RemoveOperStmt`.**

Dependencies: step 4 (type names for parameter types and return types).

---

## Step 13 — GRANT / REVOKE / Roles / Schemas / Domains / Types

| Feature | Description |
|---|---|
| `GRANT privileges ON object TO role [WITH GRANT OPTION]` | Grant |
| `REVOKE privileges ON object FROM role [CASCADE]` | Revoke |
| `GRANT role TO role [WITH ADMIN OPTION]` | Role membership |
| `CREATE/ALTER/DROP ROLE name [WITH options]` | Role management |
| `CREATE/ALTER/DROP USER` / `CREATE/ALTER/DROP GROUP` | User/group aliases |
| `CREATE SCHEMA [IF NOT EXISTS] name [AUTHORIZATION role] [elements]` | Schema creation |
| `CREATE DOMAIN name AS type [DEFAULT expr] [constraints]` | Domain creation |
| `ALTER DOMAIN ... ADD/DROP CONSTRAINT\|SET DEFAULT\|SET NOT NULL\|DROP NOT NULL` | Alter domain |
| `CREATE TYPE name AS (attributes)` | Composite type |
| `CREATE TYPE name AS ENUM ('val', ...)` | Enum type |
| `CREATE TYPE name AS RANGE (SUBTYPE = type, ...)` | Range type |
| `ALTER TYPE ... ADD VALUE\|RENAME VALUE\|RENAME TO\|SET SCHEMA\|OWNER TO` | Alter type |
| `COMMENT ON object IS 'text'` | Comments |
| `SECURITY LABEL ON object IS 'label'` | Security labels |
| `REASSIGN OWNED BY role TO role` | Ownership transfer |
| `DROP OWNED BY role [CASCADE]` | Drop owned objects |
| `ALTER DEFAULT PRIVILEGES [FOR ROLE role] [IN SCHEMA schema] grant/revoke` | Default privileges |

**~70 productions. New node types: `GrantStmt`, `GrantRoleStmt`, `CreateRoleStmt`,
`AlterRoleStmt`, `DropRoleStmt`, `CreateSchemaStmt`, `CreateDomainStmt`,
`AlterDomainStmt`, `CompositeTypeStmt`, `CreateEnumStmt`, `CreateRangeStmt`,
`AlterEnumStmt`, `CommentStmt`, `SecLabelStmt`, `ReassignOwnedStmt`,
`DropOwnedStmt`, `AlterDefaultPrivilegesStmt`.**

Dependencies: step 4 (type names for domain/type definitions).

---

## Step 14 — Sequences / Extensions / FDW / Pub-Sub / misc DDL

| Feature | Description |
|---|---|
| `CREATE/ALTER/DROP SEQUENCE name [options]` | Sequence management |
| `CREATE/ALTER/DROP EXTENSION name [options]` | Extension management |
| `CREATE/ALTER/DROP SERVER name [options]` | Foreign server |
| `CREATE/ALTER/DROP FOREIGN TABLE name (cols) SERVER name [OPTIONS]` | Foreign table |
| `CREATE/ALTER/DROP FOREIGN DATA WRAPPER name [options]` | FDW |
| `CREATE/ALTER/DROP USER MAPPING FOR role SERVER name [OPTIONS]` | User mapping |
| `IMPORT FOREIGN SCHEMA remote INTO local [LIMIT TO/EXCEPT (tables)]` | Schema import |
| `CREATE/ALTER/DROP PUBLICATION name [options]` | Logical replication pub |
| `CREATE/ALTER/DROP SUBSCRIPTION name [options]` | Logical replication sub |
| `CREATE/ALTER/DROP TABLESPACE name [options]` | Tablespace |
| `CREATE/DROP CAST (source AS target) WITH FUNCTION/INOUT/WITHOUT FUNCTION` | Cast |
| `CREATE/DROP TRANSFORM FOR type LANGUAGE lang (FROM SQL WITH FUNCTION, TO SQL WITH FUNCTION)` | Transform |
| `CREATE/ALTER/DROP STATISTICS name ON (exprs) FROM table` | Extended statistics |
| `CREATE/ALTER/DROP POLICY name ON table [options]` | Row-level security |
| `CREATE/ALTER/DROP EVENT TRIGGER name ON event [WHEN filter] EXECUTE FUNCTION func()` | Event trigger |
| `CREATE/DROP CONVERSION name FOR 'src' TO 'dst' FROM func` | Encoding conversion |
| `CREATE/DROP ACCESS METHOD name TYPE type HANDLER func` | Access method |
| `CREATE/ALTER/DROP OPERATOR name (options)` | Operator |
| `CREATE/ALTER/DROP OPERATOR CLASS/FAMILY name [options]` | Operator class/family |
| `CREATE/DROP [TRUSTED] [PROCEDURAL] LANGUAGE name [HANDLER func]` | Language |
| `ANALYZE [VERBOSE] [table [(cols)]]` | Statistics collection |
| `VACUUM [FULL\|FREEZE\|VERBOSE\|ANALYZE] [table [(cols)]]` | Maintenance |
| `REINDEX [INDEX\|TABLE\|SCHEMA\|DATABASE\|SYSTEM] name` | Index rebuild |

**~80 productions. New node types: `CreateSeqStmt`, `AlterSeqStmt`,
`CreateExtensionStmt`, `AlterExtensionStmt`, `AlterExtensionContentsStmt`,
`CreateFdwStmt`, `AlterFdwStmt`, `CreateForeignServerStmt`,
`AlterForeignServerStmt`, `CreateForeignTableStmt`, `CreateUserMappingStmt`,
`AlterUserMappingStmt`, `DropUserMappingStmt`, `ImportForeignSchemaStmt`,
`CreatePublicationStmt`, `AlterPublicationStmt`, `CreateSubscriptionStmt`,
`AlterSubscriptionStmt`, `DropSubscriptionStmt`, `CreateTableSpaceStmt`,
`DropTableSpaceStmt`, `AlterTableSpaceStmt`, `CreateCastStmt`, `DropCastStmt`,
`CreateTransformStmt`, `DropTransformStmt`, `CreateStatsStmt`,
`AlterStatsStmt`, `CreatePolicyStmt`, `AlterPolicyStmt`,
`CreateEventTrigStmt`, `AlterEventTrigStmt`, `CreateConversionStmt`,
`CreateAmStmt`, `CreateOpClassStmt`, `CreateOpFamilyStmt`,
`AlterOpFamilyStmt`, `AlterOperatorStmt`, `CreatePLangStmt`,
`VacuumStmt`, `AnalyzeStmt`, `ReindexStmt`.**

Dependencies: none.

---

## Step 15 — XML and JSON expression syntax

| Feature | Description |
|---|---|
| `XMLCONCAT(expr, ...)` | XML concatenation |
| `XMLELEMENT(NAME name [, XMLATTRIBUTES(expr AS name, ...)] [, content])` | XML element |
| `XMLFOREST(expr AS name, ...)` | XML forest |
| `XMLPARSE(DOCUMENT\|CONTENT expr [STRIP WHITESPACE])` | XML parse |
| `XMLPI(NAME name [, expr])` | XML processing instruction |
| `XMLROOT(xml, VERSION expr\|NO VALUE [, STANDALONE YES\|NO\|NO VALUE])` | XML root |
| `XMLSERIALIZE(DOCUMENT\|CONTENT expr AS type [INDENT])` | XML serialize |
| `XMLEXISTS(expr PASSING [BY REF] expr [BY REF])` | XML exists |
| `XMLTABLE(...)` | XML table |
| `IS JSON [VALUE\|ARRAY\|OBJECT\|SCALAR] [WITH\|WITHOUT UNIQUE KEYS]` | JSON type test |
| `JSON_OBJECT(key: value, ... [RETURNING type])` | JSON object constructor |
| `JSON_ARRAY(expr, ... [RETURNING type])` | JSON array constructor |
| `JSON_OBJECTAGG(key: value ... [RETURNING type])` | JSON object aggregate |
| `JSON_ARRAYAGG(expr ... [ORDER BY ...] [RETURNING type])` | JSON array aggregate |
| `JSON_QUERY(expr, path ... [RETURNING type] [behavior])` | JSON query |
| `JSON_VALUE(expr, path ... [RETURNING type] [behavior])` | JSON value extraction |
| `JSON_EXISTS(expr, path ... [behavior])` | JSON path exists |
| `JSON_TABLE(expr, path COLUMNS (...))` | JSON table |
| `json_output_clause`, `json_format_clause`, `json_behavior` | Supporting clauses |

**~80 productions. New node types: `XmlExpr`, `XmlSerialize`, `JsonConstructorExpr`,
`JsonIsPredicate`, `JsonValueExpr`, `JsonOutput`, `JsonKeyValue`,
`JsonObjectAgg`, `JsonArrayAgg`, `JsonTable`, `JsonTableColumn`.**

Dependencies: step 1 (shares SQL syntax function infrastructure).

---

## Dependency graph

```
Steps 1, 2, 3, 5, 6, 7, 10, 11, 14  (independent — can be done in any order)
         │
         ▼
       Step 4  (complete type names)
       ┌─┼──────┐
       ▼ ▼      ▼
       8 12     13   (CREATE TABLE, CREATE FUNCTION, GRANT/types)
       │
       ▼
       9              (ALTER TABLE, CREATE INDEX)
       
Step 15 depends on step 1
```

## Totals

| Metric | Value |
|---|---|
| Remaining productions | ~645 |
| New node types | ~65 |
| Steps | 15 |
| Currently implemented | ~4 statement types, ~60% of expression grammar |
| After all steps | 127 statement types, 100% of expression grammar |
