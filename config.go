package main

import (
	"errors"
	"fmt"
	"os"
	"path"
)

const (
	configFilenameBase = ".gitlab-tracker"
)

var (
	supportedConfigExtensions = []string{
		"yml",
		"yaml",
		"hcl",
		"json",
	}
)

type Config struct {
	Checks        ChecksConfig     `yaml:"checks" hcl:"checks" json:"checks"`
	Hooks         HooksConfig      `yaml:"hooks" hcl:"hooks" json:"hooks"`
	Rules         map[string]*Rule `yaml:"rules" hcl:"rules" json:"rules"`
	Matrix        []string         `yaml:"matrix" hcl:"matrix" json:"matrix"`
	MatrixFromDir string           `yaml:"matrixFromDir" hcl:"matrix_from_dir" json:"matrixFromDir"`
}

type ChecksConfig struct {
	PreFlight  map[string]*Command `yaml:"preFlight" hcl:"pre_flight" json:"preFlight"`
	PostFlight map[string]*Command `yaml:"postFlight" hcl:"post_flight" json:"postFlight"`
}

type HooksConfig struct {
	PreProcess    map[string]*Command `yaml:"preProcess" hcl:"pre_process" json:"preProcess"`
	PostCreateTag map[string]*Command `yaml:"postCreateTag" hcl:"post_create_tag" json:"postCreateTag"`
	PostUpdateTag map[string]*Command `yaml:"postUpdateTag" hcl:"post_update_tag" json:"postUpdateTag"`
	PostProcess   map[string]*Command `yaml:"postProcess" hcl:"post_process" json:"postProcess"`
}

type Command struct {
	RetryConfig         *RetryConfig `yaml:"retry" hcl:"retry" json:"retry"`
	InitialDelaySeconds int          `yaml:"initialDelaySeconds" hcl:"initial_delay_seconds" json:"initialDelaySeconds"`
	AllowFailure        bool         `yaml:"allowFailure" hcl:"allow_failure" json:"allowFailure"`
	SkipOnFailure       bool         `yaml:"skipOnFailure" hcl:"skip_on_failure" json:"skipOnFailure"`
	Command             []string     `yaml:"command" hcl:"command" json:"command"`
}

func DiscoverConfigFile(dir string) (string, error) {
	for _, ext := range supportedConfigExtensions {
		filename := path.Join(dir, fmt.Sprintf("%s.%s", configFilenameBase, ext))
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			return filename, nil
		}
	}
	return "", errors.New("configuration file not found")
}
