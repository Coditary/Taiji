#include "ui_layout.hpp"

#include <cstdio>
#include <cstdlib>

namespace {

bool path_starts_with(const std::string& path, const std::string& prefix) {
    if (path.size() < prefix.size()) {
        return false;
    }
    if (path.compare(0, prefix.size(), prefix) != 0) {
        return false;
    }
    return path.size() == prefix.size() || path[prefix.size()] == '/';
}

} // namespace

std::string shorten_display_path(const std::string& path) {
    if (path.empty()) {
        return "";
    }
    const char* home = std::getenv("HOME");
    if (home != nullptr && home[0] != '\0') {
        const std::string home_str(home);
        if (path_starts_with(path, home_str)) {
            if (path.size() == home_str.size()) {
                return "~";
            }
            return "~" + path.substr(home_str.size());
        }
    }
    return path;
}

std::string format_token_compact(int tokens) {
    char buf[32];
    if (tokens >= 1'000'000) {
        std::snprintf(buf, sizeof(buf), "%.1fM Tok", static_cast<double>(tokens) / 1'000'000.0);
    } else if (tokens >= 10'000) {
        std::snprintf(buf, sizeof(buf), "%.1fk Tok", static_cast<double>(tokens) / 1'000.0);
    } else {
        std::snprintf(buf, sizeof(buf), "%d Tok", tokens);
    }
    return buf;
}

std::string format_token_count_short(int tokens) {
    char buf[24];
    if (tokens >= 1'000'000) {
        std::snprintf(buf, sizeof(buf), "%.1fM", static_cast<double>(tokens) / 1'000'000.0);
    } else if (tokens >= 10'000) {
        std::snprintf(buf, sizeof(buf), "%.1fk", static_cast<double>(tokens) / 1'000.0);
    } else {
        std::snprintf(buf, sizeof(buf), "%d", tokens);
    }
    return buf;
}

std::string format_cost_compact(double usd) {
    char buf[24];
    if (usd < 0.01) {
        std::snprintf(buf, sizeof(buf), "$%.4f", usd);
    } else {
        std::snprintf(buf, sizeof(buf), "$%.3f", usd);
    }
    return buf;
}
