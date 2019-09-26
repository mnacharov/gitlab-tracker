# gitlab-tracker

Separate your releases and specification changes or something else.

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
    - "--server=$ARGOCD_SERVER"
    - "--auth-token=$ARGOCD_TOKEN"
rules:
  - path: application/production/**
    tag: application
    tagSuffuxFileRef:
      file: application/production/application.Deployment.yaml
      regexp: eu.gcr.io/org/proj/application:(.*)$
```
