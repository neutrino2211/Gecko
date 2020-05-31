package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
	"github.com/alecthomas/repr"
	"github.com/neutrino2211/Gecko/ast"
	"github.com/neutrino2211/Gecko/commander"
	"github.com/neutrino2211/Gecko/compiler"
	"github.com/neutrino2211/Gecko/errors"
	"github.com/neutrino2211/Gecko/tokens"
)

type buildConfig struct {
	Type         string
	Output       string
	Toolchain    string
	Dependencies []string
}

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

func streamPipe(std io.ReadCloser) {
	buf := bufio.NewReader(std) // Notice that this is not in a loop
	for {

		line, _, err := buf.ReadLine()
		if err != nil {
			break
		}
		fmt.Println(string(line))
	}
}

func streamCommand(cmd *exec.Cmd) {
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Start()
	streamPipe(stdout)
	streamPipe(stderr)
}

func parseFile(filename string) *tokens.File {
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

func build(sources []string, cfg *buildConfig) {
	cwd, _ := os.Getwd()
	outfile := cwd + string(os.PathSeparator) + cfg.Output
	format := cfg.Type
	generateHeader := cfg.Type == "library"

	inputFiles := sources

	// repr.Println(inputFiles, outfile, format, gccFlags)

	for _, inputFile := range inputFiles {
		_ast := parseFile(inputFile)
		println(inputFile[len(inputFile)-2 : len(inputFile)-1])
		if inputFile[len(inputFile)-2:len(inputFile)] != ".g" {
			break
		}
		i := 0
		for i < len(_ast.Entries) {
			entry := _ast.Entries[i]
			if len(entry.Import) > 0 {
				_ast.Imports = append(_ast.Imports, parseFile(strings.ReplaceAll(entry.Import, ".", string(os.PathSeparator))+".g"))
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

		if format == "executable" && a.Methods["Main"] == nil {
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
		if format != "object" {
			codeLines = codeLines[0 : len(codeLines)-1]
		} else {
			codeLines = codeLines[0:len(codeLines)]
		}

		code = a.CPreliminary + strings.Join(codeLines, "\n")

		if format == "executable" {
			mainArgs := compiler.CreateMethArgs(a.Methods["Main"].Arguments, a)
			passedArgs := ""

			for _, arg := range a.Methods["Main"].Arguments {
				passedArgs += arg.Name + ", "
			}

			if len(passedArgs) > 2 {
				passedArgs = passedArgs[0 : len(passedArgs)-2]
			}

			code = code + "\nint main(" + mainArgs + "){Main__Main(" + passedArgs + "); return 0;}\n"
		}

		println(code)

		directory := os.TempDir() + string(os.PathSeparator)

		if generateHeader {
			headerFile := a.CPreliminary + "\n"
			for _, m := range ctx.Methods {
				mthd := a.Methods[m.Ast.Name]
				headerFile += compiler.GetTypeAsString(m.ReturnType, a) + " " + mthd.GetFullPath() + "(" + compiler.CreateMethArgs(mthd.Arguments, a) + ");\n"
			}

			ioutil.WriteFile(inputFile+".h", []byte(headerFile), 0755)
		}

		filePath := directory + inputFile[0:len(inputFile)-1] + "c"
		paths := strings.Split(filePath, string(os.PathSeparator))
		err := os.MkdirAll(strings.Join(paths[0:len(paths)-1], string(os.PathSeparator)), 0755)
		err = ioutil.WriteFile(filePath, []byte(code), 0755)

		if err != nil {
			panic(err)
		}

		if format != "library" {
			args := []string{"gcc", filePath, "-o", outfile}
			cmd := exec.Command(args[0], args[1:len(args)]...)
			streamCommand(cmd)
		} else if format == "library" {
			args := []string{"gcc", "-c", filePath, "-I.", "-o", outfile}
			cmd := exec.Command(args[0], args[1:len(args)]...)
			streamCommand(cmd)
		}
	}
}

func readBuildJson(file string, cfg *buildConfig) {
	configFile, err := os.Open(file)
	if err != nil {
		println("opening config file", err.Error())
	}

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(cfg); err != nil {
		println("parsing config file", err.Error())
	}
}

type CompileCommand struct {
	commander.Command
}

func (c *CompileCommand) Init() {
	c.Optionals = map[string]string{
		"build":  "string",
		"output": "string",
		"64bit":  "bool",
	}

	c.Values = map[string]string{
		"output": "main",
	}
}

func (c *CompileCommand) Run() {
	cfg := &buildConfig{}

	if len(c.Values["build"]) != 0 {
		readBuildJson(c.Values["build"], cfg)
	}

	repr.Println(cfg)

	build(c.Positionals, cfg)
}
