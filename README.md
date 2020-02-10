# gitlab-tracker

[![Build Status](https://travis-ci.com/leominov/gitlab-tracker.svg?branch=master)](https://travis-ci.com/leominov/gitlab-tracker)
[![codecov](https://codecov.io/gh/leominov/gitlab-tracker/branch/master/graph/badge.svg)](https://codecov.io/gh/leominov/gitlab-tracker)

Separate your releases and specification changes or something else.

## Configuration

```hcl
checks "pre_flight" "argocd_check" {
  command = [
    "argocd",
    "cluster",
    "list"
  ]
}

hooks "pre_process" "argocd_wait" {
  command = [
    "argocd",
    "app",
    "wait",
    "{{.Tag}}-production",
    "--timeout=300"
  ]
}

hooks "post_create_tag" "argocd_set_new_revision" {
  command = [
    "argocd",
    "app",
    "set",
    "--revision={{.TagWithSuffix}}",
    "{{.Tag}}-production"
  ]
}

hooks "post_update_tag" "argocd_sync_state" {
  command = [
    "argocd",
    "app",
    "sync",
    "--async",
    "{{.Tag}}-production"
  ]
}

matrix_from_dir = "services"

rules "matrix" {
  path = "services/{{.Item}}/resources/**"
  tag = "{{.Item}}"
  tag_suffix_file_ref {
    file = "services/{{.Item}}/resources/{{.Item}}.Deployment.yaml"
    regexp = "eu.gcr.io/utilities-212509/[\\-\\w]+/[\\-\\w]+[:@](.*)$"
    regexp_group = 1
  }
}
```
