.PHONY: test vet fmt build check

test:
	go test ./...

vet:
	go vet ./...

fmt:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

build:
	mkdir -p bin
	go build -o bin/gcp-audit ./cmd/gcp-audit
	go build -o bin/gcp-audit-summarize ./cmd/gcp-audit-summarize

check: fmt vet test
