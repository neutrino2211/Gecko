package compiler

import (
	"fmt"
	"strconv"

	"github.com/alecthomas/repr"

	"github.com/neutrino2211/Gecko/ast"
	"github.com/neutrino2211/Gecko/evaluate"
	"github.com/neutrino2211/Gecko/tokens"
)

//MAJOR HACK to fix issue where compiler errors out in packages that are not main
// var isMain = true

/*
	assignSymbolVisibility:

	This function implicitly applies visibility to [Symbols](../ast.go)

	Rules:

	* All symbols are public by default

	* All symbols that have a name which start with "_" are marked as private
*/
func assignSymbolVisibility(i interface{}) {
	symbol, ok := i.(*ast.Variable)
	if !ok {
		classSymbol := i.(*ast.Class)
		if classSymbol.Class.Name[0] == '_' {
			classSymbol.Visibility = "private"
		} else {
			classSymbol.Visibility = "public"
		}
	} else {
		if symbol.Name[0] == '_' {
			symbol.Visibility = "private"
		} else {
			symbol.Visibility = "public"
		}
	}
}

func flattenArray(arr []*tokens.Literal, geckoAst *ast.Ast) {
	for _, v := range arr {
		if v.Expression != nil {
			flattenValue(v, geckoAst)
		}
	}
}

func flattenValue(value *tokens.Literal, geckoAst *ast.Ast) {
	repr.Println("======", value.ArrayIndex != nil)
	if value.Expression != nil {
		v, _ := evaluate.Evaluate(value.Expression, geckoAst)
		value.Expression = nil
		switch v.(type) {
		case int:
			value.Number = strconv.Itoa(v.(int))
		case string:
			if v.(string)[0] == '"' {
				value.String = v.(string)
			} else {
				value.Symbol = v.(string)
			}
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

func CompileClassEntries(class *tokens.Class) []*tokens.Entry {
	entries := make([]*tokens.Entry, 0)
	for _, field := range class.Fields {
		if field.Field != nil {
			newEntry := &tokens.Entry{}
			newEntry.Field = field.Field
			entries = append(entries, newEntry)
		} else if field.Method != nil {
			newEntry := &tokens.Entry{}
			newEntry.Method = field.Method
			entries = append(entries, newEntry)
		}
	}

	return entries
}

func CompileEntries(entries []*tokens.Entry, geckoAst *ast.Ast) *ast.Ast {
	for _, entry := range entries {
		if entry.Field != nil {
			variable := &ast.Variable{}
			variable.FromToken(entry.Field)
			variable.Scope = geckoAst
			repr.Println(entry.Field.Name)
			if entry.Field.Value != nil && entry.Field.Value.Expression != nil {
				flattenValue(entry.Field.Value, geckoAst)
			}

			if variable.Value.Array != nil {
				flattenArray(variable.Value.Array, geckoAst)
			}
			if variable.Visibility == "" {
				assignSymbolVisibility(variable)
			}
			geckoAst.Variables[entry.Field.Name] = variable
		} else if entry.Method != nil {
			method := &ast.Method{}
			method.FromToken(entry.Method)
			// repr.Println(method)
			method.Scope = geckoAst
			for _, argument := range method.Arguments {
				if argument.Default != nil {
					flattenValue(argument.Default, geckoAst)
				}

				if argument.Extenal {
					var argEntriesPrelim = []*tokens.Entry{
						&tokens.Entry{
							Field: &tokens.Field{
								Name: argument.Name,
								Type: argument.Type,
								Value: &tokens.Literal{
									Symbol: argument.Name,
								},
							},
						},
					}
					method.Value = append(argEntriesPrelim, method.Value...)
				}
			}
			geckoAst.Methods[method.Name] = method
		} else if entry.Class != nil {
			class := &ast.Class{}
			classAst := &ast.Ast{}
			class.Initialize()
			classAst.Initialize()
			classAst.Name = entry.Class.Name
			classEntries := CompileClassEntries(entry.Class)
			class.Merge(CompileEntries(classEntries, classAst))
			class.Parent = geckoAst
			classAst.Parent = geckoAst
			class.Class.Name = entry.Class.Name
			if class.Visibility == "" {
				assignSymbolVisibility(class)
			}
			geckoAst.Classes[class.Class.Name] = class
		} else if entry.Type != nil {
			_type := &ast.Type{}
			_type.Initialize()
			for _, f := range entry.Type.Fields {
				if f.Arguments != nil {
					method := &ast.Variable{}
					method.FromTypeField(f)
					_type.Methods[method.Name] = method
				} else {
					variable := &ast.Variable{}
					variable.FromTypeField(f)
					_type.Variables[variable.Name] = variable
				}
			}
			_type.Name = entry.Type.Name
			_type.Pos = entry.Type.Pos
			_type.Scope = geckoAst
			geckoAst.Types[_type.Name] = _type
		} else if len(entry.CCode) > 1 {
			// repr.Println(entry.CCode)
			geckoAst.CPreliminary = geckoAst.CPreliminary + entry.CCode[1:len(entry.CCode)-1] + "\n"
		}
	}

	return geckoAst
}

func CompilePass(entryFile *tokens.File, geckoAst *ast.Ast, buildAll bool) (*ast.Ast, *ExecutionContext) {
	compiledAst := &ast.Ast{}
	compiledAst.Initialize()

	var ctx *ExecutionContext

	for _, _import := range entryFile.Imports {
		importAst := &ast.Ast{}
		importAst.Name = _import.PackageName
		importAst.Initialize()
		_ast, _ := CompilePass(_import, importAst, buildAll)
		// repr.Println(c)
		geckoAst.CPreliminary = _ast.CPreliminary + geckoAst.CPreliminary
		compiledAst.MergeImport(_ast)
	}

	fmt.Println("Compiling package:", entryFile.PackageName)
	compiledAst.Name = entryFile.PackageName
	compiledAst.Merge(geckoAst)
	compiledAst.Merge(CompileEntries(entryFile.Entries, compiledAst))
	// repr.Println(compiledAst.GetFullPath(), compiledAst.Methods)
	// if compiledAst.Parent != nil {
	// 	repr.Println(compiledAst.Parent.Methods)
	// 	compiledAst.Merge(compiledAst.Parent)
	// }
	// if compiledAst.Name != "Main" && isMain {
	// 	println("Error: Main package not found")
	// 	os.Exit(2)
	// } else if !isMain {

	// }
	ctx = buildExecutionContext(entryFile.Entries, compiledAst, buildAll)
	// ctx.Ast = geckoAst
	return compiledAst, ctx
}
