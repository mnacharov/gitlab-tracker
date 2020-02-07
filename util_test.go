package main

import (
	"strings"
	"testing"
)

func TestConfigureLogging(t *testing.T) {
	err := ConfigureLogging("foobar")
	if err == nil {
		t.Error(err)
	}
	if !strings.Contains(err.Error(), "not a valid logrus Level") {
		t.Error(err)
	}
	err = ConfigureLogging("debug")
	if err != nil {
		t.Error(err)
	}
}

func TestProcessCommand(t *testing.T) {
	rule := &Rule{
		Tag:           "tag",
		TagWithSuffix: "tag@suffix",
	}
	tests := []struct {
		hookCommand []string
		cmd         string
		args        []string
	}{
		{
			hookCommand: []string{"foobar"},
			cmd:         "foobar",
			args:        []string{},
		},
		{
			hookCommand: []string{"foo", "bar"},
			cmd:         "foo",
			args:        []string{"bar"},
		},
		{
			hookCommand: []string{"foobar", "{{.Tag}}", "{{.TagWithSuffix}}"},
			cmd:         "foobar",
			args:        []string{"tag", "tag@suffix"},
		},
	}
	for _, test := range tests {
		cmd, err := ProcessCommand(rule, test.hookCommand)
		if err != nil {
			t.Error(err)
		}
		if cmd.Path != test.cmd {
			t.Errorf("Must be %s, but got %s", test.cmd, cmd.Path)
		}
		if isSimilarStringMaps(cmd.Args, test.args) {
			t.Errorf("Must be %s, but got %s", test.args, cmd.Args)
		}
	}
	_, err := ProcessCommand(rule, []string{"{{.TTTT"})
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	_, err = ProcessCommand(rule, []string{"{{.TTTT}}"})
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
}
