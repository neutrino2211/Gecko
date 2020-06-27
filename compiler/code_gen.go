package compiler

import (
	"math/rand"
	"time"

	"github.com/thoas/go-funk"

	"github.com/fatih/color"
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

var (
	types              = ""
	methods            = ""
	functionSignatures = ""
)

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
	rand.Seed(time.Now().UnixNano())
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

func GetPreludeCode() string {
	return types + "\n" + functionSignatures + "\n" + methods
}

func GetTypeAsString(v *tokens.TypeRef, geckoAst *ast.Ast) string {
	r := ""
	tmp := ""
	tyr := v

	if geckoAst.Types[tyr.Type] != nil {
		r += geckoAst.Types[tyr.Type].GetFullPath()
	} else if geckoAst.Classes[tyr.Type] != nil {
		class := geckoAst.Classes[tyr.Type]
		className := tyr.Type
		if class != nil {
			ctype := class.Variables["__ctype__"]
			if ctype != nil && ctype.Value != nil {
				className = ctype.Value.String
				className = className[1 : len(className)-1]
			} else {
				className = class.Class.Name
			}
		}
		r += className
	} else {
		r += tyr.Type
	}

	if len(typeMap[r]) > 0 {
		r = typeMap[r]
	}

	if tyr.Array != nil {
		for tyr.Array != nil {
			r += "*"
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

func codeify(v *tokens.Literal, ast *ast.Ast) string {
	if v.Array != nil {
		// refID := randomString(32)
		// s := "allocate % " + refID + "\n"
		arr := "{"
		for _, v := range v.Array {
			arr += codeify(v, ast) + ","
		}
		arr = arr[0 : len(arr)-1]
		arr += "}"

		// ar := funk.ReverseString(funk.ReverseString(strings.Join(strings.Split(s, "\n"), ","))[1:])

		return arr
	} else if v.Object != nil {
		compileLogger.Log(v.Object)
		obj := "{"
		for _, o := range v.Object {
			flattenValue(o.Value, ast)
			obj += "." + o.Key + " = " + codeify(o.Value, ast) + ","
		}
		obj += "}"
		return obj
	} else if v.ArrayIndex != nil {
		return ""
	} else if len(v.Bool) > 0 {
		return v.Bool
	} else if len(v.Number) > 0 {
		return v.Number
	} else if len(v.String) > 0 {
		return v.String
	} else if v.FuncCall != nil {
		methCall := buildMethodCallStep(v.FuncCall, ast).Code(ast)
		return methCall[0 : len(methCall)-2]
	} else {
		return v.Symbol
	}
}

func (f *LoopStep) Code(scope *ast.Ast) string {
	// scope.MergeWithParents()
	// flattenValue(f.SourceArray, scope)
	// compileLogger.Log(f.SourceArray)
	counterName := f.TargetVariable.GetFullPath() + randomString(8) + "counter"
	loopArrayName := scope.GetFullPath() + randomString(8) + "array"

	code := "int " + counterName + " = 0;"

	// f.Execution.Ast.Variables[f.TargetVariable.Name] = nil
	// scope.Variables[f.TargetVariable.Name] = nil

	delete(scope.Variables, f.TargetVariable.Name)
	delete(f.Execution.Ast.Variables, f.TargetVariable.Name)

	// scope.MergeWithParents()

	//Remove the
	for _, step := range f.Execution.Steps {
		if step.Expression != nil && step.Expression.Name == f.TargetVariable.GetFullPath() && !step.Expression.IsAssignement {
			step.Expression = nil
			break
		}
	}

	code = addCode(code, "int "+loopArrayName+"[] = "+codeify(f.SourceArray, scope)+";")
	code = addCode(code, GetTypeAsString(f.TargetVariable.Type, scope)+" "+f.TargetVariable.GetFullPath()+";")

	code = addCode(code, "while("+counterName+"< sizeof("+loopArrayName+")/sizeof("+GetTypeAsString(f.TargetVariable.Type, scope)+")){")
	code = addCode(code, f.TargetVariable.GetFullPath()+" = "+loopArrayName+"["+counterName+"];")
	code = addCode(code, f.Execution.Code(scope))
	code = addCode(code, counterName+"++;\n}")

	return code
}

func (m *MethodCall) Code(scope *ast.Ast) string {
	s := ""
	a := ""
	// repr.Println(m.Arguments)
	tmpArgs := *m.Arguments
	for _, argName := range m.ArgumentOrder {
		k := tmpArgs[argName]
		if k == nil && tmpArgs[""] != nil {
			k = tmpArgs[""]
		} else if k == nil {
			compileLogger.Fatal(color.HiRedString("transpile error: %s [%s] requires at least one unnamed argument. None passed", m.MethodFullName, m.MethodName))
		}
		instruction := codeify(k, scope)
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

	code := c.Block.Code(ast)
	s = addCode(s, code[:len(code)-2])

	s = addCode(s, "ret")

	s = addCode(s, "end "+sectionName)

	s = addCode(s, "jz "+sectionName)

	return s
}

func (e *Expression) Code(ast *ast.Ast) string {

	r := ""
	if e.Value != nil {
		flattenValue(e.Value, ast)
	}
	if e.Value.FuncCall != nil {
		r = buildMethodCallStep(e.Value.FuncCall, ast).Code(ast)
	} else {
		r = /*"auto " + strings.ReplaceAll(e.Name, "||", "::") + " = " +*/ codeify(e.Value, ast)
	}
	// repr.Println(e.Value)
	// repr.Println(e)
	// return "# TODO EVALUATE EXPRESSIONS -> " + e.Name
	if r == "" {
		compileLogger.Fatal(color.RedString("Failed to evaluate expression:"))
	}
	return r
}

func (obj *ObjectDefinition) Code(scope *ast.Ast) string {
	r := "typedef struct {\n"

	for name, variable := range obj.Variables {

		if name == "__ctype__" {
			continue
		}

		// [Removed] Reason: Caused implicit truncation
		// value := ""
		// if variable.Value != nil {
		// 	flattenValue(variable.Value, obj.Scope)
		// 	value = ":" + codeify(variable.Value, scope)
		// }
		r = addCode(r, GetTypeAsString(variable.Type, obj.Scope)+" "+name+";")
	}

	r += "} " + obj.Name + ";"
	return r
}

func (ctx *ExecutionContext) Code(scope *ast.Ast) string {
	s := ""

	for _, class := range ctx.Classes {
		types = addCode(types, class.Code(scope))
	}

	for _, mthd := range ctx.Methods {
		// mthd.As
		if funk.ContainsString(methodsGenerated, mthd.Ast.GetFullPath()) {
			continue
		}
		methodCode := mthd.Code(scope)
		compileLogger.DebugLogString("building method", mthd.Ast.Name)
		functionSignature := GetTypeAsString(mthd.ReturnType, mthd.Ast) + " " + mthd.Ast.GetFullPath() + " (" + CreateMethArgs(mthd.Ast.Parent.Methods[mthd.Ast.Name].Arguments, mthd.Ast) + ")"

		functionSignatures += functionSignature + ";\n"
		methods = addCode(methods, functionSignature+"{")
		methods = addCode(methods, methodCode)
		methods = addCode(methods, "}")
		methodsGenerated = append(methodsGenerated, mthd.Ast.GetFullPath())
	}

	for _, step := range ctx.Steps {
		var code string

		if step.Conditional != nil {
			s = addCode(s, step.Conditional.Code(ctx.Ast))
		} else if step.MethodCall != nil {
			s = addCode(s, step.MethodCall.Code(scope))
		} else if step.Expression != nil {
			if step.Expression.Value != nil && !step.Expression.IsAssignement {
				s = addCode(s, GetTypeAsString(step.Expression.Type, ctx.Ast)+" "+step.Expression.Name+" = "+step.Expression.Code(ctx.Ast)+";")
			} else if step.Expression.IsAssignement {
				s = addCode(s, step.Expression.Name+" = "+step.Expression.Code(ctx.Ast)+";")
			} else {
				s = addCode(s, GetTypeAsString(step.Expression.Type, ctx.Ast)+" "+step.Expression.Name+";")
			}
		} else if step.ReturnStep != nil {
			s = addCode(s, "return "+codeify(step.ReturnStep, scope)+";")
		} else if step.Loop != nil {
			s = addCode(s, step.Loop.Code(scope))
		}

		if len(code) != 0 {
			s = addCode(s, code)
		}
	}

	return s
}

var (
	methodsGenerated = []string{}
)
