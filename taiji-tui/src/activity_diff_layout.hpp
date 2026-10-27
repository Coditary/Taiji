#pragma once

#include <cstdint>
#include <string>

enum class FileDiffLayout : std::uint8_t { Stacked, SideBySide };

FileDiffLayout file_diff_layout_from_pref(const std::string& pref);
const char* file_diff_layout_to_pref(FileDiffLayout layout);

void ui_set_file_diff_layout(FileDiffLayout layout);
FileDiffLayout ui_file_diff_layout();
