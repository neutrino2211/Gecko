package compiler

import (
	"strings"

	"github.com/fatih/color"
	"github.com/neutrino2211/Gecko/ast"
	"github.com/neutrino2211/Gecko/errors"
	"github.com/neutrino2211/Gecko/evaluate"
	"github.com/neutrino2211/Gecko/tokens"
	"github.com/neutrino2211/Gecko/utils"

	funk "github.com/thoas/go-funk"
)

type _step interface {
	Code() string
}

type ExecutionStep struct {
	MethodCall   *MethodCall
	Conditional  *Conditional
	Expression   *Expression
	ReturnStep   *tokens.Literal
	CPreliminary string
}

type ObjectDefinition struct {
	_step
	Variables map[string]*ast.Variable
	Name      string
	Scope     *ast.Ast
}

type MethodCall struct {
	_step
	MethodName     string
	Arguments      *map[string]*tokens.Literal
	ArgumentOrder  []string
	MethodFullName string
	External       bool
}

type Conditional struct {
	_step
	Block      *ExecutionContext
	Expression *tokens.Expression
}

type Expression struct {
	_step
	Value         *tokens.Literal
	Name          string
	Type          *tokens.TypeRef
	IsAssignement bool
}

type ExecutionContext struct {
	Steps      []*ExecutionStep
	Methods    []*ExecutionContext
	Classes    []*ObjectDefinition
	Ast        *ast.Ast
	ReturnType *tokens.TypeRef
}

func (e *ExecutionContext) Init() {
	e.Ast.Initialize()
}

func (e *ExecutionContext) Merge(m *ExecutionContext) {
	if m.Steps != nil {
		e.Steps = append(e.Steps, m.Steps...)
	}

	if m.Methods != nil {
		e.Methods = append(e.Methods, m.Methods...)
	}

	if m.Ast != nil && e.Ast != nil {
		e.Ast.Merge(m.Ast)
	}
}

var builtMethods = []string{}
var builtClasses = []string{}

func methodWasBuilt(ctx *ExecutionContext, mthd *ast.Method) bool {
	for _, m := range ctx.Methods {
		if m.Ast.Name == mthd.Name {
			return true
		}
	}

	return false
}

func classWasBuilt(ctx *ExecutionContext, class string) bool {
	for _, c := range ctx.Classes {
		compileLogger.Fatal(c.Name, class)
		if c.Name == class {
			return true
		}
	}

	return false
}

func resolveTypeFunction(function string, scope *ast.Ast) *ast.Method {
	levels := strings.Split(function, ".")
	compileLogger.Log(levels)

	if scope.Parent != nil && scope.Parent.Methods[scope.Name] != nil {
		scope.Merge(CompileEntries(scope.Parent.Methods[scope.Name].Value, scope))
		scope.MergeWithParents()
	}

	var finalScope = scope
	var finalLevel string

	for _, level := range levels {
		finalLevel = level
		for v, val := range finalScope.Variables {
			compileLogger.Log(v, val.Name, val.GetFullPath(), level)
		}

		if finalScope.Variables[level] != nil {
			compileLogger.Log(level, finalScope.Variables[level].Type.Type)
			finalScope = &finalScope.Classes[finalScope.Variables[level].Type.Type].Ast
		}
	}

	return finalScope.Methods[finalLevel]
}

