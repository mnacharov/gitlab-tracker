.PHONY: test
test:
	@go test -cover

.PHONY: test-report
test-report:
	@go test -coverprofile=coverage.txt && go tool cover -html=coverage.txt

.PHONY: build
build:
	@goreleaser release --skip-publish --skip-validate --rm-dist --snapshot
	@docker build -t leominov/gitlab-tracker:latest ./
