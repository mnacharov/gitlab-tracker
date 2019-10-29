# gitlab-tracker

[![Build Status](https://travis-ci.com/leominov/gitlab-tracker.svg?branch=master)](https://travis-ci.com/leominov/gitlab-tracker)
[![codecov](https://codecov.io/gh/leominov/gitlab-tracker/branch/master/graph/badge.svg)](https://codecov.io/gh/leominov/gitlab-tracker)

Separate your releases and specification changes or something else.

## Configuration

```yaml
---
checks:
  preFlightCommand:
    - argocd
    - cluster
    - list
hooks:
  postCreateTagCommand:
    - argocd
    - app
    - set
    - "{{.Tag}}"
    - "--revision={{.TagWithSuffix}}"
  postUpdateTagCommand:
    - argocd
    - app
    - sync
    - "{{.Tag}}"
rules:
  foobar:
    path: application/production/**
      tag: application
      tagSuffixSeparator: "@"
      tagSuffixFileRef:
        file: application/production/application.Deployment.yaml
        regexp: eu.gcr.io/org/proj/application:(.*)$
        # regexp_group: 1
```
