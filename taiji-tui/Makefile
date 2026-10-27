BUILD_DIR := build
TEST_BUILD_DIR := build-test
UI_TESTS_BIN := $(TEST_BUILD_DIR)/taiji-tui-ui-tests

.PHONY: all configure build run start clean help
.PHONY: configure-tests build-ui-tests test-ui test-ui-system

all: build

help:
	@echo "taiji-tui"
	@echo ""
	@echo "  make run              Bauen und TUI starten"
	@echo "  make build            Nur bauen (Release, $(BUILD_DIR))"
	@echo "  make clean            Build-Artefakte löschen"
	@echo ""
	@echo "  make build-ui-tests   UI-Tests bauen ($(TEST_BUILD_DIR), Tuinator AppDriver)"
	@echo "  make test-ui          Alle UI-Tests ausführen"
	@echo "  make test-ui-system   Wie test-ui (inkl. tests/integration/* Systemtests)"

configure:
	cmake -B $(BUILD_DIR) -DCMAKE_BUILD_TYPE=Release

configure-tests:
	cmake -S . -B $(TEST_BUILD_DIR) -DTAIJI_TUI_BUILD_TESTS=ON -DCMAKE_BUILD_TYPE=Debug

build: configure
	cmake --build $(BUILD_DIR) -j

build-ui-tests: configure-tests
	cmake --build $(TEST_BUILD_DIR) --target taiji-tui-ui-tests -j

test-ui: build-ui-tests
	$(UI_TESTS_BIN)

test-ui-system: build-ui-tests
	@echo "System-/Integrationstests: tests/integration/test_theme_picker_*.cpp (+ gesamte UI-Suite)"
	$(UI_TESTS_BIN)

run start: build
	cd ../taiji-ai && exec ../taiji-tui/$(BUILD_DIR)/taiji-tui

clean:
	rm -rf $(BUILD_DIR) $(TEST_BUILD_DIR) highlight/target fuzzy/target
