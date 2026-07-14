# Pratt Parsing (Expressions & Types) + Recursive Descent (Statements & Declarations) — Compiler Builder's Notes

> POV: you're building a compiler front-end in Go, targeting LLVM later, own backend eventually.
> These notes are written so you can implement, not just understand. Every module has a
> practice exercise. Do them — reading a parser and writing one are different skills entirely.

---

## Module 0 — Why You Can't Just Recursive-Descent Your Way Through Expressions

Naive grammar for expressions:

```
expr := expr '+' expr
      | expr '*' expr
      | NUMBER
```

This is ambiguous (no precedence baked in) and **left-recursive** — if you write it as a
straightforward recursive descent function:

```go
func parseExpr() Node {
    left := parseExpr() // <-- infinite recursion, never consumes a token
    ...
}
```

it recurses forever before consuming anything. You *can* fix left recursion by rewriting the
grammar into a precedence-ladder form:

```
expr   := term (('+' | '-') term)*
term   := factor (('*' | '/') factor)*
factor := NUMBER | '(' expr ')'
```

This works and is what a lot of intro-compiler-course parsers do. **The problem**: every
precedence level is a separate function. Add exponentiation, ternary, assignment, bitwise
ops, member access, casts — you now have 10-15 nested functions, each hardcoding one
precedence level and its associativity. It works but doesn't scale and is a pain to extend
or refactor. That's the actual motivation for Pratt parsing: **collapse all those levels
into one function, driven by a table of numbers.**

---

## Module 1 — Core Vocabulary (get these exact, they get conflated constantly)

| Term | Meaning |
|---|---|
| **nud** (null denotation) | How to parse a token when it appears in **prefix position** — i.e. nothing to its left yet. Numbers, identifiers, unary `-`, `(`, `!`. |
| **led** (left denotation) | How to parse a token when it appears in **infix/postfix position** — i.e. there IS a left-hand expression already parsed. `+`, `*`, postfix `++`, `(` as a call, `[` as indexing. |
| **lbp** (left binding power) | How strongly this token binds to the expression on its **left**. This is what decides "does the loop continue and let this operator grab the `lhs` we already have?" |
| **rbp** (right binding power) | The **minimum bp threshold** passed down when parsing this token's right operand. Controls associativity. |
| **min_bp** (the parser argument) | Not a property of a token — it's a threshold passed into `parse_expr`, meaning "stop as soon as you see an operator whose lbp is below this." |

**The single biggest confusion point**: lbp and rbp are usually *equal or off by one from
each other for the same operator*, and people mix up "the operator's own bp" with "the bp
threshold passed to the recursive call." Keep these mentally separate:

- lbp = "should the loop in the CALLER grab me?"
- rbp = "what threshold do I hand to my own right-hand recursive parse?"

---

## Module 2 — The Algorithm

```go
// Pseudocode form first, Go implementation follows in Module 9.

func parseExpr(minBP int) Node {
    lhs := parseNud() // consume a prefix token: number, ident, '(', unary op

    for {
        op := peekToken()
        if !isInfix(op) || lbp(op) < minBP {
            break
        }
        advance() // consume operator
        rhs := parseExpr(rbp(op))
        lhs = combine(lhs, op, rhs)
    }
    return lhs
}
```

That's it. That's the whole engine. Every precedence level, every associativity rule, every
prefix/postfix/mixfix operator is expressed as **data** (a table of lbp/rbp/nud/led per
token type), not as more functions.

### Binding power table (typical arithmetic)

| Operator | lbp | rbp | Associativity |
|---|---|---|---|
| `+` `-` (infix) | 10 | 11 | left |
| `*` `/` | 20 | 21 | left |
| `^` (power) | 30 | 29 | **right** |
| unary `-` (prefix) | — | 25 | n/a (prefix has no lbp) |

Notice the pattern:
- **Left-associative**: `rbp = lbp + 1`. This means when parsing the right operand, the
  recursive call requires the *next* operator to have STRICTLY higher lbp to keep going —
  so a same-precedence operator to the right does NOT get pulled into this rhs; it gets
  picked up by the outer loop instead. That's what makes `a - b - c` parse as `(a-b)-c`.
- **Right-associative**: `rbp = lbp - 1` (or `rbp = lbp`, some implementations use equal —
  more below). This means the recursive parse of the rhs is willing to accept an operator
  of the *same* precedence, pulling it inward — so `a ^ b ^ c` parses as `a ^ (b ^ c)`.

This one delta (`+1` vs `-1`) is 100% of how associativity is encoded. No special-casing,
no "if left-assoc then... else..." branches in the parse loop itself.

---

## Module 3 — Full Trace of `3 + 3 * 5`

Entry call: `parseExpr(0)` (min_bp = 0 means "accept anything").

```
parseExpr(0)
  lhs = parseNud() -> NUMBER(3)          // lhs = 3
  loop:
    peek = '+'        lbp('+')=10, 10 >= 0 -> continue
    advance, consumed '+'
    rhs = parseExpr(rbp('+') = 11)
        |
        parseExpr(11)
          lhs = parseNud() -> NUMBER(3)   // inner lhs = 3
          loop:
            peek = '*'   lbp('*')=20,  20 >= 11 -> continue
            advance, consumed '*'
            rhs = parseExpr(rbp('*') = 21)
                |
                parseExpr(21)
                  lhs = parseNud() -> NUMBER(5)
                  loop:
                    peek = EOF (or ',' or whatever follows) -> not infix -> break
                  return 5
            lhs = combine(3, '*', 5) = (3 * 5)
          loop again:
            peek = whatever follows (say ',' or EOF), not infix -> break
          return (3 * 5)
  lhs = combine(3, '+', (3 * 5)) = (3 + (3 * 5))
  loop again:
    peek = ',' -> lbp check: is ',' even in the infix table for expression context? -> no -> break
  return (3 + (3 * 5))
```

Final AST: `3 + (3 * 5)`. The reason `*` wins over `+` is purely numeric: when we're
inside the `+`'s rhs parse with `min_bp = 11`, we hit `*` with `lbp = 20`, and
`20 >= 11` so the loop keeps going and pulls `* 5` into the right-hand side of `+`
before `+` itself gets to close. If it had instead been `3 * 3 + 5`, entering
`parseExpr(21)` after consuming `*`, we'd hit `+` with lbp 10, and `10 < 21` -> break
immediately, so `+` is NOT absorbed into `*`'s rhs — it goes back up to the outer loop
at `min_bp=0`, which happily accepts it. That's precedence, entirely encoded as number
comparisons.

---

## Module 4 — Why the Comma Doesn't Get Included

This is exactly your question: parsing `3 + 3 * 5` "stops before comma." Two equivalent
ways to think about why:

1. **Comma is often just not in the infix operator table when you're inside a plain
   expression context.** If `,` has no lbp/led registered at all, `isInfix(',')` is false
   and the loop breaks immediately regardless of min_bp. This is the simplest and most
   common real-world choice — comma is handled one level up, by whatever's calling
   `parseExpr` (e.g. "parse a comma-separated argument list" loops and calls
   `parseExpr(0)` per item, consuming the comma itself between calls).

2. **Alternative: give comma an lbp, but make it the lowest in the whole table** (e.g.
   `lbp(',') = 1`). Then in a context where comma-as-operator is meaningful (C's actual
   comma operator: `a = (b, c)` evaluates b, discards it, evaluates to c), you'd call
   `parseExpr(2)` for "parse an expression that should NOT itself consume a top-level
   comma" — i.e., you set your entry threshold *above* comma's lbp. Function-argument
   parsing almost always does this (calls `parseExpr(threshold_above_comma)` per
   argument) specifically SO that `f(3+3*5, 7)` doesn't get eaten as one massive
   comma-expression covering both arguments.

