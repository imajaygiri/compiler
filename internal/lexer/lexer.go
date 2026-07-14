package lexer

import (
	"github.com/imajaygiri/compiler/helper"
	"github.com/imajaygiri/compiler/internal/utils"
)

type Lexer struct {
	filename string
	source   string
	pos      int
	Col      int
	Line     int
	tokens   []Token
}

func New(source string, filename string) *Lexer {
	if len(source) == 0 {
		utils.Panic("source is empty\n")
	}
	return &Lexer{
		filename: filename,
		source:   source,
		pos:      0,
		Col:      0,
		Line:     0,
		tokens:   make([]Token, 0),
	}
}

func (l *Lexer) IsEof() bool {
	return l.pos == len(l.source)
}

func (l *Lexer) Cursor() byte {
	if l.IsEof() {
		utils.Panic("cursor is at EOF.")
	}
	return l.source[l.pos]
}

func (l *Lexer) Advance() byte {
	if l.IsEof() {
		utils.Panic("cursor is at EOF, can not Advance.")
	}

	if helper.IsNewLine(l.Cursor()) {
		l.Line++
		l.Col = 0
	} else {
		l.Col++
	}
	tempByte := l.source[l.pos]
	l.pos++
	return tempByte
}

func handleNumber(l *Lexer, ch byte) {
	numByte := make([]byte, 0)
	numByte = append(numByte, ch)

	col := l.Col
	for !l.IsEof() && helper.IsDigit(l.Cursor()) {
		numByte = append(numByte, l.Advance())
	}
	l.tokens = append(l.tokens, Token{
		File:  l.filename,
		Line:  l.Line,
		Col:   col,
		Kind:  NUMBER,
		Value: string(numByte),
	})
}

func handleIdentifier(l *Lexer, ch byte) {
	value := make([]byte, 0)
	value = append(value, ch)
	col := l.Col
	for !l.IsEof() && (helper.IsAlphaNum(l.Cursor()) || l.Cursor() == '_') {
		value = append(value, l.Advance())
	}

	var kind TokenKind
	reservedTokenKind, isReserved := reserved_lu[string(value)]

	if isReserved {
		kind = reservedTokenKind
	} else {
		kind = INDENTIFIER
	}

	l.tokens = append(l.tokens, Token{
		File:  l.filename,
		Line:  l.Line,
		Col:   col,
		Kind:  kind,
		Value: string(value),
	})
}

func handlerString(l *Lexer, quote byte) {
	str := make([]byte, 0)
	col := l.Col

	for !l.IsEof() && l.Cursor() != quote {
		str = append(str, l.Advance())
	}
	// eat trailing quote
	l.Advance()
	l.tokens = append(l.tokens, Token{
		File:  l.filename,
		Line:  l.Line,
		Col:   col,
		Kind:  STRING,
		Value: string(str),
	})
}

