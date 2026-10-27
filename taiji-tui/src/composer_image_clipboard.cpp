#include "composer_image_clipboard.hpp"

#include <tuinator/render/graphics_encode.hpp>

#include <cctype>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <sstream>

namespace {

bool read_binary_from_command(const char* command, std::vector<std::uint8_t>& out) {
    FILE* pipe = popen(command, "r");
    if (pipe == nullptr) {
        return false;
    }

    out.clear();
    std::uint8_t chunk[8192];
    std::size_t n = 0;
    while ((n = std::fread(chunk, 1, sizeof(chunk), pipe)) > 0) {
        out.insert(out.end(), chunk, chunk + n);
    }

    const int status = pclose(pipe);
    return status == 0 && !out.empty();
}

bool read_text_from_command(const char* command, std::string& out) {
    FILE* pipe = popen(command, "r");
    if (pipe == nullptr) {
        return false;
    }

    out.clear();
    char chunk[4096];
    while (std::fgets(chunk, sizeof(chunk), pipe) != nullptr) {
        out.append(chunk);
    }

    const int status = pclose(pipe);
    if (status != 0) {
        out.clear();
        return false;
    }

    while (!out.empty() && (out.back() == '\n' || out.back() == '\r')) {
        out.pop_back();
    }
    return true;
}

std::string temp_image_path(const char* suffix) {
    const std::filesystem::path dir = std::filesystem::temp_directory_path() / "taiji-clipboard";
    std::error_code ec;
    std::filesystem::create_directories(dir, ec);
    static int counter = 0;
    return (dir / ("clip-" + std::to_string(++counter) + suffix)).string();
}

bool write_file(const std::string& path, const std::vector<std::uint8_t>& data) {
    std::ofstream out(path, std::ios::binary);
    if (!out) {
        return false;
    }
    out.write(reinterpret_cast<const char*>(data.data()), static_cast<std::streamsize>(data.size()));
    return out.good();
}

std::optional<tuinator::TerminalImage> load_png_bytes(const std::vector<std::uint8_t>& bytes) {
    const std::string path = temp_image_path(".png");
    if (!write_file(path, bytes)) {
        return std::nullopt;
    }
    auto image = tuinator::TerminalImage::load_png(path);
    std::error_code ec;
    std::filesystem::remove(path, ec);
    return image;
}

std::optional<tuinator::TerminalImage> convert_to_png_via_ffmpeg(const std::string& input_path, bool delete_input) {
    const std::string out_path = temp_image_path(".png");
    std::ostringstream cmd;
    cmd << "ffmpeg -loglevel error -y -i \"" << input_path << "\" \"" << out_path << "\" 2>/dev/null";
    if (std::system(cmd.str().c_str()) != 0) {
        std::error_code ec;
        std::filesystem::remove(out_path, ec);
        return std::nullopt;
    }
    auto image = tuinator::TerminalImage::load_png(out_path);
    std::error_code ec;
    std::filesystem::remove(out_path, ec);
    if (delete_input) {
        std::filesystem::remove(input_path, ec);
    }
    return image;
}

std::string extension_for_mime(const std::string& mime) {
    if (mime.find("png") != std::string::npos) {
        return ".png";
    }
    if (mime.find("jpeg") != std::string::npos || mime.find("jpg") != std::string::npos) {
        return ".jpg";
    }
    if (mime.find("webp") != std::string::npos) {
        return ".webp";
    }
    return ".img";
}

bool clipboard_lists_type(const std::string& types_text, const char* needle) {
    std::istringstream stream(types_text);
    std::string line;
    while (std::getline(stream, line)) {
        while (!line.empty() && (line.back() == '\r' || line.back() == ' ')) {
            line.pop_back();
        }
        if (line == needle) {
            return true;
        }
    }
    return false;
}

std::optional<ClipboardImagePayload> read_wayland_clipboard_image() {
    std::string types;
    if (!read_text_from_command("wl-paste --list-types 2>/dev/null", types)) {
        return std::nullopt;
    }

    const char* mime = nullptr;
    if (clipboard_lists_type(types, "image/png")) {
        mime = "image/png";
    } else if (clipboard_lists_type(types, "image/jpeg")) {
        mime = "image/jpeg";
    } else if (clipboard_lists_type(types, "image/webp")) {
        mime = "image/webp";
    } else {
        return std::nullopt;
    }

    std::ostringstream cmd;
    cmd << "wl-paste --type " << mime << " 2>/dev/null";
    ClipboardImagePayload payload;
    payload.mime = mime;
    if (!read_binary_from_command(cmd.str().c_str(), payload.bytes)) {
        return std::nullopt;
    }
    return payload;
}

std::optional<ClipboardImagePayload> read_x11_clipboard_image() {
    ClipboardImagePayload payload;
    payload.mime = "image/png";
    if (read_binary_from_command("xclip -selection clipboard -t image/png -o 2>/dev/null", payload.bytes)) {
        return payload;
    }
    payload.mime = "image/jpeg";
    payload.bytes.clear();
    if (read_binary_from_command("xclip -selection clipboard -t image/jpeg -o 2>/dev/null", payload.bytes)) {
        return payload;
    }
    return std::nullopt;
}

std::string trim_copy(std::string_view text) {
    const auto begin = text.find_first_not_of(" \t\n\r");
    if (begin == std::string_view::npos) {
        return {};
    }
    const auto end = text.find_last_not_of(" \t\n\r");
    return std::string(text.substr(begin, end - begin + 1));
}

std::string strip_wrapping_quotes(std::string text) {
    if (text.size() >= 2) {
        const char a = text.front();
        const char b = text.back();
        if ((a == '\'' && b == '\'') || (a == '"' && b == '"')) {
            return text.substr(1, text.size() - 2);
        }
    }
    return text;
}

bool path_looks_like_image(const std::filesystem::path& path) {
    const std::string ext = path.extension().string();
    if (ext.empty()) {
        return false;
    }
    std::string lower = ext;
    for (char& ch : lower) {
        ch = static_cast<char>(std::tolower(static_cast<unsigned char>(ch)));
    }
    return lower == ".png" || lower == ".jpg" || lower == ".jpeg" || lower == ".webp" || lower == ".ppm";
}

} // namespace

