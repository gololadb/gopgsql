package parser

// Re-export scanner types so parser files can use them unqualified.
// This avoids adding an import to every file in the package.

import "github.com/jespino/gopgsql/scanner"

// Type aliases
type Token = scanner.Token
type LitKind = scanner.LitKind
type KeywordCategory = scanner.KeywordCategory

// Constants — re-exported via variables that are compile-time constants.
const (
	EOF            = scanner.EOF
	IDENT          = scanner.IDENT
	UIDENT         = scanner.UIDENT
	FCONST         = scanner.FCONST
	SCONST         = scanner.SCONST
	USCONST        = scanner.USCONST
	BCONST         = scanner.BCONST
	XCONST         = scanner.XCONST
	ICONST         = scanner.ICONST
	PARAM          = scanner.PARAM
	Op             = scanner.Op
	TYPECAST       = scanner.TYPECAST
	DOT_DOT        = scanner.DOT_DOT
	COLON_EQUALS   = scanner.COLON_EQUALS
	EQUALS_GREATER = scanner.EQUALS_GREATER
	LESS_EQUALS    = scanner.LESS_EQUALS
	GREATER_EQUALS = scanner.GREATER_EQUALS
	NOT_EQUALS     = scanner.NOT_EQUALS
	RIGHT_ARROW    = scanner.RIGHT_ARROW
	LESS_GREATER   = scanner.LESS_GREATER
	KEYWORD        = scanner.KEYWORD

	ReservedKeyword     = scanner.ReservedKeyword
	ColNameKeyword      = scanner.ColNameKeyword
	UnreservedKeyword   = scanner.UnreservedKeyword
	TypeFuncNameKeyword = scanner.TypeFuncNameKeyword
)

// LookupKeyword re-exports the scanner's keyword lookup.
var LookupKeyword = scanner.LookupKeyword
