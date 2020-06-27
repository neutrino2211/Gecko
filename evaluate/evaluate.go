package evaluate

import (
	"strconv"
	"strings"

	"github.com/neutrino2211/Gecko/ast"
	"github.com/neutrino2211/Gecko/errors"
	"github.com/neutrino2211/Gecko/tokens"
	"github.com/neutrino2211/Gecko/utils"
)

func flattenArray(arr []*tokens.Literal, geckoAst *ast.Ast) {
	for _, v := range arr {
		if v.Expression != nil {
			flattenValue(v, geckoAst)
		}
	}
}

func flattenValue(value *tokens.Literal, geckoAst *ast.Ast) {
	if value.Expression != nil {
		v, _ := Evaluate(value.Expression, geckoAst)
		value.Expression = nil
		switch v.(type) {
		case int:
			value.Number = strconv.Itoa(v.(int))
		case string:
			value.String = v.(string)
		case bool:
			b := v.(bool)
			if b {
				value.Bool = "true"
			} else {
				value.Bool = "false"
			}
		}
	} else if value.Array != nil {
		flattenArray(value.Array, geckoAst)
	}
}

func parseLiteral(lit *tokens.Literal) (interface{}, error) {
	var r interface{}
	var err error
	// fmt.Println(lit, "===")
	if len(lit.Bool) > 0 {
		if lit.Bool == "true" {
			r = true
			r = &r
		} else {
			r = false
			r = &r
		}
	} else if len(lit.Number) > 0 {
		r, err = strconv.Atoi(lit.Number)
	} else if len(lit.String) > 0 {
		r = lit.String
	} else if len(lit.Symbol) > 0 {
		r = lit.Symbol
	}

	return r, err
}

func unary(un *tokens.Unary, scope *ast.Ast) (interface{}, error) {
	if len(un.Op) > 0 {

	}

	var r interface{}
	var err error
	if un.Primary == nil {
		return unary(un.Unary, scope)
	}
	if len(un.Primary.Bool) > 0 {
		if un.Primary.Bool == "true" {
			r = true
		} else {
			r = false
		}
	} else if un.Primary.Nil != nil {
		r = un.Primary.Nil
	} else if len(un.Primary.Number) > 0 {
		un.Primary.Number = strings.ReplaceAll(un.Primary.Number, "_", "")
		r, err = strconv.Atoi(un.Primary.Number)
	} else if len(un.Primary.String) > 0 {
		r = un.Primary.String
	} else if un.Primary.SubExpression != nil {
		r, err = Evaluate(un.Primary.SubExpression, scope)
	} else if len(un.Primary.Symbol) > 0 {
		variable := utils.ResolveVariable(scope, un.Primary.Symbol)

		// println(color.HiYellowString("%s", variable))
		// repr.Println(variable == nil, scope.GetFullPath())

		if strings.Contains(un.Primary.Symbol, ".") && variable != nil {
			r = strings.Replace(un.Primary.Symbol, strings.Split(un.Primary.Symbol, ".")[0], variable.GetFullPath(), 1)
			return r, err
		}

		if variable != nil {
			// repr.Println(variable.Value)
			// repr.Println(variable.Value)
			if variable.Value != nil && len(variable.Value.Symbol) > 0 && variable.Scope.Parent != nil {
				// repr.Println(variable.Value)
				// variable.Scope = scope
				r = variable.GetFullPath()
			} else if len(un.Primary.Symbol) > 0 {
				r = variable.GetFullPath()
			} else {
				r, err = parseLiteral(variable.Value)
			}

		} else {
			// repr.Println(scope, un.Pos.String())
			err := errors.NewError(un.Pos, "Symbol '"+un.Primary.Symbol+"' not found", scope)
			errors.AddError(err)
		}
	} else if un.Primary.FuncCall != nil {
		r = un.Primary.FuncCall
	}

	return r, err
}

func multiplication(mult *tokens.Multiplication, scope *ast.Ast) (interface{}, error) {
	if len(mult.Op) > 0 {
		switch mult.Op {
		case "*":
			var err error
			m, err := unary(mult.Unary, scope)
			n, err := multiplication(mult.Next, scope)
			mNumber, okm := m.(int)
			nNumber, okn := n.(int)
			if okm && okn {
				return (mNumber * nNumber), err
			}
			return nil, err

		case "/":
			var err error
			m, err := unary(mult.Unary, scope)
			n, err := multiplication(mult.Next, scope)
			mNumber, okm := m.(int)
			nNumber, okn := n.(int)
			if okm && okn {
				return (mNumber / nNumber), err
			}
			return nil, err
		}
	}

	return unary(mult.Unary, scope)
}

