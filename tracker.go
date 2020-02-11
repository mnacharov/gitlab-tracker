package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/hcl"
	"github.com/sirupsen/logrus"
	gitlab "github.com/xanzy/go-gitlab"
	"gopkg.in/yaml.v2"
)

type CommandType string

const (
	tagMessage          = "Auto-generated. Do not Remove."
	errTagNotFound      = "Tag Not Found"
	configFilenameBase  = ".gitlab-tracker"
	descriptionTemplate = "<details><summary>Details</summary><pre><code>%s</code></pre></details>"

	defaultTagSuffixSeparator = "@"

	PreProcessCommandType    CommandType = "PreProcess"
	PostCreateTagCommandType CommandType = "PostCreateTag"
	PostUpdateTagCommandType CommandType = "PostUpdateTag"
	PostProcessCommandType   CommandType = "PostProcess"
	PreFlightCommandType     CommandType = "PreFlight"
	PostFlightCommandType    CommandType = "PostFlight"
)

var (
	httpCli = &http.Client{
		Timeout:   time.Second * 10,
		Transport: RetryTransport(),
	}
	tagSuffixReplacer = strings.NewReplacer("/", "", ":", "-")
)

type Tracker struct {
	dir         string
	git         string
	gitLabToken string
	gitLabURL   string
	beforeRef   string
	ref         string
	proj        string
	gitLab      gitlabClient
	config      Config
}

func NewTracker() (*Tracker, error) {
	g, err := exec.LookPath("git")
	if err != nil {
		return nil, err
	}
	d, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	t := &Tracker{
		git: g,
		dir: d,
	}
	filename, err := t.DiscoverConfigFile(d)
	if err != nil {
		return nil, err
	}
	err = t.LoadRules(filename)
	if err != nil {
		return nil, err
	}
	err = t.LoadEnvironment()
	if err != nil {
		return nil, err
	}

	cli := gitlab.NewClient(httpCli, t.gitLabToken)
	err = cli.SetBaseURL(t.gitLabURL)
	if err != nil {
		return nil, err
	}
	t.gitLab = gitlabRealClient(*cli)
	return t, nil
}

func (t *Tracker) GetTagSuffixForRule(r *Rule) (string, error) {
	separator := r.TagSuffixSeparator
	if len(separator) == 0 {
		separator = defaultTagSuffixSeparator
	}
	if len(r.TagSuffix) > 0 {
		suffix := tagSuffixReplacer.Replace(r.TagSuffix)
		return separator + suffix, nil
	}
	if r.TagSuffixFileRef != nil {
		suffix, err := r.TagSuffixFileRef.GetSuffix(t.dir)
		if err != nil {
			return "", err
		}
		if len(suffix) > 0 {
			suffix = tagSuffixReplacer.Replace(suffix)
			return separator + suffix, nil
		}
	}
	return "", nil
}

func (t *Tracker) ProcessRule(rule *Rule, force bool) error {
	suffix, err := t.GetTagSuffixForRule(rule)
	if err != nil {
		return err
	}
	rule.TagWithSuffix = rule.Tag
	if len(suffix) > 0 {
		rule.TagWithSuffix = rule.Tag + suffix
	}
	err = t.ExecCommandMap(PreProcessCommandType, t.config.Hooks.PreProcess, rule)
	if err != nil {
		return err
	}
	err = t.processRule(rule, force)
	if err != nil {
		return err
	}
	return t.ExecCommandMap(PostProcessCommandType, t.config.Hooks.PostProcess, rule)
}

func (t *Tracker) processRule(rule *Rule, force bool) error {
	var (
		matches []string
		match   bool
	)
	exists, tag, err := t.CreateTagIfNotExists(rule.TagWithSuffix)
	if err != nil {
		return err
	}
	if !exists {
		return t.ExecCommandMap(PostCreateTagCommandType, t.config.Hooks.PostCreateTag, rule)
	}
	destRef := rule.TagWithSuffix
	if len(t.beforeRef) > 0 {
		// Try to use CI_COMMIT_BEFORE_SHA
		destRef = t.beforeRef
	}
	changesHead, err := t.Diff(t.ref, destRef)
	if err != nil {
		return err
	}
	if t.ref != tag.Commit.ID {
		changesTag, err := t.Diff(t.ref, tag.Commit.ID)
		if err == nil {
			matches, match = rule.IsChangesMatch(changesTag)
		}
	}
	matchesHead, matchHead := rule.IsChangesMatch(changesHead)
	if matchHead {
		match = true
		matches = matchesHead
	}
	if !match {
		logrus.Debug("Nothing changed.")
		return nil
	}
	err = t.UpdateTag(tag, true, matches)
	if err != nil {
		return err
	}
	return t.ExecCommandMap(PostUpdateTagCommandType, t.config.Hooks.PostUpdateTag, rule)
}

