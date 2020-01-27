FROM golang:1.13-alpine as builder
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
        https://github.com/argoproj/argo-cd/releases/download/v1.4.2/argocd-linux-amd64 \
        -o /usr/local/bin/argocd \
    && chmod +x /usr/local/bin/argocd \
    && apk del curl
