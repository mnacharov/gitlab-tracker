# argo-tracker

Separate your releases and specification changes.

## Configuration

```yaml
---
hooks:
  postTagCommand:
  - argocd
  - app
  - set
  - "{{.Tag}}"
  - "--revision={{.TagWithSuffix}}"
  - "--server=ARGOCD_API_SERVER"
  - "--auth-token=ARGOCD_TOKEN"
rules:
- path: application/production/**
  tag: application
  tagSuffuxFileRef:
    file: application/production/application.Deployment.yaml
    regexp: eu.gcr.io/project/group/application:(.*)$
```
