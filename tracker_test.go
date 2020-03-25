package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"testing"
	"time"

	"github.com/xanzy/go-gitlab"
)

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
	err = tracker.LoadRules("test_data/valid.hcl")
	if err != nil {
		t.Error(err)
	}
	err = tracker.LoadRules("test_data/valid.json")
	if err != nil {
		t.Error(err)
	}
	err = tracker.LoadRules("test_data/invalid.yaml")
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	err = tracker.LoadRules("test_data/invalid.hcl")
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	err = tracker.LoadRules("test_data/invalid.json")
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
	err := tracker.LoadRules("test_data/invalid_matrix_1.yaml")
	if err == nil {
		t.Fatal("Must be an error, but got nil")
	}
	tracker = &Tracker{}
	err = tracker.LoadRules("test_data/invalid_matrix_2.yaml")
	if err == nil {
		t.Fatal("Must be an error, but got nil")
	}
	tracker = &Tracker{}
	err = tracker.LoadRules("test_data/valid_matrix.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracker.config.Rules) != len(tracker.config.Matrix) {
		t.Errorf("Must be %d, but got %d", len(tracker.config.Matrix), len(tracker.config.Rules))
	}
	tests := map[string]string{
		"foobar1": "prepare-foobar1.sh",
		"foobar2": "prepare-foobar2.sh",
	}
	for name, path := range tests {
		r, ok := tracker.config.Rules[name]
		if !ok {
			t.Errorf("Rule name %s not found", name)
			continue
		}
		if r.Path != path {
			t.Errorf("Must be %s, but got %s", path, r.Path)
		}
	}
}

