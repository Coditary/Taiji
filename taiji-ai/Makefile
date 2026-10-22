.PHONY: build proto tidy test

build:
	GOTOOLCHAIN=$${GOTOOLCHAIN:-auto} go build -o bin/taiji ./cmd/taiji

proto:
	buf generate

tidy:
	go mod tidy

test:
	go test ./...
