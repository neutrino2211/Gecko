package compiler

import (
	"math/rand"

	"github.com/alecthomas/repr"

	"github.com/neutrino2211/Gecko/ast"
	"github.com/neutrino2211/Gecko/tokens"
)

/*
  WARNING: ALL BYTECODE IS SUBJECT TO CHANGE!
*/

var typeMap = map[string]string{
	"string":   "char *",
	"char *[]": "char **",
}

func addCode(s string, a string) string {
	return s + a + "\n"
}

func randomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func GetTypeAsString(v *tokens.TypeRef, geckoAst *ast.Ast) string {
	r := ""
	tmp := ""
	tyr := v

	if geckoAst.Types[tyr.Type] != nil {
		r += geckoAst.Types[tyr.Type].GetFullPath()
	} else {
		r += tyr.Type
	}

	if len(typeMap[r]) > 0 {
		r = typeMap[r]
	}

	if tyr.Array != nil {
		for tyr.Array != nil {
			r += "[]"
			tyr = tyr.Array
		}

		if len(tyr.Type) > 0 {
			tmp = tyr.Type
		}

		if len(typeMap[tmp]) > 0 {
			tmp = typeMap[tmp]
		}

		r = tmp + r
	}

	if len(typeMap[r]) > 0 {
		r = typeMap[r]
	}

	return r
}

func CreateMethArgs(args []*tokens.Value, geckoAst *ast.Ast) string {
	r := ""
	for _, arg := range args {
		r += GetTypeAsString(arg.Type, geckoAst) + " " + arg.Name + ", "
	}

	if len(r) == 0 {
		return r
	} else {
		return r[0 : len(r)-2]
	}
}

// func generateExpression(e *tokens.Expression) string {

// }

func codeify(v *tokens.Literal) string {
	if v.Array != nil {
		// refID := randomString(32)
		// s := "allocate % " + refID + "\n"
		arr := "{"
		for _, v := range v.Array {
			arr += codeify(v) + ","
		}
		arr = arr[0 : len(arr)-1]
		arr += "}"

		// ar := funk.ReverseString(funk.ReverseString(strings.Join(strings.Split(s, "\n"), ","))[1:])

		return arr
	} else if v.ArrayIndex != nil {
		repr.Println("============", v)
		return ""
	} else if len(v.Bool) > 0 {
		return v.Bool
	} else if len(v.Number) > 0 {
		return v.Number
	} else if len(v.String) > 0 {
		return v.String
	} else {
		return v.Symbol
	}
}

func (m *MethodCall) Code() string {
	s := ""
	a := ""
	// repr.Println(m.Arguments)
	for _, k := range *m.Arguments {
		// repr.Println(a, k)
		instruction := codeify(k)
		// identification := funk.ReverseString(strings.Split(funk.ReverseString(strings.Split(instruction, "\n")[0]), " ")[0])
		// s = addCode(s, instruction)
		// s = addCode(s, "db "+m.MethodFullName+"__"+a+" @"+identification)
		a += instruction + ","
	}
	// s = addCode(s, "call "+m.MethodFullName)
	if len(a) != 0 {
		a = a[0 : len(a)-1]
	}
	s = addCode(s, m.MethodFullName+"("+a+");")
	return s
}

func (c *Conditional) Code(ast *ast.Ast) string {
	s := ""

	sectionName := randomString(32)

	s = addCode(s, "cmp 0 0")
	// repr.Println(c.Expression)

	s = addCode(s, "section "+sectionName)

	code := c.Block.Code()
	s = addCode(s, code[:len(code)-2])

	s = addCode(s, "ret")

	s = addCode(s, "end "+sectionName)

	s = addCode(s, "jz "+sectionName)

	return s
}

func (e *Expression) Code(ast *ast.Ast) string {
	flattenValue(e.Value, ast)
	// repr.Println(e.Value)
	// repr.Println(e)
	// return "# TODO EVALUATE EXPRESSIONS -> " + e.Name
	return /*"auto " + strings.ReplaceAll(e.Name, "||", "::") + " = " +*/ codeify(e.Value)
}

func (ctx *ExecutionContext) Code() string {
	s := ""

	for _, step := range ctx.Steps {
		var code string

		if step.Conditional != nil {
			s = addCode(s, step.Conditional.Code(ctx.Ast))
		} else if step.MethodCall != nil {
			s = addCode(s, step.MethodCall.Code())
		} else if step.Expression != nil {
			s = addCode(s, GetTypeAsString(step.Expression.Type, ctx.Ast)+" "+step.Expression.Name+" = "+step.Expression.Code(ctx.Ast)+";")
		}

		if len(code) != 0 {
			s = addCode(s, code)
		}
	}

	for _, mthd := range ctx.Methods {
		// mthd.As
		println(mthd.Ast.Name, mthd.Ast.Parent.Methods[mthd.Ast.Name].Arguments)
		s = addCode(s, GetTypeAsString(mthd.ReturnType, mthd.Ast)+" "+mthd.Ast.GetFullPath()+" ("+CreateMethArgs(mthd.Ast.Parent.Methods[mthd.Ast.Name].Arguments, mthd.Ast)+"){")
		s = addCode(s, mthd.Code())
		s = addCode(s, "}")
	}

	return s
}
