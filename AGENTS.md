# AGENTS.md — Taiji monorepo

Guidance for humans and coding agents working in this repository (`taiji-ai/`, `taiji-tui/`).

## TUI testing (Tuinator)

The frontend (`taiji-tui`) is built on **[Tuinator](https://github.com/Coditary/Tuinator)**. UI behavior and layout must be covered with Tuinator’s testing stack — not only manual runs or external PTY tools.

**Authoritative reference:** `docs/testing.md` in the Tuinator tree (local default: `../../../Coditary/core/Tuinator/docs/testing.md` relative to `taiji-tui/`).

### What to write when you change the TUI

1. **Prefer widget unit tests** for isolated controls and render lines (fast, no full app loop).  
   Pattern: `tests/widgets/*.cpp`, `tests/support/test_harness.hpp`, `tests/support/render_helper.hpp` in Tuinator; add Taiji-specific tests under `taiji-tui/tests/` when that target exists, or extend Tuinator only when testing generic widgets.

2. **Use `tuinator::testing::AppDriver`** when focus order, multiple widgets, hover, or **painted colors** matter (e.g. sidebar row background uniform across label + timestamp).  
   Methods: `focus`, `click`, `type_into`, `hover`, `snapshot_text`, `analyze_selector`, `ui_tree_json`, `write_snapshot_png`.  
   Examples: Tuinator `tests/core/test_testing_driver.cpp`.  
   **Note:** `find_by_selector` walks `for_each_descendant` (custom child trees). `ui_tree_json` only lists `Widget::children()` unless the widget is in that tree — prefer `find_by_selector` for Taiji’s manual layouts.

3. **Assign stable selectors** via `taiji-tui/src/ui_test_ids.hpp` (`taiji::ui::test_id::…`) and `set_test_id` on widgets.  
   Session sidebar rows/actions use dynamic IDs such as `session.sidebar.session.<id>` and `session.sidebar.project.<root>.new` (see `SessionSidebar::sync_hit_regions`).  
   `find_by_selector` prefers `test_id`, then `widget_id`. In Release ship builds, test IDs are stripped unless `TUINATOR_ENABLE_TEST_IDS=ON`; keep calls in source anyway.

4. **New UI surfaces** (sidebar rows, composer, modals, context panel sections) should get a `test_id` when agents or tests need to target them.

### Taiji TUI test layout (`taiji-tui/tests/`)

| Target | Role |
|--------|------|
| `taiji-tui-ui-tests` | Fast **component** + **integration** tests (`tests/components/*`, `tests/integration/*`). Row hover: `test_session_sidebar_hover_background.cpp` uses `analyze_selector` for uniform background (label + timestamp). |
| `taiji-tui-test-agent` | NDJSON **JSON agent** scenes (`tests/scenes/scenes.hpp`), not the full app |

```bash
cmake -S taiji-tui -B taiji-tui/build -DTAIJI_TUI_BUILD_TESTS=ON
cmake --build taiji-tui/build --target taiji-tui-ui-tests taiji-tui-test-agent
./taiji-tui/build/taiji-tui-ui-tests

# JSON agent (scene: composer | session-sidebar | sidebar-footer)
TAIJI_TEST_SCENE=session-sidebar ./taiji-tui/build/taiji-tui-test-agent
echo '{"cmd":"tree"}' | ./taiji-tui/build/taiji-tui-test-agent composer
```

`TAIJI_TUI_BUILD_TESTS=ON` forces `TUINATOR_ENABLE_TEST_IDS=ON` for the linked Tuinator library.

### UI internationalization (i18n)

- Catalog JSON under `taiji-tui/locales/`; runtime via `ui_tr` / `ui_shell` (`ui_locale.hpp`).
- **Build gate:** `scripts/check_locales.py` (complete keys) and `scripts/check_locale_usage.py` (every `ui_tr`/`ui_shell` in `src/` must exist in the English master).
- **C++ tests:** `tests/i18n/test_locale_catalog.cpp` (all referenced keys resolve for every shipped locale), `tests/i18n/test_locale_render.cpp` (German sidebar chrome in snapshots).
- Audit hardcoded English fallbacks: `python3 scripts/check_locale_usage.py locales/ --warn-fallbacks`.

### Build and run Tuinator tests

```bash
cmake -B build -DTUINATOR_BUILD_TESTS=ON -DTUINATOR_BUILD_EXAMPLES=ON
cmake --build build
./build/tuinator-tests
```

Taiji links a local Tuinator via `TAIJI_TUI_TUINATOR_DIR` (see `taiji-tui/CMakeLists.txt`). After Tuinator changes, rebuild `taiji-tui`.

---

## Interactive control for AI / agents (debugger-style)

For **exploratory testing** and **agent-driven verification**, use Tuinator’s **JSON line protocol** (NDJSON on stdin/stdout), not guesswork from screenshots alone.

- Entry: `tuinator::testing::run_json_protocol_loop(AppDriver&)` or `handle_json_command` per line.
- Demo binary: `tuinator-testing-agent` (Tuinator `examples/testing-agent.cpp`).
- Typical loop: `tree` (optionally `include_screen`, `include_render`) → `hover` / `focus` / `type` / `key` → `styles` or `tree` again to verify colors and state.

**Command cheat sheet** (one JSON object per line; responses are one JSON line each):

| `cmd` | Purpose |
|-------|---------|
| `tree` | Widget tree, bounds, `id`, focus, checked, optional screen + render summary |
| `snapshot` / `ansi` | Plain or ANSI terminal dump |
| `styles` | RGB analysis inside widget bounds (`id` required) |
| `hover`, `focus`, `click`, `activate` | Pointer / focus / activation |
| `type`, `key` | Text input and keys |
| `present`, `step` | Force render or one frame |
| `quit` | End session |

**Requirement:** Build with test IDs enabled (`Debug` or `-DTUINATOR_ENABLE_TEST_IDS=ON`) so `id` in JSON resolves.

**Taiji target:** `taiji-tui-test-agent` (`tests/agent/main.cpp`) runs isolated **scenes** from `tests/scenes/scenes.hpp`. Extend scenes before wiring the full `AppShell`. Do not ship that binary in production releases.

**Optional complement:** [agent-tui](https://www.npmjs.com/package/agent-tui) against `taiji-tui/build/taiji-tui` is useful for **full PTY smoke** (real terminal, global shortcuts). Prefer the **Tuinator JSON agent** for structure, selectors, and **per-widget colors**.

---

## Backend (`taiji-ai`)

- Go tests: `cd taiji-ai && go test ./...`
- Coverage gate (default `internal/...` ≥ 80%): `cd taiji-ai && make test-cover` (`COVER_MIN`, `COVER_PKGS` overridable).
- Mutation testing ([Gremlins](https://gremlins.dev)): `cd taiji-ai && make mutate` (default package `internal/provider`; `make mutate-module` for the full module; config in `taiji-ai/.gremlins.yaml`). From repo root: `make test-mutate-go MUTATE_PKG=./internal/agent`.
- E2E agent fixture docs: `taiji-ai/testdata/agents-e2e/AGENTS.md` (test harness only; not general project rules).

## General

- Match existing style; keep TUI test changes focused.
- For paint/snapshot stability, see Tuinator `docs/rendering-pitfalls.md`.

### Test-driven work (features and bugs)

**New features and behavior changes:** write automated tests that encode the expected outcome **before** production code; run them **red**, then implement until **green**. See `.cursor/rules/new-features-tdd.mdc`.

**User-reported bugs** (reproduce → red test → fix):

1. Reproduce it (scenes, `AppDriver`, scripts, or documented steps).
2. Add a test for the **expected** correct outcome.
3. Run the test **before** fixing product code — it must **fail** first; if it passes without a fix, fix the test.
4. Implement the fix until the test is green.

Project rules: `.cursor/rules/new-features-tdd.mdc` and `.cursor/rules/user-reported-bugs-tdd.mdc` (always applied in this repo).
