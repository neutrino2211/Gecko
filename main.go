package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"

	"github.com/neutrino2211/Gecko/ast"
	"github.com/neutrino2211/Gecko/config"

	"github.com/neutrino2211/Gecko/commander"
	"github.com/neutrino2211/Gecko/commands"
	"github.com/neutrino2211/Gecko/logger"
)

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
	for {
		_, err := stdout.Read(oneByte)
		if err != nil {
			fmt.Printf(err.Error())
			break
		}
		r := bufio.NewReader(stdout)
		line, _, _ := r.ReadLine()
		fmt.Println(string(line))
	}

	cmd.Wait()
}

func main() {
	// repr.Println(ast)

	//flags

	logger.SetDefaultChannel("Gecko")

	cmd := &commander.Commander{
		Ready: func() {
			ast.Init()
			config.Init()
		},
	}
	config.Init()
	cmd.Init()

	cmd.RegisterOption("debug", &commander.Listener{
		Option: &commander.Optional{
			Type:        "int",
			Description: "Set gecko's debug level (0 = quiet, 1 = show compile logs, 2 = verbose compile logs)",
		},

		Method: func(number interface{}) {
			logger.SetDefaultDebugMode(int(number.(int64)))
		},
	})

	cmd.RegisterOption("no-stdlib", &commander.Listener{
		Option: &commander.Optional{
			Type:        "bool",
			Description: "Do not include gecko's standard library when building",
		},

		Method: func(b interface{}) {
			(*config.GeckoConfig.Options)["no-stdlib"] = "true"
		},
	})

	cmd.RegisterCommands(commands.GeckoCommands)

	cmd.Parse(os.Args)

	return

}
