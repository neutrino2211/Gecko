package compiler

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

	"github.com/thoas/go-funk"

	"github.com/neutrino2211/Gecko/config"

	"github.com/fatih/color"
	"github.com/neutrino2211/Gecko/ast"
	"github.com/neutrino2211/Gecko/errors"
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
	compileLogger.LogString("executing command:", strings.Join(cmd.Args, " "))
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Start()
	streamPipe(stdout)
	streamPipe(stderr)
}

func BuildImportedModules(baseCfg *config.BuildConfig) {
	cfg := *baseCfg
	for _, config := range modulesToBuild[1:] {

		compileLogger.Log(builtModules)
		if funk.ContainsString(builtModules, config) {
			continue
		}

		if len(modulesToBuild) > 0 {
			modulesToBuild = modulesToBuild[1:]
		}

		compileLogger.LogString(color.HiYellowString("Building module: %s", config))
		configDir := path.Dir(config)
		os.Chdir(configDir)
		ReadBuildJson(config, &cfg)
		Build([]string{}, &cfg, make(map[string]string))
		os.Chdir(invokeDir)
		builtModules = append(builtModules, path.Join(path.Dir(config), cfg.Output))
	}
}

func Build(sources []string, cfg *config.BuildConfig, cmdLineArgs map[string]string) []string {
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
		compileLogger.Error("platform mismatch, current platform is", runtime.GOOS, "but source(s)", strings.Join(inputFiles, ", "), "require", cfg.Platform)
		return []string{}
	}

	if cfg.Arch != runtime.GOARCH {
		compileLogger.Error("arch mismatch, current arch is", runtime.GOARCH, "but source(s)", strings.Join(inputFiles, ", "), "require", cfg.Arch)
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

			if dependency.Compiler == "" {
				dependency.Compiler = cfg.Compiler
			}
			dependencyOutputs := Build([]string{}, dependency, cmdLineArgs)
			outputs = append(outputs, dependencyOutputs...)
		}
	}

	if cfg.Config != "" {
		depCfg := &config.BuildConfig{}
		depCfg.Platform = cfg.Platform
		depCfg.Arch = cfg.Arch
		depCfg.Compiler = cfg.Compiler

		ReadBuildJson(cfg.Config, depCfg)

		configDir := path.Dir(cfg.Config)

		os.Chdir(configDir)

		Build([]string{}, depCfg, cmdLineArgs)

		os.Chdir(invokeDir)
	}

	if cfg.Build != "" {
		rootDir, _ := path.Split(cmdLineArgs["build"])
		cfg.Build = strings.ReplaceAll(cfg.Build, "@{root}", rootDir)
		if len(cfg.Sources) > 0 {
			compileLogger.LogString(color.HiYellowString(
				"warning: build configuration contains both a build command and a list of sources. %s %s",
				"This might potentially cause issues during build",
				"["+cfg.Build+"]"))
		}
		if runtime.GOOS == "windows" {
			streamCommand(exec.Command("cmd", cfg.Build))
		} else {
			streamCommand(exec.Command("sh", "-c", cfg.Build))
		}
		outputs = append(outputs, cfg.Output)
	}

	for _, inputFile := range inputFiles {
		_, outFile := path.Split(inputFile)

		compileLogger.DebugLog(len(sources))
		var outputPath string

		if len(sources) == 0 {
			possibleOutDir, outputFile := path.Split(cmdLineArgs["build"])
			outputPath = outputFile
			outDir = possibleOutDir
			compileLogger.DebugLogString(possibleOutDir)
			inputFile = path.Join(possibleOutDir, inputFile)
		}

		if outDir == "" {
			outDir = "."
		}

		if cmdLineArgs["output"] != "" {
			outputPath = cmdLineArgs["output"]
		} else {
			outputPath = outDir + string(os.PathSeparator) + strings.ReplaceAll(cfg.Output, "$", outFile)
		}

		if cfg.C {

			compileLogger.LogString("compiling C file", inputFile)

			if format == "executable" {
				args := []string{cfg.Toolchain + cfg.Compiler, inputFile, "-o", outputPath}
				cmd := exec.Command(args[0], args[1:len(args)]...)
				streamCommand(cmd)
			} else if format == "library" {
				args := []string{cfg.Toolchain + cfg.Compiler, "-c", inputFile, "-I.", "-o", outputPath}
				cmd := exec.Command(args[0], args[1:len(args)]...)
				streamCommand(cmd)
			}

			outputs = append(outputs, outputPath)

			continue
		}

		if inputFile[len(inputFile)-2:len(inputFile)] != ".g" {
			continue
		}

		_ast := ParseFile(inputFile)
		compileLogger.DebugLogString(inputFile[len(inputFile)-2 : len(inputFile)-1])

		// i := 0
		// for i < len(_ast.Entries) {
		// 	entry := _ast.Entries[i]
		// 	if len(entry.Import) > 0 {

		// 	} else if entry.If != nil {
		// 		if _ast.Entries[i-1].If == nil && _ast.Entries[i-1].ElseIf == nil {
		// 			compileLogger.DebugLogString("Good if statement")
		// 		} else if _ast.Entries[i-1].If != nil {
		// 			compileLogger.DebugLogString("Double if found at", entry.If.Pos.String())
		// 		}
		// 	}
		// 	i++
		// }

		geckoAst := &ast.Ast{}
		geckoAst.Initialize()

		compileLogger.LogString("compiling gecko package", _ast.PackageName)
		a, ctx := CompilePass(_ast, geckoAst, true)

		if firstBuild {
			BuildImportedModules(cfg)
			firstBuild = false
		}

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
		code := ctx.Code(a)

		code = GetPreludeCode() + "\n" + code

		compileLogger.DebugLogString(color.HiYellowString("methods"), color.HiYellowString(GetPreludeCode()))

		codeLines := strings.Split(code, "\n")
		// fmt.Println(len(codeLines))
		if format != "object" {
			codeLines = codeLines[0 : len(codeLines)-1]
		} else {
			codeLines = codeLines[0:len(codeLines)]
		}

		code = a.CPreliminary + strings.Join(codeLines, "\n")

		if format == "executable" {
			mainArgs := CreateMethArgs(a.Methods["Main"].Arguments, a)
			passedArgs := ""

			for _, arg := range a.Methods["Main"].Arguments {
				passedArgs += arg.Name + ", "
			}

			if len(passedArgs) > 2 {
				passedArgs = passedArgs[0 : len(passedArgs)-2]
			}

			if ctx.Ast.Methods["Main"].Type != nil && ctx.Ast.Methods["Main"].Type.Type == "int" {
				code = code + "\nint main(" + mainArgs + "){return Main__Main(" + passedArgs + ");}\n"
			} else {
				code = code + "\nint main(" + mainArgs + "){Main__Main(" + passedArgs + "); return 0;}\n"
			}
		}

		compileLogger.DebugLogString(code)

		directory := os.TempDir() + string(os.PathSeparator)

		if generateHeader {
			headerFile := a.CPreliminary + "\n"
			for _, m := range ctx.Methods {
				mthd := a.Methods[m.Ast.Name]
				headerFile += GetTypeAsString(m.ReturnType, a) + " " + mthd.GetFullPath() + "(" + CreateMethArgs(mthd.Arguments, a) + ");\n"
			}

			ioutil.WriteFile(inputFile+".h", []byte(headerFile), 0755)
		}

		filePath := directory + inputFile[0:len(inputFile)-1] + (map[bool]string{true: "cc", false: "c"}[cfg.Compiler == "g++"])
		err := os.MkdirAll(path.Dir(filePath), 0755)
		err = ioutil.WriteFile(filePath, []byte(code), 0755)

		if err != nil {
			panic(err)
		}

		if format == "executable" {
			args := []string{cfg.Toolchain + cfg.Compiler, "-o", outputPath, filePath}
			args = append(args, outputs...)
			args = append(args, builtModules...)
			args = append(args, cfg.Flags...)
			cmd := exec.Command(args[0], args[1:len(args)]...)
			streamCommand(cmd)
		} else if format == "library" {
			args := []string{cfg.Toolchain + cfg.Compiler, "-I.", "-o", outputPath, "-c", filePath}
			args = append(args, outputs...)
			args = append(args, builtModules...)
			args = append(args, cfg.Flags...)
			cmd := exec.Command(args[0], args[1:len(args)]...)
			streamCommand(cmd)
		}

		outputPath = strings.Trim(outputPath, " ")

		outputs = append(outputs, outputPath)
	}

	return outputs
}

func ReadBuildJson(file string, cfg *config.BuildConfig) {
	configFile, err := os.Open(file)
	if err != nil {
		compileLogger.Fatal("opening config file", err.Error())
	}

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(cfg); err != nil {
		compileLogger.Fatal("parsing config file", err.Error()+".", file, "might not be a json file")
	}
}

var (
	invokeDir, _ = os.Getwd()
	firstBuild   = true
	builtModules = []string{}
)