func (t *Tracker) RunChecksPreFlight() error {
	if len(t.config.Checks.PreFlight) == 0 {
		return nil
	}
	err := t.ExecCommandMap(PreFlightCommandType, t.config.Checks.PreFlight, nil)
	if err != nil {
		return fmt.Errorf("pre flight checks: failed. %v", err)
	}
	logrus.Info("Pre flight checks: passed")
	return nil
}

func (t *Tracker) RunChecksPostFlight() error {
	if len(t.config.Checks.PostFlight) == 0 {
		return nil
	}
	err := t.ExecCommandMap(PostFlightCommandType, t.config.Checks.PostFlight, nil)
	if err != nil {
		return fmt.Errorf("post flight checks: failed. %v", err)
	}
	logrus.Info("Post flight checks: passed")
	return nil
}

func (t *Tracker) Run(force bool) error {
	if err := t.RunChecksPreFlight(); err != nil {
		return err
	}
	err := t.UpdateTags(force)
	if err != nil {
		return err
	}
	return t.RunChecksPostFlight()
}

func (t *Tracker) UpdateTags(force bool) error {
	var failed bool
	for _, rule := range t.config.Rules {
		err := t.ProcessRule(rule, force)
		if err != nil {
			failed = true
			logrus.Error(err)
		}
	}
	if failed {
		return errors.New("failed")
	}
	return nil
}

func (t *Tracker) ExecCommandMap(commandType CommandType, commands map[string]*Command, rule *Rule) error {
	for name, command := range commands {
		if command == nil || len(command.Command) == 0 {
			continue
		}
		if command.InitialDelaySeconds > 0 {
			time.Sleep(time.Duration(command.InitialDelaySeconds) * time.Second)
		}
		err := Retry(func(s *Stats) error {
			logrus.Debugf("Exec %v as %s command (%s).", command.Command, commandType, s)
			cmd, err := ProcessCommand(rule, command.Command)
			if err != nil {
				return err
			}
			b, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("%v: %s", err, string(b))
			}
			return nil
		}, command.RetryConfig)
		if !command.AllowFailure && err != nil {
			return fmt.Errorf("%s %s: %v", commandType, name, err)
		}
	}
	return nil
}

func (t *Tracker) CreateTagIfNotExists(tagName string) (bool, *gitlab.Tag, error) {
	tag, _, err := t.gitLab.GetTag(t.proj, tagName, nil)
	if err != nil && !strings.Contains(err.Error(), errTagNotFound) {
		return false, nil, err
	}
	if tag != nil {
		return true, tag, nil
	}
	logrus.Infof("Create '%s' tag.", tagName)
	tag, err = t.CreateTagForRef(tagName, t.ref)
	return false, tag, err
}

func (t *Tracker) CreateTagForRef(tagName, ref string) (*gitlab.Tag, error) {
	logrus.Infof("Create '%s' tag with %s ref.", tagName, ref)
	opts := &gitlab.CreateTagOptions{
		TagName: gitlab.String(tagName),
		Ref:     gitlab.String(ref),
		Message: gitlab.String(tagMessage),
	}
	tag, _, err := t.gitLab.CreateTag(t.proj, opts, nil)
	return tag, err
}

func (t *Tracker) UpdateTag(tag *gitlab.Tag, force bool, changes []string) error {
	if force {
		_, err := t.gitLab.DeleteTag(t.proj, tag.Name, nil)
		if err != nil && !strings.Contains(err.Error(), errTagNotFound) {
			return err
		}
	}
	_, err := t.CreateTagForRef(tag.Name, t.ref)
	if err != nil {
		return err
	}
	if changes == nil {
		return nil
	}
	stat, err := t.DiffStat(tag.Commit.ID, t.ref, changes)
	if err != nil {
		return err
	}
	if len(stat) == 0 {
		return nil
	}
	message := fmt.Sprintf(descriptionTemplate, stat)
	opts := &gitlab.CreateReleaseOptions{
		Name:        gitlab.String(tag.Name),
		TagName:     gitlab.String(tag.Name),
		Description: gitlab.String(message),
	}
	_, _, err = t.gitLab.CreateRelease(t.proj, opts)
	if err != nil {
		logrus.Warningf("Failed to create release: %v", err)
	}
	return nil
}

func (t *Tracker) LoadEnvironment() error {
	token := os.Getenv("GITLAB_TOKEN")
	if len(token) == 0 {
		return errors.New("GITLAB_TOKEN must be specified")
	}
	t.gitLabToken = token
	baseURL := os.Getenv("CI_API_V4_URL")
	if len(baseURL) == 0 {
		return errors.New("CI_API_V4_URL must be specified")
	}
	t.gitLabURL = baseURL
	t.beforeRef = os.Getenv("CI_COMMIT_BEFORE_SHA")
	ref := os.Getenv("CI_COMMIT_SHA")
	if len(ref) == 0 {
		return errors.New("CI_COMMIT_SHA must be specified")
	}
	t.ref = ref
	proj := os.Getenv("CI_PROJECT_PATH")
	if len(proj) == 0 {
		return errors.New("CI_PROJECT_PATH must be specified")
	}
	t.proj = proj
	return nil
}

