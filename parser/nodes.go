package parser

// Node is the interface implemented by all AST nodes.
type Node interface {
	Pos() int // byte offset in source, or -1 if unknown
	node()
}

// Stmt is the interface for statement nodes.
type Stmt interface {
	Node
	stmt()
}

// Expr is the interface for expression nodes.
type Expr interface {
	Node
	expr()
}

// baseNode provides a default Pos implementation.
type baseNode struct {
	Location int // byte offset, -1 if unknown
}

func (n *baseNode) Pos() int { return n.Location }
func (*baseNode) node()      {}

type baseStmt struct{ baseNode }

func (*baseStmt) stmt() {}

type baseExpr struct{ baseNode }

func (*baseExpr) expr() {}

// ---------------------------------------------------------------------------
// Top-level
// ---------------------------------------------------------------------------

// RawStmt wraps a statement with its source location range.
type RawStmt struct {
	baseNode
	Stmt     Stmt
	StmtEnd  int // byte offset of the trailing semicolon, or -1
}

// ---------------------------------------------------------------------------
// Expressions  (mirrors parsenodes.h / primnodes.h)
// ---------------------------------------------------------------------------

// A_Expr represents an operator expression (binary, unary, or special forms
// like IN, LIKE, BETWEEN, etc.).
type A_Expr struct {
	baseExpr
	Kind  A_Expr_Kind
	Name  []string // operator name (e.g. ["+"], ["pg_catalog","="])
	Lexpr Expr     // left operand (nil for prefix ops)
	Rexpr Expr     // right operand (nil for postfix ops)
}

type A_Expr_Kind int

const (
	AEXPR_OP             A_Expr_Kind = iota // normal operator
	AEXPR_OP_ANY                            // scalar op ANY (array)
	AEXPR_OP_ALL                            // scalar op ALL (array)
	AEXPR_DISTINCT                          // IS DISTINCT FROM
	AEXPR_NOT_DISTINCT                      // IS NOT DISTINCT FROM
	AEXPR_IN                                // [NOT] IN
	AEXPR_LIKE                              // [NOT] LIKE
	AEXPR_ILIKE                             // [NOT] ILIKE
	AEXPR_SIMILAR                           // [NOT] SIMILAR TO
	AEXPR_BETWEEN                           // BETWEEN
	AEXPR_NOT_BETWEEN                       // NOT BETWEEN
	AEXPR_BETWEEN_SYM                       // BETWEEN SYMMETRIC
	AEXPR_NOT_BETWEEN_SYM                   // NOT BETWEEN SYMMETRIC
)

// BoolExpr represents AND / OR / NOT.
type BoolExpr struct {
	baseExpr
	Op   BoolExprType
	Args []Expr
}

type BoolExprType int

const (
	AND_EXPR BoolExprType = iota
	OR_EXPR
	NOT_EXPR
)

// ColumnRef represents a column reference: colname or schema.table.col etc.
type ColumnRef struct {
	baseExpr
	Fields []Node // list of *String or *A_Star
}

// A_Star represents '*' in a column reference or target list.
type A_Star struct{ baseNode }

func (*A_Star) node() {}

// ParamRef represents a positional parameter $N.
type ParamRef struct {
	baseExpr
	Number int
}

// A_Const represents a literal constant.
type A_Const struct {
	baseExpr
	Val Value
}

// Value is a tagged union for constant values.
type Value struct {
	Type ValType
	Ival int64
	Str  string
	Bool bool
}

type ValType int

const (
	ValInt    ValType = iota // integer
	ValFloat                // numeric string (stored as string)
	ValStr                  // string
	ValBitStr               // bit string
	ValNull                 // NULL
	ValBool                 // boolean (true/false)
)

// TypeCast represents expr::type.
type TypeCast struct {
	baseExpr
	Arg      Expr
	TypeName *TypeName
}

// TypeName represents a type reference.
type TypeName struct {
	baseNode
	Names       []string // qualified name (e.g. ["pg_catalog","int4"])
	TypeOid     uint32   // 0 until analysis
	Setof       bool
	PctType     bool     // %TYPE
	Typmods     []Expr   // type modifiers (e.g. precision)
	ArrayBounds []int    // -1 for unspecified bound
}

func (*TypeName) node() {}

// Interval field constants (matching PostgreSQL's datetime.h).
// Used in INTERVAL_MASK(field) = 1 << field.
const (
	IntervalFieldMonth  = 1
	IntervalFieldYear   = 2
	IntervalFieldDay    = 3
	IntervalFieldHour   = 10
	IntervalFieldMinute = 11
	IntervalFieldSecond = 12
)

// IntervalMask returns the bitmask for an interval field.
func IntervalMask(field int) int64 { return 1 << field }

// IntervalFullRange is the bitmask for all interval fields.
const IntervalFullRange = 0x7FFF

// FuncCall represents a function invocation.
type FuncCall struct {
	baseExpr
	Funcname      []string // qualified function name
	Args          []Expr
	AggOrder      []*SortBy
	AggFilter     Expr
	AggWithinGroup []*SortBy // WITHIN GROUP (ORDER BY ...)
	Over          *WindowDef // OVER clause, if any
	AggStar       bool       // true for count(*)
	AggDistinct   bool       // true for DISTINCT in aggregate
	FuncVariadic  bool
	FuncFormat    CoercionForm
	NullTreatment NullTreatment // RESPECT NULLS / IGNORE NULLS
}

// NullTreatment for window/aggregate functions.
type NullTreatment int

const (
	NULL_TREATMENT_DEFAULT NullTreatment = iota
	NULL_TREATMENT_RESPECT               // RESPECT NULLS
	NULL_TREATMENT_IGNORE                // IGNORE NULLS
)

// NamedArgExpr represents a named argument in a function call: name := expr or name => expr.
type NamedArgExpr struct {
	baseExpr
	Arg       Expr
	Name      string
	Argnumber int // resolved during analysis, -1 initially
}

// MergeActionExpr represents MERGE_ACTION() in MERGE WHEN clauses.
type MergeActionExpr struct {
	baseExpr
}

type CoercionForm int

const (
	COERCE_EXPLICIT_CALL CoercionForm = iota
	COERCE_EXPLICIT_CAST
	COERCE_IMPLICIT_CAST
	COERCE_SQL_SYNTAX
)

// SubLink represents a subquery expression (EXISTS, IN, scalar subquery, etc.).
type SubLink struct {
	baseExpr
	SubLinkType SubLinkType
	Testexpr    Expr   // outer expression for comparison
	OperName    []string
	Subselect   Stmt   // the sub-SELECT
}

type SubLinkType int

const (
	EXISTS_SUBLINK SubLinkType = iota
	ALL_SUBLINK
	ANY_SUBLINK
	ROWCOMPARE_SUBLINK
	EXPR_SUBLINK
	MULTIEXPR_SUBLINK
	ARRAY_SUBLINK
	CTE_SUBLINK
)

// NullTest represents IS [NOT] NULL.
type NullTest struct {
	baseExpr
	Arg          Expr
	NullTestType NullTestType
}

type NullTestType int

const (
	IS_NULL NullTestType = iota
	IS_NOT_NULL
)

// BooleanTest represents IS [NOT] TRUE/FALSE/UNKNOWN.
type BooleanTest struct {
	baseExpr
	Arg           Expr
	BooltestType  BoolTestType
}

type BoolTestType int

const (
	IS_TRUE BoolTestType = iota
	IS_NOT_TRUE
	IS_FALSE
	IS_NOT_FALSE
	IS_UNKNOWN
	IS_NOT_UNKNOWN
)

// CaseExpr represents a CASE expression.
type CaseExpr struct {
	baseExpr
	Arg     Expr        // implicit equality comparison argument, if any
	Args    []*CaseWhen // list of WHEN clauses
	Defresult Expr      // ELSE clause
}

// CaseWhen represents a single WHEN clause.
type CaseWhen struct {
	baseExpr
	Expr   Expr // condition
	Result Expr // result
}

// CoalesceExpr represents COALESCE(args...).
type CoalesceExpr struct {
	baseExpr
	Args []Expr
}

// MinMaxExpr represents GREATEST/LEAST.
type MinMaxExpr struct {
	baseExpr
	Op   MinMaxOp
	Args []Expr
}

type MinMaxOp int

const (
	IS_GREATEST MinMaxOp = iota
	IS_LEAST
)

// NullIfExpr represents NULLIF(a, b).
type NullIfExpr struct {
	baseExpr
	Args []Expr // exactly 2
}

