#pragma once

#include <string>

// Horizontal inset from column dividers (cells).
constexpr int kUiPadX = 1;

std::string shorten_display_path(const std::string& path);
std::string format_token_compact(int tokens);
std::string format_token_count_short(int tokens);
std::string format_cost_compact(double usd);
