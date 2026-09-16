#pragma once

#include <tuinator/render/style.hpp>

#include <string>
#include <vector>

struct MarkdownSpan {
    std::string text;
    tuinator::Style style;
};

struct MarkdownLine {
    std::vector<MarkdownSpan> spans;
    int indent = 0;
};

std::vector<MarkdownLine> layout_markdown(const std::string& text, const tuinator::Style& base_style, int max_cols);
int markdown_display_height(const std::string& text, int max_cols);
int markdown_display_width(const std::string& text, int max_cols);
