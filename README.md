# gitlab-tracker

Separate your releases and specification changes or something else.

## Configuration

```yaml
---
hooks:
  postCreateTagCommand:
    - argocd
    - app
    - set
    - "{{.Tag}}"
    - "--revision={{.TagWithSuffix}}"
    - "--server=$ARGOCD_SERVER"
    - "--auth-token=$ARGOCD_TOKEN"
  postUpdateTagCommand:
    - argocd
    - app
    - sync
    - "{{.Tag}}"
    - "--server=$ARGOCD_SERVER"
    - "--auth-token=$ARGOCD_TOKEN"
rules:
  - path: application/production/**
    tag: application
    tagSuffixSeparator: "@"
    tagSuffuxFileRef:
      file: application/production/application.Deployment.yaml
      regexp: eu.gcr.io/org/proj/application:(.*)$
```