Either way, the mental model is: **comma is context-dependent and deliberately kept at
the boundary of your binding-power table so it never silently swallows adjacent
expressions.** Most language grammars (C, Rust, Go, TS) pick option 1 for arg lists —
comma isn't a general expression operator at all, it's list-separator syntax handled by
the surrounding grammar rule, not by the Pratt loop.

**Practice exercise 4.1**: Implement both options in the Go skeleton from Module 9 and
write test cases showing `f(1+2, 3*4)` parses as two separate args either way — then
break option 2 deliberately by setting the wrong threshold and see the bug it causes
(hint: it'll merge args into one comma-expression).

---

## Module 5 — Prefix and Postfix Operators

Prefix operators (`-x`, `!x`, `*ptr` in C, `&x`) have **no lbp** — they never appear after
an already-parsed lhs, they only appear inside `parseNud`. They DO have an rbp: how tightly
they bind their own operand.

```go
case '-': // unary minus
    advance()
    operand := parseExpr(unaryRBP) // unaryRBP typically higher than all binary ops
    return UnaryNode{Op: "-", Operand: operand}
```

Why does `-3 * 5` need unary rbp to be HIGH (higher than `*`'s lbp)? Because you want
`-3 * 5` to parse as `(-3) * 5`, not `-(3 * 5)`. If unary rbp were lower than `*`'s lbp,
the operand-parse for `-` would greedily absorb the `* 5` too.

Postfix operators (`x++`, `x!` factorial, array index `a[i]`, function call `f(x)`, member
access `a.b`) DO have an lbp (they get grabbed by the loop like normal infix ops) but their
"rhs parse" is different — often there's no rhs at all (`x++`), or the rhs is a bracketed
sub-grammar (`[i]`, `(args)`), not a recursive `parseExpr` call:

```go
case '[': // postfix array index - registered as a led
    advance()
    index := parseExpr(0)  // reset to 0 inside brackets - the bracket delimits
    expect(']')
    return IndexNode{Base: lhs, Index: index}

case '(': // postfix function call
    advance()
    args := parseArgList() // comma-loop, see Module 4
    expect(')')
    return CallNode{Callee: lhs, Args: args}
```

Notice: inside `[...]` and `(...)`, binding power **resets to 0** because the brackets
themselves are the delimiter — you don't need the outer min_bp to protect you once you're
inside an explicit close-token boundary.

**Practice exercise 5.1**: add postfix `++`/`--` (no operand, lbp very high, roughly
member-access/call level) and verify `-x++` parses sensibly (should be `-(x++)`, since
postfix binds tighter than unary prefix in C-family languages — verify this against C's
actual precedence table and explain in your own notes why that ordering makes sense).

---

## Module 6 — Ternary / Mixfix Operators (`a ? b : c`)

Ternary doesn't fit nud/led cleanly because it has THREE operands and two delimiter
tokens (`?` and `:`). Handle it as a special led for `?`:

```go
case '?': // led for ternary, right-associative overall
    advance()
    thenBranch := parseExpr(0)   // reset to 0: '?' ... ':' is fully bracketed by the ':'
    expect(':')
    elseBranch := parseExpr(ternaryRBP) // low rbp -> right-associative chaining
    return TernaryNode{Cond: lhs, Then: thenBranch, Else: elseBranch}
```

`ternaryRBP` is usually set so `a ? b : c ? d : e` parses as `a ? b : (c ? d : e)`
(right-associative chaining), matching C/JS/Go-family behavior.

**Practice exercise 6.1**: implement this and write a trace (like Module 3) for
`a ? b : c ? d : e` by hand before running your code, then verify your code matches
your hand-trace. If it doesn't, the bug is almost always in `ternaryRBP` vs `?`'s lbp.

---

## Module 7 — Assignment and Right-Associativity in Practice

`a = b = c` should parse as `a = (b = c)`. Assignment as a led:

```go
case '=':
    advance()
    rhs := parseExpr(rbp('=')) // rbp = lbp (not lbp+1!) -> right-assoc
    return AssignNode{Target: lhs, Value: rhs}
```

Here `rbp('=') == lbp('=')` (not lbp - 1) is a common convention too — as long as
`rbp <= lbp`, same-precedence chaining to the right is allowed, which is right-assoc.
The exact choice of `lbp-1` vs `lbp` only matters if you need to distinguish "equal
precedence, right-assoc" from something with subtly different rebinding behavior at the
boundary — for straightforward right-assoc operators, `rbp = lbp` is simpler and equally
correct. Pick one convention and stay consistent inside a single table.

---

## Module 8 — Type Parsing: Rust-style and TypeScript-style Types

This is the part that trips people up because **type-grammars have their own precedence
hierarchy, entirely separate from the expression grammar**, but the exact same nud/led/bp
machinery applies. You are literally instantiating a second Pratt parser with a different
token table.

### 8.1 — Why types need Pratt parsing at all

TypeScript:
```
type X = A | B & C;        // & binds tighter than |, same idea as * vs +
type Y = (A | B)[];        // array is postfix, very high bp
type Z = A extends B ? C : D;  // ternary-style conditional type - literally reuse Module 6
type W = keyof A;          // prefix operator
type V = { [K in Keys]: T[K] };  // mapped type - separate sub-grammar, not Pratt
```

Rust:
```
type X = &'a mut T;        // & is prefix, with lifetime + mut modifiers stacked
type Y = dyn Trait + Send; // '+' between trait bounds - infix, low bp
type Z = Box<dyn Trait>;   // generics <...> - like a bracketed postfix, reset bp inside
type W = fn(i32) -> bool;  // function type - prefix keyword introducing a sub-grammar
type V = [T; N];           // array with const generic length - bracketed postfix
```

### 8.2 — Type binding power table (a reasonable design, TS-flavored)

| Type-token | lbp | rbp | Position | Notes |
|---|---|---|---|---|
| `IDENT` / base type | — | — | nud | leaf |
| `keyof`, `typeof` (TS) | — | 50 | nud (prefix) | binds very tight to its operand |
| `&` (Rust reference) | — | 50 | nud (prefix) | followed by optional lifetime + mut |
| `[` `]` (array) | 60 | — | led/postfix | `T[]`, resets to 0 inside for `[N]` const-len |
| `<...>` (generics) | 60 | — | led/postfix | resets bp to 0 inside, comma-separated |
| `&` (TS intersection) | 30 | 31 | led (infix) | left-assoc |
| `\|` (TS union) | 20 | 21 | led (infix) | left-assoc, LOWER than `&` — same relationship as `*`/`+` |
| `+` (Rust trait bounds) | 20 | 21 | led (infix) | `dyn Trait + Send` |
| `extends ... ? ... :` | 10 | 9 | led (mixfix) | conditional type, right-assoc, reuse Module 6 pattern |

The union-vs-intersection relationship (`&` binds tighter than `|`) is **structurally
identical** to `*` binding tighter than `+`. Once this clicks, type-parsing stops feeling
like a separate skill — it's the same engine with a different table.

### 8.3 — Go skeleton for a type parser

```go
type TypeKind int
const (
    TKIdent TypeKind = iota
    TKUnion
    TKIntersection
    TKArray
    TKGeneric
    TKReference
    TKFunction
)

type TypeNode struct {
    Kind     TypeKind
    Name     string      // for TKIdent
    Left     *TypeNode   // for union/intersection
    Right    *TypeNode
    Elem     *TypeNode   // for array/reference
    Args     []*TypeNode // for generics
    Params   []*TypeNode // for function types
    Ret      *TypeNode
}

var typeLBP = map[TokenType]int{
    TOK_PIPE:   20, // |
    TOK_AMP:    30, // & (intersection, infix position)
    TOK_LBRACK: 60, // [ postfix
    TOK_LT:     60, // < generics postfix
}

func (p *Parser) parseType(minBP int) *TypeNode {
    left := p.parseTypeNud()

    for {
        tok := p.peek()
        lbp, ok := typeLBP[tok.Kind]
        if !ok || lbp < minBP {
            break
        }
        switch tok.Kind {
        case TOK_PIPE:
            p.advance()
            right := p.parseType(21) // left-assoc: rbp = lbp+1
            left = &TypeNode{Kind: TKUnion, Left: left, Right: right}
        case TOK_AMP:
            p.advance()
            right := p.parseType(31)
            left = &TypeNode{Kind: TKIntersection, Left: left, Right: right}
        case TOK_LBRACK:
            p.advance()
            p.expect(TOK_RBRACK) // T[] - simple array, no length expr in this sketch
            left = &TypeNode{Kind: TKArray, Elem: left}
        case TOK_LT:
            p.advance()
            args := p.parseTypeArgList() // comma loop, resets to parseType(0) per arg
            p.expect(TOK_GT)
            left = &TypeNode{Kind: TKGeneric, Elem: left, Args: args}
        }
    }
    return left
}

func (p *Parser) parseTypeNud() *TypeNode {
    tok := p.peek()
    switch tok.Kind {
    case TOK_AMP: // Rust-style &T reference, PREFIX position here
        p.advance()
        elem := p.parseType(50)
        return &TypeNode{Kind: TKReference, Elem: elem}
    case TOK_IDENT:
        p.advance()
        return &TypeNode{Kind: TKIdent, Name: tok.Text}
    case TOK_FN: // fn(A, B) -> C
        p.advance()
        p.expect(TOK_LPAREN)
        params := p.parseTypeArgList()
        p.expect(TOK_RPAREN)
        p.expect(TOK_ARROW)
        ret := p.parseType(0)
        return &TypeNode{Kind: TKFunction, Params: params, Ret: ret}
    default:
        panic("unexpected token in type position: " + tok.Text)
    }
}

func (p *Parser) parseTypeArgList() []*TypeNode {
    var args []*TypeNode
    if p.peek().Kind == TOK_RPAREN || p.peek().Kind == TOK_GT {
        return args
    }
    args = append(args, p.parseType(0))
    for p.peek().Kind == TOK_COMMA {
        p.advance()
        args = append(args, p.parseType(0))
    }
    return args
}
```

**Note the reused pattern**: `&` appears in BOTH the nud table (prefix reference) AND
conceptually could appear as a led (Rust doesn't use `&` as intersection, but TS does) —
this is exactly why nud/led are keyed by **token + position**, not by token alone. The
same token can mean different things in prefix vs infix position; your dispatch table
needs to check "am I looking for a nud or a led right now" as well as which token it is.

**Practice exercise 8.1**: extend this to parse `Box<dyn Trait + Send>` — you'll need
`dyn` as a nud-prefix keyword and `+` as an infix at trait-bound level. Write the bp table
entry and justify the number you pick relative to the existing table.

**Practice exercise 8.2**: parse `A extends B ? C : D` (TS conditional types) by
implementing it as a mixfix led exactly like Module 6's ternary, and explain in your
notes why conditional types are right-associative in real TS (`A extends B ? C : D
extends E ? F : G`).

**Practice exercise 8.3 (harder, closer to real compilers)**: Rust's actual reference
types carry lifetimes and mutability: `&'a mut T`. Redesign `parseTypeNud`'s `&` case to
optionally consume a lifetime token and optional `mut` keyword before recursing into the
element type. This is a good exercise in "prefix operator with an optional modifier
cluster before the operand," which shows up constantly in real grammars (C's
`const volatile int*`, for instance).

---

## Module 9 — Full Runnable Go Skeleton (Arithmetic + Calls + Ternary)

```go
package main

import "fmt"

type TokKind int

const (
    TOK_NUM TokKind = iota
    TOK_PLUS
    TOK_MINUS
    TOK_STAR
    TOK_SLASH
    TOK_LPAREN
    TOK_RPAREN
    TOK_COMMA
    TOK_QUESTION
    TOK_COLON
    TOK_EOF
)

type Token struct {
    Kind TokKind
    Text string
}

// --- AST ---
type Node interface{ String() string }

type NumNode struct{ Val string }
func (n NumNode) String() string { return n.Val }

type BinNode struct {
    Op          string
    Left, Right Node
}
func (n BinNode) String() string { return fmt.Sprintf("(%s %s %s)", n.Left, n.Op, n.Right) }

type UnaryNode struct {
    Op      string
    Operand Node
}
func (n UnaryNode) String() string { return fmt.Sprintf("(%s%s)", n.Op, n.Operand) }

type TernaryNode struct{ Cond, Then, Else Node }
func (n TernaryNode) String() string {
    return fmt.Sprintf("(%s ? %s : %s)", n.Cond, n.Then, n.Else)
}

// --- Parser ---
type Parser struct {
    toks []Token
    pos  int
}

func (p *Parser) peek() Token  { return p.toks[p.pos] }
func (p *Parser) advance() Token {
    t := p.toks[p.pos]
    p.pos++
    return t
}

var lbpTable = map[TokKind]int{
    TOK_PLUS:     10,
    TOK_MINUS:    10,
    TOK_STAR:     20,
    TOK_SLASH:    20,
    TOK_QUESTION: 5,
}

func rbpFor(k TokKind) int {
    switch k {
    case TOK_PLUS, TOK_MINUS, TOK_STAR, TOK_SLASH:
        return lbpTable[k] + 1 // left-associative
    case TOK_QUESTION:
        return 4 // right-associative chaining for ternary
    }
    return 0
}

func (p *Parser) parseExpr(minBP int) Node {
    left := p.parseNud()

    for {
        tok := p.peek()
        lbp, ok := lbpTable[tok.Kind]
        if !ok || lbp < minBP {
            break
        }

        switch tok.Kind {
        case TOK_QUESTION:
            p.advance()
            thenBranch := p.parseExpr(0) // reset: delimited by ':'
            if p.peek().Kind != TOK_COLON {
                panic("expected ':' in ternary")
            }
            p.advance()
            elseBranch := p.parseExpr(rbpFor(TOK_QUESTION))
            left = TernaryNode{Cond: left, Then: thenBranch, Else: elseBranch}
        default:
            p.advance()
            right := p.parseExpr(rbpFor(tok.Kind))
            left = BinNode{Op: tok.Text, Left: left, Right: right}
        }
    }
    return left
}

func (p *Parser) parseNud() Node {
    tok := p.advance()
    switch tok.Kind {
    case TOK_NUM:
        return NumNode{Val: tok.Text}
    case TOK_MINUS: // unary minus
        operand := p.parseExpr(25) // binds tighter than * (20) and + (10)
        return UnaryNode{Op: "-", Operand: operand}
    case TOK_LPAREN:
        inner := p.parseExpr(0) // reset - the ')' is the delimiter now
        if p.peek().Kind != TOK_RPAREN {
            panic("expected ')'")
        }
        p.advance()
        return inner
    default:
        panic(fmt.Sprintf("unexpected token in prefix position: %v", tok))
    }
}

func main() {
    // Manually built token stream for: 3 + 3 * 5
    toks := []Token{
        {TOK_NUM, "3"}, {TOK_PLUS, "+"}, {TOK_NUM, "3"}, {TOK_STAR, "*"}, {TOK_NUM, "5"},
        {TOK_EOF, ""},
    }
    p := &Parser{toks: toks}
    ast := p.parseExpr(0)
    fmt.Println(ast) // -> (3 + (3 * 5))
}
```

Wire a real lexer in front of this (you already know how — reuse whatever tokenizer you
built for your compiler project) and this drives straight into your existing parser stage.

**Practice exercise 9.1**: extend this skeleton with `TOK_CARET` (`^`, exponent,
right-assoc) and a postfix `(` for function calls. Write your own trace for
`f(1, 2) ^ 2` by hand first.

---

## Module 10 — Common Bugs (you will hit these, so know them in advance)

1. **Confusing "the operator's precedence" with "the min_bp threshold argument."** They
   look like the same number but play different roles — one is a property of a token, the
   other is a parameter you pass down. Name your variables so this can't get confused
   (`minBP` for the parameter, `lbp`/`rbp` map lookups for token properties — never reuse
   a variable name across both roles).

2. **Off-by-one in right-assoc rbp.** If you write `rbp = lbp + 1` for a right-associative
   operator by copy-pasting the left-assoc pattern, you silently get left-associativity
   instead — no crash, just wrong AST shape. Always write a hand-trace test for every
   right-assoc operator you add (exponent, assignment, ternary-else).

3. **Forgetting that prefix operators have no lbp.** If you accidentally register unary
   `-` with an lbp too, your loop will try to treat it as infix in some code path and
   either crash or produce garbage. Keep prefix-only tokens entirely out of the lbp table,
   or explicitly guard `isInfixPosition` checks.

4. **Not resetting bp to 0 inside explicit delimiters.** Anything inside `(...)`, `[...]`,
   `<...>`, or between `?` and `:` should reset to `parseExpr(0)` (or the type-equivalent)
   because the delimiter itself, not the outer min_bp, protects the boundary. Forgetting
   this causes weird premature-stop bugs when a low-precedence operator appears inside
   parens.

5. **Treating comma as a normal operator when it should be list-syntax.** Covered in
   Module 4 — decide explicitly which strategy you're using and be consistent across
   your whole grammar (expr-lists, type-arg-lists, function-param-lists).

---

## Module 11 — Where This Fits in Your Actual Compiler Pipeline

1. Lexer -> tokens (you likely have this already)
2. **Pratt parser (expressions) -> AST** (this document)
3. **Pratt parser (types), separate table, same engine** -> Type AST, used wherever a type
   annotation appears in your grammar (`let x: TYPE`, function signatures, struct fields)
4. Statement-level parsing (usually plain recursive descent — statements aren't really
   "expression-shaped" the same way, so Pratt buys you less there) calls into both of the
   above wherever an expr or a type is expected
5. Symbol table -> (your stated next step)
6. IR generation -> LLVM backend / your own backend later

Given your stated plan (Go frontend, LLVM later, own backend eventually): keep expression
and type parsing as clean, separately-testable functions returning a plain AST — don't let
LLVM-specific concerns leak into the parser. You'll thank yourself when you bolt on your
own backend later and don't have to touch the parser at all.

---

## Module 12 — Why Statements Use Recursive Descent, Not Pratt

Expressions needed Pratt because the SAME grammar rule (`expr`) recurses into itself with
different precedence at different points — that's what binding power solves. Statements
don't have this problem. A statement grammar is almost always **keyword-dispatched**:
you peek one token and know immediately which rule you're in.

```
stmt := 'let'    -> letDecl
      | 'fn'     -> fnDecl
      | 'struct' -> structDecl
      | 'class'  -> classDecl
      | 'return' -> returnStmt
      | 'if'     -> ifStmt
      | 'for'    -> forStmt
      | '{'      -> block
      | otherwise -> exprStmt (covers both bare calls like `foo();` and assignment `x = y;`)
```

This is plain **LL(1) recursive descent**: one function per grammar rule, dispatch on
`peek().Kind`, no binding-power table needed, because there's no precedence ambiguity at
this level — `let` can never be confused with `fn`. Where statements MEET expressions
(initializers, conditions, array sizes, call arguments) you just drop down into
`p.parseExpr(0)` or `p.parseType(0)`, the exact functions from Modules 1-9. **This is the
architectural point**: your statement parser is a thin dispatcher that calls into the
Pratt engine wherever an expression or type is expected, and never re-implements
precedence itself.

---

## Module 13 — Grammar for Your Language (EBNF)

Based on the syntax you're targeting:

```ebnf
program     := decl*

decl        := letDecl | fnDecl | structDecl | classDecl

letDecl     := 'let' IDENT ':' type ('=' expr)? ';'

fnDecl      := 'fn' IDENT '(' paramList? ')' ('->' type)? block
paramList   := param (',' param)*
param       := IDENT ':' type

block       := '{' stmt* '}'

stmt        := letDecl
             | returnStmt
             | ifStmt
             | forStmt
             | block
             | exprStmt

returnStmt  := 'return' expr? ';'
ifStmt      := 'if' expr block ('else' (ifStmt | block))?
forStmt     := 'for' ... block            // shape depends on your language design, sketched later
exprStmt    := expr ';'                    // covers bare calls AND assignment (x = y;)

structDecl  := 'struct' IDENT '{' fieldList '}'
fieldList   := field (',' field)* ','?
field       := IDENT ':' type

classDecl   := 'class' IDENT ('extends' IDENT)? '{' member* '}'
member      := 'static'? (fieldDecl ';' | methodDecl)
fieldDecl   := IDENT ':' type
methodDecl  := 'fn' IDENT '(' paramList? ')' ('->' type)? block

type        := ... (Module 8's Pratt type-grammar) ...
              | '[' expr? ']' type          // array type: '[]T' slice, '[N]T' fixed

arrayLit    := ('[' expr? ']' type)? '{' (expr (',' expr)*)? '}'
```

Notice `exprStmt` handles assignment (`x = y;`) by falling straight through to
`parseExpr(0)`, reusing the `=` **led** from Module 7 — you do NOT need a separate
`assignStmt` grammar rule. An assignment is just an expression whose top-level operator
happens to be `=`, followed by a semicolon. This is exactly how Go, Rust, and C treat it
internally. Keep this distinction straight: **declaration** (`let x: T = e;`) is a
statement-level construct because `let` is a keyword that introduces new syntax
structure; **assignment** (`x = e;`) is just an expression statement, because `x` and `e`
are both ordinary expressions and `=` is just another infix operator.

---

## Module 14 — Statement Parser: Go Implementation

Extends the `Parser` struct from Module 9. Assume `p.parseExpr(minBP)` and
`p.parseType(minBP)` already exist from earlier modules.

```go
type Stmt interface{ stmtNode() }

type LetStmt struct {
    Name string
    Type *TypeNode
    Init Node // nil if no initializer
}
func (LetStmt) stmtNode() {}

type ReturnStmt struct{ Value Node } // Value nil for bare `return;`
func (ReturnStmt) stmtNode() {}

type ExprStmt struct{ Expr Node }
func (ExprStmt) stmtNode() {}

type BlockStmt struct{ Stmts []Stmt }
func (BlockStmt) stmtNode() {}

type IfStmt struct {
    Cond       Node
    Then       *BlockStmt
    Else       Stmt // *BlockStmt or *IfStmt, nil if no else
}
func (IfStmt) stmtNode() {}

type FnDecl struct {
    Name   string
    Params []Param
    Ret    *TypeNode // nil if no return type
    Body   *BlockStmt
}
func (FnDecl) stmtNode() {}

type Param struct {
    Name string
    Type *TypeNode
}

// --- dispatcher ---
func (p *Parser) parseStmt() Stmt {
    switch p.peek().Kind {
    case TOK_LET:
        return p.parseLetStmt()
    case TOK_FN:
        return p.parseFnDecl()
    case TOK_RETURN:
        return p.parseReturnStmt()
    case TOK_IF:
        return p.parseIfStmt()
    case TOK_STRUCT:
        return p.parseStructDecl()
    case TOK_CLASS:
        return p.parseClassDecl()
    case TOK_LBRACE:
        return p.parseBlock()
    default:
        return p.parseExprStmt()
    }
}

func (p *Parser) parseLetStmt() Stmt {
    p.expect(TOK_LET)
    name := p.expect(TOK_IDENT).Text
    p.expect(TOK_COLON)
    typ := p.parseType(0)

    var init Node
    if p.peek().Kind == TOK_EQ {
        p.advance()
        init = p.parseExpr(0)
    }
    p.expect(TOK_SEMI)
    return LetStmt{Name: name, Type: typ, Init: init}
}

func (p *Parser) parseReturnStmt() Stmt {
    p.expect(TOK_RETURN)
    var val Node
    if p.peek().Kind != TOK_SEMI {
        val = p.parseExpr(0)
    }
    p.expect(TOK_SEMI)
    return ReturnStmt{Value: val}
}

func (p *Parser) parseExprStmt() Stmt {
    // covers bare calls `foo();` AND assignment `x = y;` -
    // both are just expr followed by ';'
    e := p.parseExpr(0)
    p.expect(TOK_SEMI)
    return ExprStmt{Expr: e}
}

func (p *Parser) parseBlock() *BlockStmt {
    p.expect(TOK_LBRACE)
    var stmts []Stmt
    for p.peek().Kind != TOK_RBRACE && p.peek().Kind != TOK_EOF {
        stmts = append(stmts, p.parseStmt())
    }
    p.expect(TOK_RBRACE)
    return &BlockStmt{Stmts: stmts}
}

func (p *Parser) parseIfStmt() Stmt {
    p.expect(TOK_IF)
    cond := p.parseExpr(0)
    then := p.parseBlock()
    var elseBranch Stmt
    if p.peek().Kind == TOK_ELSE {
        p.advance()
        if p.peek().Kind == TOK_IF {
            elseBranch = p.parseIfStmt() // else-if chains recursively, plain RD
        } else {
            elseBranch = p.parseBlock()
        }
    }
    return IfStmt{Cond: cond, Then: then, Else: elseBranch}
}

func (p *Parser) parseParamList() []Param {
    var params []Param
    if p.peek().Kind == TOK_RPAREN {
        return params
    }
    params = append(params, p.parseParam())
    for p.peek().Kind == TOK_COMMA {
        p.advance()
        params = append(params, p.parseParam())
    }
    return params
}

func (p *Parser) parseParam() Param {
    name := p.expect(TOK_IDENT).Text
    p.expect(TOK_COLON)
    typ := p.parseType(0)
    return Param{Name: name, Type: typ}
}

func (p *Parser) parseFnDecl() Stmt {
    p.expect(TOK_FN)
    name := p.expect(TOK_IDENT).Text
    p.expect(TOK_LPAREN)
    params := p.parseParamList()
    p.expect(TOK_RPAREN)

    var ret *TypeNode
    if p.peek().Kind == TOK_ARROW {
        p.advance()
        ret = p.parseType(0)
    }
    body := p.parseBlock()
    return FnDecl{Name: name, Params: params, Ret: ret, Body: body}
}
```

Everything here is **one-token lookahead, dispatch-and-descend** — no binding power
anywhere. The only places this code touches the Pratt machinery are the calls to
`p.parseExpr(0)` (initializers, conditions, return values) and `p.parseType(0)`
(annotations, return types, param types).

**Trace for your exact example**, `let name: string = "ajaygiri";`:

```
parseStmt() -> peek is TOK_LET -> parseLetStmt()
  expect(LET)
  name = expect(IDENT).Text = "name"
  expect(COLON)
  typ = parseType(0) -> parses "string" as a TKIdent leaf (Module 8's nud, no infix follows)
  peek == TOK_EQ -> advance
  init = parseExpr(0) -> parses the string literal "ajaygiri" as a leaf nud
  expect(SEMI)
  return LetStmt{Name:"name", Type: string, Init: "ajaygiri"}
```

No binding power decisions needed at all here — `string` and `"ajaygiri"` are each a
single nud with nothing following that would trigger a led/loop iteration.

---

## Module 15 — Struct and Class Declarations

```go
type StructDecl struct {
    Name   string
    Fields []Param // reuse Param{Name, Type} - a field is structurally identical
}

type ClassDecl struct {
    Name    string
    Extends string // "" if none
    Fields  []ClassMember
    Methods []ClassMember
}

type ClassMember struct {
    Static bool
    Field  *Param  // set if this member is a field
    Method *FnDecl // set if this member is a method
}

func (p *Parser) parseStructDecl() Stmt {
    p.expect(TOK_STRUCT)
    name := p.expect(TOK_IDENT).Text
    p.expect(TOK_LBRACE)
    var fields []Param
    for p.peek().Kind != TOK_RBRACE {
        fields = append(fields, p.parseParam()) // IDENT ':' type, same shape as a fn param
        if p.peek().Kind == TOK_COMMA {
            p.advance()
        }
    }
    p.expect(TOK_RBRACE)
    return StructDecl{Name: name, Fields: fields}
}

func (p *Parser) parseClassDecl() Stmt {
    p.expect(TOK_CLASS)
    name := p.expect(TOK_IDENT).Text

    extends := ""
    if p.peek().Kind == TOK_EXTENDS {
        p.advance()
        extends = p.expect(TOK_IDENT).Text
    }

    p.expect(TOK_LBRACE)
    var members []ClassMember
    for p.peek().Kind != TOK_RBRACE {
        isStatic := false
        if p.peek().Kind == TOK_STATIC {
            p.advance()
            isStatic = true
        }
        if p.peek().Kind == TOK_FN {
            fn := p.parseFnDecl().(FnDecl)
            members = append(members, ClassMember{Static: isStatic, Method: &fn})
        } else {
            field := p.parseParam()
            p.expect(TOK_SEMI)
            members = append(members, ClassMember{Static: isStatic, Field: &field})
        }
    }
    p.expect(TOK_RBRACE)
    return ClassDecl{Name: name, Extends: extends, Fields: nil, Methods: members}
}
```

Notice `struct` field parsing and `fn` param parsing reuse the exact same `parseParam`
(`IDENT ':' type`) — a struct field, a function parameter, and (in many languages) a
class field are all the same grammatical shape. Don't write three near-duplicate
functions for this; that's a maintenance trap the moment you need to change how types
are annotated everywhere at once.

`static` is handled as an optional-prefix-keyword check inside the member loop — this is
the recursive-descent equivalent of Module 8's exercise 8.3 (optional modifier before a
grammar rule). Same technique, different layer.

---

## Module 16 — Array Types and Array Literals (the part you specifically asked about)

You wrote three shapes:
```
let nums: []number = {}                 // slice type, empty literal
let nums: []number{}                    // (same, no top-level '=' shown — treat as expr form)
let nums: [size]number{}                // fixed-size array with explicit length
```

**Array TYPE parsing** extends Module 8's `parseType` — add a case for `[`:

```go
func (p *Parser) parseType(minBP int) *TypeNode {
    left := p.parseTypeNud()
    // ... existing led loop from Module 8 ...
    return left
}

func (p *Parser) parseTypeNud() *TypeNode {
    tok := p.peek()
    if tok.Kind == TOK_LBRACK {
        p.advance()
        var sizeExpr Node
        if p.peek().Kind != TOK_RBRACK {
            sizeExpr = p.parseExpr(0) // '[size]' - size is a full expression (could be a
                                       // const, a computed const-expr, etc. - reuses Pratt)
        }
        p.expect(TOK_RBRACK)
        elem := p.parseType(0) // element type: 'number' in '[]number'
        return &TypeNode{Kind: TKArray, SizeExpr: sizeExpr, Elem: elem}
    }
    // ... existing ident/keyof/&/fn cases from Module 8 ...
}
```

The disambiguation between `[]T` (slice — no size) and `[N]T` (fixed array — size
present) is a **one-token peek right after consuming `[`**: if the very next token is
`]`, there's no size expression; otherwise parse a full expression for the size (this
lets you write `[COMPILE_TIME_CONST]number` or even `[2 + 3]number`, not just literal
integers — because you're calling `parseExpr(0)`, the full Pratt engine, not a bespoke
integer-only parser).