func TestLoadRules_MatrixFromDir(t *testing.T) {
	tracker := &Tracker{}
	err := tracker.LoadRules("test_data/invalid_matrix_from_dir.yaml")
	if err == nil {
		t.Fatal("Must be an error, but got nil")
	}
	tracker = &Tracker{}
	err = tracker.LoadRules("test_data/valid_matrix_from_dir.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracker.config.Rules) != 3 {
		t.Errorf("Must be %d, but got %d", 3, len(tracker.config.Rules))
	}
	tests := map[string]string{
		"matrix-0": "prepare-itemA.sh",
		"matrix-1": "prepare-itemB.sh",
		"matrix-2": "prepare-itemC.sh",
	}
	for name, path := range tests {
		r, ok := tracker.config.Rules[name]
		if !ok {
			t.Errorf("Rule name %s not found", name)
			continue
		}
		if r.Path != path {
			t.Errorf("Must be %s, but got %s", path, r.Path)
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
				TagSuffix: "static",
			},
			suffix: "@static",
		},
		{
			rule: &Rule{
				TagSuffixFileRef: &TagSuffixFileRef{
					File:   "test_data/suffix_tag.yaml",
					RegExp: regexp.MustCompile(`eu.gcr.io/org/proj/application:(.*)$`),
				},
			},
			suffix: "@master-459fb2b7",
		},
		{
			rule: &Rule{
				TagSuffixFileRef: &TagSuffixFileRef{
					File:   "test_data/suffix_tag.yaml",
					RegExp: regexp.MustCompile(`foobar:(.*)$`),
				},
			},
			suffix: "",
		},
		{
			rule: &Rule{
				TagSuffixSeparator: "FOOBAR-",
				TagSuffixFileRef: &TagSuffixFileRef{
					File:   "test_data/suffix_digest.yaml",
					RegExp: regexp.MustCompile(`eu.gcr.io/org/proj/application[:@](.*)$`),
				},
			},
			suffix: "FOOBAR-sha256-391be4b7b42d1374f6578e850e74bc4977a1d35cc3adad1fcf0940f74f0ac379",
		},
		{
			rule: &Rule{
				TagSuffixSeparator: "FOOBAR-",
				TagSuffixFileRef: &TagSuffixFileRef{
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
				TagSuffixFileRef: &TagSuffixFileRef{
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
		TagSuffixFileRef: &TagSuffixFileRef{
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
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fillEnvVars()
	_, err = NewTracker(dir)
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
		PostCreateTag: nil,
	}
	err := tracker.ExecCommandMap("PostCreateTag", tracker.config.Hooks.PostCreateTag, rule)
	if err != nil {
		t.Errorf("Must be nil, but got %v", err)
	}
	tracker.config.Hooks = HooksConfig{
		PostCreateTag: map[string]*Command{
			"foobar": {
				Command: []string{"whoami"},
			},
		},
	}
	err = tracker.ExecCommandMap("PostCreateTag", tracker.config.Hooks.PostCreateTag, rule)
	if err != nil {
		t.Errorf("Must be nil, but got %v", err)
	}
	tracker.config.Hooks = HooksConfig{
		PostCreateTag: map[string]*Command{
			"foobar": {
				RetryConfig: &RetryConfig{
					Maximum: 1,
				},
				Command: []string{"{{.FOOBAR}}"},
			},
		},
	}
	err = tracker.ExecCommandMap("PostCreateTag", tracker.config.Hooks.PostCreateTag, rule)
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	tracker.config.Hooks = HooksConfig{
		PostCreateTag: map[string]*Command{
			"foobar": {
				RetryConfig: &RetryConfig{
					Maximum: 1,
				},
				Command: []string{"not-found-binary"},
			},
		},
	}
	err = tracker.ExecCommandMap("PostCreateTag", tracker.config.Hooks.PostCreateTag, rule)
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
}

func TestExecCheck_PreFlight(t *testing.T) {
	tracker := Tracker{}
	tracker.config.Checks = ChecksConfig{
		PreFlight: nil,
	}
	err := tracker.ExecCommandMap(PreFlightCommandType, tracker.config.Checks.PreFlight, nil)
	if err != nil {
		t.Error(err)
	}
	tracker.config.Checks = ChecksConfig{
		PreFlight: map[string]*Command{
			"foobar": nil,
		},
	}
	err = tracker.ExecCommandMap(PreFlightCommandType, tracker.config.Checks.PreFlight, nil)
	if err != nil {
		t.Error(err)
	}
	tracker.config.Checks = ChecksConfig{
		PreFlight: map[string]*Command{
			"foobar": {
				Command: []string{},
			},
		},
	}
	err = tracker.ExecCommandMap(PreFlightCommandType, tracker.config.Checks.PreFlight, nil)
	if err != nil {
		t.Error(err)
	}
	tracker.config.Checks = ChecksConfig{
		PreFlight: map[string]*Command{
			"foobar": {
				Command: []string{"whoami"},
			},
		},
	}
	err = tracker.ExecCommandMap(PreFlightCommandType, tracker.config.Checks.PreFlight, nil)
	if err != nil {
		t.Error(err)
	}
	err = tracker.RunChecksPreFlight()
	if err != nil {
		t.Error(err)
	}
	tracker.config.Checks = ChecksConfig{
		PreFlight: map[string]*Command{
			"foobar": {
				RetryConfig: &RetryConfig{
					Maximum: 1,
				},
				Command: []string{"not-found-binary"},
			},
		},
	}
	err = tracker.ExecCommandMap(PreFlightCommandType, tracker.config.Checks.PreFlight, nil)
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	tracker.config.Checks = ChecksConfig{
		PreFlight: map[string]*Command{
			"foobar": {
				RetryConfig: &RetryConfig{
					Maximum: 1,
				},
				Command: []string{"{{.FOOBAR}}"},
			},
		},
	}
	err = tracker.ExecCommandMap(PreFlightCommandType, tracker.config.Checks.PreFlight, nil)
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
}

func TestExecCheck_PostFlight(t *testing.T) {
	tracker := Tracker{}
	tracker.config.Checks = ChecksConfig{
		PostFlight: nil,
	}
	err := tracker.ExecCommandMap(PostFlightCommandType, tracker.config.Checks.PostFlight, nil)
	if err != nil {
		t.Error(err)
	}
	tracker.config.Checks = ChecksConfig{
		PostFlight: map[string]*Command{
			"foobar": {
				Command: []string{"whoami"},
			},
		},
	}
	err = tracker.ExecCommandMap(PostFlightCommandType, tracker.config.Checks.PostFlight, nil)
	if err != nil {
		t.Error(err)
	}
	err = tracker.RunChecksPostFlight()
	if err != nil {
		t.Error(err)
	}
	tracker.config.Checks = ChecksConfig{
		PostFlight: map[string]*Command{
			"foobar": {
				InitialDelaySeconds: 2,
				RetryConfig: &RetryConfig{
					Maximum: 1,
				},
				Command: []string{"not-found-binary"},
			},
		},
	}
	st := time.Now()
	err = tracker.ExecCommandMap(PostFlightCommandType, tracker.config.Checks.PostFlight, nil)
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	if time.Since(st) < 2*time.Second {
		t.Error("InitialDelaySeconds=2 doesn't work as expected")
	}
	tracker.config.Checks = ChecksConfig{
		PostFlight: map[string]*Command{
			"foobar": {
				RetryConfig: &RetryConfig{
					Maximum: 1,
				},
				Command: []string{"{{.FOOBAR}}"},
			},
		},
	}
	err = tracker.ExecCommandMap(PostFlightCommandType, tracker.config.Checks.PostFlight, nil)
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	tracker.config.Checks = ChecksConfig{
		PostFlight: map[string]*Command{
			"foobar": {
				AllowFailure: true,
				RetryConfig: &RetryConfig{
					Maximum: 1,
				},
				Command: []string{"{{.FOOBAR}}"},
			},
		},
	}
	err = tracker.ExecCommandMap(PostFlightCommandType, tracker.config.Checks.PostFlight, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateTagIfNotExists(t *testing.T) {
	refAtStart := "000"
	refAtChange := "111"
	tracker := &Tracker{
		gitLab: NewFakeClient(),
		proj:   "ABCD",
		ref:    refAtStart,
	}
	_, _, err := tracker.CreateTagIfNotExists("foobar")
	if err != nil {
		t.Error(err)
	}
	tracker.ref = refAtChange
	_, _, err = tracker.CreateTagIfNotExists("foobar")
	if err != nil {
		t.Error(err)
	}
	tag, _, err := tracker.gitLab.GetTag(tracker.proj, "foobar")
	if err != nil {
		t.Error(err)
	}
	if tag.Commit.ID != refAtStart {
		t.Errorf("Must be %q, but got %q", refAtStart, tag.Commit.ID)
	}
}

func TestUpdateTag(t *testing.T) {
	tracker := &Tracker{
		gitLab: NewFakeClient(),
		git:    "git",
		proj:   "ABCD",
		ref:    "a7c947751dba7fc8ec1877baa33834c09d2a5df3",
	}
	err := tracker.UpdateTag(&gitlab.Tag{
		Commit: &gitlab.Commit{
			ID: "4599ce4d09ef53a832d673fa471ecea52b69501d",
		},
		Name:    "foobar",
		Message: "ABC",
	}, false, []string{"main.go"})
	if err != nil {
		t.Error(err)
	}
	err = tracker.UpdateTag(&gitlab.Tag{
		Commit: &gitlab.Commit{
			ID: "000",
		},
		Name:    "foobar",
		Message: "DFG",
	}, true, nil)
	if err != nil {
		t.Error(err)
	}
	_, _, err = tracker.gitLab.GetTag("ABCD", "foobar")
	if err != nil {
		t.Error(err)
	}
}

func TestTrackerPipeline1(t *testing.T) {
	repoDir, err := ioutil.TempDir("", "tracker-pipeline-1")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repoDir)
	le := &localExecutor{repoDir}
	if err := le.initWorkspace(); err != nil {
		t.Fatal(err)
	}
	commit, err := le.commit()
	if err != nil {
		t.Fatal(err)
	}
	tracker := &Tracker{
		gitLab: NewFakeClient(),
		git:    "git",
		proj:   "ABCD",
		ref:    commit,
		config: Config{
			Rules: map[string]*Rule{
				"foobar": {
					Path: repoDir,
					Tag:  "1.0.0",
				},
			},
		},
	}
	if err := tracker.Run(false); err != nil {
		t.Fatalf("tracker.Run error: %v", err)
	}
	tag, _, err := tracker.gitLab.GetTag("ABCD", "1.0.0", nil)
	if err != nil {
		t.Fatalf("GetTag error: %v", err)
	}
	if tag.Commit.ID != commit {
		t.Errorf("Tag commit must be %s, but got %s", commit, tag.Commit)
	}
	// Diff returns an error
	if err := tracker.Run(false); err == nil {
		t.Fatal("Must be en error, but got nil")
	}
}

func TestTrackerPipeline2(t *testing.T) {
	repoDir, err := ioutil.TempDir("", "tracker-pipeline-2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repoDir)
	le := &localExecutor{repoDir}
	if err := le.initWorkspace(); err != nil {
		t.Fatal(err)
	}
	commit, err := le.commit()
	if err != nil {
		t.Fatal(err)
	}
	commitAtStart := commit
	tracker := &Tracker{
		gitLab: NewFakeClient(),
		git:    "git",
		proj:   "ABCD",
		ref:    commit,
		dir:    repoDir,
		config: Config{
			Checks: ChecksConfig{
				PreFlight: map[string]*Command{
					"pre": {
						RetryConfig: &RetryConfig{
							Maximum: 1,
						},
						Command: []string{
							"whoami11",
						},
					},
				},
			},
			Rules: map[string]*Rule{
				"foobar": {
					Tag: "foobar",
					TagSuffixFileRef: &TagSuffixFileRef{
						File:   "test_file",
						RegExp: regexp.MustCompile(`foobar:(.+)`),
					},
				},
			},
		},
	}
	if err := tracker.Run(false); err == nil {
		t.Error("Must be an error, but got nil")
	}
	tracker.config.Checks.PreFlight["pre"].Command = []string{"whoami"}
	if err := tracker.Run(false); err != nil {
		t.Fatalf("tracker.Run error: %v", err)
	}
	tag, _, err := tracker.gitLab.GetTag("ABCD", "foobar@1.0.0", nil)
	if err != nil {
		t.Fatalf("GetTag error: %v", err)
	}
	if tag.Commit.ID != commit {
		t.Errorf("Tag commit must be %s, but got %s", commit, tag.Commit.ID)
	}
	body := `image: foobar:2.0.0`
	if err := ioutil.WriteFile(path.Join(le.wd, "test_file"), []byte(body), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := le.addAndCommit(); err != nil {
		t.Fatal(err)
	}
	tracker.beforeRef = commit
	commit, err = le.commit()
	if err != nil {
		t.Fatal(err)
	}
	tracker.ref = commit
	if err := tracker.Run(false); err != nil {
		t.Fatalf("tracker.Run error: %v", err)
	}
	tag, _, err = tracker.gitLab.GetTag("ABCD", "foobar@2.0.0", nil)
	if err != nil {
		t.Fatalf("GetTag error: %v", err)
	}
	if tag.Commit.ID != commit {
		t.Errorf("Tag commit must be %s, but got %s", commit, tag.Commit.ID)
	}
	body = `image: foobar:1.0.0`
	if err := ioutil.WriteFile(path.Join(le.wd, "test_file"), []byte(body), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := le.addAndCommit(); err != nil {
		t.Fatal(err)
	}
	tracker.beforeRef = commit
	commit, err = le.commit()
	if err != nil {
		t.Fatal(err)
	}
	tracker.ref = commit
	if err := tracker.Run(false); err != nil {
		t.Fatalf("tracker.Run error: %v", err)
	}
	tag, _, err = tracker.gitLab.GetTag("ABCD", "foobar@1.0.0", nil)
	if err != nil {
		t.Fatalf("GetTag error: %v", err)
	}
	// After switching to existing tag we must check commit sha
	if tag.Commit.ID != commitAtStart {
		t.Errorf("Tag commit must be %s, but got %s", commitAtStart, tag.Commit.ID)
	}
}

func TestTrackerPipeline3(t *testing.T) {
	repoDir, err := ioutil.TempDir("", "tracker-pipeline-3")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repoDir)
	le := &localExecutor{repoDir}
	if _, err := le.exec([]string{"git", "init"}); err != nil {
		t.Fatal(err)
	}
	apps := []string{"app1", "app2", "app3", "app4"}
	rules := make(map[string]*Rule)
	for _, app := range apps {
		err := os.Mkdir(path.Join(repoDir, app), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
		err = ioutil.WriteFile(path.Join(repoDir, app, "application"), []byte(fmt.Sprintf("%s:1.0.0", app)), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
		rules[app] = &Rule{
			Path: path.Join(app, "**"),
			Tag:  app,
			TagSuffixFileRef: &TagSuffixFileRef{
				File:   path.Join(app, "application"),
				RegExp: regexp.MustCompile(`[\-\w]+[:@](.*)$`),
			},
		}
	}
	if err := le.addAndCommit(); err != nil {
		t.Fatal(err)
	}
	commit, err := le.commit()
	if err != nil {
		t.Fatal(err)
	}
	tracker := &Tracker{
		gitLab: NewFakeClient(),
		git:    "git",
		proj:   "ABCD",
		ref:    commit,
		dir:    repoDir,
		config: Config{
			Rules: rules,
		},
	}
	if err := tracker.Run(false); err != nil {
		t.Fatal(err)
	}
	for _, app := range apps {
		tagName := fmt.Sprintf("%s@1.0.0", app)
		tag, _, err := tracker.gitLab.GetTag("ABCD", tagName, nil)
		if err != nil {
			t.Fatal(err)
		}
		if tag.Commit.ID != tracker.ref {
			t.Errorf("Tag %s commit must be %s, but got %s", tagName, tracker.ref, tag.Commit.ID)
		}
	}
	err = ioutil.WriteFile(path.Join(repoDir, "app1", "test_file"), []byte(`test_file`), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(path.Join(repoDir, "app2", "application"), []byte(`app2:2.0.0`), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	if err := le.addAndCommit(); err != nil {
		t.Fatal(err)
	}
	tracker.beforeRef = commit
	commit, err = le.commit()
	if err != nil {
		t.Fatal(err)
	}
	tracker.ref = commit
	if err := tracker.Run(false); err != nil {
		t.Fatal(err)
	}
	tag, _, err := tracker.gitLab.GetTag("ABCD", "app1@1.0.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if tag.Commit.ID != tracker.ref {
		t.Errorf("Tag %s commit must be %s, but got %s", tag.Name, tracker.ref, tag.Commit.ID)
	}
	tagApp2, _, err := tracker.gitLab.GetTag("ABCD", "app2@1.0.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	tag, _, err = tracker.gitLab.GetTag("ABCD", "app2@2.0.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if tag.Commit.ID != tracker.ref {
		t.Errorf("Tag %s commit must be %s, but got %s", tag.Name, tracker.ref, tag.Commit.ID)
	}
	for _, app := range apps[1:] {
		tagName := fmt.Sprintf("%s@1.0.0", app)
		tag, _, err := tracker.gitLab.GetTag("ABCD", tagName, nil)
		if err != nil {
			t.Fatal(err)
		}
		if tag.Commit.ID == tracker.ref {
			t.Errorf("Tag %s commit must be %s, but got %s", tagName, tracker.ref, tag.Commit.ID)
		}
	}
	err = ioutil.WriteFile(path.Join(repoDir, "app2", "application"), []byte(`app2:1.0.0`), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	if err := le.addAndCommit(); err != nil {
		t.Fatal(err)
	}
	tracker.beforeRef = commit
	commit, err = le.commit()
	if err != nil {
		t.Fatal(err)
	}
	tracker.ref = commit
	if err := tracker.Run(false); err != nil {
		t.Fatal(err)
	}
	tag, _, err = tracker.gitLab.GetTag("ABCD", "app2@1.0.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if tag.Commit.ID != tracker.ref {
		t.Errorf("Tag %s commit must be %s, but got %s", tag.Name, tracker.ref, tag.Commit.ID)
	}
	if tagApp2.Commit.ID == tag.Commit.ID {
		t.Errorf("Tag %s commit must be %s not the same as before update", tag.Name, tag.Commit.ID)
	}
}