// A_ArrayExpr represents ARRAY[...].
type A_ArrayExpr struct {
	baseExpr
	Elements []Expr
}

// RowExpr represents ROW(a, b, c) or (a, b, c).
type RowExpr struct {
	baseExpr
	Args []Expr
}

// A_Indirection represents subscripting/field selection: expr.field or expr[sub].
type A_Indirection struct {
	baseExpr
	Arg         Expr
	Indirection []Node // list of *String (field) or *A_Indices (subscript)
}

// A_Indices represents array subscript [lower:upper] or [idx].
type A_Indices struct {
	baseNode
	IsSlice bool
	Lidx    Expr // lower bound (nil if not slice)
	Uidx    Expr // upper bound or single index
}

func (*A_Indices) node() {}

// String is a simple string node used in name lists.
type String struct {
	baseNode
	Str string
}

func (*String) node() {}

// CollateClause represents COLLATE "name".
type CollateClause struct {
	baseExpr
	Arg      Expr
	Collname []string
}

// ---------------------------------------------------------------------------
// SELECT statement
// ---------------------------------------------------------------------------

type SelectStmt struct {
	baseStmt
	// Leaf fields
	DistinctClause []Expr       // nil = no DISTINCT; empty = DISTINCT; non-empty = DISTINCT ON
	TargetList     []*ResTarget
	IntoClause     *IntoClause
	FromClause     []Node       // list of table refs (RangeVar, JoinExpr, RangeSubselect, etc.)
	WhereClause    Expr
	GroupClause    []Expr
	GroupDistinct  bool
	HavingClause   Expr
	WindowClause   []*WindowDef
	ValuesLists    [][]Expr     // for VALUES (...)

	// Shared fields
	SortClause     []*SortBy
	LimitOffset    Expr
	LimitCount     Expr
	LockingClause  []*LockingClause
	WithClause     *WithClause

	// Set operation fields
	Op   SetOperation
	All  bool
	Larg *SelectStmt
	Rarg *SelectStmt
}

type SetOperation int

const (
	SETOP_NONE SetOperation = iota
	SETOP_UNION
	SETOP_INTERSECT
	SETOP_EXCEPT
)

// ResTarget represents a result target in a SELECT list or an assignment target.
type ResTarget struct {
	baseNode
	Name        string // column label (AS name), or empty
	Indirection []Node // subscripts/field selection for assignment
	Val         Expr   // the value expression
}

func (*ResTarget) node() {}

// SortBy represents an ORDER BY item.
type SortBy struct {
	baseNode
	Node       Expr
	SortbyDir  SortByDir
	SortbyNulls SortByNulls
	UseOp      []string // USING operator, if any
}

func (*SortBy) node() {}

type SortByDir int

const (
	SORTBY_DEFAULT SortByDir = iota
	SORTBY_ASC
	SORTBY_DESC
	SORTBY_USING
)

type SortByNulls int

const (
	SORTBY_NULLS_DEFAULT SortByNulls = iota
	SORTBY_NULLS_FIRST
	SORTBY_NULLS_LAST
)

// WindowDef represents a WINDOW clause or OVER clause.
type WindowDef struct {
	baseNode
	Name            string
	Refname         string // references an existing window by name
	PartitionClause []Expr
	OrderClause     []*SortBy
	FrameOptions    int
	StartOffset     Expr
	EndOffset       Expr
}

func (*WindowDef) node() {}

// Frame option flags (matching PostgreSQL's parsenodes.h).
const (
	FRAMEOPTION_NONDEFAULT                = 0x00001
	FRAMEOPTION_RANGE                     = 0x00002
	FRAMEOPTION_ROWS                      = 0x00004
	FRAMEOPTION_GROUPS                    = 0x00008
	FRAMEOPTION_BETWEEN                   = 0x00010
	FRAMEOPTION_START_UNBOUNDED_PRECEDING = 0x00020
	FRAMEOPTION_END_UNBOUNDED_PRECEDING   = 0x00040
	FRAMEOPTION_START_UNBOUNDED_FOLLOWING = 0x00080
	FRAMEOPTION_END_UNBOUNDED_FOLLOWING   = 0x00100
	FRAMEOPTION_START_CURRENT_ROW         = 0x00200
	FRAMEOPTION_END_CURRENT_ROW           = 0x00400
	FRAMEOPTION_START_OFFSET_PRECEDING    = 0x00800
	FRAMEOPTION_END_OFFSET_PRECEDING      = 0x01000
	FRAMEOPTION_START_OFFSET_FOLLOWING    = 0x02000
	FRAMEOPTION_END_OFFSET_FOLLOWING      = 0x04000
	FRAMEOPTION_EXCLUDE_CURRENT_ROW       = 0x08000
	FRAMEOPTION_EXCLUDE_GROUP             = 0x10000
	FRAMEOPTION_EXCLUDE_TIES              = 0x20000

	FRAMEOPTION_DEFAULTS = FRAMEOPTION_RANGE |
		FRAMEOPTION_START_UNBOUNDED_PRECEDING |
		FRAMEOPTION_END_CURRENT_ROW
)

// LockingClause represents FOR UPDATE/SHARE.
type LockingClause struct {
	baseNode
	LockedRels []Node // list of RangeVar
	Strength   LockClauseStrength
	WaitPolicy LockWaitPolicy
}

func (*LockingClause) node() {}

type LockClauseStrength int

const (
	LCS_NONE LockClauseStrength = iota
	LCS_FORKEYSHARE
	LCS_FORSHARE
	LCS_FORNOKEYUPDATE
	LCS_FORUPDATE
)

type LockWaitPolicy int

const (
	LockWaitBlock LockWaitPolicy = iota
	LockWaitSkip
	LockWaitError
)

// WithClause represents a WITH (CTE) clause.
type WithClause struct {
	baseNode
	CTEs      []*CommonTableExpr
	Recursive bool
}

func (*WithClause) node() {}

// CommonTableExpr represents a single CTE.
type CommonTableExpr struct {
	baseNode
	Ctename     string
	Aliascolnames []string
	CTEMaterialized CTEMaterialize
	Ctequery    Stmt
}

func (*CommonTableExpr) node() {}

type CTEMaterialize int

const (
	CTEMaterializeDefault CTEMaterialize = iota
	CTEMaterializeAlways
	CTEMaterializeNever
)

// IntoClause represents SELECT INTO.
type IntoClause struct {
	baseNode
	Rel *RangeVar
}

func (*IntoClause) node() {}

// ---------------------------------------------------------------------------
// FROM clause items
// ---------------------------------------------------------------------------

// RangeVar represents a table reference: [schema.]tablename [alias].
type RangeVar struct {
	baseNode
	Catalogname string
	Schemaname  string
	Relname     string
	Inh         bool   // inherit? (default true)
	Alias       *Alias
}

func (*RangeVar) node() {}
func (*RangeVar) expr() {} // RangeVar can appear in expression contexts

// Alias represents an alias with optional column name list.
type Alias struct {
	baseNode
	Aliasname string
	Colnames  []string
}

func (*Alias) node() {}

// JoinExpr represents a JOIN.
type JoinExpr struct {
	baseNode
	Jointype JoinType
	IsNatural bool
	Larg     Node // left table ref
	Rarg     Node // right table ref
	UsingClause []string // USING column names
	Quals    Expr        // ON expression
	Alias    *Alias
}

func (*JoinExpr) node() {}

type JoinType int

const (
	JOIN_INNER JoinType = iota
	JOIN_LEFT
	JOIN_FULL
	JOIN_RIGHT
	JOIN_CROSS
)

// RangeSubselect represents a subquery in FROM: (SELECT ...) AS alias.
type RangeSubselect struct {
	baseNode
	Lateral    bool
	Subquery   Stmt
	Alias      *Alias
}

func (*RangeSubselect) node() {}

// RangeFunction represents a function call in FROM.
type RangeFunction struct {
	baseNode
	Lateral    bool
	Ordinality bool
	IsRowsfrom bool
	Functions  []Node // list of function calls
	Alias      *Alias
	Coldeflist []*ColumnDef
}

func (*RangeFunction) node() {}

// RangeTableSample represents table TABLESAMPLE method(args) [REPEATABLE (seed)].
type RangeTableSample struct {
	baseNode
	Relation   Node   // the table ref (usually *RangeVar)
	Method     string // sampling method name (e.g. bernoulli, system)
	Args       []Expr // sampling arguments
	Repeatable Expr   // REPEATABLE (seed) expression, or nil
}