**Array LITERAL parsing** — `{}` or `{1, 2, 3}` needs to attach to a preceding array
type. This is naturally handled as: when your STATEMENT/EXPRESSION level sees `[` in
prefix (nud) position, it's the start of an array-literal construct, not indexing
(indexing is postfix-led on an already-existing lhs, from Module 5 — completely
different grammar position, so there's no actual ambiguity, just make sure your
dispatcher checks nud vs led correctly, which it already does structurally).

```go
type ArrayLitNode struct {
    ElemType *TypeNode // the '[]number' or '[size]number' part
    Elems    []Node    // the '{1, 2, 3}' part, empty for '{}'
}

// add this case into parseNud (Module 9's expression nud), NOT parseTypeNud:
case TOK_LBRACK:
    // reuse the type parser to consume '[]number' or '[size]number' -
    // this IS a TypeNode, we're just doing it from expression context
    elemType := p.parseArrayTypePrefix() // small helper, same logic as parseTypeNud's TOK_LBRACK case
    p.expect(TOK_LBRACE)
    var elems []Node
    if p.peek().Kind != TOK_RBRACE {
        elems = append(elems, p.parseExpr(0))
        for p.peek().Kind == TOK_COMMA {
            p.advance()
            if p.peek().Kind == TOK_RBRACE { // trailing comma support
                break
            }
            elems = append(elems, p.parseExpr(0))
        }
    }
    p.expect(TOK_RBRACE)
    return ArrayLitNode{ElemType: elemType, Elems: elems}
```

