package parser

import "strings"

// parseCreateStmt dispatches CREATE TABLE, CREATE TABLE AS, etc.
func (p *Parser) parseCreateStmt() Stmt {
	p.wantKeyword("create")

	// OptTemp: TEMPORARY | TEMP | LOCAL TEMPORARY | LOCAL TEMP | GLOBAL TEMPORARY | GLOBAL TEMP | UNLOGGED
	persistence := RELPERSISTENCE_PERMANENT
	switch {
	case p.gotKeyword("temporary"), p.gotKeyword("temp"):
		persistence = RELPERSISTENCE_TEMP
	case p.gotKeyword("local"):
		if p.gotKeyword("temporary") || p.gotKeyword("temp") {
			persistence = RELPERSISTENCE_TEMP
		}
	case p.gotKeyword("global"):
		if p.gotKeyword("temporary") || p.gotKeyword("temp") {
			persistence = RELPERSISTENCE_TEMP
		}
	case p.gotKeyword("unlogged"):
		persistence = RELPERSISTENCE_UNLOGGED
	}

	if p.isKeyword("table") {
		return p.parseCreateTable(persistence)
	}

	// CREATE [UNIQUE] INDEX
	if p.isKeyword("unique") || p.isKeyword("index") {
		return p.parseCreateIndex(false)
	}

	// CREATE [OR REPLACE] VIEW
	if p.isKeyword("view") {
		return p.parseCreateView(persistence, false)
	}
	// CREATE FUNCTION / PROCEDURE
	if p.isAnyKeyword("function", "procedure") {
		return p.parseCreateFunction(false)
	}

	// CREATE TRIGGER
	if p.isKeyword("trigger") {
		return p.parseCreateTrigger(false)
	}

	// CREATE RULE
	if p.isKeyword("rule") {
		return p.parseCreateRule(false)
	}

	// CREATE MATERIALIZED VIEW
	if p.isKeyword("materialized") {
		return p.parseCreateMatView()
	}

	// CREATE STATISTICS
	if p.isKeyword("statistics") {
		return p.parseCreateStatistics()
	}

	// CREATE FOREIGN TABLE / CREATE FOREIGN DATA WRAPPER
	if p.isKeyword("foreign") {
		p.next()
		if p.isKeyword("table") {
			return p.parseCreateForeignTable()
		}
		// CREATE FOREIGN DATA WRAPPER
		return p.parseCreateFdw()
	}

	// CREATE SERVER
	if p.isKeyword("server") {
		return p.parseCreateServer()
	}

	// CREATE DATABASE
	if p.isKeyword("database") {
		return p.parseCreateDatabase()
	}

	// CREATE TABLESPACE
	if p.isKeyword("tablespace") {
		return p.parseCreateTablespace()
	}

	// CREATE SEQUENCE
	if p.isKeyword("sequence") {
		return p.parseCreateSequence(persistence == RELPERSISTENCE_TEMP)
	}

	// CREATE EXTENSION
	if p.isKeyword("extension") {
		return p.parseCreateExtension()
	}

	// CREATE POLICY
	if p.isKeyword("policy") {
		return p.parseCreatePolicy()
	}

	// CREATE PUBLICATION
	if p.isKeyword("publication") {
		return p.parseCreatePublication()
	}

	// CREATE SUBSCRIPTION
	if p.isKeyword("subscription") {
		return p.parseCreateSubscription()
	}

	// CREATE EVENT TRIGGER
	if p.isKeyword("event") {
		p.next()
		return p.parseCreateEventTrigger()
	}

	// CREATE ROLE / USER / GROUP / USER MAPPING
	if p.isAnyKeyword("role", "group") {
		return p.parseCreateRole()
	}
	if p.isKeyword("user") {
		p.next()
		if p.isKeyword("mapping") {
			return p.parseCreateUserMapping()
		}
		// It's CREATE USER name — parse as CREATE ROLE
		return p.parseCreateRoleAfterKeyword()
	}

	// CREATE SCHEMA
	if p.isKeyword("schema") {
		return p.parseCreateSchema()
	}

	// CREATE DOMAIN
	if p.isKeyword("domain") {
		return p.parseCreateDomain()
	}

	// CREATE TYPE
	if p.isKeyword("type") {
		return p.parseCreateType()
	}

	// CREATE AGGREGATE
	if p.isKeyword("aggregate") {
		return p.parseCreateAggregate(false)
	}

	// CREATE OPERATOR CLASS / FAMILY / bare OPERATOR
	if p.isKeyword("operator") {
		p.next()
		if p.isKeyword("class") {
			return p.parseCreateOpClass()
		}
		if p.isKeyword("family") {
			return p.parseCreateOpFamily()
		}
		// CREATE OPERATOR name (definition)
		return p.parseCreateOperator()
	}

	// CREATE TEXT SEARCH {PARSER|DICTIONARY|TEMPLATE|CONFIGURATION}
	if p.isKeyword("text") {
		p.next()
		return p.parseCreateTextSearch()
	}

	// CREATE COLLATION
	if p.isKeyword("collation") {
		return p.parseCreateCollation()
	}

	// CREATE CAST
	if p.isKeyword("cast") {
		return p.parseCreateCast()
	}

	// CREATE ACCESS METHOD
	if p.isKeyword("access") {
		p.next()
		return p.parseCreateAccessMethod()
	}

	// CREATE [DEFAULT] CONVERSION
	if p.isKeyword("default") {
		p.next()
		if p.isKeyword("conversion") {
			return p.parseCreateConversion(true)
		}
		p.syntaxError("expected CONVERSION after DEFAULT")
		return nil
	}
	if p.isKeyword("conversion") {
		return p.parseCreateConversion(false)
	}

	if p.isKeyword("or") {
		p.next()
		p.wantKeyword("replace")
		if p.isKeyword("view") {
			return p.parseCreateView(persistence, true)
		}
		if p.isAnyKeyword("function", "procedure") {
			return p.parseCreateFunction(true)
		}
		if p.isKeyword("trigger") {
			return p.parseCreateTrigger(true)
		}
		if p.isKeyword("rule") {
			return p.parseCreateRule(true)
		}
		if p.isKeyword("aggregate") {
			return p.parseCreateAggregate(true)
		}
		if p.isKeyword("transform") {
			return p.parseCreateTransform(true)
		}
		if p.isKeyword("trusted") || p.isKeyword("procedural") || p.isKeyword("language") {
			return p.parseCreateLanguage(true)
		}
		p.syntaxError("expected VIEW, FUNCTION, PROCEDURE, TRIGGER, RULE, AGGREGATE, TRANSFORM, or LANGUAGE after OR REPLACE")
		p.next()
		return nil
	}

	// CREATE TRANSFORM
	if p.isKeyword("transform") {
		return p.parseCreateTransform(false)
	}

	// CREATE [TRUSTED] [PROCEDURAL] LANGUAGE
	if p.isKeyword("trusted") || p.isKeyword("procedural") || p.isKeyword("language") {
		return p.parseCreateLanguage(false)
	}

	p.syntaxError("expected object type after CREATE")
	p.next()
	return nil
}

