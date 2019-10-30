# gitlab-tracker

[![Build Status](https://travis-ci.com/leominov/gitlab-tracker.svg?branch=master)](https://travis-ci.com/leominov/gitlab-tracker)
[![codecov](https://codecov.io/gh/leominov/gitlab-tracker/branch/master/graph/badge.svg)](https://codecov.io/gh/leominov/gitlab-tracker)

Separate your releases and specification changes or something else.

## Configuration

```yaml
---
checks:
  preFlight:
    command:
      - argocd
      - cluster
      - list
hooks:
  postCreateTag:
    command:
      - argocd
      - app
      - set
      - "{{.Tag}}"
      - "--revision={{.TagWithSuffix}}"
  postUpdateTag:
    command:
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
        # regexpGroup: 1
```
