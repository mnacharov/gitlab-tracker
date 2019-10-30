package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/cloudfoundry/cli/util/glob"
	"github.com/hashicorp/hcl"
	"github.com/sirupsen/logrus"
	gitlab "github.com/xanzy/go-gitlab"
	"gopkg.in/yaml.v2"
)

const (
	tagMessage         = "Auto-generated. Do not Remove."
	errTagNotFound     = "Tag Not Found"
	configFilenameBase = ".gitlab-tracker"

	defaultTagSuffixSeparator = "@"
)

var (
	httpCli = &http.Client{
		Timeout:   time.Second * 10,
		Transport: RetryTransport(),
	}
	forceFlag         = flag.Bool("force", true, "Force recreate tags.")
	logLevelFlag      = flag.String("log-level", "INFO", "Level of logging.")
	validateFlag      = flag.Bool("validate", false, "Validate config and exit")
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
	gitLab      *gitlab.Client
	config      Config
}

type Config struct {
	Checks        ChecksConfig     `yaml:"checks" hcl:"checks"`
	Hooks         HooksConfig      `yaml:"hooks" hcl:"hooks"`
	Rules         map[string]*Rule `yaml:"rules" hcl:"rules"`
	Matrix        []string         `yaml:"matrix" hcl:"matrix"`
	MatrixFromDir string           `yaml:"matrixFromDir" hcl:"matrix_from_dir"`
}

type ChecksConfig struct {
	PreFlight map[string]*Command `yaml:"preFlight" hcl:"pre_flight"`
}

type HooksConfig struct {
	PostCreateTag map[string]*Command `yaml:"postCreateTag" hcl:"post_create_tag"`
	PostUpdateTag map[string]*Command `yaml:"postUpdateTag" hcl:"post_update_tag"`
}

type Command struct {
	RetryConfig *RetryConfig `yaml:"retry" hcl:"retry"`
	Command     []string     `yaml:"command" hcl:"command"`
}

type Rule struct {
	Path               string            `yaml:"path" hcl:"path"`
	Tag                string            `yaml:"tag" hcl:"tag"`
	TagWithSuffix      string            `yaml:"-" hcl:"-"`
	TagSuffix          string            `yaml:"tagSuffix" hcl:"tag_suffix"`
	TagSuffixSeparator string            `yaml:"tagSuffixSeparator" hcl:"tag_suffix_separator"`
	TagSuffixFileRef   *TagSuffixFileRef `yaml:"tagSuffixFileRef" hcl:"tag_suffix_file_ref"`
}

type TagSuffixFileRef struct {
	File      string         `yaml:"file" hcl:"file"`
	RegExpRaw string         `yaml:"regexp" hcl:"regexp"`
	Group     int            `yaml:"regexpGroup" hcl:"regexp_group"`
	RegExp    *regexp.Regexp `yaml:"-" hcl:"-"`
}

func main() {
	flag.Parse()

	if err := ConfigureLogging(*logLevelFlag); err != nil {
		logrus.Fatal(err)
	}

	tracker, err := NewTracker()
	if err != nil {
		logrus.Fatal(err)
	}

	if *validateFlag {
		out, err := yaml.Marshal(tracker.config)
		if err != nil {
			logrus.Fatal(err)
		}
		fmt.Println(string(out))
		return
	}

	err = tracker.Run(*forceFlag)
	if err != nil {
		logrus.Fatal(err)
	}
}

func (r *Rule) ParseAsTemplate(data map[string]string) error {
	if err := r.parseTmpl(data); err != nil {
		return err
	}
	if r.TagSuffixFileRef == nil {
		return nil
	}
	if err := r.TagSuffixFileRef.parseTmpl(data); err != nil {
		return err
	}
	return nil
}

func (r *Rule) Clone() *Rule {
	dest := &Rule{
		Path:               r.Path,
		Tag:                r.Tag,
		TagSuffix:          r.TagSuffix,
		TagSuffixSeparator: r.TagSuffixSeparator,
	}
	if r.TagSuffixFileRef != nil {
		dest.TagSuffixFileRef = r.TagSuffixFileRef.Clone()
	}
	return dest
}