func addition(add *tokens.Addition, scope *ast.Ast) (interface{}, error) {
	if len(add.Op) > 0 {
		switch add.Op {

		case "+":
			var err error
			m, err := multiplication(add.Multiplication, scope)
			n, err := addition(add.Next, scope)
			mNumber, okm := m.(int)
			nNumber, okn := n.(int)
			if okm && okn {
				return (mNumber + nNumber), nil
			}

			// err = nil
			mString, okm := m.(string)
			nString, okn := n.(string)

			if okm && okn {
				return (mString[:len(mString)-1] + nString[1:]), nil
			}

			return nil, err

		case "-":
			var err error
			m, err := multiplication(add.Multiplication, scope)
			n, err := addition(add.Next, scope)
			mNumber, okm := m.(int)
			nNumber, okn := n.(int)
			if okm && okn {
				return (mNumber - nNumber), err
			}
			return nil, err
		}

	}

	return multiplication(add.Multiplication, scope)
}

func comparison(cmp *tokens.Comparison, scope *ast.Ast) (interface{}, error) {
	if len(cmp.Op) > 0 {
		switch cmp.Op {

		case ">":
			var err error
			m, err := addition(cmp.Addition, scope)
			n, err := comparison(cmp.Next, scope)
			mNumber, okm := m.(int)
			nNumber, okn := n.(int)
			if okm && okn {
				return (mNumber > nNumber), nil
			}

			return nil, err

		case "<":
			var err error
			m, err := addition(cmp.Addition, scope)
			n, err := comparison(cmp.Next, scope)
			mNumber, okm := m.(int)
			nNumber, okn := n.(int)
			if okm && okn {
				return (mNumber < nNumber), err
			}
			return nil, err

		case ">=":
			var err error
			m, err := addition(cmp.Addition, scope)
			n, err := comparison(cmp.Next, scope)
			mNumber, okm := m.(int)
			nNumber, okn := n.(int)
			if okm && okn {
				return (mNumber >= nNumber), nil
			}

			return nil, err

		case "<=":
			var err error
			m, err := addition(cmp.Addition, scope)
			n, err := comparison(cmp.Next, scope)
			mNumber, okm := m.(int)
			nNumber, okn := n.(int)
			if okm && okn {
				return (mNumber <= nNumber), nil
			}

			return nil, err
		}
	}

	return addition(cmp.Addition, scope)
}

func equality(eq *tokens.Equality, scope *ast.Ast) (interface{}, error) {
	if len(eq.Op) > 0 {
		switch eq.Op {
		case "!=":
			var err error
			eql, err := equality(eq.Next, scope)
			cmp, err := comparison(eq.Comparison, scope)
			eqlNum, okeqlnum := eql.(int)
			cmpNum, okcmpnum := cmp.(int)
			if okeqlnum && okcmpnum {
				return (eqlNum != cmpNum), err
			}

			eqlStr, okeqlstring := eql.(string)
			cmpStr, okcmpstring := cmp.(string)
			if okeqlstring && okcmpstring {
				return (eqlStr != cmpStr), err
			}

			return nil, err
		case "==":
			var err error
			eql, err := equality(eq.Next, scope)
			cmp, err := comparison(eq.Comparison, scope)
			eqlNum, okeqlnum := eql.(int)
			cmpNum, okcmpnum := cmp.(int)
			if okeqlnum && okcmpnum {
				return (eqlNum == cmpNum), err
			}

			eqlStr, okeqlstring := eql.(string)
			cmpStr, okcmpstring := cmp.(string)
			if okeqlstring && okcmpstring {
				return (eqlStr == cmpStr), nil
			}

			return nil, err
		}
	}

	return comparison(eq.Comparison, scope)
}

func Evaluate(expr *tokens.Expression, scope *ast.Ast) (interface{}, error) {
	// repr.Println(expr.Pos)
	v, err := equality(expr.Equality, scope)
	return v, err
}
