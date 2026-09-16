BUILD_DIR := build

.PHONY: all configure build run start clean help

all: build

help:
	@echo "taiji-tui"
	@echo ""
	@echo "  make run      Bauen und TUI starten"
	@echo "  make build    Nur bauen"
	@echo "  make clean    Build-Artefakte löschen"

configure:
	cmake -B $(BUILD_DIR) -DCMAKE_BUILD_TYPE=Release

build: configure
	cmake --build $(BUILD_DIR) -j

run start: build
	cd ../taiji-ai && exec ../taiji-tui/$(BUILD_DIR)/taiji-tui

clean:
	rm -rf $(BUILD_DIR) highlight/target
