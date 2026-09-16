# Taiji

Monorepo for the Taiji agent stack: Go backend and C++ terminal UI, kept as separate modules inside one project.

## Layout

```
taiji/
├── Makefile          # build orchestration (root)
├── taiji-ai/         # backend — Go CLI, agent runtime, embed layer
└── taiji-tui/        # frontend — C++ TUI (Tuinator), links taiji-ai/embed
```

Wuji lives in a sibling tree (`../wuji-ai/`) and is not part of this repo.

## Requirements

- Go 1.23+ with CGO enabled
- C++17 compiler, CMake 3.16+
- ncursesw dev headers (Fedora: `ncurses-devel`)

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

## Architecture

```
taiji-tui (C++/Tuinator)
    └── libtaiji.a (Go c-archive, taiji-ai/embed)
            └── Taiji agent runtime, providers, tools, RAG
```
