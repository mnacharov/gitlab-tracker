package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	forceFlag    = flag.Bool("force", GetBoolEnv("GT_FORCE", false), "Force recreate tags.")
	logLevelFlag = flag.String("log-level", GetStringEnv("GT_LOG_LEVEL", "INFO"), "Level of logging.")
	validateFlag = flag.Bool("validate", false, "Validate config and exit.")
	versionFlag  = flag.Bool("version", false, "Prints version and exit.")
)

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Println(GetVersion())
		return
	}

	if err := ConfigureLogging(*logLevelFlag); err != nil {
		logrus.Fatal(err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		logrus.Fatal(err)
	}

	tracker, err := NewTracker(workDir)
	if err != nil {
		logrus.Fatal(err)
	}

	if *validateFlag {
		out, err := yaml.Marshal(tracker.config)
		if err != nil {
			logrus.Fatal(err)
		}
		fmt.Println(string(out))
		return
	}

	err = tracker.Run(*forceFlag)
	if err != nil {
		logrus.Fatal(err)
	}
}
