package compiler

import (
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
	CPreliminary string
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
	Value *tokens.Literal
	Name  string
	Type  *tokens.TypeRef
}

type ExecutionContext struct {
	Steps      []*ExecutionStep
	Methods    []*ExecutionContext
	Ast        *ast.Ast
	ReturnType *tokens.TypeRef
}

var builtMethods = []string{}

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

	// repr.Println(geckoAst.GetFullPath(), call.Function, geckoAst.Parent.Methods)
	// if geckoAst.Parent != nil {
	// 	geckoAst.Merge(geckoAst.Parent)
	// }
	mthd := geckoAst.Methods[call.Function]
	compileLogger.DebugLogString("building call step for", call.Function)

	if mthd == nil {
		compileLogger.Fatal(color.HiRedString("Could not find method %s", call.Function))
	}

	for _, arg := range mthd.Arguments {
		if arg.Default != nil {
			flattenValue(arg.Default, geckoAst)
			args[arg.Name] = arg.Default
		}
	}

	for _, arg := range call.Arguments {
		argsOrder = append(argsOrder, arg.Name)
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

	// ctx.Methods = []*ExecutionContext{}
	// ctx.Steps = []*ExecutionStep{}
	// repr.Println(entries)

	for _, entry := range entries {
		if entry.FuncCall != nil {
			mthd := geckoAst.Methods[entry.FuncCall.Function]
			if mthd != nil && !funk.Contains(builtMethods, mthd.GetFullPath()) && mthd.Visibility != "external" {
				methodContext := buildExecutionContext(mthd.Method.Value, mthd.ToAst(), buildAll)
				methodContext.ReturnType = mthd.Type
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
					Name:  name,
					Value: entry.Field.Value,
					Type:  entry.Field.Type,
				},
			})
		} else if entry.Method != nil && entry.Method.Visibility != "external" && buildAll {
			mthd := geckoAst.Methods[entry.Method.Name]
			compileLogger.DebugLogString("building execution context for method", color.HiYellowString("'%s'", entry.Method.Name))
			if mthd != nil && !funk.Contains(builtMethods, mthd.GetFullPath()) {
				mthdAst := mthd.ToAst()
				mthdAst.MergeWithParents()
				methodContext := buildExecutionContext(mthd.Method.Value, mthdAst, buildAll)
				methodContext.ReturnType = mthd.Type
				ctx.Methods = append(ctx.Methods, methodContext)
				builtMethods = append(builtMethods, mthd.GetFullPath())
			}
		}
	}

	ctx.Ast = geckoAst

	return ctx
}
