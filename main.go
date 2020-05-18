package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"

	"github.com/neutrino2211/Gecko/ast"
	"github.com/neutrino2211/Gecko/compiler"
	"github.com/neutrino2211/Gecko/errors"
	"github.com/neutrino2211/Gecko/tokens"
)

var (
	graphQLLexer = lexer.Must(ebnf.New(`
Comment = "//"  { "\u0000"…"\uffff"-"\n" } .
CCode = "#"  { "\u0000"…"\uffff"-"\n" } .
Ident = (alpha | "_" | ".") { "_" | "." | alpha | digit } .
String = "\"" [ { "\u0000"…"\uffff"-"\""-"\\" | "\\" any } ] "\"" .
Number = ( "0x" | "." | "_" | digit) { "0x" |"." | "_" | digit} .
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

	cli struct {
		Files []string `arg:"" type:"existingfile" required:"" help:"GraphQL schema files to parse."`
	}
)

func ParseFile(filename string) *tokens.File {
	ast := &tokens.File{}
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	r, err := os.Open(wd + string(os.PathSeparator) + filename)
	if err != nil {
		panic(err)
	}
	err = parser.Parse(r, ast)
	r.Close()

	return ast
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func runCmdWithOutput(cmd *exec.Cmd) {
	stdout, _ := cmd.StdoutPipe()
	cmd.Start()
	oneByte := make([]byte, 100)
	num := 1
	for {
		_, err := stdout.Read(oneByte)
		if err != nil {
			fmt.Printf(err.Error())
			break
		}
		r := bufio.NewReader(stdout)
		line, _, _ := r.ReadLine()
		fmt.Println(string(line))
		num = num + 1
		if num > 3 {
			os.Exit(0)
		}
	}

	cmd.Wait()
}

func main() {
	// repr.Println(ast)

	//flags

	outfile := flag.String("o", "a", "Path to ouptut file")
	format := flag.String("f", "executable", "File out put format")
	gccFlags := flag.String("gflags", "", "Additional flags to pass on to gcc")
	// linkObjects := flag.String("l-objects", "", "Additional object files to link together with output (Used only with -f=object)")
	generateHeader := flag.String("header", "", "Generate header file for the output file (Mostly used with '-f=object')")
	flag.Parse()

	inputFiles := flag.Args()

	// repr.Println(inputFiles, outfile, format, gccFlags)

	for _, inputFile := range inputFiles {
		_ast := ParseFile(inputFile)
		println(inputFile[len(inputFile)-2 : len(inputFile)-1])
		if inputFile[len(inputFile)-2:len(inputFile)] != ".g" {
			break
		}
		i := 0
		for i < len(_ast.Entries) {
			entry := _ast.Entries[i]
			if len(entry.Import) > 0 {
				_ast.Imports = append(_ast.Imports, ParseFile(strings.ReplaceAll(entry.Import, ".", string(os.PathSeparator))+".g"))
			} else if entry.If != nil {
				if _ast.Entries[i-1].If == nil && _ast.Entries[i-1].ElseIf == nil {
					fmt.Println("Good if statement")
				} else if _ast.Entries[i-1].If != nil {
					fmt.Println("Double if found at", entry.If.Pos.String())
				}
			}
			i++
		}

		geckoAst := &ast.Ast{}
		geckoAst.Initialize()

		// v := _ast.Imports[0].Entries[1].Field.Value
		_ast.Entries = append(_ast.Entries,
			&tokens.Entry{
				Field: &tokens.Field{
					Name: "__version",
					Type: &tokens.TypeRef{
						Type: "string",
					},
					Value: &tokens.Literal{
						String: "\"0.0.1\"",
					},
				},
			},
		)

		// if *format != "object" {
		// 	_ast.Entries = append(_ast.Entries, &tokens.Entry{
		// 		FuncCall: &tokens.FuncCall{
		// 			Function:  "Main",
		// 			Arguments: []*tokens.Argument{},
		// 		},
		// 	})
		// }

		a, ctx := compiler.CompilePass(_ast, geckoAst, true)
		// compiledAst, _ = compiler.CompilePass(_ast, compiledAst)
		// compiledAst, ctx := compiler.CompilePass(_ast, compiledAst)
		// if v.Expression != nil {
		// 	fmt.Println(evaluate.Evaluate(v.Expression, compiledAst))
		// } else {
		// 	fmt.Println(v.String)
		// }

		if *format == "executable" && a.Methods["Main"] == nil {
			errors.AddError(&errors.Error{
				Pos:    _ast.Entries[len(_ast.Entries)-1].Pos,
				Reason: "No 'Main' function in file. Did you mean to build an object file?",
				Scope:  a,
			})
		}

		// repr.Println(ctx)

		if errors.HaveErrors() {
			for _, e := range errors.GetErrors() {
				fmt.Println(e.String())
			}

			os.Exit(1)
		}

		// repr.Println(ctx.Steps, ctx.Code(), a.CPreliminary)
		// repr.Println(ctx)
		code := ctx.Code()

		codeLines := strings.Split(code, "\n")
		// fmt.Println(len(codeLines))
		if *format != "object" {
			codeLines = codeLines[0 : len(codeLines)-1]
		} else {
			codeLines = codeLines[0:len(codeLines)]
		}

		code = a.CPreliminary + strings.Join(codeLines, "\n")

		if *format == "executable" {
			mainArgs := compiler.CreateMethArgs(a.Methods["Main"].Arguments, a)
			passedArgs := ""

			for _, arg := range a.Methods["Main"].Arguments {
				passedArgs += arg.Name + ", "
			}

			passedArgs = passedArgs[0 : len(passedArgs)-2]

			code = code + "\nint main(" + mainArgs + "){Main__Main(" + passedArgs + "); return 0;}\n"
		}

		println(code)

		directory := os.TempDir() + string(os.PathSeparator)

		if generateHeader != nil {
			headerFile := a.CPreliminary + "\n"
			for _, m := range ctx.Methods {
				mthd := a.Methods[m.Ast.Name]
				headerFile += compiler.GetTypeAsString(m.ReturnType, a) + " " + mthd.GetFullPath() + "(" + compiler.CreateMethArgs(mthd.Arguments, a) + ");\n"
				// compiler.GetTypeAsString()
			}

			ioutil.WriteFile(*generateHeader, []byte(headerFile), 0644)
		}

		if outfile != nil && *format != "object" {
			filePath := directory + inputFile[0:len(inputFile)-1] + "c"
			ioutil.WriteFile(filePath, []byte(code), 0644)
			// fmt.Println(err)
			// gccCmd := "gcc " + filePath + " -o " + *outfile + " " + *gccFlags
			// fmt.Println("Error:", err)
			args := []string{"gcc", filePath, "-o", *outfile, *gccFlags}
			// fmt.Println(strings.Join(args, " "))
			cmd := exec.Command(args[0], args[1:len(args)-1]...)
			runCmdWithOutput(cmd)
		} else if *format == "object" {
			filePath := directory + inputFile[0:len(inputFile)-1] + "c"
			ioutil.WriteFile(filePath, []byte(code), 0644)
			// fmt.Println(err)
			// gccCmd := "gcc " + filePath + " -o " + *outfile + " " + gccFlags
			args := []string{"gcc", "-c", filePath, "-I.", "-o", *outfile, *gccFlags}
			fmt.Println(strings.Join(args, " "))
			cmd := exec.Command(args[0], args[1:len(args)-1]...)
			runCmdWithOutput(cmd)
		}
	}
}
