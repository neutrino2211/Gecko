package commander

import (
	"strconv"
	"strings"
)

func confirmType(registeredType string, variable interface{}) bool {
	r := false
	switch variable.(type) {
	case string:
		r = registeredType == "string"
	case int64:
		r = registeredType == "int"
	case int32:
		r = registeredType == "int"
	case float64:
		r = registeredType == "float"
	case float32:
		r = registeredType == "float"
	case bool:
		r = registeredType == "bool"
	}

	return r
}

func fetchType(variable interface{}) string {
	r := "unknown"
	switch variable.(type) {
	case string:
		r = "string"
	case int64:
		r = "int"
	case int32:
		r = "int"
	case float64:
		r = "float"
	case float32:
		r = "float"
	case bool:
		r = "bool"
	}

	return r
}

func getValue(v string) interface{} {
	var r interface{}
	r, err := strconv.ParseBool(v)

	if err == nil {
		return r
	}

	r, err = strconv.ParseInt(v, 0, 32)

	if err == nil {
		return r
	}

	r, err = strconv.ParseFloat(v, 32)

	if err == nil {
		return r
	}

	return v
}

//Command : Interface describing properties held by command
type Command struct {
	Positionals []string
	Optionals   map[string]string
	Values      map[string]string
}

func (c *Command) RegisterOptional(option string, value string) {
	optType := c.Optionals[option]
	optValue := getValue(value)
	isRightType := confirmType(optType, optValue)

	if !isRightType {
		panic("Error: expected type " + optType + " for option '" + option + "' but got " + fetchType(optValue))
	}

	c.Values[option] = value
}

func (c *Command) RegisterPositionals(positionals []string) {
	c.Positionals = positionals
}

type Commandable interface {
	Run()
	Init()
	RegisterOptional(string, string)
	RegisterPositionals([]string)
}

//Commander : Command line parser
type Commander struct {
	commands map[string]Commandable
}

func (c *Commander) Init() {
	c.commands = make(map[string]Commandable)
}

func (c *Commander) Register(name string, cmd Commandable) {
	c.commands[name] = cmd
}

// Parse : Parses command line arguments
func (c *Commander) Parse(cmds []string) {
	cmdName := cmds[1]
	registeredCmd := c.commands[cmdName]
	registeredCmd.Init()
	// Check if we have that command registered
	if registeredCmd == nil {
		panic("Error: command '" + cmdName + "' not found")
	}

	positionals := []string{}

	for i := 2; i < len(cmds); i++ {
		cmd := cmds[i]
		if !strings.HasPrefix(cmd, "-") {
			positionals = append(positionals, cmd)
		} else if strings.HasPrefix(cmd, "--") {
			i++
			registeredCmd.RegisterOptional(cmd[2:len(cmd)], cmds[i])
		}
	}

	registeredCmd.RegisterPositionals(positionals)

	registeredCmd.Run()
}