func (*RangeTableSample) node() {}

// ColumnDef represents a column definition in CREATE TABLE or function FROM.
type ColumnDef struct {
	baseNode
	Colname     string
	TypeName    *TypeName
	Constraints []*Constraint
	CollClause  *CollateClause
	IsNotNull   bool
}

func (*ColumnDef) node() {}

// ---------------------------------------------------------------------------
// INSERT statement
// ---------------------------------------------------------------------------

type InsertStmt struct {
	baseStmt
	Relation      *RangeVar
	Cols          []*ResTarget // target columns (nil = all)
	SelectStmt    Stmt         // SELECT or VALUES
	OnConflict    *OnConflictClause
	ReturningList []*ResTarget
	WithClause    *WithClause
	Override      OverridingKind
}

type OverridingKind int

const (
	OVERRIDING_NOT_SET OverridingKind = iota
	OVERRIDING_USER_VALUE
	OVERRIDING_SYSTEM_VALUE
)

// OnConflictClause represents ON CONFLICT.
type OnConflictClause struct {
	baseNode
	Action      OnConflictAction
	Infer       *InferClause
	TargetList  []*ResTarget
	WhereClause Expr
}

func (*OnConflictClause) node() {}

type OnConflictAction int

const (
	ONCONFLICT_NONE OnConflictAction = iota
	ONCONFLICT_NOTHING
	ONCONFLICT_UPDATE
)

// InferClause represents the conflict target in ON CONFLICT.
type InferClause struct {
	baseNode
	IndexElems  []Node
	WhereClause Expr
	Conname     string // constraint name
}

func (*InferClause) node() {}

// ---------------------------------------------------------------------------
// UPDATE statement
// ---------------------------------------------------------------------------

type UpdateStmt struct {
	baseStmt
	Relation      *RangeVar
	TargetList    []*ResTarget
	WhereClause   Expr
	FromClause    []Node
	ReturningList []*ResTarget
	WithClause    *WithClause
}

// ---------------------------------------------------------------------------
// DELETE statement
// ---------------------------------------------------------------------------

type DeleteStmt struct {
	baseStmt
	Relation      *RangeVar
	UsingClause   []Node
	WhereClause   Expr
	ReturningList []*ResTarget
	WithClause    *WithClause
}

// MergeStmt represents a MERGE statement.
type MergeStmt struct {
	baseStmt
	Relation       *RangeVar
	SourceRelation Node // table_ref (RangeVar, JoinExpr, RangeSubselect, etc.)
	JoinCondition  Expr
	WhenClauses    []*MergeWhenClause
	WithClause     *WithClause
	ReturningList  []*ResTarget
}

// MergeWhenClause represents a single WHEN clause in a MERGE statement.
type MergeWhenClause struct {
	baseNode
	MatchKind   MergeMatchKind
	CommandType MergeCommandType
	Condition   Expr         // AND condition, or nil
	TargetList  []*ResTarget // SET clause for UPDATE, column list for INSERT
	Values      []Expr       // VALUES for INSERT
	Override    OverridingKind
}

func (*MergeWhenClause) node() {}

type MergeMatchKind int

const (
	MERGE_WHEN_MATCHED                MergeMatchKind = iota
	MERGE_WHEN_NOT_MATCHED_BY_TARGET
	MERGE_WHEN_NOT_MATCHED_BY_SOURCE
)

type MergeCommandType int

const (
	MERGE_CMD_UPDATE  MergeCommandType = iota
	MERGE_CMD_DELETE
	MERGE_CMD_INSERT
	MERGE_CMD_NOTHING
)

// ---------------------------------------------------------------------------
// DDL statements
// ---------------------------------------------------------------------------

// CreateStmt represents CREATE TABLE.
type CreateStmt struct {
	baseStmt
	Relation      *RangeVar
	TableElts     []Node       // list of ColumnDef, Constraint
	InhRelations  []Node       // INHERITS list (RangeVar)
	IfNotExists   bool
	Persistence   RelPersistence // TEMP, UNLOGGED, or permanent
	OnCommit      OnCommitAction
	PartitionSpec *PartitionSpec // PARTITION BY clause, or nil
	PartitionOf   *RangeVar          // PARTITION OF parent, or nil
	PartBound     *PartitionBoundSpec // FOR VALUES clause, or nil
}

// PartitionSpec represents a PARTITION BY clause.
type PartitionSpec struct {
	baseNode
	Strategy string          // "range", "list", or "hash"
	PartParams []*PartitionElem // partition key columns/expressions
}

func (*PartitionSpec) node() {}

// PartitionElem represents a single partition key element.
type PartitionElem struct {
	baseNode
	Name     string   // column name (empty if expression)
	Expr     Expr     // expression (nil if column name)
	Collation []string // COLLATE clause
	OpClass  []string // operator class
}

func (*PartitionElem) node() {}

// PartitionBoundSpec represents partition bounds (FOR VALUES ... or DEFAULT).
type PartitionBoundSpec struct {
	baseNode
	Strategy   string   // "list", "range", or ""
	IsDefault  bool     // DEFAULT partition
	ListValues []Expr   // FOR VALUES IN (...)
	LowerBound []Expr   // FOR VALUES FROM (...)
	UpperBound []Expr   // FOR VALUES TO (...)
}

func (*PartitionBoundSpec) node() {}

// PartitionCmd holds the child table and optional bound for ATTACH/DETACH.
type PartitionCmd struct {
	baseNode
	Name  *RangeVar          // child partition table
	Bound *PartitionBoundSpec // partition bounds (nil for DETACH)
}

func (*PartitionCmd) node() {}

// ClusterStmt represents CLUSTER [table_name [USING index_name]].
type ClusterStmt struct {
	baseStmt
	Relation  *RangeVar // table to cluster, or nil for all
	IndexName string    // index to use, or ""
}

// CreateTableAsStmt represents CREATE TABLE ... AS SELECT.
type CreateTableAsStmt struct {
	baseStmt
	Into        *IntoClause
	Query       Stmt // the SELECT
	IfNotExists bool
	Persistence RelPersistence
	WithData    bool // WITH DATA (true) or WITH NO DATA (false)
}

type RelPersistence int

const (
	RELPERSISTENCE_PERMANENT RelPersistence = iota
	RELPERSISTENCE_TEMP
	RELPERSISTENCE_UNLOGGED
)

type OnCommitAction int

const (
	ONCOMMIT_NOOP          OnCommitAction = iota
	ONCOMMIT_PRESERVE_ROWS
	ONCOMMIT_DELETE_ROWS
	ONCOMMIT_DROP
)

// Constraint represents a column or table constraint.
type Constraint struct {
	baseNode
	Contype        ConstrType
	Conname        string // constraint name (from CONSTRAINT name)
	RawExpr        Expr   // CHECK expression or DEFAULT expression
	Keys           []string // column names for PK, UNIQUE
	FkAttrs        []string // FK local columns
	PkTable        *RangeVar // REFERENCES table
	PkAttrs        []string  // REFERENCES columns
	FkMatchType    string    // FULL, PARTIAL, SIMPLE
	FkUpdAction    string    // ON UPDATE action
	FkDelAction    string    // ON DELETE action
	IsNoInherit    bool
	IsEnforced     bool
	Deferrable     bool
	InitDeferred   bool
	NullsNotDistinct bool
}

func (*Constraint) node() {}

type ConstrType int

const (
	CONSTR_NULL       ConstrType = iota // NULL (not a real constraint)
	CONSTR_NOTNULL                      // NOT NULL
	CONSTR_DEFAULT                      // DEFAULT expr
	CONSTR_CHECK                        // CHECK (expr)
	CONSTR_PRIMARY                      // PRIMARY KEY
	CONSTR_UNIQUE                       // UNIQUE
	CONSTR_FOREIGN                      // FOREIGN KEY / REFERENCES
	CONSTR_EXCLUSION                    // EXCLUDE
	CONSTR_GENERATED                    // GENERATED ALWAYS AS (expr) STORED
	CONSTR_IDENTITY                     // GENERATED { ALWAYS | BY DEFAULT } AS IDENTITY
)

// IndexStmt represents CREATE INDEX.
type IndexStmt struct {
	baseStmt
	Idxname     string
	Relation    *RangeVar
	AccessMethod string
	IndexParams []*IndexElem
	Unique      bool
	Concurrent  bool
	IfNotExists bool
	WhereClause Expr
	NullsNotDistinct bool
}

