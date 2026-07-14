package lexer

import "fmt"

type TokenKind int

const (
	EOF         TokenKind = iota
	NUMBER                // 0-9
	STRING                // ""
	INDENTIFIER           //a-zA-Z
	// brackets
	OPEN_BRACKET  // [
	CLOSE_BRACKET // ]
	OPEN_CURLY    // {
	CLOSE_CURLY   //}
	OPEN_PAREN    // (
	CLOSE_PAREN   // )
	// assugment
	ASSIGNMENT // =
	// maths, comparison,
	EQUALS         // ==
	NOT            // !
	NOT_EQUALS     // !=
	LESS           // <
	LESS_EQUALS    // <=
	GREATER        // >
	GREATER_EQUALS // >=
	// logical
	OR      // |
	AND     // &
	XOR     // ^
	DOT     // .
	DOT_DOT // ..

	SEMI_COLON // ,
	COLON      // :
	QUESTION   // ?
	COMMA      // ,

	//uninary
	PLUS_PLUS   // ++
	MINUS_MINUS // --

	//maths operator
	PLUS_EQUALS  // +=
	MINUS_EQUALS // -=

	NULLISH_ASSIGNMENT

	SLASH_EQUALS // /=
	STAR_EQUALS  // *=

	//maths operator
	PLUS     // +
	DASH     // -
	SLASH    // /
	STAR     // *
	PERCENT  // %
	DOLLAR   // $
	HASH     // #
	AT       // @
	TILDE    // ~
	BACKTICK // `
	SINGLE_QUOTE // '
	// Reserved keyyord
	LET
	CONST
	CLASS
	NEW
	IMPORT
	FROM
	FN
	IF
	ELSE
	FOREACH
	WHILE
	FOR
	EXPORT
	TYPEOF
	IN
	STRUCT
	STATIC
)

var reserved_lu map[string]TokenKind = map[string]TokenKind{
	"let":     LET,
	"const":   CONST,
	"class":   CLASS,
	"new":     NEW,
	"import":  IMPORT,
	"from":    FROM,
	"fn":      FN,
	"if":      IF,
	"else":    ELSE,
	"foreach": FOREACH,
	"while":   WHILE,
	"for":     FOR,
	"export":  EXPORT,
	"typeof":  TYPEOF,
	"in":      IN,
	"struct":  STRUCT,
	"static":  STATIC,
}

func (t TokenKind) ToString() string {
	kindInfo := []string{
		"EOF ",
		"NUMBER",
		"STRING",
		"INDENTIFIER",
		"OPEN_BRACKET",
		"CLOSE_BRACKET",
		"OPEN_CURLY",
		"CLOSE_CURLY",
		"OPEN_PAREN",
		"CLOSE_PAREN",
		"ASSIGNMENT",
		"EQUALS",
		"NOT",
		"NOT_EQUALS",
		"LESS",
		"LESS_EQUALS",
		"GREATER",
		"GREATER_EQUALS",
		"OR",
		"AND",
		"XOR",
		"DOT",
		"DOT_DOT",
		"SEMI_COLON",
		"COLON",
		"QUESTION",
		"COMMA",
		"PLUS_PLUS",
		"MINUS_MINUS",
		"PLUS_EQUALS",
		"MINUS_EQUALS",
		"NULLISH_ASSIGNMENT",
		"SLASH_EQUALS",
		"STAR_EQUALS",
		"PLUS",
		"DASH",
		"SLASH",
		"STAR",
		"PERCENT",
		"DOLLAR",
		"HASH",
		"AT",
		"TILDE",
		"BACKTICK",
		"SINGLE_QUOTE",
		"LET",
		"CONST",
		"CLASS",
		"NEW",
		"IMPORT",
		"FROM",
		"FN",
		"IF",
		"ELSE",
		"FOREACH",
		"WHILE",
		"FOR",
		"EXPORT",
		"TYPEOF",
		"IN",
		"STRUCT",
		"STATIC",
	}
	return kindInfo[t]
}

type Token struct {
	File  string
	Line  int
	Col   int
	Kind  TokenKind
	Value string
}

func (t Token) Debug() {
	fmt.Printf("File: 	%s\n"+
		"Line: 	%d\n"+
		"Col:  	%d\n"+
		"Kind:   %s\n"+
		"Value:  %s\n",
		t.File,
		t.Line,
		t.Col,
		t.Kind.ToString(),
		t.Value,
	)
	fmt.Println()
}
