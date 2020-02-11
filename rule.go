package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"path"
	"regexp"
	"strings"

	"github.com/cloudfoundry/cli/util/glob"
)

type Rule struct {
	Path               string            `yaml:"path" hcl:"path" json:"path"`
	Tag                string            `yaml:"tag" hcl:"tag" json:"tag"`
	TagWithSuffix      string            `yaml:"-" hcl:"-" json:"-"`
	TagSuffix          string            `yaml:"tagSuffix" hcl:"tag_suffix" json:"tagSuffix"`
	TagSuffixSeparator string            `yaml:"tagSuffixSeparator" hcl:"tag_suffix_separator" json:"tagSuffixSeparator"`
	TagSuffixFileRef   *TagSuffixFileRef `yaml:"tagSuffixFileRef" hcl:"tag_suffix_file_ref" json:"tagSuffixFileRef"`
}

type TagSuffixFileRef struct {
	File      string         `yaml:"file" hcl:"file" json:"file"`
	RegExpRaw string         `yaml:"regexp" hcl:"regexp" json:"regexp"`
	Group     int            `yaml:"regexpGroup" hcl:"regexp_group" json:"regexpGroup"`
	RegExp    *regexp.Regexp `yaml:"-" hcl:"-" json:"-"`
}

func (r *Rule) ParseAsTemplate(data map[string]string) error {
	if err := r.parseTmpl(data); err != nil {
		return err
	}
	if r.TagSuffixFileRef == nil {
		return nil
	}
	return r.TagSuffixFileRef.parseTmpl(data)
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
	return err
}

func (r *Rule) IsChangesMatch(changes []string) ([]string, bool) {
	var matches []string
	gl := glob.MustCompileGlob(r.Path)
	for _, change := range changes {
		if gl.Match(change) {
			matches = append(matches, change)
		}
	}
	return matches, len(matches) > 0
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
	return err
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