// IndexElem represents a single index column/expression.
type IndexElem struct {
	baseNode
	Name       string // column name, or empty if expression
	Expr       Expr   // expression, or nil if column name
	Ordering   SortByDir
	NullsOrder SortByNulls
	Opclass    []string // operator class, if any
}

func (*IndexElem) node() {}

// AlterTableStmt represents ALTER TABLE.
type AlterTableStmt struct {
	baseStmt
	Relation  *RangeVar
	Cmds      []*AlterTableCmd
	MissingOk bool // IF EXISTS
}

// AlterTableCmd represents a single ALTER TABLE sub-command.
type AlterTableCmd struct {
	baseNode
	Subtype AlterTableType
	Name    string     // column name for ADD/DROP/ALTER COLUMN
	Def     Node       // ColumnDef for ADD COLUMN, Constraint for ADD CONSTRAINT, TypeName for SET DATA TYPE
	MissingOk bool     // IF EXISTS / IF NOT EXISTS
	Behavior  DropBehavior
}

func (*AlterTableCmd) node() {}

type AlterTableType int

const (
	AT_AddColumn       AlterTableType = iota
	AT_DropColumn
	AT_AlterColumnType
	AT_SetDefault
	AT_DropDefault
	AT_SetNotNull
	AT_DropNotNull
	AT_AddConstraint
	AT_DropConstraint
	AT_RenameColumn    // ALTER TABLE ... RENAME COLUMN
	AT_RenameTable     // ALTER TABLE ... RENAME TO
	AT_SetSchema       // ALTER TABLE ... SET SCHEMA
	AT_AddIndex        // (internal)
	AT_SetStatistics   // ALTER COLUMN ... SET STATISTICS n
	AT_SetStorage      // ALTER COLUMN ... SET STORAGE type
	AT_SetCompression  // ALTER COLUMN ... SET COMPRESSION method
	AT_AddIdentity     // ALTER COLUMN ... ADD GENERATED ... AS IDENTITY
	AT_DropIdentity    // ALTER COLUMN ... DROP IDENTITY
	AT_SetExpression   // ALTER COLUMN ... SET EXPRESSION AS (expr)
	AT_DropExpression  // ALTER COLUMN ... DROP EXPRESSION
	AT_AlterConstraint // ALTER CONSTRAINT name ...
	AT_ValidateConstraint // VALIDATE CONSTRAINT name
	AT_ClusterOn       // CLUSTER ON index
	AT_DropCluster     // SET WITHOUT CLUSTER
	AT_SetLogged       // SET LOGGED
	AT_SetUnLogged     // SET UNLOGGED
	AT_SetAccessMethod // SET ACCESS METHOD name
	AT_SetTableSpace   // SET TABLESPACE name
	AT_SetRelOptions   // SET (opt=val, ...)
	AT_ResetRelOptions // RESET (opt, ...)
	AT_EnableTrig      // ENABLE TRIGGER name
	AT_EnableAlwaysTrig // ENABLE ALWAYS TRIGGER name
	AT_EnableReplicaTrig // ENABLE REPLICA TRIGGER name
	AT_DisableTrig     // DISABLE TRIGGER name
	AT_EnableTrigAll   // ENABLE TRIGGER ALL
	AT_DisableTrigAll  // DISABLE TRIGGER ALL
	AT_EnableTrigUser  // ENABLE TRIGGER USER
	AT_DisableTrigUser // DISABLE TRIGGER USER
	AT_EnableRule      // ENABLE RULE name
	AT_EnableAlwaysRule // ENABLE ALWAYS RULE name
	AT_EnableReplicaRule // ENABLE REPLICA RULE name
	AT_DisableRule     // DISABLE RULE name
	AT_AddInherit      // INHERIT parent
	AT_DropInherit     // NO INHERIT parent
	AT_AddOf           // OF type
	AT_DropOf          // NOT OF
	AT_ChangeOwner     // OWNER TO role
	AT_ReplicaIdentity // REPLICA IDENTITY ...
	AT_EnableRowSecurity  // ENABLE ROW LEVEL SECURITY
	AT_DisableRowSecurity // DISABLE ROW LEVEL SECURITY
	AT_ForceRowSecurity   // FORCE ROW LEVEL SECURITY
	AT_NoForceRowSecurity // NO FORCE ROW LEVEL SECURITY
	AT_AttachPartition    // ATTACH PARTITION child FOR VALUES ...
	AT_DetachPartition    // DETACH PARTITION child
)

// DropStmt represents DROP TABLE/INDEX/VIEW/etc.
type DropStmt struct {
	baseStmt
	Objects    [][]string // list of qualified names
	RemoveType ObjectType
	MissingOk  bool // IF EXISTS
	Behavior   DropBehavior
	Concurrent bool
}

type ObjectType int

const (
	OBJECT_TABLE     ObjectType = iota
	OBJECT_INDEX
	OBJECT_VIEW
	OBJECT_MATVIEW
	OBJECT_SEQUENCE
	OBJECT_TYPE
	OBJECT_SCHEMA
	OBJECT_FUNCTION
	OBJECT_PROCEDURE
	OBJECT_EXTENSION
	OBJECT_TRIGGER
	OBJECT_RULE
	OBJECT_DOMAIN
	OBJECT_FOREIGN_TABLE
	OBJECT_ROLE
	OBJECT_DATABASE
	OBJECT_TABLESPACE
	OBJECT_POLICY
	OBJECT_PUBLICATION
	OBJECT_SUBSCRIPTION
	OBJECT_AGGREGATE
	OBJECT_OPERATOR
	OBJECT_COLLATION
	OBJECT_CONVERSION
	OBJECT_LANGUAGE
	OBJECT_CAST
	OBJECT_COLUMN
	OBJECT_CONSTRAINT
	OBJECT_EVENT_TRIGGER
	OBJECT_ACCESS_METHOD
	OBJECT_FDW
	OBJECT_FOREIGN_SERVER
	OBJECT_TSPARSER
	OBJECT_TSDICTIONARY
	OBJECT_TSTEMPLATE
	OBJECT_TSCONFIGURATION
	OBJECT_STATISTICS
	OBJECT_TRANSFORM
	OBJECT_OPCLASS
	OBJECT_OPFAMILY
	OBJECT_LARGEOBJECT
	OBJECT_USER_MAPPING
)

type DropBehavior int

const (
	DROP_RESTRICT DropBehavior = iota
	DROP_CASCADE
)

// TruncateStmt represents TRUNCATE.
type TruncateStmt struct {
	baseStmt
	Relations   []*RangeVar
	RestartSeqs bool
	Behavior    DropBehavior
}

// RenameStmt represents ALTER TABLE ... RENAME.
type RenameStmt struct {
	baseStmt
	RenameType ObjectType
	Relation   *RangeVar
	Subname    string // old column name
	Newname    string // new name
	MissingOk  bool
}

// ViewStmt represents CREATE [OR REPLACE] VIEW.
type ViewStmt struct {
	baseStmt
	View            *RangeVar
	Aliases         []string // column aliases
	Query           Stmt     // the SELECT
	Replace         bool     // OR REPLACE
	Persistence     RelPersistence
	WithCheckOption ViewCheckOption
}

type ViewCheckOption int

const (
	NO_CHECK_OPTION    ViewCheckOption = iota
	LOCAL_CHECK_OPTION
	CASCADED_CHECK_OPTION
)

// ExplainStmt represents EXPLAIN.
type ExplainStmt struct {
	baseStmt
	Query   Stmt
	Options []*DefElem // ANALYZE, VERBOSE, FORMAT, etc.
}

// DefElem represents a generic name=value option.
type DefElem struct {
	baseNode
	Defname string
	Arg     Node // value, or nil for boolean options
}

func (*DefElem) node() {}

// CopyStmt represents COPY ... FROM/TO.
type CopyStmt struct {
	baseStmt
	Relation    *RangeVar
	Query       Stmt     // for COPY (query) TO
	Attlist     []string // column list
	IsFrom      bool     // FROM (true) or TO (false)
	IsProgram   bool
	Filename    string   // file name, or empty for STDIN/STDOUT
	WhereClause Expr
	Options     []*DefElem
}

// TransactionStmt represents BEGIN, COMMIT, ROLLBACK, SAVEPOINT, etc.
type TransactionStmt struct {
	baseStmt
	Kind    TransactionStmtKind
	Options []string // transaction modes, savepoint name, etc.
}

type TransactionStmtKind int

