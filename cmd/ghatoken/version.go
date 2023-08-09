package main

import (
	"runtime/debug"
)

var Version string
var Revision string

func GetVersion() (string, string) {
	info, ok := debug.ReadBuildInfo()

	var version string

	if len(Version) > 0 {
		version = Version
	} else {
		if ok && len(info.Main.Version) > 0 {
			version = info.Main.Version
		} else {
			version = "(devel)"
		}
	}

	var revision string

	// prefer "vcs.revision" in BuildInfo.Settings to embedded value
	if ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && len(setting.Value) > 0 {
				revision = setting.Value[:7]
			}
		}
	}

	if len(revision) == 0 {
		revision = Revision
	}

	return version, revision
}
