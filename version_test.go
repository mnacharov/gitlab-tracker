package main

import "testing"

func TestGetVersion(t *testing.T) {
	ver := GetVersion()
	if len(ver) == 0 {
		t.Error("Version can't be empty")
	}
	Version = "1.0.0"
	ver = GetVersion()
	if ver != Version {
		t.Errorf("Version must be %s, but got %s", Version, ver)
	}
}