func buildConditional(ctx *ExecutionContext, ifBlock interface{}, geckoAst *ast.Ast) *Conditional {
	conditional := &Conditional{}
	var b = false
	switch ifBlock.(type) {
	case *tokens.If:
		ifBlock := ifBlock.(*tokens.If)
		_bool, err := evaluate.Evaluate(ifBlock.Expression, geckoAst)
		if err != nil {
			panic(err)
		} else if utils.IsBool(_bool) {
			b = _bool.(bool)
			if b != false {
				conditional.Block = buildExecutionContext(ifBlock.Value, geckoAst, false)
				conditional.Expression = ifBlock.Expression
				ctx.Steps = append(ctx.Steps, &ExecutionStep{
					Conditional: conditional,
				})
			}
		} else if utils.IsFuncCall(_bool) {
			conditional.Block = buildExecutionContext(ifBlock.Value, geckoAst, false)
			conditional.Expression = ifBlock.Expression
			ctx.Steps = append(ctx.Steps, &ExecutionStep{
				Conditional: conditional,
			})
		}

	case *tokens.ElseIf:
		ifBlock := ifBlock.(*tokens.ElseIf)
		_bool, err := evaluate.Evaluate(ifBlock.Expression, geckoAst)

		if err != nil {
			panic(err)
		} else if utils.IsBool(_bool) {
			b = _bool.(bool)
			if b != false {
				conditional.Block = buildExecutionContext(ifBlock.Value, geckoAst, false)
				conditional.Expression = ifBlock.Expression
				ctx.Steps = append(ctx.Steps, &ExecutionStep{
					Conditional: conditional,
				})
			}
		} else if utils.IsFuncCall(_bool) {
			conditional.Block = buildExecutionContext(ifBlock.Value, geckoAst, false)
			conditional.Expression = ifBlock.Expression
			ctx.Steps = append(ctx.Steps, &ExecutionStep{
				Conditional: conditional,
			})
		}

	case *tokens.Else:
		ifBlock := ifBlock.(*tokens.Else)
		conditional.Block = buildExecutionContext(ifBlock.Value, geckoAst, false)
		//TODO: Set conditional expression to true
		ctx.Steps = append(ctx.Steps, &ExecutionStep{
			Conditional: conditional,
		})
	}

	return conditional
}

func buildMethodCallStep(call *tokens.FuncCall, geckoAst *ast.Ast) *MethodCall {
	mthdStep := &MethodCall{}
	args := make(map[string]*tokens.Literal)
	argsOrder := []string{}
	geckoAst.MergeWithParents()

	// repr.Println(geckoAst.GetFullPath(), call.Function, geckoAst.Parent.Methods)
	// if geckoAst.Parent != nil {
	// 	geckoAst.Merge(geckoAst.Parent)
	// }
	mthd := geckoAst.Methods[call.Function]
	compileLogger.DebugLogString("building call step for", call.Function)

	if mthd == nil {
		mthd = resolveTypeFunction(call.Function, geckoAst)
		if mthd != nil { // This is a type function, add the self variable
			varsList := strings.Split(call.Function, ".")
			self := &tokens.Argument{
				Name: "self",
				Value: &tokens.Literal{
					Symbol: geckoAst.Variables[strings.Join(varsList[0:len(varsList)-1], ".")].GetFullPath(),
				},
			}

			call.Arguments = append([]*tokens.Argument{self}, call.Arguments...)
		}
	}

	if mthd == nil && geckoAst.Classes[call.Function] != nil {
		mthd = geckoAst.Classes[call.Function].Methods["constructor"]
		if mthd != nil { // This is a type function, add the self variable
			for v := range geckoAst.Variables {
				compileLogger.Log(v)
			}
			varsList := strings.Split(call.Function, ".")
			self := &tokens.Argument{
				Name: "self",
				Value: &tokens.Literal{
					Symbol: geckoAst.Variables[strings.Join(varsList[0:len(varsList)-1], ".")].GetFullPath(),
				},
			}

			call.Arguments = append([]*tokens.Argument{self}, call.Arguments...)
		}
	}

	if mthd == nil {
		compileLogger.Fatal(color.HiRedString("Could not find method %s", call.Function))
	}

	for _, arg := range mthd.Arguments {
		argsOrder = append(argsOrder, arg.Name)
		if arg.Default != nil {
			flattenValue(arg.Default, geckoAst)
			args[arg.Name] = arg.Default
			compileLogger.DebugLogString("adding default variable", arg.Name)
		}
	}

	for _, arg := range call.Arguments {
		if arg.Value != nil {
			mthdAst := mthd.ToAst()
			// mthdAst.MergeWithParents()
			mthdAst.Name = geckoAst.Name
			// Hack for adding variables to function calls when their scope has not been finalized
			if geckoAst.Parent != nil && geckoAst.Parent.Methods[geckoAst.Name] != nil {
				mthdAst.Merge(CompileEntries(geckoAst.Parent.Methods[geckoAst.Name].Value, geckoAst))
			}

			compileLogger.DebugLogString("adding variable", arg.Name, "to", mthd.Name, "call")

			// if arg.Value.Expression != nil {
			// 	// repr.Println(evaluate.Evaluate(arg.Value.Expression, mthdAst))
			// 	i := 0
			// 	for n, mthdArg := range mthd.Arguments {
			// 		if mthdArg.Name == arg.Name {
			// 			i = n
			// 		}
			// 	}

			// 	repr.Println(arg.Value, mthd.Arguments[i])
			// }
			// repr.Println(arg.Value)
			valTmp := *arg.Value
			errors.IgnoreNextError()
			flattenValue(&valTmp, mthdAst)
			if errors.ErrorWasIgnored() {
				flattenValue(arg.Value, geckoAst)
			} else {
				arg.Value = &valTmp
			}
			args[arg.Name] = arg.Value
		}
	}
	mthdStep.MethodName = call.Function
	mthdStep.Arguments = &args
	mthdStep.External = mthd.Visibility == "external"
	mthdStep.ArgumentOrder = argsOrder
	if mthdStep.External {
		// compileLogger.DebugLogString("method", mthd.Name, "is external", call.Function)
		mthdStep.MethodFullName = mthd.Name
	} else {
		mthdStep.MethodFullName = mthd.GetFullPath()
	}

	return mthdStep
}