Full trace for `let nums: []number = {};`:

```
parseLetStmt():
  name = "nums"
  typ  = parseType(0)
       -> parseTypeNud() sees '[' -> advance
          peek == ']' -> no sizeExpr
          expect(']')
          elem = parseType(0) -> "number" (TKIdent leaf)
          -> TypeNode{Kind: TKArray, SizeExpr: nil, Elem: number}
  peek == '=' -> advance
  init = parseExpr(0)
       -> parseNud() sees '[' ... wait, next token is actually '{' directly (no '[]' here
          because the type annotation already consumed it) -> parses as a bare
          brace-literal with inferred/already-known element type from `typ` above.
```

Important subtlety you should notice from this trace: because `let nums: []number = {}`
already has the ARRAY TYPE on the left of `=` (in the annotation), the right-hand side
`{}` doesn't need to repeat `[]number` — it's just a bare brace literal whose element
type comes from context (the declared type). This is different from a case with no
annotation, `let nums = []number{1,2,3};` (Go-style), where the type-prefix DOES need to
appear on the initializer expression itself, because there's no separate annotation to
infer it from. **Decide which of these two forms your language actually supports** (or
both) — it changes whether `ArrayLitNode.ElemType` is ever nil (inferred-from-annotation
case) or always present (Go-style bare literal case). This is a real language-design
fork, not just a parsing detail — write down your decision in your own notes before
coding it, because it changes your type-checker's job later too.

