#include "code_highlight.hpp"

#include <algorithm>

std::vector<HighlightedSpan> highlight_code(const std::string& lang, const std::string& code) {
    std::vector<HighlightedSpan> result;
    if (code.empty()) {
        return result;
    }

    taiji_hl_span* spans = nullptr;
    std::size_t count = 0;
    const char* lang_ptr = lang.empty() ? nullptr : lang.c_str();
    if (taiji_hl_highlight(lang_ptr, code.c_str(), &spans, &count) != 0 || spans == nullptr || count == 0) {
        return result;
    }

    result.reserve(count);
    for (std::size_t i = 0; i < count; ++i) {
        const taiji_hl_span& span = spans[i];
        if (span.end <= span.start || span.start >= code.size()) {
            continue;
        }
        result.push_back(HighlightedSpan{
            .start = span.start,
            .end = std::min<std::size_t>(span.end, code.size()),
            .kind = span.kind,
        });
    }

    taiji_hl_free(spans, count);
    return result;
}

tuinator::Style highlight_kind_style(const tuinator::Style& base, uint8_t kind) {
    tuinator::Style style = base;
    style.background = tuinator::Color::Default;
    style.bold = false;
    style.dim = false;

    switch (kind) {
    case TAIJI_HL_KEYWORD:
        style.foreground = tuinator::Color::Magenta;
        style.bold = true;
        break;
    case TAIJI_HL_STRING:
        style.foreground = tuinator::Color::Green;
        break;
    case TAIJI_HL_NUMBER:
        style.foreground_rgb = tuinator::rgb(255, 180, 80);
        break;
    case TAIJI_HL_COMMENT:
        style.foreground = tuinator::Color::White;
        style.dim = true;
        break;
    case TAIJI_HL_FUNCTION:
        style.foreground_rgb = tuinator::rgb(120, 180, 255);
        style.bold = true;
        break;
    case TAIJI_HL_TYPE:
        style.foreground = tuinator::Color::Cyan;
        style.bold = true;
        break;
    case TAIJI_HL_OPERATOR:
    case TAIJI_HL_PUNCTUATION:
        style.foreground = tuinator::Color::White;
        style.dim = true;
        break;
    case TAIJI_HL_PROPERTY:
    case TAIJI_HL_ATTRIBUTE:
    case TAIJI_HL_LABEL:
        style.foreground_rgb = tuinator::rgb(180, 200, 255);
        break;
    default:
        style.foreground = tuinator::Color::Cyan;
        break;
    }

    return style;
}
