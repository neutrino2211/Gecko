package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/neutrino2211/Gecko/commander"

	"github.com/mitchellh/go-homedir"
	"github.com/neutrino2211/Gecko/logger"
)

var (
	configLogger  = &logger.Logger{}
	defaultConfig = `
{
	"stdlibpath": "$root/std",
	"modulespath": "$root/src",
	"toolchainpath": "$root/toolchains",
	"version": "0.0.1",
	"defaultcompiler": "g++"
}
	`
	GeckoConfig = &Config{}
)

func readConfigJson(file string, cfg *Config) {
	configFile, err := os.Open(file)
	if err != nil {
		configLogger.LogString("opening config file", err.Error())
	}

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(cfg); err != nil {
		configLogger.LogString("parsing config file", err.Error())
	}
}

type Config struct {
	StdLibPath      string
	ModulesPath     string
	ToolchainPath   string
	Version         string
	DefaultCompiler string
	Options         *map[string]string
}

type BuildConfig struct {
	Type         string
	Output       string
	Sources      []string
	Headers      []string
	Toolchain    string
	Dependencies []*BuildConfig
	C            bool
	Root         bool
	Platform     string
	Arch         string
	Build        string
	Config       string
	Flags        []string
	Compiler     string

	Command commander.Commandable
}

func Init() {
	configLogger.Init("config", 6)
	home, err := homedir.Dir()
	geckoPath := path.Join(home, "gecko")
	configFilePath := path.Join(geckoPath, "config.json")
	GeckoConfig.Options = &map[string]string{}

	if err != nil {
		configLogger.Fatal(err.Error())
	}

	if _, err := os.Stat(geckoPath); os.IsNotExist(err) {
		os.MkdirAll(geckoPath, 0755)
	}

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		defaultConfig = strings.ReplaceAll(defaultConfig, "$root", geckoPath)
		err = ioutil.WriteFile(configFilePath, []byte(defaultConfig), 0755)

		if err != nil {
			configLogger.Fatal(err.Error())
		}
	}

	readConfigJson(configFilePath, GeckoConfig)
}
