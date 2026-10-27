# Taiji monorepo — build orchestration
#
#   taiji-ai/   backend (Go)
#   taiji-tui/  frontend (C++)
#
# Usage:
#   make run      Build and start taiji-tui
#   make build    Build Taiji CLI + TUI
#   make clean    Remove build artifacts

ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
TAIJI_DIR := $(ROOT)/taiji-ai
TUI_DIR := $(ROOT)/taiji-tui
TUI_BIN := $(TUI_DIR)/build/taiji-tui
TAIJI_CLI_BIN := $(TAIJI_DIR)/bin/taiji
EMBED_ARCHIVE := $(TUI_DIR)/build/libtaiji.a

LLAMA_DRIVER_SRC := $(HOME)/Dev/Coditary/plugins/wuji/llama
LLAMA_DRIVER_BIN := $(LLAMA_DRIVER_SRC)/bin/wuji-driver-llama

INSTALL_PREFIX ?= $(HOME)/.local
INSTALL_DIR ?= $(INSTALL_PREFIX)/opt/taiji
ifeq ($(origin TAIJI_OPT_DIR), environment)
INSTALL_DIR := $(TAIJI_OPT_DIR)
endif
INSTALL_BIN_DIR := $(INSTALL_PREFIX)/bin
INSTALL_BETA_DIR := $(INSTALL_PREFIX)/opt/beta-bin
INSTALL_TEMPLATES_DIR := $(INSTALL_DIR)/share/taiji/templates
INSTALL_BETA_TEMPLATES_DIR := $(INSTALL_BETA_DIR)/share/taiji/templates
TAIJI_GLOBAL_TEMPLATES := $(TAIJI_DIR)/global/templates
TAIJI_GLOBAL_GRAPHS := $(TAIJI_DIR)/global/graphs
INSTALL_GRAPHS_DIR := $(INSTALL_DIR)/share/taiji/graphs
INSTALL_BETA_GRAPHS_DIR := $(INSTALL_BETA_DIR)/share/taiji/graphs
INSTALL_CLI := $(INSTALL_DIR)/taiji
INSTALL_TUI := $(INSTALL_DIR)/taiji-tui
INSTALL_CLI_BIN := $(INSTALL_BIN_DIR)/taiji
INSTALL_TUI_BIN := $(INSTALL_BIN_DIR)/taiji-tui
INSTALL_CLI_BETA := $(INSTALL_BETA_DIR)/taiji
INSTALL_TUI_BETA := $(INSTALL_BETA_DIR)/taiji-tui

.PHONY: all build rebuild build-taiji build-tui build-tui-tests ensure-wuji-driver clean run tui
.PHONY: test-tui test-tui-system test-mutate-go test-cover-go test-docker-build-no-wuji install uninstall help

all: build

help:
	@echo "Taiji"
	@echo ""
	@echo "  make run      Build and start taiji-tui"
	@echo "  make install    Install taiji + taiji-tui to ~/.local/opt/taiji, ~/.local/opt/beta-bin, ~/.local/bin"
	@echo "  make uninstall  Remove install from ~/.local/opt/taiji"
	@echo "  make build            Build Taiji CLI + TUI"
	@echo "  make rebuild          clean + build"
	@echo "  make clean            Remove build artifacts"
	@echo ""
	@echo "  make build-tui-tests  Build taiji-tui-ui-tests (taiji-tui/build-test)"
	@echo "  make test-tui         Run all TUI UI tests"
	@echo "  make test-tui-system  Run TUI tests (includes integration/system tests)"
	@echo "  make test-docker-build-no-wuji  Docker: build/install without Wuji CLI on PATH"
	@echo "  make test-mutate-go             Go mutation tests (gremlins, default: taiji-ai/internal/provider)"
	@echo "  make test-cover-go              Go statement coverage gate (default: internal/... >= 80%%)"
	@echo ""
	@echo "Wuji: install wuji/wuji-core to PATH. Taiji auto-starts wuji-core + llama driver."

ensure-wuji-driver:
	@if [ -f "$(LLAMA_DRIVER_SRC)/cmd/wuji-driver-llama/main.go" ]; then \
		mkdir -p "$(LLAMA_DRIVER_SRC)/bin"; \
		cd "$(LLAMA_DRIVER_SRC)" && go build -o bin/wuji-driver-llama ./cmd/wuji-driver-llama; \
	fi