std::optional<ClipboardImagePayload> read_clipboard_image() {
    if (std::getenv("WAYLAND_DISPLAY") != nullptr) {
        if (auto payload = read_wayland_clipboard_image()) {
            return payload;
        }
    }
    if (std::getenv("DISPLAY") != nullptr) {
        return read_x11_clipboard_image();
    }
    return std::nullopt;
}

std::optional<tuinator::TerminalImage> decode_image_bytes(const ClipboardImagePayload& payload) {
    if (payload.bytes.empty()) {
        return std::nullopt;
    }

    if (payload.mime.find("png") != std::string::npos) {
        return load_png_bytes(payload.bytes);
    }

    const std::string path = temp_image_path(extension_for_mime(payload.mime).c_str());
    if (!write_file(path, payload.bytes)) {
        return std::nullopt;
    }

    if (auto converted = convert_to_png_via_ffmpeg(path, true)) {
        return converted;
    }

    std::error_code ec;
    std::filesystem::remove(path, ec);
    return std::nullopt;
}

std::optional<tuinator::TerminalImage> load_image_file(const std::string& path) {
    if (path.empty() || !std::filesystem::is_regular_file(path)) {
        return std::nullopt;
    }

    const std::filesystem::path p(path);
    const std::string ext = p.extension().string();
    std::string lower = ext;
    for (char& ch : lower) {
        ch = static_cast<char>(std::tolower(static_cast<unsigned char>(ch)));
    }

    if (lower == ".png") {
        return tuinator::TerminalImage::load_png(path);
    }
    if (lower == ".ppm") {
        return tuinator::TerminalImage::load_ppm(path);
    }
    if (lower == ".jpg" || lower == ".jpeg" || lower == ".webp") {
        return convert_to_png_via_ffmpeg(path, false);
    }
    return std::nullopt;
}

std::optional<std::string> image_path_from_pasted_text(std::string_view pasted) {
    std::string text = strip_wrapping_quotes(trim_copy(pasted));
    if (text.empty() || text.find('\n') != std::string::npos) {
        return std::nullopt;
    }

    std::filesystem::path path(text);
    if (!path.is_absolute()) {
        path = std::filesystem::absolute(path);
    }
    if (!std::filesystem::is_regular_file(path) || !path_looks_like_image(path)) {
        return std::nullopt;
    }
    return path.string();
}