// parseCreateTable parses CREATE [TEMP|UNLOGGED] TABLE ...
func (p *Parser) parseCreateTable(persistence RelPersistence) Stmt {
	p.wantKeyword("table")

	ifNotExists := false
	if p.isKeyword("if") {
		p.next()
		p.wantKeyword("not")
		p.wantKeyword("exists")
		ifNotExists = true
	}

	rel := p.parseRangeVar()

	// CREATE TABLE ... AS SELECT — detect by absence of '('
	if p.isKeyword("as") {
		return p.parseCreateTableAs(rel, persistence, ifNotExists)
	}

	// CREATE TABLE ... (column_defs, constraints)
	cs := &CreateStmt{
		baseStmt:    baseStmt{baseNode{rel.Pos()}},
		Relation:    rel,
		IfNotExists: ifNotExists,
		Persistence: persistence,
	}

	p.wantSelf('(')
	if p.tok != Token(')') {
		cs.TableElts = p.parseTableElementList()
	}
	p.wantSelf(')')

	// INHERITS (parent, ...)
	if p.gotKeyword("inherits") {
		p.wantSelf('(')
		for {
			cs.InhRelations = append(cs.InhRelations, p.parseRangeVar())
			if !p.gotSelf(',') {
				break
			}
		}
		p.wantSelf(')')
	}

	// PARTITION BY { RANGE | LIST | HASH } (partition_elem, ...)
	if p.isKeyword("partition") {
		cs.PartitionSpec = p.parsePartitionSpec()
	}

	// ON COMMIT { PRESERVE ROWS | DELETE ROWS | DROP }
	if p.isKeyword("on") {
		p.next()
		p.wantKeyword("commit")
		switch {
		case p.gotKeyword("preserve"):
			p.wantKeyword("rows")
			cs.OnCommit = ONCOMMIT_PRESERVE_ROWS
		case p.gotKeyword("delete"):
			p.wantKeyword("rows")
			cs.OnCommit = ONCOMMIT_DELETE_ROWS
		case p.gotKeyword("drop"):
			cs.OnCommit = ONCOMMIT_DROP
		}
	}

	return cs
}

