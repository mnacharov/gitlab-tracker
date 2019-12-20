package main

import (
	"errors"
	"fmt"

	"github.com/xanzy/go-gitlab"
)

type gitlabFake struct {
	tags map[string]*gitlab.Tag
}

func (g gitlabFake) GetTag(_ interface{}, tag string, _ ...gitlab.OptionFunc) (*gitlab.Tag, *gitlab.Response, error) {
	t, ok := g.tags[tag]
	if !ok {
		return nil, nil, errors.New(errTagNotFound)
	}
	return t, nil, nil
}

func (g gitlabFake) CreateTag(_ interface{}, opts *gitlab.CreateTagOptions, _ ...gitlab.OptionFunc) (*gitlab.Tag, *gitlab.Response, error) {
	tagName := *opts.TagName
	_, ok := g.tags[tagName]
	if ok {
		return nil, nil, fmt.Errorf("tag %q already exists", tagName)
	}
	tag := &gitlab.Tag{
		Commit: &gitlab.Commit{
			ID: *opts.Ref,
		},
		Name:    *opts.TagName,
		Message: *opts.Message,
	}
	g.tags[tagName] = tag
	return tag, nil, nil
}

func (g gitlabFake) DeleteTag(_ interface{}, tag string, _ ...gitlab.OptionFunc) (*gitlab.Response, error) {
	_, ok := g.tags[tag]
	if !ok {
		return nil, errors.New(errTagNotFound)
	}
	delete(g.tags, tag)
	return nil, nil
}

func (g gitlabFake) CreateRelease(_ interface{}, _ *gitlab.CreateReleaseOptions, _ ...gitlab.OptionFunc) (*gitlab.Release, *gitlab.Response, error) {
	return nil, nil, nil
}

func NewFakeClient() gitlabClient {
	return &gitlabFake{
		tags: make(map[string]*gitlab.Tag),
	}
}
