# Taiji monorepo — build orchestration
#
#   taiji-ai/   backend (Go)
#   taiji-tui/  frontend (C++)
#
# Wuji lives in ../wuji-ai (separate repo tree).
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
WUJI_DIR := $(ROOT)/wuji-ai/wuji-ai

.PHONY: all build rebuild build-taiji build-tui clean run tui help

all: build

help:
	@echo "Taiji"
	@echo ""
	@echo "  make run      Build and start taiji-tui"
	@echo "  make build    Build Taiji CLI + TUI"
	@echo "  make rebuild  clean + build"
	@echo "  make clean    Remove build artifacts"
	@echo ""
	@echo "Wuji (separate): cd wuji-ai && make build"
	@echo "Start Wuji core:   cd $(WUJI_DIR) && ./bin/wuji serve"

build: build-taiji build-tui

rebuild: clean build

build-taiji:
	$(MAKE) -C $(TAIJI_DIR) build

build-tui: build-taiji
	rm -f $(EMBED_ARCHIVE)
	$(MAKE) -C $(TUI_DIR) build

clean:
	$(MAKE) -C $(TUI_DIR) clean
	rm -rf $(TAIJI_DIR)/bin

run tui: build-tui
	cd $(TAIJI_DIR) && exec $(TUI_BIN)