**Practice exercise 16.1**: implement both forms (`let x: []T = {...}` with inferred elem
type, and `let x = []T{...}` with explicit elem type on the literal) and make your
`ArrayLitNode` handle `ElemType == nil` gracefully by falling back to the enclosing
`LetStmt.Type`.

**Practice exercise 16.2**: add fixed-size arrays with a runtime-checkable size mismatch
placeholder — parse `let nums: [5]number = {1,2,3,4,5};`, count `len(Elems)` against the
parsed `SizeExpr`, and stub out where you'd emit a compile error if they don't match
(you don't need constant-folding yet, just wire the check point).

---

## Module 17 — `for` Loops (sketch, since your syntax wasn't fully specified)

Recursive descent for `for` depends entirely on which style you pick — decide this
explicitly since your example didn't specify:

```ebnf
(* C-style *)
forStmt := 'for' '(' (letStmt | exprStmt)? ';' expr? ';' expr? ')' block

(* range-style, Go-flavored *)
forStmt := 'for' IDENT 'in' expr block
```

Both are pure recursive descent — peek `for`, then peek the token after it to decide
which shape you're in (`(` -> C-style, `IDENT` followed by `in` -> range-style). No
binding power involved; the only Pratt calls are the embedded `expr` slots.

**Practice exercise 17.1**: pick one style (or support both, dispatching on the token
after `for`) and implement it, reusing `parseBlock` for the body.

---

## Module 18 — Instantiation: The Ambiguity Nobody Warns You About

Instantiation (`Point{x: 1, y: 2}`, `new Point(1, 2)`, `Point::new(1, 2)`) sits at the
**expression** level, not the statement level — it's a nud, same category as a number
literal or a parenthesized expr. Where it gets genuinely hard is a real ambiguity that
Go/Rust/C-family designers have all had to explicitly solve:

```
if x == Point { x: 1, y: 2 } { ... }
                ^ is this a struct literal starting here, or is '{' the start
                  of the if-block, with 'Point' being some earlier expression?
```

This is not a hypothetical — **Rust actually forbids bare struct literals in condition
position** for exactly this reason (`if Point { x: 1 } == other { ... }` is illegal
without wrapping the literal in parens: `if (Point { x: 1 }) == other { ... }`). This is
implemented as a **parser-context flag** — a boolean the parser threads through
recursive calls, something like "am I currently in a position where `{` must mean
struct-literal-start vs block-start." You will need the same mechanism:

```go
type Parser struct {
    toks             []Token
    pos              int
    noStructLiteral  bool // context flag, NOT a token property
}

func (p *Parser) parseIfStmt() Stmt {
    p.expect(TOK_IF)

    prev := p.noStructLiteral
    p.noStructLiteral = true      // disable struct literals while parsing the condition
    cond := p.parseExpr(0)
    p.noStructLiteral = prev      // restore before parsing the block body

    then := p.parseBlock()
    // ...
}
```

Then in your nud dispatch for identifiers:

```go
case TOK_IDENT:
    if p.peekAt(1).Kind == TOK_LBRACE && !p.noStructLiteral {
        return p.parseStructLit()
    }
    p.advance()
    return IdentNode{Name: tok.Text}
```

**This is the single most important lesson in this module**: not every parsing decision
is a binding-power number. Some are genuine **context-sensitivity** that has to be
carried as explicit parser state, restored on the way back out (classic save/restore-a-
flag pattern, same shape as saving/restoring `minBP` implicitly via the call stack, except
here you're doing it explicitly because the flag isn't naturally scoped by recursion
depth alone — both the condition-parse and the block-parse call `parseExpr` internally,
so you must toggle and restore around exactly the condition, not globally).

**Practice exercise 18.1**: add the `noStructLiteral` flag to `for` and `while`
conditions too, and write a test case proving `for x in Point{x:1,y:2}.field { }` is
correctly rejected (or requires parens) under your rule.

---

## Module 19 — Struct Literal Instantiation (Go/Rust-style `Point{x: 1, y: 2}`)

```go
type FieldInit struct {
    Name  string
    Value Node
}

type StructLitNode struct {
    TypeName string
    Fields   []FieldInit
}
func (n StructLitNode) String() string {
    s := n.TypeName + "{"
    for i, f := range n.Fields {
        if i > 0 { s += ", " }
        s += f.Name + ": " + f.Value.String()
    }
    return s + "}"
}

func (p *Parser) parseStructLit() Node {
    name := p.expect(TOK_IDENT).Text
    p.expect(TOK_LBRACE)

    var fields []FieldInit
    for p.peek().Kind != TOK_RBRACE {
        fname := p.expect(TOK_IDENT).Text
        p.expect(TOK_COLON)
        fval := p.parseExpr(0) // full expr - field values can be arbitrary expressions,
                                // including nested struct literals: Point{x: origin.x + 1, y: 2}
        fields = append(fields, FieldInit{Name: fname, Value: fval})

        if p.peek().Kind == TOK_COMMA {
            p.advance()
        } else {
            break
        }
    }
    p.expect(TOK_RBRACE)
    return StructLitNode{TypeName: name, Fields: fields}
}
```

Trace for `Point{x: 1, y: origin.y}`:

```
parseNud(): tok = IDENT("Point"), peekAt(1) = '{' -> parseStructLit()
  name = "Point"
  expect('{')
  loop:
    fname = "x", expect(':'), fval = parseExpr(0) -> NumNode(1)
    fields = [{x, 1}]
    peek == ',' -> advance
    fname = "y", expect(':'), fval = parseExpr(0)
        -> parseNud(): IDENT("origin"), peekAt(1) = '.' (NOT '{') -> plain IdentNode
        -> loop: peek = '.', lbp('.')=70 >= 0 -> led: MemberAccess(origin, "y")
        -> peek = '}' -> not infix -> break, return MemberAccess(origin, y)
    fields = [{x,1}, {y, origin.y}]
    peek == '}' -> loop's inner "else break" fires (no comma) -> exit field loop
  expect('}')
  return StructLitNode{Point, [{x,1},{y,origin.y}]}
```

Notice this is why the **2-token lookahead** (`peekAt(1)`) matters: at the moment you see
`IDENT`, you don't yet know if it's a plain variable reference or the start of a struct
literal — you have to peek one token further to see if `{` follows. Most recursive
descent parsers need this occasionally; it's not a violation of "LL(1)-ish," it's just
LL(2) at this one decision point.

**Practice exercise 19.1**: add support for **positional** struct literals too (some
languages allow `Point{1, 2}` without field names, matched by declaration order) — this
needs you to detect, right after `{`, whether the first token pattern is `IDENT ':'`
(named) or a bare expression (positional), and branch your field-loop accordingly.
Decide whether your language allows mixing the two styles in one literal (most real
languages don't — pick "no" unless you have a strong reason).

---

## Module 20 — Class Instantiation: Three Real-World Patterns, Pick One

Unlike structs (which are almost universally the brace-literal shape), classes have
THREE common instantiation idioms across real languages. You need to choose which your
language uses (or support more than one deliberately, not by accident):

### Pattern A — `new` keyword (Java/C#/TS/C++ style)

```go
type NewExprNode struct {
    ClassName string
    TypeArgs  []*TypeNode // for generics: new Box<int>(5)
    Args      []Node
}

// add to parseNud:
case TOK_NEW:
    p.advance()
    className := p.expect(TOK_IDENT).Text

    var typeArgs []*TypeNode
    if p.peek().Kind == TOK_LT {
        p.advance()
        for p.peek().Kind != TOK_GT {
            typeArgs = append(typeArgs, p.parseType(0))
            if p.peek().Kind == TOK_COMMA { p.advance() }
        }
        p.expect(TOK_GT)
    }

    p.expect(TOK_LPAREN)
    args := p.parseArgList()
    p.expect(TOK_RPAREN)
    return NewExprNode{ClassName: className, TypeArgs: typeArgs, Args: args}
```

`new` is a clean **prefix keyword nud** — no ambiguity at all, because `new` can never
mean anything else, so there's no lookahead trickery needed here (contrast with Module
18's struct-literal problem, which exists precisely because `IDENT {` is ambiguous but
`new IDENT (` never is).

### Pattern B — Associated/static factory function (Rust `Type::new(...)`)

This needs **zero new grammar** if you already have member-access (`.`) — you just need
a second postfix operator, `::`, with the same shape:

```go
var exprLBP = map[TokKind]int{
    // ... existing entries ...
    TOK_DOT:        70,
    TOK_COLONCOLON: 70,
    TOK_LPAREN:     70, // call
}

// in the led loop:
case TOK_COLONCOLON:
    p.advance()
    member := p.expect(TOK_IDENT).Text
    left = PathAccessNode{Base: left, Member: member}
```

`Point::new(1, 2)` then just falls out of existing machinery: `Point` is an nud (plain
ident), `::new` is a led (PathAccess), `(1, 2)` is another led (Call) chained right after
— **this is exactly Module 21's chaining pattern**, nothing instantiation-specific about
it at the grammar level at all. This is the cleanest option if you already have `.` and
call parsing, which you do.

### Pattern C — Struct-literal-style for classes too (Go has no classes, but if your
language's "class" is really just "struct + attached methods," reuse Module 19 directly)

```go
Vector{x: 1, y: 2}   // identical grammar to a struct literal
```

**The actual decision you need to make** (this is a real language-design fork, write it
down before coding): does your language distinguish "plain data with methods attached"
(→ Pattern C, reuse struct literal) from "objects requiring constructor logic to run on
creation" (→ Pattern A or B, which imply a constructor call actually executes code, not
just field-assignment)? If classes need constructor bodies to run (validation, computed
defaults, side effects), you need Pattern A or B — a brace literal can't express "call a
function." If classes are just structs-with-methods and you don't need constructor
logic, Pattern C is simpler and you can drop `new` entirely.

