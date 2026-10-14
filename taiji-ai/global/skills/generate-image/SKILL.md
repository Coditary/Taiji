---
name: generate-image
description: Generate images from text prompts via Wuji and display them in the chat.
---

# Generate Image

Use this when the user asks to create, draw, generate, or visualize an image (German: "Bild", "zeichne", "generiere ein Bild").

## Workflow

1. Call `generate_image` with a clear English or German prompt describing the scene, style, and subject.
2. Wait for the tool result. It includes `path` and a ready-made `markdown` line.
3. Reply to the user with a short caption and the markdown image from the tool output, e.g. `![sunset over mountains](/absolute/path.png)`.
4. Use the **absolute** path from the tool output so the TUI can render the image inline.

## Rules

- Do not use `shell` or `wuji image` for generation — use `generate_image` only.
- Do not invent paths; only use paths returned by the tool.
- Image generation can take several minutes; do not retry unless the tool failed with an error.
- Optional tool args: `negative_prompt`, `width`, `height`, `steps`, `model`.
