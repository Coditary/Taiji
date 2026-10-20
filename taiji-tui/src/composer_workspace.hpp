#pragma once

#include <string>
#include <vector>

struct WorkspaceFileEntry {
    std::string path;
    std::string display;
};

std::vector<WorkspaceFileEntry> list_workspace_files(const std::string& root, int max_entries = 4000);
std::string capture_git_diff(const std::string& repo_path, int max_bytes = 120000);
std::string read_clipboard_text();
std::string read_last_terminal_capture();
