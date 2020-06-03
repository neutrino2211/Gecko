// Package tokens contains the definitions for all gecko symbols/tokens
package tokens

import (
	"github.com/alecthomas/participle/lexer"
)

type baseToken struct {
	Pos   lexer.Position
	RefID string
}

// File tokens

type File struct {
	PackageName string   `["package" @Ident]`
	Entries     []*Entry `@@*`
	Imports     []*File
	Name        string
}

type Entry struct {
	baseToken
	CCode    string    `@CCode`
	ElseIf   *ElseIf   `| @@`
	Else     *Else     `| @@`
	If       *If       `| @@`
	FuncCall *FuncCall `| @@`
	Method   *Method   `| @@`
	Class    *Class    `| @@`
	Type     *Type     `| @@`
	Schema   *Schema   `| @@`
	Enum     *Enum     `| @@`
	Field    *Field    `| @@`
	Import   string    `| "import" @Ident`
}

// Class tokens

type Class struct {
	baseToken
	Visibility string             `[ @"private" | @"public" | @"protected" ]`
	Name       string             `"class" @Ident`
	Extends    []string           `"extends" @Ident { "," @Ident }`
	Fields     []*ClassBlockField `"{" { @@ } "}"`
}

type ClassBlockField struct {
	baseToken
	Method *Method `@@`
	Field  *Field  `| @@`
}

type ClassField struct {
	baseToken
	Field
}

type ClassMethod struct {
	baseToken
	Method
}

// Conditionals

type If struct {
	baseToken
	Expression *Expression `"if" "(" @@ ")"`
	Value      []*Entry    `"{" { @@ } "}"`
}

type ElseIf struct {
	baseToken
	Expression *Expression `"elif" "(" @@ ")"`
	Value      []*Entry    `"{" { @@ } "}"`
}

type Else struct {
	baseToken
	Value []*Entry `"else" "{" { @@ } "}"`
}

// Expressions

type Expression struct {
	baseToken
	Equality *Equality `@@`
}

type Equality struct {
	baseToken
	Comparison *Comparison `@@`
	Op         string      `[ @( "!" "=" | "=" "=" )`
	Next       *Equality   `  @@ ]`
}

type Comparison struct {
	baseToken
	Addition *Addition   `@@`
	Op       string      `[ @( ">" | ">" "=" | "<" | "<" "=" )`
	Next     *Comparison `  @@ ]`
}

type Addition struct {
	baseToken
	Multiplication *Multiplication `@@`
	Op             string          `[ @( "-" | "+" )`
	Next           *Addition       `  @@ ]`
}

type Multiplication struct {
	baseToken
	Unary *Unary          `@@`
	Op    string          `[ @( "/" | "*" )`
	Next  *Multiplication `  @@ ]`
}

type Unary struct {
	baseToken
	Op      string   `  ( @( "!" | "-" | "+" )`
	Unary   *Unary   `    @@ )`
	Primary *Primary `| @@`
}

type Primary struct {
	baseToken
	FuncCall      *FuncCall   `@@`
	Bool          string      `| ( @"true" | @"false" )`
	Nil           *bool       `| @"nil"`
	String        string      `| @String`
	Symbol        string      `| @Ident`
	Number        string      `| @Number`
	SubExpression *Expression `| "(" @@ ")" `
}

// Misc TODO: Sort

type Enum struct {
	baseToken
	Name  string   `"enum" @Ident`
	Cases []string `"{" { @Ident } "}"`
}

type Schema struct {
	baseToken
	Fields []*Field `"schema" "{" { @@ } "}"`
}

type Type struct {
	baseToken
	Visibility string       `[ @"private" | @"public" | @"protected" ]`
	Name       string       `"type" @Ident`
	Implements string       `[ "implements" @Ident ]`
	Fields     []*TypeField `"{" { @@ } "}"`
}

type Field struct {
	baseToken
	Visibility string   `[ @"private" | @"public" | @"protected" | @"external" ]`
	Name       string   `@Ident`
	Type       *TypeRef `":" @@`
	Value      *Literal `[ "=" @@ ]`
}

type TypeField struct {
	baseToken
	Name      string   `@Ident`
	Arguments []*Value `[ "(" [ @@ { "," @@ } ] ")" ]`
	Type      *TypeRef `":" @@`
	Value     *Literal `[ "=" @@ ]`
}

type Method struct {
	baseToken
	Visibility string   `[ @"private" | @"public" | @"protected" | @"external" ]`
	Name       string   `"func" @Ident`
	Arguments  []*Value `"(" [ @@ { "," @@ } ] ")"`
	Type       *TypeRef `":" @@`
	Value      []*Entry `"{" { @@ } "}"`
}

type Value struct {
	baseToken
	Extenal bool     `[ @"external" ]`
	Name    string   `@Ident`
	Type    *TypeRef `":" @@`
	Default *Literal `[ "=" @@ ]`
}

type Argument struct {
	baseToken
	Name  string   `[ @Ident ":" ]`
	Value *Literal `@@`
}

type TypeRef struct {
	baseToken
	Array       *TypeRef `(   "[" @@ "]"`
	Type        string   `  | @Ident )`
	NonNullable bool     `[ @"!" ]`
}

type Literal struct {
	baseToken
	FuncCall   *FuncCall   `( @@`
	Bool       string      ` | @( "true" | "false" )`
	Nil        *bool       ` | @"nil"`
	Expression *Expression ` | @@`
	String     string      ` | @String`
	Symbol     string      ` | @Ident`
	Number     string      ` | @Number`
	Array      []*Literal  ` | "[" [ @@ { "," @@ } ] "]" )`
	ArrayIndex *Literal    `[ "[" @@ "]" ]`
}

type FuncCall struct {
	baseToken
	Function  string      `@Ident`
	Arguments []*Argument `"(" [ @@ { "," @@ } ] ")"`
}