const (
	TRANS_STMT_BEGIN TransactionStmtKind = iota
	TRANS_STMT_COMMIT
	TRANS_STMT_ROLLBACK
	TRANS_STMT_SAVEPOINT
	TRANS_STMT_RELEASE
	TRANS_STMT_ROLLBACK_TO
	TRANS_STMT_PREPARE // PREPARE TRANSACTION
	TRANS_STMT_COMMIT_PREPARED
	TRANS_STMT_ROLLBACK_PREPARED
	TRANS_STMT_START // START TRANSACTION
)

// VariableSetStmt represents SET name = value.
type VariableSetStmt struct {
	baseStmt
	Name    string
	Args    []Expr
	IsLocal bool
	IsReset bool
}

// VariableShowStmt represents SHOW name.
type VariableShowStmt struct {
	baseStmt
	Name string
}

// ListenStmt represents LISTEN channel.
type ListenStmt struct {
	baseStmt
	Conditionname string
}

// NotifyStmt represents NOTIFY channel [, payload].
type NotifyStmt struct {
	baseStmt
	Conditionname string
	Payload       string
}

// UnlistenStmt represents UNLISTEN channel | *.
type UnlistenStmt struct {
	baseStmt
	Conditionname string // empty for *
}

// VacuumStmt represents VACUUM or ANALYZE.
type VacuumStmt struct {
	baseStmt
	Options   []*DefElem
	Relations []*RangeVar
	IsVacuum  bool // true=VACUUM, false=ANALYZE
}

// LockStmt represents LOCK TABLE.
type LockStmt struct {
	baseStmt
	Relations []*RangeVar
	Mode      string
	Nowait    bool
}

// PrepareStmt represents PREPARE name [(types)] AS stmt.
type PrepareStmt struct {
	baseStmt
	Name     string
	Argtypes []*TypeName
	Query    Stmt
}

// ExecuteStmt represents EXECUTE name [(params)].
type ExecuteStmt struct {
	baseStmt
	Name   string
	Params []Expr
}

// DeallocateStmt represents DEALLOCATE [PREPARE] name | ALL.
type DeallocateStmt struct {
	baseStmt
	Name  string // empty for ALL
	IsAll bool
}

// DiscardStmt represents DISCARD ALL | PLANS | SEQUENCES | TEMP.
type DiscardStmt struct {
	baseStmt
	Target string
}

// CreateFunctionStmt represents CREATE [OR REPLACE] FUNCTION/PROCEDURE.
type CreateFunctionStmt struct {
	baseStmt
	Funcname    []string
	Parameters  []*FunctionParameter
	ReturnType  *TypeName
	Options     []*DefElem // LANGUAGE, AS, IMMUTABLE, etc.
	Replace     bool
	IsProcedure bool
}

// FunctionParameter represents a single function parameter.
type FunctionParameter struct {
	baseNode
	Name    string
	ArgType *TypeName
	Mode    FuncParamMode
	DefExpr Expr // DEFAULT value
}

func (*FunctionParameter) node() {}

type FuncParamMode int

const (
	FUNC_PARAM_IN      FuncParamMode = iota
	FUNC_PARAM_OUT
	FUNC_PARAM_INOUT
	FUNC_PARAM_VARIADIC
	FUNC_PARAM_DEFAULT // no explicit mode
)

// DoStmt represents DO $$ ... $$ [LANGUAGE lang].
type DoStmt struct {
	baseStmt
	Args []*DefElem // body and language
}

// CallStmt represents CALL procedure_name(args).
type CallStmt struct {
	baseStmt
	FuncCall *FuncCall
}

// CreateTrigStmt represents CREATE TRIGGER.
type CreateTrigStmt struct {
	baseStmt
	Trigname   string
	Relation   *RangeVar
	Funcname   []string
	Args       []Expr
	Row        bool // FOR EACH ROW vs STATEMENT
	Timing     int  // BEFORE, AFTER, INSTEAD OF
	Events     int  // INSERT, UPDATE, DELETE, TRUNCATE
	Columns    []string // UPDATE OF columns
	WhenClause Expr
	Replace    bool
}

// Trigger timing flags
const (
	TRIGGER_TYPE_BEFORE    = 1 << 1
	TRIGGER_TYPE_AFTER     = 1 << 2
	TRIGGER_TYPE_INSTEAD   = 1 << 3
	TRIGGER_TYPE_INSERT    = 1 << 4
	TRIGGER_TYPE_DELETE    = 1 << 5
	TRIGGER_TYPE_UPDATE    = 1 << 6
	TRIGGER_TYPE_TRUNCATE  = 1 << 7
)

// RuleStmt represents CREATE RULE.
type RuleStmt struct {
	baseStmt
	Rulename  string
	WhereClause Expr
	Event     CmdType
	Instead   bool
	Actions   []Stmt
	Replace   bool
	Relation  *RangeVar
}

type CmdType int

const (
	CMD_SELECT CmdType = iota
	CMD_UPDATE
	CMD_INSERT
	CMD_DELETE
	CMD_NOTHING
)

// GrantStmt represents GRANT/REVOKE privileges.
type GrantStmt struct {
	baseStmt
	IsGrant    bool
	Privileges []string   // ALL, SELECT, INSERT, etc.
	PrivCols   [][]string // per-privilege column lists (nil entry = no columns)
	TargetType ObjectType // TABLE, SCHEMA, FUNCTION, etc.
	Objects    [][]string // qualified names
	Grantees   []string   // role names
	GrantOption bool      // WITH GRANT OPTION / CASCADE
}

// GrantRoleStmt represents GRANT/REVOKE role TO/FROM role.
type GrantRoleStmt struct {
	baseStmt
	IsGrant     bool
	GrantedRoles []string
	Grantees     []string
	AdminOption  bool
}

// CreateRoleStmt represents CREATE ROLE/USER/GROUP.
type CreateRoleStmt struct {
	baseStmt
	RoleName string
	Options  []*DefElem
	StmtType string // ROLE, USER, GROUP
}

// AlterRoleStmt represents ALTER ROLE/USER.
type AlterRoleStmt struct {
	baseStmt
	RoleName string
	Options  []*DefElem
}

// CreateSchemaStmt represents CREATE SCHEMA.
type CreateSchemaStmt struct {
	baseStmt
	Schemaname  string
	AuthRole    string
	IfNotExists bool
}

// CreateDomainStmt represents CREATE DOMAIN.
type CreateDomainStmt struct {
	baseStmt
	Domainname  []string
	TypeName    *TypeName
	Constraints []*Constraint
	CollClause  *CollateClause
}

// CreateEnumStmt represents CREATE TYPE name AS ENUM (...).
type CreateEnumStmt struct {
	baseStmt
	TypeName []string
	Vals     []string
}

// CompositeTypeStmt represents CREATE TYPE name AS (col type, ...).
type CompositeTypeStmt struct {
	baseStmt
	TypeName []string
	ColDefs  []*ColumnDef
}

// AlterEnumStmt represents ALTER TYPE name ADD VALUE / RENAME VALUE.
type AlterEnumStmt struct {
	baseStmt
	TypeName    []string
	NewVal      string
	NewValNeighbor string
	NewValIsAfter  bool
	IfNotExists    bool
	RenameOldVal   string
}

// TableLikeClause represents LIKE source_table in CREATE TABLE.
type TableLikeClause struct {
	baseNode
	Relation *RangeVar
	Options  uint32 // bitmask of LIKE options
}

func (*TableLikeClause) node() {}

// ---------------------------------------------------------------------------
// Utility: list of expressions used in IN (...) etc.
// ---------------------------------------------------------------------------

// ExprList is a wrapper for a list of expressions, used as a Node.
type ExprList struct {
	baseExpr
	Items []Expr
}

// SQLValueFunction represents parameterless SQL value functions like
// CURRENT_DATE, CURRENT_USER, etc.
type SQLValueFunction struct {
	baseExpr
	Op      SQLValueFunctionOp
	Typmod  int32 // type modifier (precision for time types), -1 if none
}

type SQLValueFunctionOp int

const (
	SVFOP_CURRENT_DATE         SQLValueFunctionOp = iota
	SVFOP_CURRENT_TIME
	SVFOP_CURRENT_TIME_N
	SVFOP_CURRENT_TIMESTAMP
	SVFOP_CURRENT_TIMESTAMP_N
	SVFOP_LOCALTIME
	SVFOP_LOCALTIME_N
	SVFOP_LOCALTIMESTAMP
	SVFOP_LOCALTIMESTAMP_N
	SVFOP_CURRENT_ROLE
	SVFOP_CURRENT_USER
	SVFOP_USER
	SVFOP_SESSION_USER
	SVFOP_CURRENT_CATALOG
	SVFOP_CURRENT_SCHEMA
)

