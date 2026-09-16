.PHONY: build proto tidy test

build:
	go build -mod=mod -o bin/taiji ./cmd/taiji

proto:
	buf generate

tidy:
	go mod tidy

test:
	go test ./...