**Practice exercise 20.1**: implement all three patterns in the same parser (yes, all
three — this is a good exercise in "same underlying AST shapes, different surface
syntax") and write a short note in your own words on which one your actual language will
use and why.

---

## Module 21 — Method Calls and Chaining (Postfix `.` and `(`, all one bp tier)

This is what makes `obj.method1().field.method2(a, b)` parse correctly — it's ALL one
mechanism: postfix operators at (roughly) the same high lbp, left-associative, so the
led-loop just keeps grabbing them one at a time, left to right.

```go
type MemberAccessNode struct {
    Object Node
    Field  string
}
func (n MemberAccessNode) String() string { return n.Object.String() + "." + n.Field }

type CallNode struct {
    Callee Node
    Args   []Node
}
func (n CallNode) String() string {
    s := n.Callee.String() + "("
    for i, a := range n.Args {
        if i > 0 { s += ", " }
        s += a.String()
    }
    return s + ")"
}

// led loop additions (goes alongside existing '+' '*' '?' cases from Module 9):
case TOK_DOT:
    p.advance()
    field := p.expect(TOK_IDENT).Text
    left = MemberAccessNode{Object: left, Field: field}

case TOK_LPAREN:
    p.advance()
    args := p.parseArgList() // comma-loop, Module 4 pattern, resets to parseExpr(0) per arg
    p.expect(TOK_RPAREN)
    left = CallNode{Callee: left, Args: args}
```

Full trace for `obj.method1().field.method2(a, b)`, entry `parseExpr(0)`:

```
lhs = parseNud() -> IdentNode("obj")     // peekAt(1) after 'obj' is '.', not '{', so plain ident
loop:
  peek='.'  lbp=70>=0 -> advance, field="method1" -> lhs = MemberAccess(obj, method1)
  peek='('  lbp=70>=0 -> advance, args=[] (empty), expect(')')
            -> lhs = Call(MemberAccess(obj,method1), [])
  peek='.'  lbp=70>=0 -> advance, field="field" -> lhs = MemberAccess(Call(...), field)
  peek='.'  lbp=70>=0 -> advance, field="method2" -> lhs = MemberAccess(MemberAccess(Call..),method2)
  peek='('  lbp=70>=0 -> advance, args=[parseExpr(0)->a, parseExpr(0)->b], expect(')')
            -> lhs = Call(MemberAccess(...), [a, b])
  peek= ';' or EOF -> not infix -> break
return Call(MemberAccess(MemberAccess(Call(MemberAccess(obj,"method1"),[]),"field"),"method2"),[a,b])
```

Read that final AST from the inside out and it matches exactly what you'd expect:
`obj.method1()` first, `.field` off the result, `.method2(a, b)` off that. **All of this
"just happens" from one loop with one bp number for `.`, `(`, and `[` — you never wrote
special chaining logic.** That's the payoff of postfix operators sharing the same
mechanism as infix ones: chaining is not a separate feature, it's the natural consequence
of the led-loop running more than once.

**Practice exercise 21.1**: add `?.` (optional chaining, TS-style) at the same lbp tier
and make it short-circuit at the AST level (`OptMemberAccessNode`, evaluated later as
"return null immediately if Object is null" — you don't need to implement evaluation
yet, just get the parse + AST shape right).

---

## Module 22 — Full Integration: Declaration + Instantiation + Chaining

```
struct Point {
    x: number,
    y: number
}

class Vector {
    static origin: Point;

    fn add(self: Vector, other: Vector) -> Vector {
        return other;
    }
}

fn main() -> number {
    let p: Point = Point { x: 1, y: 2 };
    let v: Vector = new Vector();
    let sum: Vector = v.add(v).add(v);
    return p.x;
}
```

Parsing `let sum: Vector = v.add(v).add(v);` exercises nearly every module in this file:
- `let` dispatch → Module 14
- `Vector` type annotation → Module 8 (type nud, plain ident, no infix follows)
- `v.add(v).add(v)` → Module 21's chaining: `v` (nud) → `.add` (led) → `(v)` (led, `v`
  itself parsed via `parseExpr(0)` inside the arg list) → `.add` (led again) → `(v)`
  (led again)
