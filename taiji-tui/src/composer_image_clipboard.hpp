#pragma once

#include <tuinator/render/terminal_image.hpp>

#include <optional>
#include <string>
#include <vector>

struct ClipboardImagePayload {
    std::string mime;
    std::vector<std::uint8_t> bytes;
};

/// Reads image/png or image/jpeg from the OS clipboard (wl-paste on Wayland, xclip on X11).
std::optional<ClipboardImagePayload> read_clipboard_image();

std::optional<tuinator::TerminalImage> decode_image_bytes(const ClipboardImagePayload& payload);

std::optional<tuinator::TerminalImage> load_image_file(const std::string& path);

/// If pasted text is a single path to an existing image file, returns the normalized path.
std::optional<std::string> image_path_from_pasted_text(std::string_view pasted);
