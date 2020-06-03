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
	"path"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/neutrino2211/Gecko/ast"
	"github.com/neutrino2211/Gecko/commander"
	"github.com/neutrino2211/Gecko/compiler"
	"github.com/neutrino2211/Gecko/errors"
	"github.com/neutrino2211/Gecko/tokens"
)

type buildConfig struct {
	Type         string
	Output       string
	Sources      []string
	Headers      []string
	Toolchain    string
	Dependencies []*buildConfig
	C            bool
	Root         bool
	Platform     string
	Arch         string

	Command *CompileCommand
}

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

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func build(sources []string, cfg *buildConfig) []string {
	outDir, _ := os.Getwd()
	format := cfg.Type
	generateHeader := cfg.Type == "library"

	var inputFiles []string

	if len(sources) > 0 {
		inputFiles = sources
	} else {
		inputFiles = cfg.Sources
	}

	if cfg.Platform != runtime.GOOS {
		cfg.Command.Error("platform mismatch, current platform is", runtime.GOOS, "but source(s)", strings.Join(inputFiles, ", "), "require", cfg.Platform)
		return []string{}
	}

	if cfg.Arch != runtime.GOARCH {
		cfg.Command.Error("arch mismatch, current arch is", runtime.GOARCH, "but source(s)", strings.Join(inputFiles, ", "), "require", cfg.Arch)
		return []string{}
	}

	outputs := []string{}

	if len(cfg.Dependencies) > 0 {
		for _, dependency := range cfg.Dependencies {
			if dependency.Platform == "" {
				dependency.Platform = cfg.Platform
			}

			if dependency.Arch == "" {
				dependency.Arch = cfg.Arch
			}
			dependency.Command = cfg.Command
			dependencyOutputs := build([]string{}, dependency)
			outputs = append(outputs, dependencyOutputs...)
		}
	}

	for _, inputFile := range inputFiles {
		_, outFile := path.Split(inputFile)

		cfg.Command.DebugLog(len(sources))
		var outputPath string

		if len(sources) == 0 {
			possibleOutDir, outputFile := path.Split(cfg.Command.Values["build"])
			outputPath = outputFile
			outDir = possibleOutDir
			cfg.Command.DebugLogString(possibleOutDir)
			inputFile = path.Join(possibleOutDir, inputFile)
		}

		if outDir == "" {
			outDir = "."
		}

		if cfg.Command.Values["output"] != "" {
			outputPath = cfg.Command.Values["output"]
		} else {
			outputPath = outDir + string(os.PathSeparator) + strings.ReplaceAll(cfg.Output, "$", outFile)
		}

		if cfg.C {

			cfg.Command.LogString("compiling C file", inputFile)

			if format == "executable" {
				args := []string{"gcc", inputFile, "-o", outputPath}
				cmd := exec.Command(args[0], args[1:len(args)]...)
				streamCommand(cmd)
			} else if format == "library" {
				args := []string{"gcc", "-c", inputFile, "-I.", "-o", outputPath}
				cmd := exec.Command(args[0], args[1:len(args)]...)
				streamCommand(cmd)
			}

			outputs = append(outputs, outputPath)

			continue
		}

		if inputFile[len(inputFile)-2:len(inputFile)] != ".g" {
			break
		}

		_ast := compiler.ParseFile(inputFile)
		cfg.Command.DebugLogString(inputFile[len(inputFile)-2 : len(inputFile)-1])

		// i := 0
		// for i < len(_ast.Entries) {
		// 	entry := _ast.Entries[i]
		// 	if len(entry.Import) > 0 {

		// 	} else if entry.If != nil {
		// 		if _ast.Entries[i-1].If == nil && _ast.Entries[i-1].ElseIf == nil {
		// 			cfg.Command.DebugLogString("Good if statement")
		// 		} else if _ast.Entries[i-1].If != nil {
		// 			cfg.Command.DebugLogString("Double if found at", entry.If.Pos.String())
		// 		}
		// 	}
		// 	i++
		// }

		geckoAst := &ast.Ast{}
		geckoAst.Initialize()

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

		cfg.Command.LogString("compiling gecko package", _ast.PackageName)
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

		cfg.Command.DebugLogString(code)

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
		err := os.MkdirAll(path.Dir(filePath), 0755)
		err = ioutil.WriteFile(filePath, []byte(code), 0755)

		if err != nil {
			panic(err)
		}

		if format == "executable" {
			args := []string{"gcc", "-o", outputPath, filePath}
			args = append(args, outputs...)
			cmd := exec.Command(args[0], args[1:len(args)]...)
			streamCommand(cmd)
		} else if format == "library" {
			args := []string{"gcc", "-I.", "-o", outputPath, "-c", filePath}
			args = append(args, outputs...)
			cfg.Command.DebugLogString("ARGS:", strings.Join(args, " "))
			cmd := exec.Command(args[0], args[1:len(args)]...)
			streamCommand(cmd)
		}

		outputs = append(outputs, outputPath)
	}

	return outputs
}

func readBuildJson(file string, cfg *buildConfig) {
	configFile, err := os.Open(file)
	if err != nil {
		cfg.Command.Fatal("opening config file", err.Error())
	}

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(cfg); err != nil {
		cfg.Command.Fatal("parsing config file", err.Error()+".", file, "might not be a json file")
	}
}

type CompileCommand struct {
	commander.Command
}

func (c *CompileCommand) Init() {
	c.Optionals = map[string]*commander.Optional{
		"build": &commander.Optional{
			Type:        "string",
			Description: "Path to the project's build.json file.",
		},
		"output": &commander.Optional{
			Type:        "string",
			Description: "Output file path " + color.HiYellowString("(warning: this overrides the build configuration's output path)"),
		},
	}

	c.Usage = "gecko compile sources... [options]"

	c.Values = map[string]string{}

	c.Logger.Init(c.CommandName, 2)
	c.Description = c.BuildHelp(help)
}

func (c *CompileCommand) Run() {
	cfg := &buildConfig{}
	compiler.Init()

	cfg.Platform = runtime.GOOS
	cfg.Arch = runtime.GOARCH
	cfg.Command = c

	if len(c.Values["build"]) != 0 {
		readBuildJson(c.Values["build"], cfg)
	} else if len(c.Positionals) == 0 {
		c.Help()
		return
	}

	if c.Values["output"] != "" {
		cfg.Output = c.Values["output"]
	}

	c.DebugLog(cfg)
	cfg.Root = true

	outputs := build(c.Positionals, cfg)

	c.DebugLog(outputs)

	if fileExists(outputs[len(outputs)-1]) {
		color.Set(color.FgGreen)
		c.LogString("output saved to", outputs[len(outputs)-1])
		color.Unset()
	} else {
		c.Error("failed to save output to", outputs[len(outputs)-1], ". There should additional information above above")
	}
}

var (
	help = `compiles a gecko source file or a gecko project via a build.json file`
)