// GroupingFunc represents GROUPING(expr, ...).
type GroupingFunc struct {
	baseExpr
	Args []Expr
}

// GroupingSet represents ROLLUP(...), CUBE(...), GROUPING SETS(...), or ().
type GroupingSet struct {
	baseExpr
	Kind    GroupingSetKind
	Content []Node // list of expressions or nested GroupingSets
}

type GroupingSetKind int

const (
	GROUPING_SET_EMPTY  GroupingSetKind = iota // ()
	GROUPING_SET_ROLLUP                        // ROLLUP(...)
	GROUPING_SET_CUBE                          // CUBE(...)
	GROUPING_SET_SETS                          // GROUPING SETS(...)
)

// SetToDefault represents the DEFAULT keyword in INSERT/UPDATE value contexts.
type SetToDefault struct {
	baseExpr
}

// ---------------------------------------------------------------------------
// Step 14: Sequences, Extensions, FDW, Pub/Sub, misc DDL
// ---------------------------------------------------------------------------

// CreateSeqStmt represents CREATE SEQUENCE.
type CreateSeqStmt struct {
	baseStmt
	Name        []string
	Options     []*DefElem
	IfNotExists bool
	Temp        bool
}

// AlterSeqStmt represents ALTER SEQUENCE.
type AlterSeqStmt struct {
	baseStmt
	Name      []string
	Options   []*DefElem
	IfExists  bool
	MissingOk bool
}

// CreateExtensionStmt represents CREATE EXTENSION.
type CreateExtensionStmt struct {
	baseStmt
	Extname     string
	IfNotExists bool
	Options     []*DefElem
}

// AlterExtensionStmt represents ALTER EXTENSION ... UPDATE.
type AlterExtensionStmt struct {
	baseStmt
	Extname string
	Options []*DefElem
}

// CreatePolicyStmt represents CREATE POLICY.
type CreatePolicyStmt struct {
	baseStmt
	PolicyName string
	Table      []string
	CmdName    string
	Permissive bool
	Roles      []string
	Qual       Expr
	WithCheck  Expr
}

// AlterPolicyStmt represents ALTER POLICY.
type AlterPolicyStmt struct {
	baseStmt
	PolicyName string
	Table      []string
	Roles      []string
	Qual       Expr
	WithCheck  Expr
}

// CreatePublicationStmt represents CREATE PUBLICATION.
type CreatePublicationStmt struct {
	baseStmt
	Pubname      string
	Options      []*DefElem
	Tables       [][]string
	ForAllTables bool
}

// CreateSubscriptionStmt represents CREATE SUBSCRIPTION.
type CreateSubscriptionStmt struct {
	baseStmt
	Subname     string
	Conninfo    string
	Publication []string
	Options     []*DefElem
}

// CreateEventTrigStmt represents CREATE EVENT TRIGGER.
type CreateEventTrigStmt struct {
	baseStmt
	Trigname   string
	Eventname  string
	WhenClause []*DefElem
	Funcname   []string
}

// CoercionContext for CREATE CAST.
type CoercionContext int

const (
	COERCION_IMPLICIT   CoercionContext = iota
	COERCION_ASSIGNMENT
	COERCION_EXPLICIT
)

// ---------------------------------------------------------------------------
// Step 15: XML and JSON expression syntax
// ---------------------------------------------------------------------------

// XmlExpr represents XML expression functions.
type XmlExpr struct {
	baseExpr
	Op        XmlExprOp
	Name      string // element/PI name
	NamedArgs []Node // XMLATTRIBUTES or XMLFOREST items (ResTarget nodes)
	Args      []Expr // content arguments
	Xmloption XmlOptionType
	TypeName  *TypeName // for XMLSERIALIZE
	Indent    bool      // for XMLSERIALIZE INDENT
}

// XmlExprOp identifies the XML operation.
type XmlExprOp int

const (
	IS_XMLCONCAT    XmlExprOp = iota // XMLCONCAT(...)
	IS_XMLELEMENT                    // XMLELEMENT(...)
	IS_XMLFOREST                     // XMLFOREST(...)
	IS_XMLPARSE                      // XMLPARSE(...)
	IS_XMLPI                         // XMLPI(...)
	IS_XMLROOT                       // XMLROOT(...)
	IS_XMLSERIALIZE                  // XMLSERIALIZE(...)
	IS_XMLEXISTS                     // XMLEXISTS(...)
)

// XmlOptionType for DOCUMENT vs CONTENT.
type XmlOptionType int

const (
	XMLOPTION_DOCUMENT XmlOptionType = iota
	XMLOPTION_CONTENT
)

// JsonIsPredicate represents expr IS [NOT] JSON [type] [WITH|WITHOUT UNIQUE KEYS].
type JsonIsPredicate struct {
	baseExpr
	Expr       Expr
	ItemType   JsonValueType
	UniqueKeys bool
}

// JsonValueType for IS JSON predicates.
type JsonValueType int

const (
	JS_TYPE_ANY    JsonValueType = iota // IS JSON
	JS_TYPE_OBJECT                      // IS JSON OBJECT
	JS_TYPE_ARRAY                       // IS JSON ARRAY
	JS_TYPE_SCALAR                      // IS JSON SCALAR
)

// JsonObjectConstructor represents JSON_OBJECT(key: value, ...).
type JsonObjectConstructor struct {
	baseExpr
	Exprs      []*JsonKeyValue
	Output     *JsonOutput
	AbsentOnNull bool
	UniqueKeys   bool
}

// JsonKeyValue represents a key:value pair in JSON_OBJECT.
type JsonKeyValue struct {
	baseNode
	Key   Expr
	Value Expr
}

// JsonOutput represents RETURNING type [FORMAT JSON].
type JsonOutput struct {
	baseNode
	TypeName *TypeName
	// Format is always JSON in standard SQL
}

// JsonArrayConstructor represents JSON_ARRAY(expr, ...).
type JsonArrayConstructor struct {
	baseExpr
	Exprs        []Expr
	Output       *JsonOutput
	AbsentOnNull bool
}

// JsonObjectAgg represents JSON_OBJECTAGG(key: value ...).
type JsonObjectAgg struct {
	baseExpr
	Arg          *JsonKeyValue
	Output       *JsonOutput
	AbsentOnNull bool
	UniqueKeys   bool
}

// JsonArrayAgg represents JSON_ARRAYAGG(expr ... [ORDER BY ...]).
type JsonArrayAgg struct {
	baseExpr
	Arg          Expr
	Order        []*SortBy
	Output       *JsonOutput
	AbsentOnNull bool
}

// JsonFuncExpr represents JSON_QUERY, JSON_VALUE, JSON_EXISTS.
type JsonFuncExpr struct {
	baseExpr
	Op       JsonFuncOp
	Expr     Expr
	PathSpec Expr // path expression (string literal)
	Passing  []*JsonArgument
	Output   *JsonOutput
	OnEmpty  *JsonBehavior
	OnError  *JsonBehavior
	Wrapper  JsonWrapper
}

// JsonFuncOp identifies the JSON function.
type JsonFuncOp int

const (
	JSON_QUERY_OP  JsonFuncOp = iota
	JSON_VALUE_OP
	JSON_EXISTS_OP
)

// JsonArgument represents a PASSING argument.
type JsonArgument struct {
	baseNode
	Val  Expr
	Name string
}

// JsonBehavior represents ON EMPTY / ON ERROR behavior.
type JsonBehavior struct {
	baseNode
	BType JsonBehaviorType
	Expr  Expr // for DEFAULT expr
}

// JsonBehaviorType enumerates behavior options.
type JsonBehaviorType int

const (
	JSON_BEHAVIOR_NULL    JsonBehaviorType = iota
	JSON_BEHAVIOR_ERROR
	JSON_BEHAVIOR_EMPTY
	JSON_BEHAVIOR_TRUE
	JSON_BEHAVIOR_FALSE
	JSON_BEHAVIOR_UNKNOWN
	JSON_BEHAVIOR_EMPTY_ARRAY
	JSON_BEHAVIOR_EMPTY_OBJECT
	JSON_BEHAVIOR_DEFAULT
)

// JsonWrapper for JSON_QUERY wrapping.
type JsonWrapper int

const (
	JSW_NONE          JsonWrapper = iota
	JSW_UNCONDITIONAL             // WITH WRAPPER
	JSW_CONDITIONAL               // WITH CONDITIONAL WRAPPER
)

