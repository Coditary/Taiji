#include "composer_workspace.hpp"

#include "ui_theme.hpp"

#include <cstdio>
#include <filesystem>
#include <fstream>
#include <tuinator/platform/clipboard.hpp>

namespace fs = std::filesystem;

namespace {

bool is_ignored_path(const fs::path& path) {
    static const char* kIgnored[] = {".git", "node_modules", "build", "dist", ".cache", "target", "__pycache__"};
    for (const auto& part : path) {
        const std::string name = part.string();
        for (const char* ign : kIgnored) {
            if (name == ign) {
                return true;
            }
        }
    }
    return false;
}

} // namespace

std::vector<WorkspaceFileEntry> list_workspace_files(const std::string& root, int max_entries) {
    std::vector<WorkspaceFileEntry> out;
    if (root.empty()) {
        return out;
    }
    std::error_code ec;
    const fs::path base = fs::path(root);
    if (!fs::exists(base, ec) || !fs::is_directory(base, ec)) {
        return out;
    }

    for (auto it = fs::recursive_directory_iterator(base, fs::directory_options::skip_permission_denied, ec);
         it != fs::recursive_directory_iterator(); it.increment(ec)) {
        if (ec) {
            ec.clear();
            continue;
        }
        if (is_ignored_path(it->path())) {
            it.disable_recursion_pending();
            continue;
        }
        if (!it->is_regular_file(ec)) {
            continue;
        }
        const std::string rel = fs::relative(it->path(), base, ec).string();
        if (rel.empty() || rel == ".") {
            continue;
        }
        out.push_back({rel, rel});
        if (static_cast<int>(out.size()) >= max_entries) {
            break;
        }
    }
    return out;
}

std::string capture_git_diff(const std::string& repo_path, int max_bytes) {
    if (repo_path.empty()) {
        return {};
    }
    const std::string quoted = shell_quote_single(repo_path);
    std::string diff = run_command_capture("git -C " + quoted + " diff --no-color 2>/dev/null");
    if (diff.empty()) {
        diff = run_command_capture("git -C " + quoted + " diff --cached --no-color 2>/dev/null");
    }
    if (static_cast<int>(diff.size()) > max_bytes) {
        diff.resize(static_cast<std::size_t>(max_bytes));
        diff += "\n… (diff truncated)";
    }
    return diff;
}

std::string read_clipboard_text() {
    return tuinator::clipboard::get();
}

std::string read_last_terminal_capture() {
    const char* path = std::getenv("TAIJI_LAST_BUILD_LOG");
    if (path == nullptr || path[0] == '\0') {
        return {};
    }
    std::ifstream in(path);
    if (!in) {
        return {};
    }
    std::string content((std::istreambuf_iterator<char>(in)), std::istreambuf_iterator<char>());
    if (content.size() > 120000) {
        content.resize(120000);
        content += "\n… (log truncated)";
    }
    return content;
}