func (r *Rule) parseTmpl(data map[string]string) error {
	var err error
	r.Path, err = gotmpl(r.Path, data)
	if err != nil {
		return err
	}
	r.Tag, err = gotmpl(r.Tag, data)
	if err != nil {
		return err
	}
	r.TagSuffix, err = gotmpl(r.TagSuffix, data)
	if err != nil {
		return err
	}
	r.TagSuffixSeparator, err = gotmpl(r.TagSuffixSeparator, data)
	if err != nil {
		return err
	}
	return nil
}

func (t *TagSuffixFileRef) Clone() *TagSuffixFileRef {
	return &TagSuffixFileRef{
		File:      t.File,
		RegExpRaw: t.RegExpRaw,
		Group:     t.Group,
	}
}

func (t *TagSuffixFileRef) parseTmpl(data map[string]string) error {
	var err error
	t.File, err = gotmpl(t.File, data)
	if err != nil {
		return err
	}
	t.RegExpRaw, err = gotmpl(t.RegExpRaw, data)
	if err != nil {
		return err
	}
	return nil
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
		return "", nil
	}
	return "", nil
}

func (t *TagSuffixFileRef) GetSuffix(dir string) (string, error) {
	filename := path.Join(dir, t.File)
	output, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	scan := bufio.NewScanner(bytes.NewReader(output))
	scan.Split(bufio.ScanLines)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if !t.RegExp.MatchString(line) {
			continue
		}
		results := t.RegExp.FindStringSubmatch(line)
		groupID := t.Group
		if groupID == 0 {
			groupID = 1
		}
		if len(results) > groupID {
			return results[groupID], nil
		}
	}
	return "", err
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
	exists, tag, err := t.CreateTagIfNotExists(rule.TagWithSuffix)
	if err != nil {
		return err
	}
	if !exists {
		err = t.ExecCommandMap("PostCreateTag", t.config.Hooks.PostCreateTag, rule)
		if err != nil {
			return err
		}
		return nil
	}
	destRef := rule.TagWithSuffix
	if len(t.beforeRef) > 0 {
		// Try to use CI_COMMIT_BEFORE_SHA
		destRef = t.beforeRef
	}
	changes, err := t.Diff(t.ref, destRef)
	if err != nil {
		return err
	}
	matches, match := rule.IsChangesMatch(changes)
	if !match {
		logrus.Debug("Nothing changed.")
		return nil
	}
	err = t.UpdateTag(tag, force, matches)
	if err != nil {
		return err
	}
	err = t.ExecCommandMap("PostUpdateTag", t.config.Hooks.PostUpdateTag, rule)
	if err != nil {
		return err
	}
	return nil
}

func (t *Tracker) Run(force bool) error {
	err := t.ExecCommandMap("PreFlight", t.config.Checks.PreFlight, nil)
	if err != nil {
		return fmt.Errorf("Pre flight check: failed. %v", err)
	}
	logrus.Info("Pre flight check: passed")

	err = t.UpdateTags(force)
	if err != nil {
		return err
	}
	return nil
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
		return errors.New("Failed")
	}
	return nil
}

func ProcessCommand(rule *Rule, args []string) (*exec.Cmd, error) {
	argsExec := []string{}
	for _, templ := range args {
		arg, err := gotmpl(templ, rule)
		if err != nil {
			return nil, err
		}
		argsExec = append(argsExec, os.ExpandEnv(arg))
	}
	if len(argsExec) > 1 {
		c := exec.Command(argsExec[0], argsExec[1:]...)
		c.Env = os.Environ()
		return c, nil
	}
	c := exec.Command(argsExec[0])
	c.Env = os.Environ()
	return c, nil
}

func (t *Tracker) ExecCommandMap(commandType string, commands map[string]*Command, rule *Rule) error {
	for name, command := range commands {
		if command == nil || len(command.Command) == 0 {
			continue
		}
		err := Retry(func(s *Stats) error {
			logrus.Debugf("Exec %v as %s command (%s).", command.Command, commandType, s)
			cmd, err := ProcessCommand(rule, command.Command)
			if err != nil {
				return err
			}
			bytes, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("%v: %s", err, string(bytes))
			}
			return nil
		}, command.RetryConfig)
		if err != nil {
			return fmt.Errorf("%s %s: %v", commandType, name, err)
		}
	}
	return nil
}