func buildExecutionContext(entries []*tokens.Entry, geckoAst *ast.Ast, buildAll bool) *ExecutionContext {
	ctx := &ExecutionContext{}

	ctx.Methods = []*ExecutionContext{}
	ctx.Steps = []*ExecutionStep{}
	ctx.Classes = []*ObjectDefinition{}
	// repr.Println(entries)

	classMethods := make(map[string]*ast.Method)

	for _, class := range geckoAst.Classes {

		var name string
		ctype := class.Variables["__ctype__"]
		if ctype != nil && ctype.Value != nil {
			name = ctype.Value.String
			name = name[1 : len(name)-1]
		} else {
			name = class.Class.Name
		}

		if funk.ContainsString(builtClasses, name) {
			continue
		}

		ctx.Classes = append(ctx.Classes, &ObjectDefinition{
			Variables: class.Variables,
			Name:      name,
			Scope:     geckoAst,
		})

		typeMap[class.Class.Name] = name

		builtClasses = append(builtClasses, name)

		for _, mthd := range class.Methods {
			compileLogger.DebugLogString("building execution context for method", color.HiYellowString("'%s'", mthd.Name), "in class", color.HiYellowString("'%s'", class.Class.Name))
			classMethods[mthd.GetFullPath()] = mthd
		}
	}

	for _, variable := range geckoAst.Variables {
		ctx.Steps = append(ctx.Steps, &ExecutionStep{
			Expression: &Expression{
				Name:  variable.GetFullPath(),
				Value: variable.Value,
				Type:  variable.Type,
			},
		})
	}

	for _, mthd := range classMethods {
		mthdAst := mthd.ToAst()
		methodContext := buildExecutionContext(mthd.Method.Value, mthdAst, buildAll)
		if mthd.Type != nil {
			methodContext.ReturnType = mthd.Type
		} else {
			methodContext.ReturnType = &tokens.TypeRef{
				Type:        "void",
				NonNullable: false,
			}
		}
		ctx.Methods = append(ctx.Methods, methodContext)
		builtMethods = append(builtMethods, mthd.GetFullPath())
	}

	for _, entry := range entries {
		if entry.FuncCall != nil {
			mthd := geckoAst.Methods[entry.FuncCall.Function]
			if mthd != nil && !methodWasBuilt(ctx, mthd) && mthd.Visibility != "external" {
				methodContext := buildExecutionContext(mthd.Method.Value, mthd.ToAst(), buildAll)
				if mthd.Type != nil {
					methodContext.ReturnType = mthd.Type
				} else {
					methodContext.ReturnType = &tokens.TypeRef{
						Type:        "void",
						NonNullable: false,
					}
				}
				ctx.Methods = append(ctx.Methods, methodContext)
				builtMethods = append(builtMethods, mthd.GetFullPath())
			}

			ctx.Steps = append(ctx.Steps, &ExecutionStep{
				MethodCall: buildMethodCallStep(entry.FuncCall, geckoAst),
			})
		} else if entry.If != nil {
			isBool := evaluate.CouldBeBool(entry.If.Expression, geckoAst)
			if isBool {
				buildConditional(ctx, entry.If, geckoAst)
			} else {
				errors.AddError(errors.NewError(entry.If.Pos, "Expression does not evaluate to a bool", geckoAst))
			}
		} else if entry.ElseIf != nil {
			isBool := evaluate.CouldBeBool(entry.ElseIf.Expression, geckoAst)
			if isBool {
				buildConditional(ctx, entry.ElseIf, geckoAst)
			} else {
				errors.AddError(errors.NewError(entry.ElseIf.Pos, "Expression does not evaluate to a bool", geckoAst))
			}
		} else if entry.Else != nil {
			buildConditional(ctx, entry.Else, geckoAst)
		} else if entry.Field != nil {
			name := ""
			if entry.Field.Visibility == "external" {
				name = entry.Field.Name
			} else {
				name = geckoAst.GetFullPath() + "__" + entry.Field.Name
			}
			ctx.Steps = append(ctx.Steps, &ExecutionStep{
				Expression: &Expression{
					Name:          name,
					Value:         entry.Field.Value,
					Type:          entry.Field.Type,
					IsAssignement: false,
				},
			})
		} else if entry.Assignment != nil {
			name := entry.Assignment.Name
			if geckoAst.Variables[name] != nil && geckoAst.Variables[name].Visibility == "external" {
				name = geckoAst.Variables[name].Name
			} else {
				name = geckoAst.GetFullPath() + "__" + name
			}
			ctx.Steps = append(ctx.Steps, &ExecutionStep{
				Expression: &Expression{
					Name:          name,
					Value:         entry.Assignment.Value,
					IsAssignement: true,
				},
			})
		} else if entry.Method != nil && entry.Method.Visibility != "external" && buildAll {
			mthd := geckoAst.Methods[entry.Method.Name]
			compileLogger.DebugLogString("building execution context for method", color.HiYellowString("'%s'", entry.Method.Name))
			if mthd != nil && !funk.Contains(builtMethods, mthd.GetFullPath()) {
				mthdAst := mthd.ToAst()
				mthdAst.MergeWithParents()
				methodContext := buildExecutionContext(mthd.Method.Value, mthdAst, buildAll)
				if mthd.Type != nil {
					methodContext.ReturnType = mthd.Type
				} else {
					methodContext.ReturnType = &tokens.TypeRef{
						Type:        "void",
						NonNullable: false,
					}
				}
				ctx.Methods = append(ctx.Methods, methodContext)
				builtMethods = append(builtMethods, mthd.GetFullPath())
			}
		} else if entry.Return != nil {
			flattenValue(entry.Return, geckoAst)
			ctx.Steps = append(ctx.Steps, &ExecutionStep{
				ReturnStep: entry.Return,
			})
		}
	}

	ctx.Ast = geckoAst

	return ctx
}
