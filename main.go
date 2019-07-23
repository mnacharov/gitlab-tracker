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
	"strings"
	"time"

	"github.com/cloudfoundry/cli/util/glob"
	"github.com/sirupsen/logrus"
	gitlab "github.com/xanzy/go-gitlab"
	"gopkg.in/yaml.v2"
)

const (
	tagMessage = "Auto-generated. Do not Remove."
)

var (
	requiredVariables = []string{
		"CI_COMMIT_SHA",
		"CI_API_V4_URL",
		"GITLAB_TOKEN",
		"CI_COMMIT_SHORT_SHA",
		"CI_PROJECT_PATH",
	}
	httpCli = &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
	forceFlag = flag.Bool("force", true, "Force recreate tags.")
)

type Tracker struct {
	dir    string
	git    string
	gitLab *gitlab.Client
	logger *logrus.Entry
	rules  []*Rule
}

type Rule struct {
	Path string `yaml:"path"`
	Tag  string `yaml:"tag"`
}

func main() {
	flag.Parse()
	errs := ValidateEnvironment()
	if len(errs) > 0 {
		for _, err := range errs {
			logrus.Error(err)
		}
		logrus.Fatal("Validation failed")
	}
	tracker, err := NewTracker()
	if err != nil {
		logrus.Fatal(err)
	}
	err = tracker.UpdateTags(*forceFlag)
	if err != nil {
		logrus.Fatal(err)
	}
}

func ValidateEnvironment() []error {
	errs := []error{}
	for _, env := range requiredVariables {
		if len(os.Getenv(env)) == 0 {
			errs = append(errs, fmt.Errorf("Variable %s must be specified", env))
		}
	}
	return errs
}

func (t *Tracker) UpdateTags(force bool) error {
	var failed bool
	for _, rule := range t.rules {
		tag, err := t.CreateTagIfNotExists(rule.Tag)
		if err != nil {
			failed = true
			t.logger.Error(err)
			continue
		}
		changes, err := t.Diff(os.Getenv("CI_COMMIT_SHA"), rule.Tag)
		if err != nil {
			failed = true
			t.logger.Error(err)
			continue
		}
		matches, match := rule.IsChangesMatch(changes)
		if !match {
			continue
		}
		err = t.UpdateTag(tag, force, matches)
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

func (t *Tracker) CreateTagIfNotExists(tagName string) (*gitlab.Tag, error) {
	tag, _, err := t.gitLab.Tags.GetTag(os.Getenv("CI_PROJECT_PATH"), tagName, nil)
	if err != nil && !strings.Contains(err.Error(), "Tag Not Found") {
		return nil, err
	}
	if err == nil {
		return tag, nil
	}
	t.logger.Infof("Create '%s' tag.", tagName)
	tag, err = t.CreateTagForRef(tagName, os.Getenv("CI_COMMIT_SHA"))
	if err != nil {
		return nil, err
	}
	return tag, nil
}

func (t Tracker) CreateTagForRef(tagName, ref string) (*gitlab.Tag, error) {
	t.logger.Infof("Create '%s' tag with %s ref.", tagName, ref)
	opts := &gitlab.CreateTagOptions{
		TagName: gitlab.String(tagName),
		Ref:     gitlab.String(ref),
		Message: gitlab.String(tagMessage),
	}
	tag, _, err := t.gitLab.Tags.CreateTag(
		os.Getenv("CI_PROJECT_PATH"),
		opts,
		nil,
	)
	return tag, err
}

func (t *Tracker) UpdateTag(tag *gitlab.Tag, force bool, changes []string) error {
	if force {
		_, err := t.gitLab.Tags.DeleteTag(
			os.Getenv("CI_PROJECT_PATH"),
			tag.Name,
			nil,
		)
		if err != nil && !strings.Contains(err.Error(), "Tag Not Found") {
			return err
		}
	}
	stat, err := t.DiffStat(tag.Commit.ID, os.Getenv("CI_COMMIT_SHA"), changes)
	if err != nil {
		return err
	}
	_, err = t.CreateTagForRef(tag.Name, os.Getenv("CI_COMMIT_SHA"))
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
	t.gitLab.Tags.CreateReleaseNote(os.Getenv("CI_PROJECT_PATH"), tag.Name, opts, nil)
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
	cli := gitlab.NewClient(httpCli, os.Getenv("GITLAB_TOKEN"))
	err = cli.SetBaseURL(os.Getenv("CI_API_V4_URL"))
	if err != nil {
		return nil, err
	}
	t := &Tracker{
		git:    g,
		gitLab: cli,
		dir:    d,
		logger: logrus.WithField("client", "git"),
	}
	err = t.LoadRules(path.Join(d, ".argo-tracker.yml"))
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Tracker) LoadRules(filename string) error {
	t.logger.Infof("Load %s configuration file.", filename)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(b, &t.rules)
	if err != nil {
		return err
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
	args := []string{
		"diff",
		"--stat",
		head,
		sha,
	}
	args = append(args, files...)
	output, err := t.gitCommand(args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
