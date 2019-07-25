FROM golang:1.12.6-alpine as builder
WORKDIR /go/src/github.com/leominov/argo-tracker
COPY . .
RUN go build ./

FROM alpine:3.9
ENV USER=argocd
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/leominov/argo-tracker/argo-tracker /usr/local/bin/argo-tracker
RUN apk --no-cache add \
        curl \
        git \
    && curl -sfL \
        https://github.com/argoproj/argo-cd/releases/download/v1.1.1/argocd-linux-amd64 \
        -o /usr/local/bin/argocd \
    && chmod +x /usr/local/bin/argocd \
    && apk del curl
