FROM alpine:3.11
ENV USER=argocd
COPY /_output/gitlab-tracker_linux_amd64/gitlab-tracker /usr/local/bin/gitlab-tracker
RUN apk --no-cache add \
        curl \
        git \
        jq \
    && curl -sfL \
        https://github.com/argoproj/argo-cd/releases/download/v1.6.2/argocd-linux-amd64 \
        -o /usr/local/bin/argocd \
    && chmod +x /usr/local/bin/argocd \
    && apk del curl
