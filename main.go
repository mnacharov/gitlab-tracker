package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/cloudfoundry/cli/util/glob"
	"github.com/sirupsen/logrus"
	gitlab "github.com/xanzy/go-gitlab"
	"gopkg.in/yaml.v2"
)

const (
	tagMessage         = "Auto-generated. Do not Remove."
	errTagNotFound     = "Tag Not Found"
	configFilename     = ".argo-tracker.yml"
	tagSuffixSeparator = "@"
)

var (
	httpCli = &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
	forceFlag         = flag.Bool("force", true, "Force recreate tags.")
	tagSuffixReplacer = strings.NewReplacer("/", "", ":", "")
)

type Tracker struct {
	dir         string
	git         string
	gitLabToken string
	gitLabURL   string
	ref         string
	proj        string
	gitLab      *gitlab.Client
	logger      *logrus.Entry
	config      Config
}

type Config struct {
	Hooks HooksConfig `yaml:"hooks"`
	Rules []*Rule     `yaml:"rules"`
}

type HooksConfig struct {
	PostTagCommand []string `yaml:"postTagCommand"`
}

type Rule struct {
	Path             string            `yaml:"path"`
	Tag              string            `yaml:"tag"`
	TagWithSuffix    string            `yaml:"-"`
	TagSuffux        string            `yaml:"tagSuffux"`
	TagSuffuxFileRef *TagSuffuxFileRef `yaml:"tagSuffuxFileRef"`
}

type TagSuffuxFileRef struct {
	File      string         `yaml:"file"`
	RegExpRaw string         `yaml:"regexp"`
	RegExp    *regexp.Regexp `yaml:"-"`
}

func main() {
	flag.Parse()
	tracker, err := NewTracker()
	if err != nil {
		logrus.Fatal(err)
	}
	err = tracker.UpdateTags(*forceFlag)
	if err != nil {
		logrus.Fatal(err)
	}
}

func (t *Tracker) GetTagSuffixForRule(r *Rule) (string, error) {
	if len(r.TagSuffux) > 0 {
		suffix := tagSuffixReplacer.Replace(r.TagSuffux)
		return tagSuffixSeparator + suffix, nil
	}
	if r.TagSuffuxFileRef != nil {
		suffix, err := r.TagSuffuxFileRef.GetSuffix(t.dir)
		if err != nil {
			return "", err
		}
		if len(suffix) > 0 {
			suffix = tagSuffixReplacer.Replace(suffix)
			return tagSuffixSeparator + suffix, nil
		}
		return "", nil
	}
	return "", nil
}

func (t *TagSuffuxFileRef) GetSuffix(dir string) (string, error) {
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
		if len(results) > 1 {
			return results[1], nil
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
		err = t.PostTagHooks(rule)
		if err != nil {
			return err
		}
		return nil
	}
	changes, err := t.Diff(t.ref, rule.TagWithSuffix)
	if err != nil {
		return err
	}
	matches, match := rule.IsChangesMatch(changes)
	if !match {
		t.logger.Info("Nothing changed.")
		return nil
	}
	err = t.UpdateTag(tag, force, matches)
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
			t.logger.Error(err)
		}
	}
	if failed {
		return errors.New("Failed")
	}
	return nil
}

func ProcessTagHookCommand(args []string, rule *Rule) (*exec.Cmd, error) {
	for i, templ := range args {
		arg, err := gotmpl(templ, rule)
		if err != nil {
			return nil, err
		}
		args[i] = arg
	}
	if len(args) > 1 {
		c := exec.Command(args[0], args[1:]...)
		c.Env = os.Environ()
		return c, nil
	}
	c := exec.Command(args[0])
	c.Env = os.Environ()
	return c, nil
}

func (t *Tracker) PostTagHooks(rule *Rule) error {
	if len(t.config.Hooks.PostTagCommand) == 0 {
		return nil
	}
	t.logger.Infof("Exec %v as PostTag command.", t.config.Hooks.PostTagCommand)
	cmd, err := ProcessTagHookCommand(t.config.Hooks.PostTagCommand, rule)
	if err != nil {
		return err
	}
	_, err = cmd.CombinedOutput()
	return err
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
	t.logger.Infof("Create '%s' tag.", tagName)
	tag, err = t.CreateTagForRef(tagName, t.ref)
	if err != nil {
		return false, nil, err
	}
	return false, tag, nil
}

func (t Tracker) CreateTagForRef(tagName, ref string) (*gitlab.Tag, error) {
	t.logger.Infof("Create '%s' tag with %s ref.", tagName, ref)
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
	message := fmt.Sprintf(
		"<details><summary>Details</summary><pre><code>%s</code></pre></details>",
		stat,
	)
	opts := &gitlab.CreateReleaseNoteOptions{
		Description: gitlab.String(message),
	}
	// It's okay if it fail
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
		git:    g,
		dir:    d,
		logger: logrus.WithField("client", "git"),
	}
	err = t.LoadRules(path.Join(d, configFilename))
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
		return errors.New("gitlab token must be specified")
	}
	t.gitLabToken = token
	baseURL := os.Getenv("CI_API_V4_URL")
	if len(baseURL) == 0 {
		return errors.New("gitlab api url bust be specified")
	}
	t.gitLabURL = baseURL
	ref := os.Getenv("CI_COMMIT_SHA")
	if len(ref) == 0 {
		return errors.New("commit sha must be specified")
	}
	t.ref = ref
	proj := os.Getenv("CI_PROJECT_PATH")
	if len(proj) == 0 {
		return errors.New("project must be specified")
	}
	t.proj = proj
	return nil
}

func (t *Tracker) LoadRules(filename string) error {
	t.logger.Infof("Load %s configuration file.", filename)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(b, &t.config)
	if err != nil {
		return err
	}
	for _, rule := range t.config.Rules {
		if rule.TagSuffuxFileRef == nil {
			continue
		}
		re, err := regexp.Compile(rule.TagSuffuxFileRef.RegExpRaw)
		if err != nil {
			return fmt.Errorf("Failed to parse '%s': %v", rule.TagSuffuxFileRef.RegExpRaw, err)
		}
		rule.TagSuffuxFileRef.RegExp = re
	}
	return nil
}

func (t *Tracker) gitCommand(arg ...string) *exec.Cmd {
	cmd := exec.Command(t.git, arg...)
	cmd.Dir = t.dir
	return cmd
}

func (t *Tracker) Diff(head, sha string) (changes []string, err error) {
	t.logger.Infof("Diff head with %s.", sha)
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
	t.logger.Infof("Diff stat head with %s for %s.", sha, strings.Join(files, ", "))
	args := []string{"diff", "--stat", head, sha}
	args = append(args, files...)
	output, err := t.gitCommand(args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
