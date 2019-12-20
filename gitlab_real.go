package main

import "github.com/xanzy/go-gitlab"

type gitlabRealClient gitlab.Client

// GetTag alias for Tags.GetTag
func (g gitlabRealClient) GetTag(pid interface{}, tag string, options ...gitlab.OptionFunc) (*gitlab.Tag, *gitlab.Response, error) {
	return g.Tags.GetTag(pid, tag, options...)
}

// CreateTag alias for Tags.CreateTag
func (g gitlabRealClient) CreateTag(pid interface{}, opt *gitlab.CreateTagOptions, options ...gitlab.OptionFunc) (*gitlab.Tag, *gitlab.Response, error) {
	return g.Tags.CreateTag(pid, opt, options...)
}

// DeleteTag alias for Tags.DeleteTag
func (g gitlabRealClient) DeleteTag(pid interface{}, tag string, options ...gitlab.OptionFunc) (*gitlab.Response, error) {
	return g.Tags.DeleteTag(pid, tag, options...)
}

// CreateRelease alias for Releases.CreateRelease
func (g gitlabRealClient) CreateRelease(pid interface{}, opts *gitlab.CreateReleaseOptions, options ...gitlab.OptionFunc) (*gitlab.Release, *gitlab.Response, error) {
	return g.Releases.CreateRelease(pid, opts, options...)
}
