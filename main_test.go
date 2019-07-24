package main

import (
	"os/exec"
	"regexp"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestIsChangesMatch(t *testing.T) {
	tests := []struct {
		rule    *Rule
		changes []string
		match   bool
	}{
		{
			rule: &Rule{
				Path: "abcd",
			},
			changes: []string{"foobar"},
			match:   false,
		},
		{
			rule: &Rule{
				Path: "foobar",
			},
			changes: []string{"foobar"},
			match:   true,
		},
		{
			rule: &Rule{
				Path: "foobar/**",
			},
			changes: []string{"foobar/a", "foobar/b"},
			match:   true,
		},
	}
	for _, test := range tests {
		_, match := test.rule.IsChangesMatch(test.changes)
		if match != test.match {
			t.Errorf("Must be %v, but got %v", test.match, match)
		}
	}
}

func TestDiff(t *testing.T) {
	g, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	tracker := &Tracker{
		logger: logrus.WithField("client", "git"),
		git:    g,
	}
	changes, err := tracker.Diff("4599ce4d09ef53a832d673fa471ecea52b69501d", "a7c947751dba7fc8ec1877baa33834c09d2a5df3")
	if err != nil {
		t.Error(err)
	}
	if len(changes) != 1 {
		t.Errorf("Must be 1, but got %d", len(changes))
	}
	tracker.git = "foobar"
	_, err = tracker.Diff("4599ce4d09ef53a832d673fa471ecea52b69501d", "a7c947751dba7fc8ec1877baa33834c09d2a5df3")
	if err == nil {
		t.Error("Must be an error")
	}
}

func TestDiffStat(t *testing.T) {
	g, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	tracker := &Tracker{
		logger: logrus.WithField("client", "git"),
		git:    g,
	}
	stat, err := tracker.DiffStat("4599ce4d09ef53a832d673fa471ecea52b69501d", "a7c947751dba7fc8ec1877baa33834c09d2a5df3", []string{"main.go"})
	if err != nil {
		t.Error(err)
	}
	if len(stat) == 0 {
		t.Error("Must be > 1, but got 0")
	}
	tracker.git = "foobar"
	_, err = tracker.DiffStat("4599ce4d09ef53a832d673fa471ecea52b69501d", "a7c947751dba7fc8ec1877baa33834c09d2a5df3", []string{"main.go"})
	if err == nil {
		t.Error("Must be an error")
	}
}

func TestLoadRules(t *testing.T) {
	tracker := &Tracker{
		logger: logrus.WithField("client", "git"),
	}
	err := tracker.LoadRules("test_data/not-found.yaml")
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	err = tracker.LoadRules("test_data/valid.yaml")
	if err != nil {
		t.Error(err)
	}
	err = tracker.LoadRules("test_data/invalid.yaml")
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	err = tracker.LoadRules("test_data/invalid_tag.yaml")
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
}

func TestGetTagSuffixForRule(t *testing.T) {
	tracker := &Tracker{
		dir: "./",
	}
	tests := []struct {
		rule   *Rule
		suffix string
	}{
		{
			rule: &Rule{
				TagSuffux: "static",
			},
			suffix: "@static",
		},
		{
			rule: &Rule{
				TagSuffuxFileRef: &TagSuffuxFileRef{
					File:   "test_data/suffix.yaml",
					RegExp: regexp.MustCompile(`eu.gcr.io/utilities-212509/argo/application:(.*)$`),
				},
			},
			suffix: "@master-459fb2b7",
		},
		{
			rule: &Rule{
				TagSuffuxFileRef: &TagSuffuxFileRef{
					File:   "test_data/suffix.yaml",
					RegExp: regexp.MustCompile(`foobar:(.*)$`),
				},
			},
			suffix: "",
		},
	}
	for _, test := range tests {
		suffix, err := tracker.GetTagSuffixForRule(test.rule)
		if err != nil {
			t.Error(err)
		}
		if suffix != test.suffix {
			t.Errorf("Must be %s, but got %s", test.suffix, suffix)
		}
	}
	rule := &Rule{}
	suffix, err := tracker.GetTagSuffixForRule(rule)
	if err != nil {
		t.Error(err)
	}
	if suffix != "" {
		t.Errorf("Must be empty, but got %s", suffix)
	}
	rule = &Rule{
		TagSuffuxFileRef: &TagSuffuxFileRef{
			File: "test_data/not-found.yaml",
		},
	}
	_, err = tracker.GetTagSuffixForRule(rule)
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
}
