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

3. **Assign stable selectors** on interactive or assertable widgets:
   ```cpp
   widget->set_test_id("session-sidebar-row-active");
   ```
   `find_by_selector` prefers `test_id`, then `widget_id`. In Release ship builds, test IDs are stripped unless `TUINATOR_ENABLE_TEST_IDS=ON`; keep calls in source anyway.

4. **New UI surfaces** (sidebar rows, composer, modals, context panel sections) should get a `test_id` when agents or tests need to target them.

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

**Taiji target (when added):** a dedicated binary such as `taiji-tui-test-agent` should call `run_json_protocol_loop` with the same UI root as the app (see “Wire your own app” in Tuinator `docs/testing.md`). Do not ship that binary in production releases.

**Optional complement:** [agent-tui](https://www.npmjs.com/package/agent-tui) against `taiji-tui/build/taiji-tui` is useful for **full PTY smoke** (real terminal, global shortcuts). Prefer the **Tuinator JSON agent** for structure, selectors, and **per-widget colors**.

---

## Backend (`taiji-ai`)

- Go tests: `cd taiji-ai && go test ./...`
- E2E agent fixture docs: `taiji-ai/testdata/agents-e2e/AGENTS.md` (test harness only; not general project rules).

## General

- Match existing style; keep TUI test changes focused.
- For paint/snapshot stability, see Tuinator `docs/rendering-pitfalls.md`.
