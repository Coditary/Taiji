# taiji-tui

C++ Terminal UI for [Taiji](https://github.com/Coditary/Taiji) using [Tuinator](https://github.com/Coditary/Tuinator).

Part of the Taiji monorepo (`../`); backend lives in `../taiji-ai/`. Taiji is linked statically via the Go `c-archive` embed layer in `../taiji-ai/embed`.

## Layout

- **Main area (left):** chat history — your message, then Taiji's reply below it
- **Input (bottom):** full-width text field (`Enter` to send)
- **Sidebar (right):** session, context, MCP/LSP status

## Requirements

- C++17 compiler
- CMake 3.16+
- Go 1.23+ with CGO enabled
- ncursesw development headers (Fedora: `ncurses-devel`)

## Build

```bash
make
```

Or manually:

```bash
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -j
```

Run from `taiji-ai` (needs `.taiji/config.yaml` there):

```bash
# from taiji-tui/
make run

# or manually from taiji-ai/
cd ../taiji-ai
../taiji-tui/build/taiji-tui
```

## Architecture

```
taiji-tui (C++/Tuinator)
    └── libtaiji.a (Go c-archive, taiji-ai/embed)
            └── Taiji agent runtime, providers, tools, RAG
```