// JsonScalarExpr represents JSON_SCALAR(expr).
type JsonScalarExpr struct {
	baseExpr
	Expr   Expr
	Output *JsonOutput
}

// JsonSerializeExpr represents JSON_SERIALIZE(expr [FORMAT JSON] [RETURNING type]).
type JsonSerializeExpr struct {
	baseExpr
	Expr   Expr
	Output *JsonOutput
}

// JsonTableColumnType distinguishes JSON_TABLE column kinds.
type JsonTableColumnType int

const (
	JTC_FOR_ORDINALITY JsonTableColumnType = iota
	JTC_REGULAR                            // type PATH path
	JTC_EXISTS                             // type EXISTS PATH path
	JTC_NESTED                             // NESTED PATH path COLUMNS (...)
)

// JsonTableColumn represents a single column in JSON_TABLE COLUMNS clause.
type JsonTableColumn struct {
	baseNode
	Coltype  JsonTableColumnType
	Name     string
	TypeName *TypeName
	PathSpec Expr // path string
	Wrapper  JsonWrapper
	OnEmpty  *JsonBehavior
	OnError  *JsonBehavior
	// For NESTED
	Columns []*JsonTableColumn
}

// JsonTable represents JSON_TABLE(expr, path COLUMNS (...)) in FROM.
type JsonTable struct {
	baseNode
	Expr     Expr
	PathSpec Expr // path string
	Passing  []*JsonArgument
	Columns  []*JsonTableColumn
	OnEmpty  *JsonBehavior
	OnError  *JsonBehavior
	Alias    *Alias
	Lateral  bool
}

func (*JsonTable) node() {}

// XmlTableColumn represents a single column in XMLTABLE COLUMNS clause.
type XmlTableColumn struct {
	baseNode
	Name       string
	TypeName   *TypeName
	ForOrdinality bool
	PathExpr   Expr // xpath string
	DefExpr    Expr // DEFAULT expr
	IsNotNull  bool
}

// XmlTable represents XMLTABLE(xpath PASSING xml COLUMNS (...)) in FROM.
type XmlTable struct {
	baseNode
	Xmlexpr  Expr   // row xpath expression
	Docexpr  Expr   // the XML document (PASSING arg)
	Columns  []*XmlTableColumn
	Namespaces []Node // optional WITH XMLNAMESPACES
	Alias    *Alias
	Lateral  bool
}

func (*XmlTable) node() {}

// ---------------------------------------------------------------------------
// PLAN2 Step 2: Cursor statements
// ---------------------------------------------------------------------------

// DeclareCursorStmt represents DECLARE cursor_name ... CURSOR FOR query.
type DeclareCursorStmt struct {
	baseStmt
	Portalname string
	Options    int  // bitmask of CURSOR_OPT_*
	Query      Stmt // the SELECT query
}

// Cursor option flags (matching PG's CURSOR_OPT_* constants).
const (
	CURSOR_OPT_BINARY      = 1 << 0
	CURSOR_OPT_SCROLL      = 1 << 1
	CURSOR_OPT_NO_SCROLL   = 1 << 2
	CURSOR_OPT_INSENSITIVE = 1 << 3
	CURSOR_OPT_ASENSITIVE  = 1 << 4
	CURSOR_OPT_HOLD        = 1 << 5
)

// FetchStmt represents FETCH or MOVE.
type FetchStmt struct {
	baseStmt
	Direction  FetchDirection
	HowMany   int64
	Portalname string
	IsMove     bool // true for MOVE, false for FETCH
}

// FetchDirection enumerates FETCH/MOVE directions.
type FetchDirection int

const (
	FETCH_FORWARD  FetchDirection = iota // FORWARD / NEXT / positive count
	FETCH_BACKWARD                       // BACKWARD / PRIOR / negative count
	FETCH_ABSOLUTE                       // ABSOLUTE n
	FETCH_RELATIVE                       // RELATIVE n
)

// ClosePortalStmt represents CLOSE cursor or CLOSE ALL.
type ClosePortalStmt struct {
	baseStmt
	Portalname string // empty string means CLOSE ALL
}

// ---------------------------------------------------------------------------
// PLAN2 Step 3: Database & tablespace DDL
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// PLAN2 Step 4: ALTER dispatch expansion — new node types
// ---------------------------------------------------------------------------

// AlterRoleSetStmt represents ALTER ROLE ... SET/RESET config.
type AlterRoleSetStmt struct {
	baseStmt
	RoleName string
	SetStmt  Stmt
}

// AlterDomainStmt represents ALTER DOMAIN.
type AlterDomainStmt struct {
	baseStmt
	Subtype    byte   // 'T' = SET DEFAULT, 'N' = DROP DEFAULT, 'O' = SET NOT NULL, 'C' = ADD CONSTRAINT, 'X' = DROP CONSTRAINT, 'V' = VALIDATE CONSTRAINT
	TypeName   []string
	Name       string // constraint name for DROP/VALIDATE CONSTRAINT
	Def        Expr   // new default for SET DEFAULT
	Constraint *Constraint
	MissingOk  bool
}

// AlterFunctionStmt represents ALTER FUNCTION/PROCEDURE.
type AlterFunctionStmt struct {
	baseStmt
	ObjType byte // 'f' = FUNCTION, 'p' = PROCEDURE
	Func    *FuncWithArgs
	Actions []*DefElem
}

// FuncWithArgs identifies a function by name and argument types.
type FuncWithArgs struct {
	baseNode
	Funcname []string
	Funcargs []*TypeName
}

// AlterPublicationStmt represents ALTER PUBLICATION.
type AlterPublicationStmt struct {
	baseStmt
	Pubname      string
	Options      []*DefElem
	Tables       [][]string
	ForAllTables bool
	TableAction  string // "add", "drop", "set"
}

// AlterSubscriptionStmt represents ALTER SUBSCRIPTION.
type AlterSubscriptionStmt struct {
	baseStmt
	Subname     string
	Kind        string // "options", "connection", "refresh", "publication", "enable", "disable"
	Conninfo    string
	Publication []string
	Options     []*DefElem
}

// AlterEventTrigStmt represents ALTER EVENT TRIGGER.
type AlterEventTrigStmt struct {
	baseStmt
	Trigname  string
	Tgenabled byte // 'O' = enable, 'D' = disable, 'R' = replica, 'A' = always
}

// AlterTypeStmt represents generic ALTER TYPE operations (ADD/DROP/ALTER ATTRIBUTE, SET SCHEMA, OWNER TO, RENAME).
type AlterTypeStmt struct {
	baseStmt
	TypeName []string
	Options  []*DefElem
}

// AlterSystemStmt represents ALTER SYSTEM SET/RESET.
type AlterSystemStmt struct {
	baseStmt
	SetStmt Stmt
}

// ---------------------------------------------------------------------------
// PLAN2 Step 6: DROP completions, RENAME, REASSIGN OWNED
// ---------------------------------------------------------------------------

// DropRoleStmt represents DROP ROLE/USER/GROUP.
type DropRoleStmt struct {
	baseStmt
	Roles     []string
	MissingOk bool
}

// DropOwnedStmt represents DROP OWNED BY role, ... [CASCADE|RESTRICT].
type DropOwnedStmt struct {
	baseStmt
	Roles    []string
	Behavior DropBehavior
}

// ReassignOwnedStmt represents REASSIGN OWNED BY role, ... TO newrole.
type ReassignOwnedStmt struct {
	baseStmt
	Roles   []string
	NewRole string
}

// RemoveFuncStmt represents DROP FUNCTION/PROCEDURE/AGGREGATE.
type RemoveFuncStmt struct {
	baseStmt
	ObjType    ObjectType
	Funcname   []string
	Funcargs   []*TypeName
	MissingOk  bool
	Behavior   DropBehavior
}

// ---------------------------------------------------------------------------
// PLAN2 Step 7: Foreign data wrappers & foreign tables
// ---------------------------------------------------------------------------

// CreateFdwStmt represents CREATE FOREIGN DATA WRAPPER.
type CreateFdwStmt struct {
	baseStmt
	Fdwname    string
	FuncOptions []*DefElem // HANDLER, VALIDATOR
	Options    []*DefElem  // generic options
}

// CreateForeignServerStmt represents CREATE SERVER.
type CreateForeignServerStmt struct {
	baseStmt
	Servername  string
	ServerType  string
	Version     string
	Fdwname     string
	IfNotExists bool
	Options     []*DefElem
}