build: build-taiji ensure-wuji-driver build-tui

rebuild: clean build

build-taiji:
	cd $(TAIJI_DIR) && GOTOOLCHAIN=$${GOTOOLCHAIN:-auto} $(MAKE) build

build-tui: build-taiji
	rm -f $(EMBED_ARCHIVE)
	$(MAKE) -C $(TUI_DIR) build

build-tui-tests:
	$(MAKE) -C $(TUI_DIR) build-ui-tests

test-tui:
	$(MAKE) -C $(TUI_DIR) test-ui

test-tui-system:
	$(MAKE) -C $(TUI_DIR) test-ui-system

test-docker-build-no-wuji:
	./scripts/test-docker-build-without-wuji.sh

test-mutate-go:
	$(MAKE) -C $(TAIJI_DIR) mutate MUTATE_PKG="$(MUTATE_PKG)"

test-cover-go:
	$(MAKE) -C $(TAIJI_DIR) test-cover COVER_MIN="$(COVER_MIN)"

clean:
	$(MAKE) -C $(TUI_DIR) clean
	rm -rf $(TAIJI_DIR)/bin

# test-tui* targets do not require ensure-wuji-driver or a Release TUI build.

run tui: build-tui ensure-wuji-driver
	cd $(TAIJI_DIR) && PATH="$(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)" exec $(TUI_BIN)

install: build-tui
	mkdir -p "$(INSTALL_DIR)" "$(INSTALL_BIN_DIR)" "$(INSTALL_BETA_DIR)"
	mkdir -p "$(INSTALL_TEMPLATES_DIR)" "$(INSTALL_BETA_TEMPLATES_DIR)" "$(INSTALL_GRAPHS_DIR)" "$(INSTALL_BETA_GRAPHS_DIR)"
	install -m 755 "$(TAIJI_CLI_BIN)" "$(INSTALL_CLI)"
	install -m 755 "$(TUI_BIN)" "$(INSTALL_TUI)"
	install -m 755 "$(TAIJI_CLI_BIN)" "$(INSTALL_CLI_BIN)"
	install -m 755 "$(TUI_BIN)" "$(INSTALL_TUI_BIN)"
	install -m 755 "$(TAIJI_CLI_BIN)" "$(INSTALL_CLI_BETA)"
	install -m 755 "$(TUI_BIN)" "$(INSTALL_TUI_BETA)"
	cp -a "$(TAIJI_GLOBAL_TEMPLATES)/." "$(INSTALL_TEMPLATES_DIR)/"
	cp -a "$(TAIJI_GLOBAL_TEMPLATES)/." "$(INSTALL_BETA_TEMPLATES_DIR)/"
	cp -a "$(TAIJI_GLOBAL_GRAPHS)/." "$(INSTALL_GRAPHS_DIR)/"
	cp -a "$(TAIJI_GLOBAL_GRAPHS)/." "$(INSTALL_BETA_GRAPHS_DIR)/"
	@echo "Installed $(INSTALL_CLI) and $(INSTALL_TUI)"
	@echo "Installed $(INSTALL_CLI_BIN) and $(INSTALL_TUI_BIN)"
	@echo "Installed $(INSTALL_CLI_BETA) and $(INSTALL_TUI_BETA)"
	@echo "Installed templates to $(INSTALL_TEMPLATES_DIR) and $(INSTALL_BETA_TEMPLATES_DIR)"

uninstall:
	rm -f "$(INSTALL_CLI)" "$(INSTALL_TUI)" "$(INSTALL_CLI_BIN)" "$(INSTALL_TUI_BIN)"
	rm -f "$(INSTALL_CLI_BETA)" "$(INSTALL_TUI_BETA)"
	rm -rf "$(INSTALL_TEMPLATES_DIR)" "$(INSTALL_BETA_TEMPLATES_DIR)"
	@rmdir "$(INSTALL_DIR)" 2>/dev/null || true
	@rmdir "$(INSTALL_BETA_DIR)" 2>/dev/null || true
	@echo "Removed install from $(INSTALL_DIR), $(INSTALL_BIN_DIR), and $(INSTALL_BETA_DIR)"
