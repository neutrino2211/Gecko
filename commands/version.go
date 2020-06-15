package commands

import (
	"github.com/neutrino2211/Gecko/commander"
	"github.com/neutrino2211/Gecko/config"
)

type VersionCommand struct {
	commander.Command
}

func (v *VersionCommand) Init() {
	v.Logger.Init(v.CommandName, 0)
	v.Usage = "gecko version"
	v.Description = v.BuildHelp(versionHelp)
}

func (v *VersionCommand) Run() {
	v.LogString("gecko", config.GeckoConfig.Version)
}

var (
	versionHelp = `shows the current working gecko language version`
)
