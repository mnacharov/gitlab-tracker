package main

import "testing"

func TestDiscoverConfigFile(t *testing.T) {
	_, err := DiscoverConfigFile("test_data/discover_rules")
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	_, err = DiscoverConfigFile("test_data/discover_rules/yml")
	if err != nil {
		t.Error(err)
	}
	_, err = DiscoverConfigFile("test_data/discover_rules/hcl")
	if err != nil {
		t.Error(err)
	}
	_, err = DiscoverConfigFile("test_data/discover_rules/json")
	if err != nil {
		t.Error(err)
	}
}
