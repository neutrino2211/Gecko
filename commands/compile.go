package commands

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/neutrino2211/Gecko/config"
	"github.com/neutrino2211/Gecko/logger"
	"github.com/neutrino2211/Gecko/utils"

	"github.com/fatih/color"
	"github.com/neutrino2211/Gecko/commander"
	"github.com/neutrino2211/Gecko/compiler"
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
	compileCommandLogger.LogString("executing command:", strings.Join(cmd.Args, " "))
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Start()
	streamPipe(stdout)
	streamPipe(stderr)
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
		"type": &commander.Optional{
			Type:        "string",
			Description: "Output type for program. (executable | library)",
		},
	}

	c.Usage = "gecko compile sources... [options]"

	c.Values = map[string]string{}

	compileCommandLogger.Init(c.CommandName, 2)
	c.Logger = *compileCommandLogger
	c.Description = c.BuildHelp(compileHelp)
}

func (c *CompileCommand) Run() {
	cfg := &config.BuildConfig{}
	compiler.Init()

	cfg.Platform = runtime.GOOS
	cfg.Arch = runtime.GOARCH
	cfg.Command = c
	cfg.Compiler = config.GeckoConfig.DefaultCompiler
	cfg.Type = "executable"

	if len(c.Values["build"]) != 0 {
		compiler.ReadBuildJson(c.Values["build"], cfg)
	} else if len(c.Positionals) == 0 {
		c.Help()
		return
	}

	if c.Values["output"] != "" {
		cfg.Output = c.Values["output"]
	} else if cfg.Output == "" {
		cfg.Output = "gecko.out"
	}

	if c.Values["type"] != "" {
		cfg.Type = c.Values["type"]
	}

	if cfg.Toolchain != "" {
		cfg.Toolchain += "-"
	}

	c.DebugLog(cfg)
	cfg.Root = true

	outputs := compiler.Build(c.Positionals, cfg, c.Values)

	c.DebugLog(outputs)

	if len(outputs) > 0 && utils.FileExists(outputs[len(outputs)-1]) {
		color.Set(color.FgGreen)
		c.LogString("output saved to", outputs[len(outputs)-1])
		color.Unset()
	} else if len(outputs) == 0 {
		c.Error("No outputs. There should additional information above above")
	} else {
		c.Error("failed to save output to", outputs[len(outputs)-1], ". There should be additional information above above")
	}
}

var (
	compileHelp          = `compiles a gecko source file or a gecko project`
	compileCommandLogger = &logger.Logger{}
	invokeDir, _         = os.Getwd()
)
