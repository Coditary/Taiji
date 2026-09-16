#include "markdown_render.hpp"

#include "code_highlight.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cctype>
#include <sstream>
#include <string_view>

namespace {

tuinator::Style with_bold(const tuinator::Style& base, bool enabled) {
    tuinator::Style style = base;
    style.bold = enabled;
    return style;
}

tuinator::Style with_dim(const tuinator::Style& base, bool enabled) {
    tuinator::Style style = base;
    style.dim = enabled;
    return style;
}

tuinator::Style code_style(const tuinator::Style& base) {
    tuinator::Style style = base;
    style.foreground = tuinator::Color::Cyan;
    style.background = tuinator::Color::Default;
    style.bold = false;
    return style;
}

tuinator::Style header_style(const tuinator::Style& base, int level) {
    tuinator::Style style = base;
    style.bold = true;
    if (level == 1) {
        style.foreground_rgb = tuinator::rgb(120, 180, 255);
    } else if (level == 2) {
        style.foreground_rgb = tuinator::rgb(150, 190, 255);
    }
    return style;
}

tuinator::Style link_style(const tuinator::Style& base) {
    tuinator::Style style = base;
    style.foreground = tuinator::Color::Cyan;
    style.bold = true;
    return style;
}

int span_width(const MarkdownSpan& span) {
    return tuinator::text_display_width(span.text);
}

int line_width(const MarkdownLine& line) {
    int width = line.indent;
    for (const auto& span : line.spans) {
        width += span_width(span);
    }
    return width;
}

bool is_lang_token(const std::string& token) {
    if (token.empty() || token.size() > 12) {
        return false;
    }

    return std::all_of(token.begin(), token.end(), [](unsigned char ch) {
        return std::isalnum(ch) != 0 || ch == '+' || ch == '#';
    });
}

std::pair<std::string, std::string> split_inline_code(const std::string& raw) {
    const std::size_t colon = raw.find(':');
    if (colon == std::string::npos || colon == 0) {
        return {"", raw};
    }

    const std::string lang = raw.substr(0, colon);
    if (!is_lang_token(lang)) {
        return {"", raw};
    }

    return {lang, raw.substr(colon + 1)};
}

void append_span(std::vector<MarkdownSpan>& spans, const std::string& text, const tuinator::Style& style) {
    if (text.empty()) {
        return;
    }
    if (!spans.empty() && spans.back().style.foreground == style.foreground &&
        spans.back().style.background == style.background && spans.back().style.bold == style.bold &&
        spans.back().style.dim == style.dim && spans.back().style.foreground_rgb == style.foreground_rgb &&
        spans.back().style.background_rgb == style.background_rgb && spans.back().style.reverse == style.reverse) {
        spans.back().text += text;
        return;
    }
    spans.push_back(MarkdownSpan{.text = text, .style = style});
}

void append_highlighted_code(std::vector<MarkdownSpan>& out, const std::string& lang, const std::string& code,
                             const tuinator::Style& base) {
    const tuinator::Style fallback = code_style(base);
    const auto spans = highlight_code(lang, code);
    if (spans.empty()) {
        append_span(out, code, fallback);
        return;
    }

    std::size_t cursor = 0;
    for (const HighlightedSpan& span : spans) {
        if (span.start > cursor) {
            append_span(out, code.substr(cursor, span.start - cursor), fallback);
        }
        if (span.start < span.end) {
            append_span(out, code.substr(span.start, span.end - span.start),
                        highlight_kind_style(base, span.kind));
        }
        cursor = std::max(cursor, span.end);
    }

    if (cursor < code.size()) {
        append_span(out, code.substr(cursor), fallback);
    }
}

std::vector<std::string> split_words(const std::string& text) {
    std::vector<std::string> words;
    std::istringstream stream(text);
    std::string word;
    while (stream >> word) {
        words.push_back(word);
    }
    return words;
}

bool parse_bold(const std::string& input, std::size_t& index, const tuinator::Style& base,
                std::vector<MarkdownSpan>& out) {
    if (index + 1 >= input.size() || input[index] != '*' || input[index + 1] != '*') {
        return false;
    }
    const std::size_t end = input.find("**", index + 2);
    if (end == std::string::npos) {
        return false;
    }
    append_span(out, input.substr(index + 2, end - index - 2), with_bold(base, true));
    index = end + 2;
    return true;
}

bool parse_italic(const std::string& input, std::size_t& index, const tuinator::Style& base,
                  std::vector<MarkdownSpan>& out) {
    if (input[index] != '*' || (index + 1 < input.size() && input[index + 1] == '*')) {
        return false;
    }
    const std::size_t end = input.find('*', index + 1);
    if (end == std::string::npos) {
        return false;
    }
    append_span(out, input.substr(index + 1, end - index - 1), with_dim(base, true));
    index = end + 1;
    return true;
}

bool parse_code(const std::string& input, std::size_t& index, const tuinator::Style& base,
                std::vector<MarkdownSpan>& out) {
    if (input[index] != '`') {
        return false;
    }
    const std::size_t end = input.find('`', index + 1);
    if (end == std::string::npos) {
        return false;
    }
    const std::string raw = input.substr(index + 1, end - index - 1);
    const auto [lang, code] = split_inline_code(raw);
    append_highlighted_code(out, lang, code, base);
    index = end + 1;
    return true;
}

bool parse_link(const std::string& input, std::size_t& index, const tuinator::Style& base,
                std::vector<MarkdownSpan>& out) {
    if (input[index] != '[') {
        return false;
    }
    const std::size_t text_end = input.find(']', index + 1);
    if (text_end == std::string::npos || text_end + 1 >= input.size() || input[text_end + 1] != '(') {
        return false;
    }
    const std::size_t url_end = input.find(')', text_end + 2);
    if (url_end == std::string::npos) {
        return false;
    }
    append_span(out, input.substr(index + 1, text_end - index - 1), link_style(base));
    index = url_end + 1;
    return true;
}

std::vector<MarkdownSpan> parse_inline(const std::string& input, const tuinator::Style& base) {
    std::vector<MarkdownSpan> spans;
    std::size_t index = 0;
    while (index < input.size()) {
        if (parse_bold(input, index, base, spans) || parse_code(input, index, base, spans) ||
            parse_link(input, index, base, spans) || parse_italic(input, index, base, spans)) {
            continue;
        }

        const std::size_t next_special = input.find_first_of("*[`", index);
        const std::size_t end = next_special == std::string::npos ? input.size() : next_special;
        append_span(spans, input.substr(index, end - index), base);
        index = end;
    }
    return spans;
}

std::vector<MarkdownLine> wrap_spans(std::vector<MarkdownSpan> spans, int indent, int max_cols) {
    if (max_cols <= 0) {
        return {MarkdownLine{.spans = std::move(spans), .indent = indent}};
    }

    const int usable = std::max(1, max_cols - indent);
    std::vector<MarkdownLine> lines;
    MarkdownLine current;
    current.indent = indent;
    int width = 0;

    auto flush = [&]() {
        if (!current.spans.empty()) {
            lines.push_back(current);
            current = MarkdownLine{.indent = indent};
            width = 0;
        }
    };

    for (const auto& span : spans) {
        const auto words = split_words(span.text);
        for (const auto& word : words) {
            const int word_width = tuinator::text_display_width(word);
            const int extra = width == 0 ? word_width : word_width + 1;
            if (width > 0 && width + extra > usable) {
                flush();
            }
            if (width > 0) {
                append_span(current.spans, " ", span.style);
                ++width;
            }
            append_span(current.spans, word, span.style);
            width += word_width;
        }
    }

    flush();
    if (lines.empty()) {
        lines.push_back(MarkdownLine{.indent = indent});
    }
    return lines;
}

std::vector<MarkdownLine> layout_block(const std::string& text, const tuinator::Style& base, int indent, int max_cols) {
    return wrap_spans(parse_inline(text, base), indent, max_cols);
}

std::vector<MarkdownLine> layout_code_block(const std::string& text, const std::string& lang, const tuinator::Style& base,
                                            int indent, int max_cols) {
    std::vector<MarkdownSpan> spans;
    append_highlighted_code(spans, lang, text, base);
    return wrap_spans(std::move(spans), indent, max_cols);
}

std::string trim_copy(const std::string& value) {
    const auto begin = value.find_first_not_of(" \t\r\n");
    if (begin == std::string::npos) {
        return {};
    }
    const auto end = value.find_last_not_of(" \t\r\n");
    return value.substr(begin, end - begin + 1);
}

int header_level(std::string_view line, std::size_t& prefix_len) {
    if (line.empty() || line.front() != '#') {
        return 0;
    }
    int level = 0;
    while (level < static_cast<int>(line.size()) && line[level] == '#') {
        ++level;
    }
    if (level == 0 || level > 6 || level >= static_cast<int>(line.size()) || line[level] != ' ') {
        return 0;
    }
    prefix_len = static_cast<std::size_t>(level + 1);
    return level;
}

} // namespace

