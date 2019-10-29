package main

import (
	"os"
	"os/exec"
	"regexp"
	"testing"
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
		git: g,
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
		git: g,
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

func TestLoadRules_Basic(t *testing.T) {
	tracker := &Tracker{}
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

func TestLoadRules_Matrix(t *testing.T) {
	tracker := &Tracker{}
	err := tracker.LoadRules("test_data/invalid_matrix.yaml")
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	err = tracker.LoadRules("test_data/valid_matrix.yaml")
	if err != nil {
		t.Error(err)
	}
	if len(tracker.config.Rules) != len(tracker.config.Matrix) {
		t.Errorf("Must be %d, but got %d", len(tracker.config.Matrix), len(tracker.config.Rules))
	}
	tests := []string{
		"prepare-foobar1.sh",
		"prepare-foobar2.sh",
	}
	for i, path := range tests {
		if tracker.config.Rules[i].Path != path {
			t.Errorf("Must be %s, but got %s", path, tracker.config.Rules[0].Path)
		}
	}
}

func TestLoadRules_MatrixFromDir(t *testing.T) {
	tracker := &Tracker{}
	err := tracker.LoadRules("test_data/invalid_matrix_from_dir.yaml")
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	err = tracker.LoadRules("test_data/valid_matrix_from_dir.yaml")
	if err != nil {
		t.Error(err)
	}
	if len(tracker.config.Rules) != 3 {
		t.Errorf("Must be %d, but got %d", 3, len(tracker.config.Rules))
	}
	tests := []string{
		"prepare-itemA.sh",
		"prepare-itemB.sh",
		"prepare-itemC.sh",
	}
	for i, path := range tests {
		if tracker.config.Rules[i].Path != path {
			t.Errorf("Must be %s, but got %s", path, tracker.config.Rules[0].Path)
		}
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
					File:   "test_data/suffix_tag.yaml",
					RegExp: regexp.MustCompile(`eu.gcr.io/org/proj/application:(.*)$`),
				},
			},
			suffix: "@master-459fb2b7",
		},
		{
			rule: &Rule{
				TagSuffuxFileRef: &TagSuffuxFileRef{
					File:   "test_data/suffix_tag.yaml",
					RegExp: regexp.MustCompile(`foobar:(.*)$`),
				},
			},
			suffix: "",
		},
		{
			rule: &Rule{
				TagSuffixSeparator: "FOOBAR-",
				TagSuffuxFileRef: &TagSuffuxFileRef{
					File:   "test_data/suffix_digest.yaml",
					RegExp: regexp.MustCompile(`eu.gcr.io/org/proj/application[:@](.*)$`),
				},
			},
			suffix: "FOOBAR-sha256-391be4b7b42d1374f6578e850e74bc4977a1d35cc3adad1fcf0940f74f0ac379",
		},
		{
			rule: &Rule{
				TagSuffixSeparator: "FOOBAR-",
				TagSuffuxFileRef: &TagSuffuxFileRef{
					File:   "test_data/regexp_group_1.yaml",
					RegExp: regexp.MustCompile(`(application|eu.gcr.io/org/proj/application)[:@](.*)$`),
					Group:  2,
				},
			},
			suffix: "FOOBAR-1.0.0",
		},
		{
			rule: &Rule{
				TagSuffixSeparator: "FOOBAR-",
				TagSuffuxFileRef: &TagSuffuxFileRef{
					File:   "test_data/regexp_group_2.yaml",
					RegExp: regexp.MustCompile(`(application|eu.gcr.io/org/proj/application)[:@](.*)$`),
					Group:  2,
				},
			},
			suffix: "FOOBAR-sha256-391be4b7b42d1374f6578e850e74bc4977a1d35cc3adad1fcf0940f74f0ac379",
		},
	}
	for i, test := range tests {
		suffix, err := tracker.GetTagSuffixForRule(test.rule)
		if err != nil {
			t.Error(i, err)
		}
		if suffix != test.suffix {
			t.Errorf("%d. Must be %s, but got %s", i, test.suffix, suffix)
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

func isSimilarStringMaps(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, labelA := range a {
		found := false
		for _, labelB := range b {
			if labelA == labelB {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
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

var (
	vars = []string{
		"GITLAB_TOKEN",
		"CI_API_V4_URL",
		"CI_COMMIT_SHA",
		"CI_PROJECT_PATH",
	}
)

func fillEnvVars() {
	for _, v := range vars {
		os.Setenv(v, "1")
	}
}

func cleanupEnvVars() {
	for _, v := range vars {
		os.Unsetenv(v)
	}
}

func TestLoadEnvironment(t *testing.T) {
	fillEnvVars()
	tracker := &Tracker{}
	err := tracker.LoadEnvironment()
	if err != nil {
		t.Errorf("Must be nil, but got %v", err)
	}
	for id := len(vars) - 1; id >= 0; id-- {
		os.Unsetenv(vars[id])
		err := tracker.LoadEnvironment()
		if err == nil {
			t.Error("Must be an error, but got nil")
		}
	}
}

func TestNewTracker(t *testing.T) {
	fillEnvVars()
	_, err := NewTracker()
	if err != nil {
		t.Error(err)
	}
	cleanupEnvVars()
}

func TestPostTagHooks(t *testing.T) {
	tracker := &Tracker{}
	rule := &Rule{
		Path: "test_data/**",
		Tag:  "latest",
	}
	tracker.config.Hooks = HooksConfig{
		PostCreateTagCommand: []string{},
	}
	err := tracker.ExecHook(rule, tracker.config.Hooks.PostCreateTagCommand)
	if err != nil {
		t.Errorf("Must be nil, but got %v", err)
	}
	tracker.config.Hooks = HooksConfig{
		PostCreateTagCommand: []string{"whoami"},
	}
	err = tracker.ExecHook(rule, tracker.config.Hooks.PostCreateTagCommand)
	if err != nil {
		t.Errorf("Must be nil, but got %v", err)
	}
	tracker.config.Hooks = HooksConfig{
		PostCreateTagCommand: []string{"{{.FOOBAR}}"},
	}
	err = tracker.ExecHook(rule, tracker.config.Hooks.PostCreateTagCommand)
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	tracker.config.Hooks = HooksConfig{
		PostCreateTagCommand: []string{"not-found-binary"},
	}
	err = tracker.ExecHook(rule, tracker.config.Hooks.PostCreateTagCommand)
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
}

func TestExecCheck(t *testing.T) {
	tracker := Tracker{}
	tracker.config.Checks = ChecksConfig{
		RetryConfig:      nil,
		PreFlightCommand: []string{},
	}
	err := tracker.ExecCheck(tracker.config.Checks.PreFlightCommand)
	if err != nil {
		t.Error(err)
	}
	tracker.config.Checks = ChecksConfig{
		RetryConfig:      nil,
		PreFlightCommand: []string{"whoami"},
	}
	err = tracker.ExecCheck(tracker.config.Checks.PreFlightCommand)
	if err != nil {
		t.Error(err)
	}
	tracker.config.Checks = ChecksConfig{
		RetryConfig: &RetryConfig{
			Maximum: 1,
		},
		PreFlightCommand: []string{"not-found-binary"},
	}
	err = tracker.ExecCheck(tracker.config.Checks.PreFlightCommand)
	if err == nil {
		t.Errorf("Must be an error, but got nil")
	}
	tracker.config.Checks = ChecksConfig{
		RetryConfig: &RetryConfig{
			Maximum: 1,
		},
		PreFlightCommand: []string{"{{.FOOBAR}}"},
	}
	err = tracker.ExecCheck(tracker.config.Checks.PreFlightCommand)
	if err == nil {
		t.Errorf("Must be an error, but got nil")
	}
}
