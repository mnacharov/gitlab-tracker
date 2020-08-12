package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
	_, err := ProcessCommand(rule, []string{})
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	_, err = ProcessCommand(rule, []string{"{{.TTTT"})
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	_, err = ProcessCommand(rule, []string{"{{.TTTT}}"})
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
}

func TestGetBoolEnv(t *testing.T) {
	os.Unsetenv("FOOBAR")
	assert.Equal(t, true, GetBoolEnv("FOOBAR", true))
	os.Setenv("FOOBAR", "false")
	assert.Equal(t, false, GetBoolEnv("FOOBAR", true))
	os.Setenv("FOOBAR", "1")
	assert.Equal(t, true, GetBoolEnv("FOOBAR", false))
	os.Setenv("FOOBAR", "ABCD")
	assert.Equal(t, false, GetBoolEnv("FOOBAR", false))
}

func TestGetStringEnv(t *testing.T) {
	os.Unsetenv("FOOBAR")
	assert.Equal(t, "default", GetStringEnv("FOOBAR", "default"))
	os.Setenv("FOOBAR", "specified")
	assert.Equal(t, "specified", GetStringEnv("FOOBAR", "default"))
}
