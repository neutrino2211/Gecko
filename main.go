package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"

	"github.com/neutrino2211/Gecko/commander"
	"github.com/neutrino2211/Gecko/commands"
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

	cmd := &commander.Commander{}
	cmd.Init()

	cmd.Register("compile", &commands.CompileCommand{})

	cmd.Parse(os.Args)

	return

}