std::vector<MarkdownLine> layout_markdown(const std::string& text, const tuinator::Style& base_style, int max_cols) {
    std::vector<MarkdownLine> lines;
    bool in_code_block = false;
    std::string fence_lang;
    std::size_t start = 0;

    while (start <= text.size()) {
        const std::size_t end = text.find('\n', start);
        const std::string raw = end == std::string::npos ? text.substr(start) : text.substr(start, end - start);
        start = end == std::string::npos ? text.size() + 1 : end + 1;

        if (raw.rfind("```", 0) == 0) {
            if (!in_code_block) {
                fence_lang = trim_copy(raw.substr(3));
            } else {
                fence_lang.clear();
            }
            in_code_block = !in_code_block;
            continue;
        }

        if (in_code_block) {
            auto code_lines = layout_code_block(raw, fence_lang, base_style, 2, max_cols);
            for (auto& line : code_lines) {
                lines.push_back(std::move(line));
            }
            continue;
        }

        if (raw.empty()) {
            lines.push_back(MarkdownLine{.indent = 0});
            continue;
        }

        std::size_t prefix_len = 0;
        const int level = header_level(raw, prefix_len);
        if (level > 0) {
            auto header_lines = layout_block(raw.substr(prefix_len), header_style(base_style, level), 0, max_cols);
            for (auto& line : header_lines) {
                lines.push_back(std::move(line));
            }
            continue;
        }

        if (raw.rfind("> ", 0) == 0) {
            auto quote_lines = layout_block(raw.substr(2), with_dim(base_style, true), 2, max_cols);
            for (auto& line : quote_lines) {
                lines.push_back(std::move(line));
            }
            continue;
        }

        if (raw.rfind("- ", 0) == 0 || raw.rfind("* ", 0) == 0) {
            MarkdownLine bullet;
            bullet.indent = 0;
            append_span(bullet.spans, "•", base_style);
            auto item_lines = layout_block(raw.substr(2), base_style, 2, max_cols);
            if (!item_lines.empty()) {
                if (!item_lines.front().spans.empty()) {
                    bullet.spans.insert(bullet.spans.end(), item_lines.front().spans.begin(),
                                        item_lines.front().spans.end());
                }
                lines.push_back(std::move(bullet));
                for (std::size_t i = 1; i < item_lines.size(); ++i) {
                    lines.push_back(std::move(item_lines[i]));
                }
            } else {
                lines.push_back(std::move(bullet));
            }
            continue;
        }

        auto block_lines = layout_block(raw, base_style, 0, max_cols);
        for (auto& line : block_lines) {
            lines.push_back(std::move(line));
        }
    }

    if (lines.empty()) {
        lines.push_back(MarkdownLine{.indent = 0});
    }
    return lines;
}

int markdown_display_height(const std::string& text, int max_cols) {
    return static_cast<int>(layout_markdown(text, tuinator::Style{}, max_cols).size());
}

int markdown_display_width(const std::string& text, int max_cols) {
    int max_width = 0;
    for (const auto& line : layout_markdown(text, tuinator::Style{}, max_cols)) {
        max_width = std::max(max_width, line_width(line));
    }
    return max_width;
}
