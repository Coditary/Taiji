#pragma once

#include <taiji_highlight.h>

#include <tuinator/render/style.hpp>

#include <cstdint>
#include <string>
#include <vector>

struct HighlightedSpan {
    std::size_t start = 0;
    std::size_t end = 0;
    uint8_t kind = TAIJI_HL_DEFAULT;
};

std::vector<HighlightedSpan> highlight_code(const std::string& lang, const std::string& code);

tuinator::Style highlight_kind_style(const tuinator::Style& base, uint8_t kind);