// parseCreateTableAs parses CREATE TABLE name AS SELECT ...
func (p *Parser) parseCreateTableAs(rel *RangeVar, persistence RelPersistence, ifNotExists bool) Stmt {
	p.wantKeyword("as")

	query := p.parseSelectStmt()

	withData := true
	if p.isKeyword("with") {
		p.next()
		if p.gotKeyword("no") {
			p.wantKeyword("data")
			withData = false
		} else {
			p.wantKeyword("data")
			withData = true
		}
	}

	return &CreateTableAsStmt{
		baseStmt:    baseStmt{baseNode{rel.Pos()}},
		Into:        &IntoClause{baseNode: baseNode{rel.Pos()}, Rel: rel},
		Query:       query,
		IfNotExists: ifNotExists,
		Persistence: persistence,
		WithData:    withData,
	}
}

// parsePartitionSpec parses: PARTITION BY { RANGE | LIST | HASH } ( partition_elem, ... )
func (p *Parser) parsePartitionSpec() *PartitionSpec {
	pos := p.pos
	p.wantKeyword("partition")
	p.wantKeyword("by")

	spec := &PartitionSpec{baseNode: baseNode{pos}}

	// RANGE is a keyword, but LIST and HASH are plain identifiers in
	// PostgreSQL's grammar, so match on the literal text.
	switch strings.ToLower(p.lit) {
	case "range", "list", "hash":
		spec.Strategy = strings.ToLower(p.lit)
		p.next()
	default:
		p.syntaxError("expected RANGE, LIST, or HASH")
		return spec
	}

	p.wantSelf('(')
	for {
		spec.PartParams = append(spec.PartParams, p.parsePartitionElem())
		if !p.gotSelf(',') {
			break
		}
	}
	p.wantSelf(')')

	return spec
}

// parsePartitionElem parses a single partition key element:
//   column_name [COLLATE collation] [opclass]
//   ( expression ) [COLLATE collation] [opclass]
func (p *Parser) parsePartitionElem() *PartitionElem {
	pos := p.pos
	elem := &PartitionElem{baseNode: baseNode{pos}}

	if p.tok == Token('(') {
		// Expression in parentheses.
		p.next()
		elem.Expr = p.parseExpr()
		p.wantSelf(')')
	} else {
		// Column name.
		elem.Name = p.colId()
	}

	// Optional COLLATE collation.
	if p.gotKeyword("collate") {
		elem.Collation = p.parseQualifiedName()
	}

	// Optional operator class (an identifier, possibly qualified).
	// Only consume if the next token looks like an identifier and not
	// a keyword that ends the partition element list.
	if p.tok == IDENT && !p.isAnyKeyword(")", ",") {
		elem.OpClass = p.parseQualifiedName()
	}

	return elem
}

// parseTableElementList parses a comma-separated list of column defs and table constraints.
func (p *Parser) parseTableElementList() []Node {
	var elts []Node
	elts = append(elts, p.parseTableElement())
	for p.gotSelf(',') {
		elts = append(elts, p.parseTableElement())
	}
	return elts
}

// parseTableElement parses a single column definition, table constraint, or LIKE clause.
func (p *Parser) parseTableElement() Node {
	pos := p.pos

	// LIKE source_table
	if p.gotKeyword("like") {
		rv := p.parseRangeVar()
		return &TableLikeClause{baseNode: baseNode{pos}, Relation: rv}
	}

	// Table constraint: CONSTRAINT name ... | CHECK | UNIQUE | PRIMARY KEY | FOREIGN KEY | EXCLUDE
	if p.isKeyword("constraint") || p.isKeyword("check") || p.isKeyword("unique") ||
		p.isKeyword("primary") || p.isKeyword("foreign") || p.isKeyword("exclude") {
		return p.parseTableConstraint()
	}

	// Column definition: colname typename [constraints...]
	return p.parseColumnDef()
}

