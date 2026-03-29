// Package pgscan implements a hand-written lexical scanner for PostgreSQL SQL,
// modeled after the Go compiler's scanner architecture but targeting the
// lexical grammar defined in src/backend/parser/scan.l of PostgreSQL.
package scanner

type Token uint

//go:generate stringer -type Token -linecomment tokens.go

const (
	_ Token = iota

	EOF // EOF

	// Identifiers and literals
	IDENT   // IDENT
	UIDENT  // UIDENT
	FCONST  // FCONST
	SCONST  // SCONST
	USCONST // USCONST
	BCONST  // BCONST
	XCONST  // XCONST
	ICONST  // ICONST
	PARAM   // PARAM
	Op      // Op

	// Multi-character fixed tokens
	TYPECAST       // ::
	DOT_DOT        // ..
	COLON_EQUALS   // :=
	EQUALS_GREATER // =>
	LESS_EQUALS    // <=
	GREATER_EQUALS // >=
	NOT_EQUALS     // !=
	RIGHT_ARROW    // ->
	LESS_GREATER   // <>

	// Keywords (placeholder start; actual keyword tokens are looked up
	// from the keyword table and returned as IDENT if unreserved, or
	// as the keyword's own token if reserved).
	// We represent all keywords with a single token type; the parser
	// distinguishes them via the literal string.
	KEYWORD // KEYWORD

	// Single-character self tokens are returned as their rune value
	// cast to Token. They are not enumerated here.

	tokenCount //
)

// LitKind describes the kind of a literal token.
type LitKind uint8

const (
	IntLit    LitKind = iota // integer literal (decimal, hex, octal, binary)
	FloatLit                 // numeric/real literal
	StringLit                // standard or extended string literal
	BitLit                   // bit string literal (B'...')
	HexLit                   // hex string literal (X'...')
	DollarLit                // dollar-quoted string literal
	UStringLit               // Unicode string literal (U&'...')
	UIdentLit                // Unicode identifier literal (U&"...")
	IdentLit                 // delimited (double-quoted) identifier
)

// KeywordCategory mirrors PostgreSQL's keyword classification.
type KeywordCategory uint8

const (
	UnreservedKeyword   KeywordCategory = iota // can always be used as identifier
	ColNameKeyword                             // can be column name
	TypeFuncNameKeyword                        // can be type or function name
	ReservedKeyword                            // fully reserved
)