// CreateForeignTableStmt represents CREATE FOREIGN TABLE.
type CreateForeignTableStmt struct {
	baseStmt
	Base       CreateStmt // reuse table definition
	Servername string
	Options    []*DefElem
}

// CreateUserMappingStmt represents CREATE USER MAPPING.
type CreateUserMappingStmt struct {
	baseStmt
	User        string // role name or PUBLIC or CURRENT_USER
	Servername  string
	IfNotExists bool
	Options     []*DefElem
}

// ImportForeignSchemaStmt represents IMPORT FOREIGN SCHEMA.
type ImportForeignSchemaStmt struct {
	baseStmt
	ServerName   string
	RemoteSchema string
	LocalSchema  string
	ListType     string // "limit_to", "except", or ""
	TableList    []string
	Options      []*DefElem
}

// ---------------------------------------------------------------------------
// PLAN2 Step 8: Materialized views & statistics
// ---------------------------------------------------------------------------

// CreateMatViewStmt represents CREATE MATERIALIZED VIEW.
type CreateMatViewStmt struct {
	baseStmt
	Relation    *RangeVar
	Query       Stmt
	IfNotExists bool
	WithData    bool // WITH DATA (true) or WITH NO DATA (false)
	AccessMethod string
	Options     []*DefElem
	TableSpace  string
}

// RefreshMatViewStmt represents REFRESH MATERIALIZED VIEW.
type RefreshMatViewStmt struct {
	baseStmt
	Relation     *RangeVar
	Concurrent   bool
	SkipData     bool // WITH NO DATA
}

// CreateStatsStmt represents CREATE STATISTICS.
type CreateStatsStmt struct {
	baseStmt
	Defnames    []string // qualified stat name
	StatTypes   []string // ndistinct, dependencies, mcv
	Exprs       []Expr
	Relations   [][]string
	IfNotExists bool
}

// AlterOwnerStmt represents ALTER object_type name OWNER TO newowner.
type AlterOwnerStmt struct {
	baseStmt
	ObjectType ObjectType
	Object     []string
	NewOwner   string
}

// AlterStatsStmt represents ALTER STATISTICS name SET STATISTICS n / RENAME / SET SCHEMA / OWNER TO.
type AlterStatsStmt struct {
	baseStmt
	Defnames []string
	Stxstattarget int // SET STATISTICS value; -1 if not set
}

// CreatedbStmt represents CREATE DATABASE.
type CreatedbStmt struct {
	baseStmt
	Dbname  string
	Options []*DefElem
}

// DropdbStmt represents DROP DATABASE.
type DropdbStmt struct {
	baseStmt
	Dbname    string
	MissingOk bool
	Options   []*DefElem // WITH (FORCE)
}

// AlterDatabaseStmt represents ALTER DATABASE ... WITH options.
type AlterDatabaseStmt struct {
	baseStmt
	Dbname  string
	Options []*DefElem
}

// AlterDatabaseSetStmt represents ALTER DATABASE ... SET/RESET config.
type AlterDatabaseSetStmt struct {
	baseStmt
	Dbname string
	SetStmt Stmt // a VariableSetStmt or VariableResetStmt
}

// CreateTableSpaceStmt represents CREATE TABLESPACE.
type CreateTableSpaceStmt struct {
	baseStmt
	Tablespacename string
	Owner          string
	Location       string
	Options        []*DefElem
}




// CommentStmt represents COMMENT ON object_type name IS 'text'.
type CommentStmt struct {
	baseStmt
	ObjType ObjectType
	Object  []string // qualified name
	Comment string   // the comment text; empty means NULL (drop comment)
	IsNull  bool     // true if IS NULL (drop comment)
}

// SecLabelStmt represents SECURITY LABEL [FOR provider] ON object_type name IS 'label'.
type SecLabelStmt struct {
	baseStmt
	Provider string
	ObjType  ObjectType
	Object   []string
	Label    string
	IsNull   bool
}

// CheckPointStmt represents CHECKPOINT.
type CheckPointStmt struct {
	baseStmt
}

// LoadStmt represents LOAD 'filename'.
type LoadStmt struct {
	baseStmt
	Filename string
}

// ReindexObjectType distinguishes REINDEX targets.
type ReindexObjectType int

const (
	REINDEX_OBJECT_INDEX    ReindexObjectType = iota
	REINDEX_OBJECT_TABLE
	REINDEX_OBJECT_SCHEMA
	REINDEX_OBJECT_DATABASE
	REINDEX_OBJECT_SYSTEM
)

// ReindexStmt represents REINDEX [(options)] {INDEX|TABLE|SCHEMA|DATABASE|SYSTEM} [CONCURRENTLY] name.
type ReindexStmt struct {
	baseStmt
	Kind       ReindexObjectType
	Relation   *RangeVar // for INDEX, TABLE
	Name       string    // for SCHEMA, DATABASE, SYSTEM
	Options    []*DefElem
	Concurrent bool
}

// ConstraintsSetStmt represents SET CONSTRAINTS {ALL|name,...} {DEFERRED|IMMEDIATE}.
type ConstraintsSetStmt struct {
	baseStmt
	Constraints [][]string // nil means ALL
	Deferred    bool
}

// AlterDefaultPrivilegesStmt represents ALTER DEFAULT PRIVILEGES ... GRANT/REVOKE.
type AlterDefaultPrivilegesStmt struct {
	baseStmt
	Options []*DefElem // FOR ROLE, IN SCHEMA
	Action  Stmt       // a GrantStmt
}

// DefineStmt represents CREATE AGGREGATE, CREATE OPERATOR, CREATE TYPE (range/shell),
// CREATE TEXT SEARCH {PARSER|DICTIONARY|TEMPLATE|CONFIGURATION}, CREATE COLLATION.
type DefineStmt struct {
	baseStmt
	Kind        ObjectType // OBJECT_AGGREGATE, OBJECT_OPERATOR, OBJECT_TYPE, etc.
	Defnames    []string   // qualified name
	Args        []Node     // function arguments (for AGGREGATE)
	Definition  []*DefElem // (name = value, ...) pairs
	IfNotExists bool
	Replace     bool // OR REPLACE
	OldStyle    bool // old-style aggregate syntax
}

// CreateCastStmt represents CREATE CAST (source AS target) ...
type CreateCastStmt struct {
	baseStmt
	SourceType *TypeName
	TargetType *TypeName
	Func       *FuncWithArgs // nil for WITHOUT FUNCTION or WITH INOUT
	Context    CoercionContext
	Inout      bool // WITH INOUT
}

// CreateTransformStmt represents CREATE [OR REPLACE] TRANSFORM FOR type LANGUAGE lang (...).
type CreateTransformStmt struct {
	baseStmt
	Replace  bool
	TypeName *TypeName
	Lang     string
	FromSQL  *FuncWithArgs
	ToSQL    *FuncWithArgs
}

// CreateAmStmt represents CREATE ACCESS METHOD name TYPE {INDEX|TABLE} HANDLER func.
type CreateAmStmt struct {
	baseStmt
	AmName    string
	AmType    string // "INDEX" or "TABLE"
	HandlerName []string
}

// CreateOpClassStmt represents CREATE OPERATOR CLASS.
type CreateOpClassStmt struct {
	baseStmt
	OpClassName []string
	IsDefault   bool
	DataType    *TypeName
	AmName      string
	OpFamily    []string
	Items       []*CreateOpClassItem
}

// CreateOpClassItem represents a single item in CREATE OPERATOR CLASS AS (...).
type CreateOpClassItem struct {
	baseNode
	ItemType int    // 1=OPERATOR, 2=FUNCTION, 3=STORAGE
	Name     []string
	Number   int
	OrderFamily []string
	ClassArgs   []*TypeName
	StoredType  *TypeName
}

// CreateOpFamilyStmt represents CREATE OPERATOR FAMILY name USING method.
type CreateOpFamilyStmt struct {
	baseStmt
	OpFamilyName []string
	AmName       string
}

// CreatePLangStmt represents CREATE [OR REPLACE] [TRUSTED] [PROCEDURAL] LANGUAGE name ...
type CreatePLangStmt struct {
	baseStmt
	Replace   bool
	PLName    string
	Trusted   bool
	PLHandler []string
	PLInline  []string
	PLValidator []string
}

// CreateConversionStmt represents CREATE [DEFAULT] CONVERSION name FOR 'src' TO 'dst' FROM func.
type CreateConversionStmt struct {
	baseStmt
	ConvName    []string
	ForEncoding string
	ToEncoding  string
	FuncName    []string
	IsDefault   bool
}