// parseColumnDef parses: ColId Typename [column_constraints...]
func (p *Parser) parseColumnDef() *ColumnDef {
	pos := p.pos
	cd := &ColumnDef{baseNode: baseNode{pos}}
	cd.Colname = p.colId()
	cd.TypeName = p.parseTypeName()

	// Column constraints
	for {
		c := p.parseColConstraint()
		if c == nil {
			break
		}
		cd.Constraints = append(cd.Constraints, c)
	}

	return cd
}

// parseColConstraint parses a single column constraint, or returns nil if none.
func (p *Parser) parseColConstraint() *Constraint {
	pos := p.pos

	// CONSTRAINT name
	var conname string
	if p.gotKeyword("constraint") {
		conname = p.colId()
	}

	c := p.parseColConstraintElem()
	if c == nil {
		// If we consumed CONSTRAINT name but no constraint follows, that's an error.
		// In practice this shouldn't happen with valid SQL.
		if conname != "" {
			p.syntaxError("expected constraint after CONSTRAINT name")
		}
		return nil
	}
	c.Conname = conname
	c.Location = pos
	return c
}

// parseColConstraintElem parses the actual constraint keyword.
func (p *Parser) parseColConstraintElem() *Constraint {
	pos := p.pos

	switch {
	case p.isKeyword("not"):
		p.next()
		p.wantKeyword("null")
		return &Constraint{baseNode: baseNode{pos}, Contype: CONSTR_NOTNULL}

	case p.isKeyword("null"):
		p.next()
		return &Constraint{baseNode: baseNode{pos}, Contype: CONSTR_NULL}

	case p.isKeyword("default"):
		p.next()
		p.inColDefault = true
		expr := p.parseExpr()
		p.inColDefault = false
		return &Constraint{baseNode: baseNode{pos}, Contype: CONSTR_DEFAULT, RawExpr: expr}

	case p.isKeyword("check"):
		p.next()
		p.wantSelf('(')
		expr := p.parseExpr()
		p.wantSelf(')')
		return &Constraint{baseNode: baseNode{pos}, Contype: CONSTR_CHECK, RawExpr: expr}

	case p.isKeyword("primary"):
		p.next()
		p.wantKeyword("key")
		return &Constraint{baseNode: baseNode{pos}, Contype: CONSTR_PRIMARY}

	case p.isKeyword("unique"):
		p.next()
		c := &Constraint{baseNode: baseNode{pos}, Contype: CONSTR_UNIQUE}
		if p.gotKeyword("nulls") {
			p.wantKeyword("not")
			p.wantKeyword("distinct")
			c.NullsNotDistinct = true
		}
		return c

	case p.isKeyword("references"):
		p.next()
		c := &Constraint{baseNode: baseNode{pos}, Contype: CONSTR_FOREIGN}
		c.PkTable = p.parseRangeVar()
		if p.tok == Token('(') {
			p.next()
			c.PkAttrs = p.parseNameList()
			p.wantSelf(')')
		}
		p.parseFKActions(c)
		return c

	case p.isKeyword("generated"):
		p.next()
		c := &Constraint{baseNode: baseNode{pos}}
		if p.gotKeyword("always") {
			if p.isKeyword("as") {
				p.next()
				if p.isKeyword("identity") {
					// GENERATED ALWAYS AS IDENTITY
					p.next()
					c.Contype = CONSTR_IDENTITY
				} else {
					// GENERATED ALWAYS AS (expr) STORED
					p.wantSelf('(')
					c.RawExpr = p.parseExpr()
					p.wantSelf(')')
					p.wantKeyword("stored")
					c.Contype = CONSTR_GENERATED
				}
			}
		} else if p.gotKeyword("by") {
			p.wantKeyword("default")
			p.wantKeyword("as")
			p.wantKeyword("identity")
			c.Contype = CONSTR_IDENTITY
		}
		return c

	case p.isKeyword("collate"):
		// COLLATE is not really a constraint but appears in column qualifiers.
		// We skip it here and let the caller handle it.
		return nil

	case p.isKeyword("deferrable"):
		p.next()
		c := &Constraint{baseNode: baseNode{pos}, Deferrable: true}
		if p.gotKeyword("initially") {
			if p.gotKeyword("deferred") {
				c.InitDeferred = true
			} else {
				p.wantKeyword("immediate")
			}
		}
		return c
	}

	return nil
}

// parseTableConstraint parses a table-level constraint.
func (p *Parser) parseTableConstraint() *Constraint {
	pos := p.pos

	var conname string
	if p.gotKeyword("constraint") {
		conname = p.colId()
	}

	c := p.parseConstraintElem()
	c.Conname = conname
	c.Location = pos
	return c
}