func gotmpl(templ string, data interface{}) (string, error) {
	var templateEng *template.Template
	buf := bytes.NewBufferString("")
	templateEng = template.New("hook")
	if messageTempl, err := templateEng.Parse(templ); err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	} else if err := messageTempl.Execute(buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	return buf.String(), nil
}

func (t *Tracker) CreateTagIfNotExists(tagName string) (bool, *gitlab.Tag, error) {
	tag, _, err := t.gitLab.Tags.GetTag(t.proj, tagName, nil)
	if err != nil && !strings.Contains(err.Error(), errTagNotFound) {
		return false, nil, err
	}
	if err == nil {
		return true, tag, nil
	}
	logrus.Infof("Create '%s' tag.", tagName)
	tag, err = t.CreateTagForRef(tagName, t.ref)
	if err != nil {
		return false, nil, err
	}
	return false, tag, nil
}

func (t Tracker) CreateTagForRef(tagName, ref string) (*gitlab.Tag, error) {
	logrus.Infof("Create '%s' tag with %s ref.", tagName, ref)
	opts := &gitlab.CreateTagOptions{
		TagName: gitlab.String(tagName),
		Ref:     gitlab.String(ref),
		Message: gitlab.String(tagMessage),
	}
	tag, _, err := t.gitLab.Tags.CreateTag(t.proj, opts, nil)
	return tag, err
}

func (t *Tracker) UpdateTag(tag *gitlab.Tag, force bool, changes []string) error {
	if force {
		_, err := t.gitLab.Tags.DeleteTag(t.proj, tag.Name, nil)
		if err != nil && !strings.Contains(err.Error(), errTagNotFound) {
			return err
		}
	}
	_, err := t.CreateTagForRef(tag.Name, t.ref)
	if err != nil {
		return err
	}
	stat, err := t.DiffStat(tag.Commit.ID, t.ref, changes)
	if err != nil {
		return err
	}
	if len(stat) == 0 {
		return nil
	}
	message := fmt.Sprintf(
		"<details><summary>Details</summary><pre><code>%s</code></pre></details>",
		stat,
	)
	opts := &gitlab.CreateReleaseNoteOptions{
		Description: gitlab.String(message),
	}
	// It's okay if it fails
	t.gitLab.Tags.CreateReleaseNote(t.proj, tag.Name, opts, nil)
	return nil
}

func (r *Rule) IsChangesMatch(changes []string) ([]string, bool) {
	matches := []string{}
	gl := glob.MustCompileGlob(r.Path)
	for _, change := range changes {
		if gl.Match(change) {
			matches = append(matches, change)
		}
	}
	if len(matches) > 0 {
		return matches, true
	}
	return matches, false
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
	t.gitLab = cli
	return t, nil
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
		ref.ParseAsTemplate(map[string]string{
			"Item": path.Base(item.Name()),
		})
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
		ref.ParseAsTemplate(map[string]string{
			"Item": item,
		})
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
		return errors.New("Matrix can be used only with single rule")
	}
	matrixRule, ok := t.config.Rules["matrix"]
	if !ok {
		return errors.New("Matrix can be used only with rule that named as `matrix`")
	}
	if len(t.config.Matrix) > 0 {
		return t.templateRulesWithMatrixRaw(matrixRule)
	}
	return t.templateRulesWithMatrixFromDir(matrixRule)
}

func (t *Tracker) DiscoverConfigFile(dir string) (string, error) {
	filename := path.Join(dir, fmt.Sprintf("%s.yml", configFilenameBase))
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return filename, nil
	}
	filename = path.Join(dir, fmt.Sprintf("%s.hcl", configFilenameBase))
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return filename, nil
	}
	return "", errors.New("Configuration file not found")
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
			return fmt.Errorf("Failed to parse '%s': %v", rule.TagSuffixFileRef.RegExpRaw, err)
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
	args := []string{"diff", "--stat", head, sha}
	args = append(args, files...)
	output, err := t.gitCommand(args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, string(output))
	}
	return string(output), nil
}
