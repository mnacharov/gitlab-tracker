package main

import (
	"os/exec"
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
}
