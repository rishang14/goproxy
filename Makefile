.PHONY: run fmt vet test build check

run:
	go run ./cmd/server

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

build:
	go build -o bin/web-server .

check: fmt vet test