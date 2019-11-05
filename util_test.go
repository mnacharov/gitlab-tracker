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