- `Point { x: 1, y: 2 }` → Module 19's struct literal, guarded by Module 18's
  `noStructLiteral` context flag (here it's fine since we're not in a condition
  position)
- `new Vector()` → Module 20 Pattern A

**Practice exercise 22.1 (capstone)**: implement the full program above end-to-end —
lexer through statement parser through expression/type Pratt parsers — and print the
full AST. If any single module in this file has a bug, this program will surface it,
because it deliberately stacks every construct on top of every other one.

**Practice exercise 22.2**: deliberately break your `noStructLiteral` flag (comment out
the restore line) and write down, precisely, which line of the integration program above
would parse incorrectly and why. This is the single best way to actually understand why
save/restore-around-a-scope matters instead of just copying the pattern.

---

## Instantiation Exercise Set (do after the Statement & Declaration set)

1. Implement Module 18's `noStructLiteral` context flag and the `if`/`for` guard.
2. Implement Module 19's named-field struct literal parser; extend with positional
   fields (19.1) and decide + document whether mixing is allowed.
3. Implement all three instantiation patterns from Module 20 and write down which one
   your language actually uses and why (20.1).
4. Implement Module 21's method-chaining led cases and verify the exact trace shown
   above matches your own parser's output on `obj.method1().field.method2(a, b)`.
5. Add `?.` optional chaining (21.1).
6. Do the full capstone integration (22.1) and the deliberate-bug exercise (22.2).

