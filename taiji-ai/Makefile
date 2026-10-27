.PHONY: build proto tidy test test-cover mutate mutate-dry mutate-module mutate-diff taiji-tools globwalk

TAIJI_TOOLS_BIN := taiji-tools/target/release/taiji-tools
GLOBWALK_BIN := globwalk/target/release/taiji-globwalk

taiji-tools:
	CARGO_TARGET_DIR=$(CURDIR)/taiji-tools/target cargo build --release --manifest-path taiji-tools/Cargo.toml

# Legacy alias (superseded by taiji-tools glob).
globwalk: taiji-tools

build: taiji-tools
	GOTOOLCHAIN=$${GOTOOLCHAIN:-auto} go build -o bin/taiji ./cmd/taiji
	cp -f $(TAIJI_TOOLS_BIN) bin/taiji-tools

proto:
	buf generate

tidy:
	go mod tidy

test:
	go test ./...

COVER_MIN ?= 80
COVER_PKGS ?= ./internal/...

test-cover:
	chmod +x ./scripts/check_go_coverage.sh
	COVER_MIN=$(COVER_MIN) COVER_PKGS='$(COVER_PKGS)' ./scripts/check_go_coverage.sh

# Mutation testing (gremlins). Default: ./internal/provider (fast). Override: make mutate MUTATE_PKG=./internal/agent
MUTATE_PKG ?= ./internal/provider

mutate:
	./scripts/mutate.sh $(MUTATE_PKG)

mutate-dry:
	./scripts/mutate.sh --dry-run $(MUTATE_PKG)

mutate-module:
	./scripts/mutate.sh --module

mutate-diff:
	./scripts/mutate.sh --diff
