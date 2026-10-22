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
INSTALL_CLI := $(INSTALL_DIR)/taiji
INSTALL_TUI := $(INSTALL_DIR)/taiji-tui
INSTALL_CLI_BIN := $(INSTALL_BIN_DIR)/taiji
INSTALL_TUI_BIN := $(INSTALL_BIN_DIR)/taiji-tui

.PHONY: all build rebuild build-taiji build-tui ensure-wuji-driver clean run tui install uninstall help

all: build

help:
	@echo "Taiji"
	@echo ""
	@echo "  make run      Build and start taiji-tui"
	@echo "  make install    Install taiji + taiji-tui to ~/.local/opt/taiji and ~/.local/bin"
	@echo "  make uninstall  Remove install from ~/.local/opt/taiji"
	@echo "  make build    Build Taiji CLI + TUI"
	@echo "  make rebuild  clean + build"
	@echo "  make clean    Remove build artifacts"
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

clean:
	$(MAKE) -C $(TUI_DIR) clean
	rm -rf $(TAIJI_DIR)/bin

run tui: build-tui ensure-wuji-driver
	cd $(TAIJI_DIR) && PATH="$(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)" exec $(TUI_BIN)

install: build-tui
	mkdir -p "$(INSTALL_DIR)" "$(INSTALL_BIN_DIR)"
	install -m 755 "$(TAIJI_CLI_BIN)" "$(INSTALL_CLI)"
	install -m 755 "$(TUI_BIN)" "$(INSTALL_TUI)"
	install -m 755 "$(TAIJI_CLI_BIN)" "$(INSTALL_CLI_BIN)"
	install -m 755 "$(TUI_BIN)" "$(INSTALL_TUI_BIN)"
	@echo "Installed $(INSTALL_CLI) and $(INSTALL_TUI)"
	@echo "Installed $(INSTALL_CLI_BIN) and $(INSTALL_TUI_BIN)"

uninstall:
	rm -f "$(INSTALL_CLI)" "$(INSTALL_TUI)" "$(INSTALL_CLI_BIN)" "$(INSTALL_TUI_BIN)"
	rm -f "$(HOME)/.local/opt/beta-bin/taiji" "$(HOME)/.local/opt/beta-bin/taiji-tui"
	@rmdir "$(INSTALL_DIR)" 2>/dev/null || true
	@rmdir "$(HOME)/.local/opt/beta-bin" 2>/dev/null || true
	@echo "Removed install from $(INSTALL_DIR) and $(INSTALL_BIN_DIR)"