func Tokenize(source string, filename string) []Token {
	lexer := New(source, filename)

	for !lexer.IsEof() {
		ch := lexer.Advance()
		switch {
		case helper.IsWhiteSpace(ch):
			continue
		case helper.IsDigit(ch):
			// handlerNumber
			handleNumber(lexer, ch)
		case helper.IsAlpha(ch) || ch == '_':
			handleIdentifier(lexer, ch)

		case ch == '"' || ch == '\'' || ch == '`':
			handlerString(lexer, ch)

			// ------------------------------------
			// 		All bracket kinds
			// ------------------------------------

		case ch == '[':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  OPEN_BRACKET,
				Value: "[",
			})

		case ch == ']':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  CLOSE_BRACKET,
				Value: "]",
			})

		case ch == '{':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  OPEN_CURLY,
				Value: "{",
			})

		case ch == '}':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  CLOSE_CURLY,
				Value: "}",
			})

		case ch == '(':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  OPEN_PAREN,
				Value: "(",
			})
		case ch == ')':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  CLOSE_PAREN,
				Value: ")",
			})

			// ------------------------------------
			// 		equality and one ASSIGNMENT
			// ------------------------------------
		case ch == '>':
			if !lexer.IsEof() && lexer.Cursor() == '=' {
				col := lexer.Col
				lexer.Advance()

				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   col,
					Kind:  GREATER_EQUALS,
					Value: ">=",
				})
			} else {

				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   lexer.Col,
					Kind:  GREATER,
					Value: string(ch),
				})
			}

		case ch == '!':
			if !lexer.IsEof() && lexer.Cursor() == '=' {
				col := lexer.Col
				lexer.Advance()
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   col,
					Kind:  NOT_EQUALS,
					Value: "!=",
				})
			} else {
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   lexer.Col,
					Kind:  NOT,
					Value: string(ch),
				})
			}
		case ch == '<':
			if !lexer.IsEof() && lexer.Cursor() == '=' {
				col := lexer.Col
				lexer.Advance()

				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   col,
					Kind:  LESS_EQUALS,
					Value: "<=",
				})
			} else {
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   lexer.Col,
					Kind:  LESS,
					Value: string(ch),
				})
			}
		case ch == '=':
			col := lexer.Col
			if !lexer.IsEof() && lexer.Cursor() == '=' {
				// eat =
				lexer.Advance()
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   col,
					Kind:  EQUALS,
					Value: "==",
				})
			} else {
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   lexer.Col,
					Kind:  ASSIGNMENT,
					Value: string(ch),
				})
			}
			// ------------------------------------
			// 		Maths Operator +, +=, -, -=, *, *=, /, */, %, %=
			// ------------------------------------
		case ch == '+':
			col := lexer.Col
			if !lexer.IsEof() && lexer.Cursor() == '+' {
				lexer.Advance()
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   col,
					Kind:  PLUS_PLUS,
					Value: "++",
				})
			} else if !lexer.IsEof() && lexer.Cursor() == '=' {
				lexer.Advance()
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   col,
					Kind:  PLUS_EQUALS,
					Value: "+=",
				})
			} else {
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   lexer.Col,
					Kind:  PLUS,
					Value: string(ch),
				})
			}

		case ch == '-':
			col := lexer.Col
			if !lexer.IsEof() && lexer.Cursor() == '-' {
				lexer.Advance()
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   col,
					Kind:  MINUS_MINUS,
					Value: "--",
				})
			} else if !lexer.IsEof() && lexer.Cursor() == '=' {
				lexer.Advance()
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   col,
					Kind:  MINUS_EQUALS,
					Value: "-=",
				})
			} else {
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   lexer.Col,
					Kind:  DASH,
					Value: string(ch),
				})
			}
		case ch == '*':
			if !lexer.IsEof() && lexer.Cursor() == '=' {
				col := lexer.Col
				lexer.Advance()
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   col,
					Kind:  STAR_EQUALS,
					Value: "*=",
				})
			} else {
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   lexer.Col,
					Kind:  STAR,
					Value: string(ch),
				})
			}
		case ch == '%':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  PERCENT,
				Value: string(ch),
			})

		case ch == '/':
			if !lexer.IsEof() && lexer.Cursor() == '=' {
				col := lexer.Col
				lexer.Advance()
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   col,
					Kind:  SLASH_EQUALS,
					Value: "/=",
				})
			} else {
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   lexer.Col,
					Kind:  SLASH,
					Value: string(ch),
				})
			}

		// ------------------------------------
		// 		logical operator & , | and conditonal &&, ||
		// ------------------------------------

		case ch == '&':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  AND,
				Value: string(ch),
			})

		case ch == '|':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  OR,
				Value: string(ch),
			})

			// ------------------------------------
			// 		MISC
			// ------------------------------------
		case ch == ';':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  SEMI_COLON,
				Value: string(ch),
			})
		case ch == ':':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  COLON,
				Value: string(ch),
			})
		case ch == '?':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  QUESTION,
				Value: string(ch),
			})
		case ch == '.':
			if !lexer.IsEof() && lexer.Cursor() == '.' {
				col := lexer.Col
				lexer.Advance()
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   col,
					Kind:  DOT_DOT,
					Value: "..",
				})
			} else {
				lexer.tokens = append(lexer.tokens, Token{
					File:  lexer.filename,
					Line:  lexer.Line,
					Col:   lexer.Col,
					Kind:  DOT,
					Value: string(ch),
				})
			}
		case ch == ',':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  COMMA,
				Value: string(ch),
			})
		case ch == '$':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  DOLLAR,
				Value: string(ch),
			})
		case ch == '#':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  HASH,
				Value: string(ch),
			})
		case ch == '@':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  AT,
				Value: string(ch),
			})
		case ch == '~':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  TILDE,
				Value: string(ch),
			})
		case ch == '^':
			lexer.tokens = append(lexer.tokens, Token{
				File:  lexer.filename,
				Line:  lexer.Line,
				Col:   lexer.Col,
				Kind:  XOR,
				Value: string(ch),
			})
		}

	}

	return lexer.tokens
}
