# Taiji

Monorepo for the Taiji agent stack: Go backend and C++ terminal UI, kept as separate modules inside one project.

## Layout

```
taiji/
├── Makefile          # build orchestration (root)
├── taiji-ai/         # backend — Go CLI, agent runtime, embed layer
└── taiji-tui/        # frontend — C++ TUI (Tuinator), links taiji-ai/embed
```

Wuji is a separate install. Put `wuji` and `wuji-core` on your `PATH` (or set
`WUJI_ROOT`). Taiji auto-starts `wuji-core` when the socket is down. The Go
library dependency is `github.com/coditary/wuji-core` (fetched from GitHub via
`go mod`).

## Requirements

- Go 1.26+ with CGO enabled (`GOTOOLCHAIN=auto` is set by `make` and `scripts/install.sh` if your distro Go is older)
- Rust (`cargo`) for the TUI fuzzy matcher
- C++17 compiler, CMake 3.16+, Python 3
- ncursesw dev headers (Fedora: `ncurses-devel`)
- Git (CMake may fetch [Tuinator](https://github.com/Coditary/Tuinator) on first build)

## Install

From anywhere (clone, build, install to `~/.local/bin`):

```bash
curl -fsSL https://raw.githubusercontent.com/Coditary/Taiji/main/scripts/install.sh | bash
```

Install OS build dependencies first (Fedora or Debian/Ubuntu; uses `sudo`):

```bash
curl -fsSL https://raw.githubusercontent.com/Coditary/Taiji/main/scripts/install.sh | bash -s -- --deps
```

From a git checkout:

```bash
./scripts/install.sh
```

Options: `./scripts/install.sh --help` (custom `--prefix`, `--ref`, `--uninstall`).

Ensure `~/.local/bin` is on your `PATH`. Wuji remains a [separate install](#layout).

## Build & run

From the repo root:

```bash
make build    # backend CLI + TUI
make run      # build and start taiji-tui
make clean
```

Individual modules:

```bash
make -C taiji-ai build
make -C taiji-tui build
make -C taiji-tui run
```

Runtime config: `taiji-ai/.taiji/config.yaml` (created on first run or copied from `taiji-ai/global/`).

## Hooks

Taiji supports layered lifecycle hooks (OpenCode-style interception + Claude-compatible shell hooks).

- **YAML:** `.taiji/hooks.yaml` (project) and `~/.taiji/hooks.yaml` (global)
- **JSON:** `.taiji/hooks.json` — Claude Code `PreToolUse` / `PostToolUse` / `Stop` subset
- **Config:** `hooks.enabled: true` in `config.yaml` (default on)

Events: `prompt.before`, `context.before`, `tool.before.*`, `tool.after.*`, `file.changed`, `session.start`, `session.idle`, `session.error`.

Shell hooks receive JSON on stdin; exit code **2** blocks. JSON stdout can return `updated_prompt`, `updated_args`, `updated_output`, `additional_system`, or `block`.

See `taiji-ai/global/hooks.example.yaml` for samples (gofmt on write, plugin hooks).

## Saved decision profiles

Reusable decision templates live alongside skills under `.taiji/decisions/<name>/DECISION.md` (project), `~/.taiji/decisions/` (global), or bundled defaults in `taiji-ai/global/decisions/`.

Each profile defines YAML front matter (`name`, `description`, `when`, `questions`, optional `model`) plus optional markdown instructions. The agent calls:

```json
decide({"profile": "buy", "text": "MacBook 2800€, altes Laptop langsam, Budget 3000€"})
```

Ad-hoc decisions still work with inline `questions` and `text`. In the TUI, `/decisions` lists available profiles; the sidebar shows loaded profile names.

Example profile: `taiji-ai/global/decisions/buy/DECISION.md`.

## Architecture

```
taiji-tui (C++/Tuinator)
    └── libtaiji.a (Go c-archive, taiji-ai/embed)
            └── Taiji agent runtime, providers, tools, RAG
```
