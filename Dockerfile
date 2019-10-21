FROM golang:1.12.6-alpine as builder
WORKDIR /go/src/github.com/leominov/gitlab-tracker
COPY . .
RUN go build ./

FROM alpine:3.9
ENV USER=argocd
COPY --from=builder /go/src/github.com/leominov/gitlab-tracker/gitlab-tracker /usr/local/bin/gitlab-tracker
RUN apk --no-cache add \
        curl \
        git \
    && curl -sfL \
        https://github.com/argoproj/argo-cd/releases/download/v1.2.1/argocd-linux-amd64 \
        -o /usr/local/bin/argocd \
    && chmod +x /usr/local/bin/argocd \
    && apk del curl