// parseConstraintElem parses CHECK, UNIQUE, PRIMARY KEY, FOREIGN KEY, EXCLUDE.
func (p *Parser) parseConstraintElem() *Constraint {
	pos := p.pos

	switch {
	case p.isKeyword("check"):
		p.next()
		p.wantSelf('(')
		expr := p.parseExpr()
		p.wantSelf(')')
		c := &Constraint{baseNode: baseNode{pos}, Contype: CONSTR_CHECK, RawExpr: expr}
		p.parseConstraintAttrs(c)
		return c

	case p.isKeyword("unique"):
		p.next()
		c := &Constraint{baseNode: baseNode{pos}, Contype: CONSTR_UNIQUE}
		if p.gotKeyword("nulls") {
			p.wantKeyword("not")
			p.wantKeyword("distinct")
			c.NullsNotDistinct = true
		}
		p.wantSelf('(')
		c.Keys = p.parseNameList()
		p.wantSelf(')')
		p.parseConstraintAttrs(c)
		return c

	case p.isKeyword("primary"):
		p.next()
		p.wantKeyword("key")
		p.wantSelf('(')
		c := &Constraint{baseNode: baseNode{pos}, Contype: CONSTR_PRIMARY}
		c.Keys = p.parseNameList()
		p.wantSelf(')')
		p.parseConstraintAttrs(c)
		return c

	case p.isKeyword("foreign"):
		p.next()
		p.wantKeyword("key")
		p.wantSelf('(')
		c := &Constraint{baseNode: baseNode{pos}, Contype: CONSTR_FOREIGN}
		c.FkAttrs = p.parseNameList()
		p.wantSelf(')')
		p.wantKeyword("references")
		c.PkTable = p.parseRangeVar()
		if p.tok == Token('(') {
			p.next()
			c.PkAttrs = p.parseNameList()
			p.wantSelf(')')
		}
		p.parseFKActions(c)
		p.parseConstraintAttrs(c)
		return c

	case p.isKeyword("exclude"):
		p.next()
		// Simplified: skip EXCLUDE details for now
		c := &Constraint{baseNode: baseNode{pos}, Contype: CONSTR_EXCLUSION}
		// Skip until we find a closing context
		return c
	}

	p.syntaxError("expected CHECK, UNIQUE, PRIMARY KEY, FOREIGN KEY, or EXCLUDE")
	return &Constraint{baseNode: baseNode{pos}}
}

// parseFKActions parses ON UPDATE/DELETE actions for foreign keys.
func (p *Parser) parseFKActions(c *Constraint) {
	for p.isKeyword("on") {
		p.next()
		var action *string
		if p.gotKeyword("update") {
			action = &c.FkUpdAction
		} else if p.gotKeyword("delete") {
			action = &c.FkDelAction
		} else {
			break
		}
		switch {
		case p.gotKeyword("cascade"):
			*action = "CASCADE"
		case p.gotKeyword("restrict"):
			*action = "RESTRICT"
		case p.isKeyword("set"):
			p.next()
			if p.gotKeyword("null") {
				*action = "SET NULL"
			} else if p.gotKeyword("default") {
				*action = "SET DEFAULT"
			}
		case p.isKeyword("no"):
			p.next()
			p.wantKeyword("action")
			*action = "NO ACTION"
		}
	}

	// MATCH FULL | MATCH PARTIAL | MATCH SIMPLE
	if p.gotKeyword("match") {
		switch {
		case p.gotKeyword("full"):
			c.FkMatchType = "FULL"
		case p.gotKeyword("partial"):
			c.FkMatchType = "PARTIAL"
		case p.gotKeyword("simple"):
			c.FkMatchType = "SIMPLE"
		}
	}
}

// parseConstraintAttrs parses DEFERRABLE / NOT DEFERRABLE / INITIALLY DEFERRED / INITIALLY IMMEDIATE.
func (p *Parser) parseConstraintAttrs(c *Constraint) {
	if p.gotKeyword("deferrable") {
		c.Deferrable = true
	} else if p.isKeyword("not") {
		// Could be NOT DEFERRABLE — but we need to be careful not to consume
		// NOT that belongs to something else. Only consume if followed by DEFERRABLE.
		// We can't peek, so skip this for now.
		return
	}
	if p.gotKeyword("initially") {
		if p.gotKeyword("deferred") {
			c.InitDeferred = true
		} else {
			p.wantKeyword("immediate")
		}
	}
}

