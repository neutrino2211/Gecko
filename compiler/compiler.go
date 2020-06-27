package compiler

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/neutrino2211/Gecko/config"
	"github.com/neutrino2211/Gecko/errors"
	"github.com/neutrino2211/Gecko/utils"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
	"github.com/fatih/color"
	"github.com/neutrino2211/Gecko/logger"

	"github.com/neutrino2211/Gecko/ast"
	"github.com/neutrino2211/Gecko/evaluate"
	"github.com/neutrino2211/Gecko/tokens"
)

var (
	compileLogger = &logger.Logger{}
	graphQLLexer  = lexer.Must(ebnf.New(`
Comment = "//"  { "\u0000"…"\uffff"-"\n" } .
CCode = "#"  { "\u0000"…"\uffff"-"\n" } .
Ident = (alpha | "_" | ".") { "_" | "." | alpha | digit } .
String = "\"" [ { "\u0000"…"\uffff"-"\""-"\\" | "\\" any } ] "\"" .
Number = ( digit | "0x" | "." | "_" ) { digit | "." | "_" } .
Whitespace = " " | "\t" | "\n" | "\r" .
Digit = digit .
Punct = "!"…"/" | ":"…"@" | "["…` + "\"`\"" + ` | "{"…"~" .
alpha = "a"…"z" | "A"…"Z" .
digit = "0"…"9" .
EOL = ( "\n" | "\r" ) { "\n" | "\r" } .
any = "\u0000"…"\uffff" .
`))

	parser = participle.MustBuild(&tokens.File{},
		participle.Lexer(graphQLLexer),
		participle.Elide("Comment", "Whitespace"),
	)

	modulesToBuild = []string{}

	cli struct {
		Files []string `arg:"" type:"existingfile" required:"" help:"GraphQL schema files to parse."`
	}
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
		flattenValue(v, geckoAst)
	}
}

func flattenValue(value *tokens.Literal, geckoAst *ast.Ast) {
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

func updateMethodAst(a *ast.Ast) {
	if a.Parent != nil && a.Parent.Methods[a.Name] != nil {
		a.Merge(CompileEntries(a.Parent.Methods[a.Name].Value, a))
	}
}

func ParseFile(filename string) *tokens.File {
	file := &tokens.File{}
	baseDirectory, _ := path.Split(filename)
	filePath := string(os.PathSeparator) + filename
	wd, err := os.Getwd()
	searchDirectories := []string{wd, baseDirectory, config.GeckoConfig.StdLibPath, config.GeckoConfig.ModulesPath}

	var r *os.File

	for _, searchPath := range searchDirectories {
		compileLogger.DebugLogString("Trying import path", searchPath+filePath)
		r, err = os.Open(searchPath + filePath)
		if err != nil {
			compileLogger.DebugLogString("Trying import path", searchPath+filePath[:len(filePath)-2]+"/index.g")
			r, err = os.Open(searchPath + filePath[:len(filePath)-2] + "/index.g")
		}

		if err == nil {
			break
		}
	}

	if err != nil {
		compileLogger.Error("Couldn't resolve import", filename[:len(filename)-2])
		os.Exit(1)
	} else {
		finalFileName := r.Name()
		potentialBuildPath := path.Join(path.Dir(finalFileName), "build.json")
		compileLogger.LogString(finalFileName)

		if utils.FileExists(potentialBuildPath) && !funk.ContainsString(modulesToBuild, potentialBuildPath) {
			modulesToBuild = append(modulesToBuild, potentialBuildPath)
		}
	}

	// compileLogger.DebugLogString("Trying import path", wd+filePath)
	// if err != nil {
	// 	r, err = os.Open(baseDirectory + filePath)
	// 	compileLogger.DebugLogString("Trying import path", baseDirectory+filePath)
	// 	if err != nil {
	// 		r, err = os.Open(config.GeckoConfig.StdLibPath + filePath)
	// 		compileLogger.DebugLogString("Trying import path", config.GeckoConfig.StdLibPath+filePath)
	// 		if err != nil {
	// 			r, err = os.Open(config.GeckoConfig.ModulesPath + filePath)
	// 			compileLogger.DebugLogString("Trying import path", config.GeckoConfig.ModulesPath+filePath)
	// 			if err != nil {
	// 				compileLogger.Error("Couldn't resolve import", filename)
	// 				os.Exit(1)
	// 			}
	// 		}
	// 	}
	// }
	err = parser.Parse(r, file)
	if err != nil {
		tokErr := err.(participle.UnexpectedTokenError)
		geckoErr := errors.NewError(tokErr.Unexpected.Pos, "unexpected token "+tokErr.Unexpected.Value+" expected "+tokErr.Expected, &ast.Ast{
			Name: filename,
		})
		compileLogger.Error(geckoErr.String())
		os.Exit(1)
	}
	r.Close()
	file.Name = filename

	return file
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
			if entry.Field.Value != nil {
				if entry.Field.Value.FuncCall != nil {
					entry.Field.Value.Symbol = entry.Field.Name
				} else {
					flattenValue(entry.Field.Value, geckoAst)
				}
			}

			if variable.Value != nil && variable.Value.Array != nil {
				flattenArray(variable.Value.Array, geckoAst)
			}
			if variable.Visibility == "" {
				assignSymbolVisibility(variable)
			}
			geckoAst.Variables[entry.Field.Name] = variable
		} else if entry.Loop != nil {
			variable := &ast.Variable{}
			if entry.Loop.ForOf != nil {
				variable.FromToken(entry.Loop.ForOf.Variable)
				geckoAst.Variables[variable.Name] = variable
				variable.Scope = geckoAst
			}
		} else if entry.Method != nil {
			method := &ast.Method{}
			method.FromToken(entry.Method)
			// compileLogger.DebugLog(method)
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
			classAst = CompileEntries(classEntries, classAst)
			class.Merge(classAst)
			class.Ast = *classAst
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
			geckoAst.CPreliminary = geckoAst.CPreliminary + entry.CCode[1:len(entry.CCode)] + "\n"
		}
	}

	return geckoAst
}

