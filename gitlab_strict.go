package main

import "github.com/xanzy/go-gitlab"

// Strict subset of gitlab.Client methods
type gitlabClient interface {
	GetTag(pid interface{}, tag string, options ...gitlab.OptionFunc) (*gitlab.Tag, *gitlab.Response, error)
	CreateTag(pid interface{}, opt *gitlab.CreateTagOptions, options ...gitlab.OptionFunc) (*gitlab.Tag, *gitlab.Response, error)
	DeleteTag(pid interface{}, tag string, options ...gitlab.OptionFunc) (*gitlab.Response, error)
	CreateRelease(pid interface{}, opts *gitlab.CreateReleaseOptions, options ...gitlab.OptionFunc) (*gitlab.Release, *gitlab.Response, error)
}
