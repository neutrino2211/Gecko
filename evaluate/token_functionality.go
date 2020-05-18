package evaluate

import (
	"github.com/neutrino2211/Gecko/ast"
	"github.com/neutrino2211/Gecko/tokens"
)

func IsFalse(expr *tokens.Expression, ast *ast.Ast) bool {
	e, _ := Evaluate(expr, ast)
	r := false

	switch e.(type) {
	case bool:
		b := e.(bool)
		if b == false {
			r = true
		}

	}

	return r
}

func IsBool(expr *tokens.Expression, ast *ast.Ast) bool {
	e, _ := Evaluate(expr, ast)

	switch e.(type) {
	case bool:
		return true
	}

	return false
}

func CouldBeBool(expr *tokens.Expression, ast *ast.Ast) bool {
	e, _ := Evaluate(expr, ast)
	r := false
	switch e.(type) {
	case bool:
		r = true
		break
	case *tokens.FuncCall:
		e := e.(*tokens.FuncCall)
		mthd := ast.Methods[e.Function]
		if mthd.Method.Type.Type == "bool" && mthd.Method.Type.Array == nil {
			r = true
		}
		break
	case int:
		e := e.(int)
		r = e > -1 && e < 2
		break
	}
	// fmt.Println(r)
	return r
}