func Init() {
	compileLogger.Init("compiler engine", 2)
}

func CompilePass(entryFile *tokens.File, geckoAst *ast.Ast, buildAll bool) (*ast.Ast, *ExecutionContext) {
	compiledAst := &ast.Ast{}
	compiledAst.Initialize()
	compiledAst.Name = entryFile.PackageName

	compileLogger.DebugLogString("Transpiling", color.HiYellowString("'%s'", entryFile.Name))

	// if (*config.GeckoConfig.Options)["no-stdlib"] != "true" {
	// 	compileLogger.Log(config.GeckoConfig)
	// 	// entryFile.Entries = append([]*tokens.Entry{&tokens.Entry{
	// 	// 	Import: "std",
	// 	// }}, entryFile.Entries...)
	// }

	for _, entry := range entryFile.Entries {
		if len(entry.Import) > 0 {
			importedFilePath := strings.ReplaceAll(entry.Import, ".", string(os.PathSeparator)) + ".g"
			entryFile.Imports = append(entryFile.Imports, ParseFile(importedFilePath))
		}
	}

	ctx := &ExecutionContext{}
	importedContexts := []*ExecutionContext{}
	for _, _import := range entryFile.Imports {
		importAst := &ast.Ast{}
		importAst.Name = _import.PackageName
		importAst.Initialize()
		compileLogger.DebugLogString("Branching into imported package", color.HiYellowString("'%s'", _import.PackageName))
		_ast, compileCtx := CompilePass(_import, importAst, buildAll)
		geckoAst.CPreliminary = _ast.CPreliminary + geckoAst.CPreliminary
		if _ast.Name == compiledAst.Name {
			compileLogger.DebugLogString("imported package", _ast.Name, "is part of the base package", compiledAst.Name)
			compiledAst.Merge(_ast)
		} else {
			compiledAst.MergeImport(_ast)
		}
		// ctx.Merge(compileCtx)
		importedContexts = append(importedContexts, compileCtx)
	}
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
	ctx.Ast = compiledAst
	for _, importedContext := range importedContexts {
		ctx.Merge(importedContext)
	}
	ctx.Merge(buildExecutionContext(entryFile.Entries, compiledAst, buildAll))
	return compiledAst, ctx
}
