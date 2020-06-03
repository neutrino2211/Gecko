package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/mitchellh/go-homedir"
	"github.com/neutrino2211/Gecko/logger"
)

var (
	configLogger  = &logger.Logger{}
	defaultConfig = `
{
	"stdlibpath": "std",
	"modulespath": "src",
	"toolchainpath": "toolchains"
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
	StdLibPath    string
	ModulesPath   string
	ToolchainPath string
}

func Init() {
	home, err := homedir.Dir()
	geckoPath := path.Join(home, "gecko")
	configFilePath := path.Join(geckoPath, "config.json")

	if err != nil {
		configLogger.Fatal(err.Error())
	}

	if _, err := os.Stat(geckoPath); os.IsNotExist(err) {
		os.MkdirAll(geckoPath, 0755)
	}

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		err = ioutil.WriteFile(configFilePath, []byte(defaultConfig), 0755)

		if err != nil {
			configLogger.Fatal(err.Error())
		}
	}

	readConfigJson(configFilePath, GeckoConfig)
}
