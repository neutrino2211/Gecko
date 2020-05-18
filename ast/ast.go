package ast

import (
	"github.com/neutrino2211/Gecko/tokens"
)

type Variable struct {
	tokens.Field
	Scope *Ast
}

type Method struct {
	tokens.Method
	Scope *Ast
}

type Type struct {
	tokens.Type
	Scope     *Ast
	Variables map[string]*Variable
	Methods   map[string]*Variable
}

type Class struct {
	tokens.Class
	Ast
	Visibility string
	Scope      *Ast
}

// Ast : Structure for .g ASTs not participle ASTs
type Ast struct {
	Variables    map[string]*Variable
	Methods      map[string]*Method
	Types        map[string]*Type
	Classes      map[string]*Class
	Name         string
	Parent       *Ast
	CPreliminary string
}

// Initialize : Initialize default Ast fields
func (a *Ast) Initialize() {
	a.Variables = make(map[string]*Variable)
	a.Methods = make(map[string]*Method)
	a.Types = make(map[string]*Type)
	a.Classes = make(map[string]*Class)
	a.CPreliminary = ""
}

func (a *Ast) Merge(m *Ast) {
	if m.Variables != nil {
		for _, v := range m.Variables {
			if a.Variables[v.Name] == nil {
				a.Variables[v.Name] = v
			}
		}
	}

	if m.Methods != nil {
		for _, m := range m.Methods {
			if a.Methods[m.Name] == nil {
				a.Methods[m.Name] = m
			}
		}
	}

	if m.Variables != nil {
		for _, t := range m.Types {
			if a.Types[t.Name] == nil {
				a.Types[t.Name] = t
			}
		}
	}

	if m.Classes != nil {
		for _, c := range m.Classes {
			if a.Classes[c.Class.Name] == nil {
				a.Classes[c.Class.Name] = c
			}
		}
	}
}

func (a *Ast) MergeImport(m *Ast) {
	for _, v := range m.Variables {
		if v.Scope.Name == a.Name && v.Visibility == "protected" {
			a.Variables[v.Scope.Name+"."+v.Name] = v
		} else if v.Visibility == "public" {
			a.Variables[v.Scope.Name+"."+v.Name] = v
		}
	}

	for _, m := range m.Methods {
		a.Methods[m.Scope.Name+"."+m.Name] = m
	}

	for _, t := range m.Types {
		a.Types[t.Scope.Name+"."+t.Name] = t
	}

	for _, c := range m.Classes {
		a.Classes[c.Parent.Name+"."+c.Class.Name] = c
	}
}

func (a *Ast) GetFullPath() string {
	if a.Parent != nil {
		return a.Parent.GetFullPath() + "__" + a.Name
	}

	return a.Name
}

func (t *Type) Initialize() {
	t.Variables = make(map[string]*Variable)
	t.Methods = make(map[string]*Variable)
}

func (t *Type) GetFullPath() string {
	if t.Scope != nil {
		return t.Scope.GetFullPath() + "__" + t.Name
	}

	return t.Name
}

func (v *Variable) FromToken(tok *tokens.Field) {
	v.Name = tok.Name
	v.Pos = tok.Pos
	v.Type = tok.Type
	v.Value = tok.Value
	v.Visibility = tok.Visibility
}

func (v *Variable) FromTypeField(tok *tokens.TypeField) {
	v.Name = tok.Name
	v.Pos = tok.Pos
	v.Type = tok.Type
	v.Value = tok.Value
}

func (v *Variable) GetFullPath() string {
	return v.Scope.GetFullPath() + "__" + v.Name
}

func (m *Method) FromToken(tok *tokens.Method) {
	m.Arguments = tok.Arguments
	m.Name = tok.Name
	m.Type = tok.Type
	m.Value = tok.Value
	m.Pos = tok.Pos
	m.Visibility = tok.Visibility
}

func (m *Method) ToAst() *Ast {
	ast := &Ast{}
	ast.Initialize()
	ast.Name = m.Name
	ast.Parent = m.Scope
	return ast
}

func (m *Method) GetFullPath() string {
	return m.Scope.GetFullPath() + "__" + m.Name
}