func (t *Tracker) templateRulesWithMatrixFromDir(rule *Rule) error {
	fi, err := ioutil.ReadDir(t.config.MatrixFromDir)
	if err != nil {
		return err
	}
	parsedRules := map[string]*Rule{}
	var i int
	for _, item := range fi {
		if !item.IsDir() {
			continue
		}
		ref := rule.Clone()
		err := ref.ParseAsTemplate(map[string]string{
			"Item": path.Base(item.Name()),
		})
		if err != nil {
			return err
		}
		parsedRules[fmt.Sprintf("matrix-%d", i)] = ref
		i++
	}
	t.config.Rules = parsedRules
	return nil
}

func (t *Tracker) templateRulesWithMatrixRaw(rule *Rule) error {
	parsedRules := map[string]*Rule{}
	for _, item := range t.config.Matrix {
		ref := rule.Clone()
		err := ref.ParseAsTemplate(map[string]string{
			"Item": item,
		})
		if err != nil {
			return err
		}
		parsedRules[item] = ref
	}
	t.config.Rules = parsedRules
	return nil
}

func (t *Tracker) TemplateRulesWithMatrix() error {
	if len(t.config.Matrix) == 0 && len(t.config.MatrixFromDir) == 0 {
		return nil
	}
	if len(t.config.Rules) > 1 || len(t.config.Rules) == 0 {
		return errors.New("matrix can be used only with single rule")
	}
	matrixRule, ok := t.config.Rules["matrix"]
	if !ok {
		return errors.New("matrix can be used only with rule that named as `matrix`")
	}
	if len(t.config.Matrix) > 0 {
		return t.templateRulesWithMatrixRaw(matrixRule)
	}
	return t.templateRulesWithMatrixFromDir(matrixRule)
}

func (t *Tracker) DiscoverConfigFile(dir string) (string, error) {
	exts := []string{"yml", "yaml", "hcl", "json"}
	for _, ext := range exts {
		filename := path.Join(dir, fmt.Sprintf("%s.%s", configFilenameBase, ext))
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			return filename, nil
		}
	}
	return "", errors.New("configuration file not found")
}

func (t *Tracker) LoadRules(filename string) error {
	logrus.Debugf("Configuration file: %s", filename)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	if strings.HasSuffix(filename, "hcl") {
		err := hcl.Unmarshal(b, &t.config)
		if err != nil {
			return err
		}
	} else if strings.HasSuffix(filename, "json") {
		err := json.Unmarshal(b, &t.config)
		if err != nil {
			return err
		}
	} else {
		err := yaml.Unmarshal(b, &t.config)
		if err != nil {
			return err
		}
	}

	if err := t.TemplateRulesWithMatrix(); err != nil {
		return err
	}
	for _, rule := range t.config.Rules {
		if rule.TagSuffixFileRef == nil {
			continue
		}
		re, err := regexp.Compile(rule.TagSuffixFileRef.RegExpRaw)
		if err != nil {
			return fmt.Errorf("failed to parse '%s': %v", rule.TagSuffixFileRef.RegExpRaw, err)
		}
		rule.TagSuffixFileRef.RegExp = re
	}
	return nil
}

func (t *Tracker) gitCommand(arg ...string) *exec.Cmd {
	cmd := exec.Command(t.git, arg...)
	cmd.Dir = t.dir
	return cmd
}

func (t *Tracker) Diff(head, sha string) (changes []string, err error) {
	logrus.Debugf("Diff head with %s.", sha)
	output, err := t.gitCommand("diff", head, sha, "--name-only").CombinedOutput()
	if err != nil {
		return nil, err
	}
	scan := bufio.NewScanner(bytes.NewReader(output))
	scan.Split(bufio.ScanLines)
	for scan.Scan() {
		changes = append(changes, scan.Text())
	}
	return
}

func (t *Tracker) DiffStat(head, sha string, files []string) (string, error) {
	logrus.Debugf("Diff stat head with %s for %s.", sha, strings.Join(files, ", "))
	// fatal: ambiguous argument 'README.md2': unknown revision or path not in the working tree.
	// Use '--' to separate paths from revisions, like this:
	// 'git <command> [<revision>...] -- [<file>...]'
	args := append([]string{"diff", "--stat", head, sha, "--"}, files...)
	output, err := t.gitCommand(args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, string(output))
	}
	return string(output), nil
}