## Statement & Declaration Exercise Set (do after the Master Exercise Set above)

1. Implement Module 14's full statement parser and parse a small program combining
   `let`, `fn`, `return`, and nested `if`/`else`.
2. Implement Module 15's struct/class parser, including `static` members, on a class with
   at least one static field and one instance method.
3. Implement Module 16's array type + array literal parsing for both the
   annotation-inferred and Go-style-explicit forms (exercises 16.1, 16.2 above).
4. Implement one `for` loop style (exercise 17.1).
5. **Integration exercise**: parse this full program end-to-end and print the AST:
   ```
   struct Point {
       x: number,
       y: number
   }

   class Vector {
       static origin: Point;

       fn length(self: Vector) -> number {
           return self.x;
       }
   }

   fn main() -> number {
       let nums: []number = {1, 2, 3};
       let fixed: [3]number = {1, 2, 3};
       let p: Point = Point{ x: 1, y: 2 };
       return 0;
   }
   ```
   This forces every module in this file to work together: struct fields (Module 15),
   class + static + methods (Module 15), array type + literal both forms (Module 16),
   function decl with params/return/body (Module 14), and a struct-literal construct
   you'll need to design yourself by extension (`Point{ x: 1, y: 2 }` — same shape as
   array literal, but with named fields instead of positional elements; this is
   intentionally left for you to design as a capstone, not spec'd out above).

## Master Exercise Set (do these in order, don't skip)

1. Implement Module 9's skeleton exactly, verify `3 + 3 * 5` output.
2. Add `^` (exponent, right-assoc) and hand-trace `2 ^ 3 ^ 2` before coding, then verify.
3. Add postfix function calls `f(a, b)` with comma-separated args (Module 4/5).
4. Add postfix array indexing `a[i]`.
5. Add ternary (Module 6), hand-trace `a ? b : c ? d : e`.
6. Add assignment as right-assoc led (Module 7), hand-trace `a = b = c`.
7. Build the type-parser skeleton from Module 8.3, test on `A | B & C` and `A[]`.
8. Extend the type-parser for `Box<dyn Trait + Send>` (exercise 8.1).
9. Extend the type-parser for TS conditional types `A extends B ? C : D` (exercise 8.2).
10. Extend Rust reference types with lifetimes + `mut` (exercise 8.3) — hardest one, closest
    to what a real `rustc`-style parser actually has to deal with.
11. Write a "bp tracer" mode: modify `parseExpr`/`parseType` to print `minBP` and each
    token's `lbp`/`rbp` as it runs, so you can visually debug precedence bugs on any new
    operator you add going forward. This pays for itself the moment you add your 6th or
    7th operator and something parses wrong.

Once you can do all 11 without looking back at this file, you understand Pratt parsing at
implementation depth, not just conceptually — which is the actual bar for using it in a
real compiler front-end.
