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
EMBED_ARCHIVE := $(TUI_DIR)/build/libtaiji.a

LLAMA_DRIVER_SRC := $(HOME)/Dev/Coditary/plugins/wuji/llama
LLAMA_DRIVER_BIN := $(LLAMA_DRIVER_SRC)/bin/wuji-driver-llama

INSTALL_BIN := $(HOME)/.local/bin/taiji-tui

.PHONY: all build rebuild build-taiji build-tui ensure-wuji-driver clean run tui install help

all: build

help:
	@echo "Taiji"
	@echo ""
	@echo "  make run      Build and start taiji-tui"
	@echo "  make install  Install taiji-tui to ~/.local/bin"
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
	$(MAKE) -C $(TAIJI_DIR) build

build-tui: build-taiji
	rm -f $(EMBED_ARCHIVE)
	$(MAKE) -C $(TUI_DIR) build

clean:
	$(MAKE) -C $(TUI_DIR) clean
	rm -rf $(TAIJI_DIR)/bin

run tui: build-tui ensure-wuji-driver
	cd $(TAIJI_DIR) && PATH="$(HOME)/.local/bin:$(HOME)/go/bin:$(PATH)" exec $(TUI_BIN)

install: build-tui
	mkdir -p "$(HOME)/.local/bin"
	install -m 755 "$(TUI_BIN)" "$(INSTALL_BIN)"
	@echo "Installed $(INSTALL_BIN)"
