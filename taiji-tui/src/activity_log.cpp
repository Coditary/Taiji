#include "activity_log.hpp"

#include "ui_test_ids.hpp"
#include "activity_diff_layout.hpp"
#include "ui_chat_spacing.hpp"
#include "ui_anim.hpp"
#include "ui_chat_column.hpp"
#include "ui_locale.hpp"
#include "ui_theme.hpp"

#include <libtaiji.h>

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cctype>
#include <cstdio>
#include <cstdlib>
#include <sstream>

namespace {

std::string json_string_field(const std::string& json, const std::string& key);

bool is_invoke_skill_tool_name(const std::string& name) {
    return name == "invoke_skill";
}

std::vector<std::string> split_lines(const std::string& text) {
    std::vector<std::string> lines;
    std::istringstream stream(text);
    std::string line;
    while (std::getline(stream, line)) {
        lines.push_back(line);
    }
    return lines;
}

std::string skill_name_from_args(const std::string& args);

std::string question_header_from_args(const std::string& args) {
    const std::string header = json_string_field(args, "header");
    if (!header.empty()) {
        return header;
    }
    const std::string question = json_string_field(args, "question");
    if (!question.empty()) {
        return question.size() > 42 ? question.substr(0, 39) + "..." : question;
    }
    const std::string fallback = ui_tr("tools", "scopes.question");
    return fallback.empty() ? "Question" : fallback;
}

std::string decision_state_from_args(const std::string& args) {
    for (const char* key : {"state", "text"}) {
        const std::string value = json_string_field(args, key);
        if (!value.empty()) {
            return value;
        }
    }
    return {};
}

std::string json_string_field(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return {};
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    if (i >= json.size() || json[i] != '"') {
        return {};
    }
    ++i;
    std::string out;
    while (i < json.size()) {
        const char ch = json[i++];
        if (ch == '"') {
            return out;
        }
        if (ch == '\\' && i < json.size()) {
            const char esc = json[i++];
            switch (esc) {
            case 'n':
                out.push_back('\n');
                break;
            case 'r':
                out.push_back('\r');
                break;
            case 't':
                out.push_back('\t');
                break;
            case '"':
                out.push_back('"');
                break;
            case '\\':
                out.push_back('\\');
                break;
            default:
                out.push_back(esc);
                break;
            }
            continue;
        }
        out.push_back(ch);
    }
    return {};
}

std::string trim_copy(const std::string& text) {
    const auto begin = text.find_first_not_of(" \t\n\r");
    if (begin == std::string::npos) {
        return {};
    }
    const auto end = text.find_last_not_of(" \t\n\r");
    return text.substr(begin, end - begin + 1);
}

std::string question_answer_summary(const std::string& text) {
    const std::string trimmed = trim_copy(text);
    if (trimmed.empty()) {
        return {};
    }
    const std::string prefix = "User has answered your questions: ";
    if (trimmed.rfind(prefix, 0) != 0) {
        return trimmed;
    }
    std::string rest = trimmed.substr(prefix.size());
    const std::size_t cont = rest.find(". Continue");
    if (cont != std::string::npos) {
        rest = rest.substr(0, cont);
    }
    return trim_copy(rest);
}

std::string skill_name_from_args(const std::string& args) {
    if (args.empty()) {
        return {};
    }
    for (const char* key : {"name", "skill", "skill_name"}) {
        const std::string value = json_string_field(args, key);
        if (!value.empty() && !is_invoke_skill_tool_name(value)) {
            return value;
        }
    }
    const std::string nested = json_string_field(args, "arguments");
    if (!nested.empty()) {
        return skill_name_from_args(nested);
    }
    return {};
}

std::string skill_name_from_result_text(const std::string& text) {
    const std::string marker = "## Skill: ";
    const std::size_t pos = text.find(marker);
    if (pos == std::string::npos) {
        return {};
    }
    const std::size_t start = pos + marker.size();
    const std::size_t end = text.find('\n', start);
    const std::string name = end == std::string::npos ? text.substr(start) : text.substr(start, end - start);
    const std::string trimmed = trim_copy(name);
    if (trimmed.empty() || is_invoke_skill_tool_name(trimmed)) {
        return {};
    }
    return trimmed;
}

constexpr int kTreeTextCol = 5; // "├─ ✓ "
constexpr int kBodyBarCol = 2;
constexpr int kBodyIndent = 4;
constexpr int kSpinnerPeriodMs = 80;
constexpr std::size_t kMaxThinkingLines = 120;
constexpr std::size_t kMaxResultLines = 48;
constexpr std::size_t kMaxParamLines = 40;
constexpr std::size_t kMaxToolExpandSourceChars = 48 * 1024;

const char* kSpinnerFrames[] = {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"};
constexpr int kSpinnerFrameCount = 10;

constexpr tuinator::Rgb kActivityOkBadgeBg{52, 78, 62};
constexpr tuinator::Rgb kActivityFailBadgeBg{78, 48, 48};
constexpr tuinator::Rgb kActivityAddLineBg{45, 68, 52};
constexpr tuinator::Rgb kActivityDelLineBg{72, 48, 48};
constexpr tuinator::Rgb kActivityRunningBadgeBg{58, 48, 72};
constexpr tuinator::Rgb kActivityDiffHeaderBg{38, 40, 48};
constexpr tuinator::Rgb kActivityDiffAddStat{120, 210, 140};
constexpr tuinator::Rgb kActivityDiffDelStat{235, 120, 120};
constexpr std::size_t kMaxDiffCodeLines = 80;
constexpr std::size_t kMaxInlineDiffTokens = 512;

using DiffTokenList = std::vector<std::string>;

DiffTokenList tokenize_diff_line(const std::string& text) {
    DiffTokenList tokens;
    std::string word;
    for (unsigned char ch : text) {
        if (ch == ' ' || ch == '\t') {
            if (!word.empty()) {
                tokens.push_back(word);
                word.clear();
            }
            tokens.push_back(std::string(1, static_cast<char>(ch)));
        } else {
            word.push_back(static_cast<char>(ch));
        }
    }
    if (!word.empty()) {
        tokens.push_back(word);
    }
    return tokens;
}

void append_diff_segment(std::vector<ActivityDiffTextSegment>& out, const std::string& text, bool highlight) {
    if (text.empty()) {
        return;
    }
    if (!out.empty() && out.back().highlight == highlight) {
        out.back().text += text;
        return;
    }
    out.push_back(ActivityDiffTextSegment{.text = text, .highlight = highlight});
}

enum class InlineDiffOp : std::uint8_t { Equal, Delete, Insert };

void build_inline_diff_segments(const std::string& old_text, const std::string& new_text,
                                std::vector<ActivityDiffTextSegment>& old_segments,
                                std::vector<ActivityDiffTextSegment>& new_segments) {
    old_segments.clear();
    new_segments.clear();
    const DiffTokenList old_tokens = tokenize_diff_line(old_text);
    const DiffTokenList new_tokens = tokenize_diff_line(new_text);
    if (old_tokens.empty() && new_tokens.empty()) {
        return;
    }
    if (old_tokens.size() > kMaxInlineDiffTokens || new_tokens.size() > kMaxInlineDiffTokens) {
        append_diff_segment(old_segments, old_text, true);
        append_diff_segment(new_segments, new_text, true);
        return;
    }

    const int n = static_cast<int>(old_tokens.size());
    const int m = static_cast<int>(new_tokens.size());
    const int cols = m + 1;
    std::vector<int> dp(static_cast<std::size_t>((n + 1) * cols), 0);
    auto at = [&](int i, int j) -> int { return dp[static_cast<std::size_t>(i * cols + j)]; };
    auto set = [&](int i, int j, int value) { dp[static_cast<std::size_t>(i * cols + j)] = value; };

    for (int i = 1; i <= n; ++i) {
        for (int j = 1; j <= m; ++j) {
            if (old_tokens[static_cast<std::size_t>(i - 1)] == new_tokens[static_cast<std::size_t>(j - 1)]) {
                set(i, j, at(i - 1, j - 1) + 1);
            } else {
                set(i, j, std::max(at(i, j - 1), at(i - 1, j)));
            }
        }
    }

    std::vector<InlineDiffOp> ops;
    int i = n;
    int j = m;
    while (i > 0 || j > 0) {
        if (i > 0 && j > 0 && old_tokens[static_cast<std::size_t>(i - 1)] == new_tokens[static_cast<std::size_t>(j - 1)]) {
            ops.push_back(InlineDiffOp::Equal);
            --i;
            --j;
        } else if (j > 0 && (i == 0 || at(i, j - 1) >= at(i - 1, j))) {
            ops.push_back(InlineDiffOp::Insert);
            --j;
        } else {
            ops.push_back(InlineDiffOp::Delete);
            --i;
        }
    }
    std::reverse(ops.begin(), ops.end());

    std::size_t oi = 0;
    std::size_t ni = 0;
    for (InlineDiffOp op : ops) {
        switch (op) {
        case InlineDiffOp::Equal:
            append_diff_segment(old_segments, old_tokens[oi], false);
            append_diff_segment(new_segments, new_tokens[ni], false);
            ++oi;
            ++ni;
            break;
        case InlineDiffOp::Delete:
            append_diff_segment(old_segments, old_tokens[oi], true);
            ++oi;
            break;
        case InlineDiffOp::Insert:
            append_diff_segment(new_segments, new_tokens[ni], true);
            ++ni;
            break;
        }
    }
}

bool is_file_edit_tool(const std::string& tool_name) {
    return tool_name == "edit_file" || tool_name == "write_file" || tool_name == "batch_edit" ||
           tool_name == "manage_file";
}

std::string path_basename(const std::string& path) {
    const std::size_t slash = path.find_last_of("/\\");
    if (slash == std::string::npos) {
        return path;
    }
    return path.substr(slash + 1);
}

std::string path_extension_lower(const std::string& path) {
    const std::string base = path_basename(path);
    const std::size_t dot = base.find_last_of('.');
    if (dot == std::string::npos || dot + 1 >= base.size()) {
        return {};
    }
    std::string ext = base.substr(dot + 1);
    for (char& ch : ext) {
        if (ch >= 'A' && ch <= 'Z') {
            ch = static_cast<char>(ch - 'A' + 'a');
        }
    }
    return ext;
}

struct FileTypeBadge {
    std::string label;
    tuinator::Rgb color;
};

FileTypeBadge file_type_badge(const std::string& path) {
    const std::string ext = path_extension_lower(path);
    if (ext == "go") {
        return {kNfLangGo, tuinator::rgb(97, 192, 235)};
    }
    if (ext == "rs") {
        return {kNfLangRust, tuinator::rgb(255, 138, 66)};
    }
    if (ext == "cpp" || ext == "cc" || ext == "cxx" || ext == "hpp") {
        return {kNfLangCpp, tuinator::rgb(120, 170, 255)};
    }
    if (ext == "c" || ext == "h") {
        return {kNfLangC, tuinator::rgb(120, 170, 255)};
    }
    if (ext == "js" || ext == "jsx") {
        return {kNfLangJavascript, tuinator::rgb(247, 223, 108)};
    }
    if (ext == "ts" || ext == "tsx") {
        return {kNfLangTypescript, tuinator::rgb(97, 160, 255)};
    }
    if (ext == "py") {
        return {kNfLangPython, tuinator::rgb(90, 170, 230)};
    }
    if (ext == "json") {
        return {kNfLangJson, tuinator::rgb(210, 180, 120)};
    }
    if (ext == "md" || ext == "markdown") {
        return {kNfLangMarkdown, tuinator::rgb(160, 200, 255)};
    }
    if (ext == "yaml" || ext == "yml") {
        return {kNfLangYaml, tuinator::rgb(200, 160, 230)};
    }
    if (ext == "toml") {
        return {kNfLangToml, tuinator::rgb(180, 150, 120)};
    }
    if (ext == "css" || ext == "scss") {
        return {kNfLangCss, tuinator::rgb(120, 170, 255)};
    }
    if (ext == "html") {
        return {kNfLangHtml, tuinator::rgb(230, 110, 90)};
    }
    if (ext == "sh" || ext == "bash" || ext == "zsh") {
        return {kNfLangShell, tuinator::rgb(180, 220, 160)};
    }
    if (ext == "txt") {
        return {kNfFile, ui_secondary_text()};
    }
    return {kNfFile, ui_secondary_text()};
}

int diff_card_width_cols(const ChatStreamLayout& stream) {
    return chat_bubble_max_cols(stream.content_width);
}

std::string strip_embedded_unified_diff(std::string body) {
    const std::string marker = "\n\n--- a/";
    const std::size_t pos = body.find(marker);
    if (pos != std::string::npos) {
        body.resize(pos);
    }
    return trim_copy(body);
}

std::string unified_diff_embedded_in_text(const std::string& text) {
    const std::string marker = "\n\n--- a/";
    const std::size_t pos = text.find(marker);
    if (pos == std::string::npos) {
        return {};
    }
    return text.substr(pos + 2);
}

std::string build_unified_diff_new_file(const std::string& path, const std::string& content) {
    if (path.empty()) {
        return {};
    }
    std::ostringstream out;
    out << "--- a/" << path << "\n+++ b/" << path << "\n";
    const std::vector<std::string> lines = split_lines(content);
    out << "@@ -0,0 +1," << lines.size() << " @@\n";
    for (const std::string& line : lines) {
        out << "+" << line << "\n";
    }
    return out.str();
}

std::string format_diff_gutter(int old_line, int new_line) {
    auto format_num = [](int n) -> std::string {
        if (n <= 0) {
            return "    ";
        }
        char buf[16];
        std::snprintf(buf, sizeof(buf), "%4d", n);
        return buf;
    };
    return format_num(old_line) + " " + format_num(new_line) + " │ ";
}

bool parse_hunk_header(const std::string& line, int& old_start, int& new_start) {
    if (line.rfind("@@", 0) != 0) {
        return false;
    }
    const std::size_t minus = line.find('-');
    const std::size_t plus = line.find('+');
    if (minus == std::string::npos || plus == std::string::npos || plus <= minus) {
        return false;
    }
    try {
        old_start = std::stoi(line.substr(minus + 1));
        new_start = std::stoi(line.substr(plus + 1));
        return new_start > 0;
    } catch (...) {
        return false;
    }
}

tuinator::Style activity_on_panel(tuinator::Style style) {
    return ui_resolve_main_text(style);
}

bool activity_preserve_text_background(tuinator::Rgb bg) {
    if (bg == ui_badge_bg()) {
        return true;
    }
    if (bg == kActivityOkBadgeBg || bg == kActivityFailBadgeBg || bg == kActivityRunningBadgeBg ||
        bg == kActivityAddLineBg || bg == kActivityDelLineBg || bg == kActivityDiffHeaderBg) {
        return true;
    }
    return false;
}

tuinator::Style activity_draw_style(tuinator::Style style) {
    tuinator::Style out = activity_on_panel(style);
    if (style.background_rgb.has_value() && activity_preserve_text_background(*style.background_rgb)) {
        out.background_rgb = *style.background_rgb;
    }
    return out;
}

void paint_activity_row_bg(tuinator::PaintContext& ctx, int y, int width,
                           tuinator::Rgb background = ui_main_bg()) {
    if (width <= 0) {
        return;
    }
    tuinator::Style fill;
    fill.background_rgb = background;
    ctx.canvas.fill_rect({0, y, width, 1}, ' ', fill);
}

void paint_activity_tree_trunk_clear(tuinator::PaintContext& ctx, int stream_x, int y) {
    if (kTreeTextCol <= 0) {
        return;
    }
    tuinator::Style fill;
    fill.background_rgb = ui_main_bg();
    ctx.canvas.fill_rect({stream_x, y, kTreeTextCol, 1}, ' ', fill);
}

void paint_activity_running_prefix_bg(tuinator::PaintContext& ctx, int stream_x, int y, int panel_width) {
    if (panel_width <= stream_x || kTreeTextCol <= 0) {
        return;
    }
    tuinator::Style fill;
    fill.background_rgb = ui_badge_bg();
    ctx.canvas.fill_rect({stream_x, y, kTreeTextCol, 1}, ' ', fill);
}

void paint_text_with_background(tuinator::PaintContext& ctx, int x, int y, const std::string& text,
                                tuinator::Style style) {
    if (text.empty()) {
        return;
    }
    const tuinator::Style draw = activity_draw_style(style);
    const int width = tuinator::text_display_width(text);
    if (width > 0) {
        ctx.canvas.fill_rect({x, y, width, 1}, ' ', draw);
    }
    ctx.canvas.draw_text({x, y}, text, draw);
}

void paint_activity_tree_trunk_continuation(tuinator::PaintContext& ctx, int stream_x, int y) {
    if (kTreeTextCol <= 0) {
        return;
    }
    tuinator::Style line;
    line.background_rgb = ui_main_bg();
    line.foreground_rgb = ui_secondary_text();
    line.dim = true;
    paint_text_with_background(ctx, stream_x, y, "│", line);
}

void paint_diff_header_line(tuinator::PaintContext& ctx, int card_x, int y, int card_width,
                            const std::string& filename, const std::string& icon, tuinator::Rgb icon_rgb,
                            int adds, int dels) {
    const int x0 = card_x;
    const int width = std::max(0, card_width);
    if (width <= 0) {
        return;
    }
    tuinator::Style fill;
    fill.background_rgb = kActivityDiffHeaderBg;
    ctx.canvas.fill_rect({x0, y, width, 1}, ' ', fill);

    int right = x0 + width - 1;
    if (dels > 0) {
        const std::string del_text = "-" + std::to_string(dels);
        right -= tuinator::text_display_width(del_text);
        tuinator::Style del_stat;
        del_stat.background_rgb = kActivityDiffHeaderBg;
        del_stat.foreground_rgb = kActivityDiffDelStat;
        del_stat.bold = true;
        paint_text_with_background(ctx, right, y, del_text, del_stat);
        right -= 1;
    }
    if (adds > 0) {
        const std::string add_text = "+" + std::to_string(adds);
        right -= tuinator::text_display_width(add_text);
        tuinator::Style add_stat;
        add_stat.background_rgb = kActivityDiffHeaderBg;
        add_stat.foreground_rgb = kActivityDiffAddStat;
        add_stat.bold = true;
        paint_text_with_background(ctx, right, y, add_text, add_stat);
    }

    int cursor = x0 + 1;
    if (!icon.empty()) {
        tuinator::Style icon_style;
        icon_style.background_rgb = kActivityDiffHeaderBg;
        icon_style.foreground_rgb = icon_rgb;
        icon_style.bold = true;
        paint_text_with_background(ctx, cursor, y, icon, icon_style);
        cursor += tuinator::text_display_width(icon) + 1;
    }

    const int name_max = std::max(0, right - cursor - 1);
    const std::size_t name_bytes = tuinator::text_byte_length_for_width(filename, name_max);
    if (name_bytes > 0) {
        tuinator::Style name_style;
        name_style.background_rgb = kActivityDiffHeaderBg;
        name_style.foreground_rgb = tuinator::rgb(230, 232, 240);
        name_style.bold = true;
        paint_text_with_background(ctx, cursor, y, filename.substr(0, name_bytes), name_style);
    }
}

tuinator::Style diff_intraline_context_style(bool deleted) {
    tuinator::Style style;
    style.background_rgb = ui_main_bg();
    style.foreground_rgb = deleted ? tuinator::rgb(195, 175, 175) : tuinator::rgb(175, 195, 180);
    style.dim = true;
    return style;
}

void paint_diff_text_segments(tuinator::PaintContext& ctx, int x, int y, int max_cols,
                              const std::vector<ActivityDiffTextSegment>& segments,
                              tuinator::Style line_style, tuinator::Style highlight_style, bool deleted) {
    if (max_cols <= 0 || segments.empty()) {
        return;
    }
    const tuinator::Style context_style = diff_intraline_context_style(deleted);
    int cursor = x;
    const int end_x = x + max_cols;
    for (const ActivityDiffTextSegment& segment : segments) {
        if (cursor >= end_x || segment.text.empty()) {
            continue;
        }
        const int remaining = end_x - cursor;
        const std::size_t bytes = tuinator::text_byte_length_for_width(segment.text, remaining);
        if (bytes == 0) {
            break;
        }
        tuinator::Style draw = segment.highlight ? highlight_style : context_style;
        if (!segment.highlight && line_style.background_rgb.has_value()) {
            draw.background_rgb = line_style.background_rgb;
        }
        const std::string piece = segment.text.substr(0, bytes);
        paint_text_with_background(ctx, cursor, y, piece, draw);
        cursor += tuinator::text_display_width(piece);
    }
}

void paint_diff_code_line(tuinator::PaintContext& ctx, int card_x, int y, int card_width, const std::string& text,
                          tuinator::Style style,
                          const std::vector<ActivityDiffTextSegment>* segments = nullptr) {
    const int x0 = card_x;
    const int width = std::max(0, card_width);
    if (width > 0) {
        tuinator::Style row = style;
        if (!row.background_rgb.has_value()) {
            row.background_rgb = ui_main_bg();
        }
        ctx.canvas.fill_rect({x0, y, width, 1}, ' ', row);
    }
    const int body_columns = std::max(0, card_width - 1);
    if (segments != nullptr && !segments->empty()) {
        std::string segment_text;
        for (const ActivityDiffTextSegment& segment : *segments) {
            segment_text += segment.text;
        }
        int gutter_cols = 0;
        if (!segment_text.empty() && text.size() >= segment_text.size()) {
            const std::string gutter = text.substr(0, text.size() - segment_text.size());
            gutter_cols = tuinator::text_display_width(gutter);
            if (gutter_cols > 0) {
                tuinator::Style gutter_style = style;
                gutter_style.dim = true;
                const std::size_t gutter_bytes = tuinator::text_byte_length_for_width(gutter, gutter_cols);
                if (gutter_bytes > 0) {
                    paint_text_with_background(ctx, x0 + 1, y, gutter.substr(0, gutter_bytes), gutter_style);
                }
            }
        }
        const bool deleted = style.background_rgb.has_value() && *style.background_rgb == kActivityDelLineBg;
        paint_diff_text_segments(ctx, x0 + 1 + gutter_cols, y, body_columns - gutter_cols, *segments, style, style,
                                 deleted);
        return;
    }
    const std::size_t body_bytes = tuinator::text_byte_length_for_width(text, body_columns);
    if (body_bytes > 0) {
        paint_text_with_background(ctx, x0 + 1, y, text.substr(0, body_bytes), style);
    }
}

std::string format_single_diff_gutter(int line_no) {
    if (line_no <= 0) {
        return "    │ ";
    }
    char buf[16];
    std::snprintf(buf, sizeof(buf), "%4d │ ", line_no);
    return buf;
}

void paint_diff_pane(tuinator::PaintContext& ctx, int x, int y, int width, const std::string& gutter,
                     const std::string& content, tuinator::Style style,
                     const std::vector<ActivityDiffTextSegment>* segments = nullptr) {
    if (width <= 0) {
        return;
    }
    tuinator::Style row = style;
    if (!row.background_rgb.has_value()) {
        row.background_rgb = ui_main_bg();
    }
    ctx.canvas.fill_rect({x, y, width, 1}, ' ', row);
    int cursor = x + 1;
    const int gutter_w = tuinator::text_display_width(gutter);
    if (gutter_w > 0 && cursor + gutter_w <= x + width) {
        tuinator::Style gutter_style = row;
        gutter_style.dim = true;
        paint_text_with_background(ctx, cursor, y, gutter, gutter_style);
        cursor += gutter_w;
    }
    const int content_cols = std::max(0, x + width - cursor);
    if (segments != nullptr && !segments->empty()) {
        const bool deleted = style.background_rgb.has_value() && *style.background_rgb == kActivityDelLineBg;
        paint_diff_text_segments(ctx, cursor, y, content_cols, *segments, style, style, deleted);
        return;
    }
    const std::size_t content_bytes = tuinator::text_byte_length_for_width(content, content_cols);
    if (content_bytes > 0) {
        paint_text_with_background(ctx, cursor, y, content.substr(0, content_bytes), style);
    }
}

void paint_diff_split_row(tuinator::PaintContext& ctx, int card_x, int y, int card_w, int old_line, int new_line,
                          const std::string& left_text, const std::string& right_text, tuinator::Style left_style,
                          tuinator::Style right_style,
                          const std::vector<ActivityDiffTextSegment>* left_segments = nullptr,
                          const std::vector<ActivityDiffTextSegment>* right_segments = nullptr) {
    const int width = std::max(0, card_w);
    if (width <= 0) {
        return;
    }
    const int split_x = card_x + width / 2;
    const int left_w = split_x - card_x;
    const int right_x = split_x + 1;
    const int right_w = card_x + width - right_x;

    paint_diff_pane(ctx, card_x, y, left_w, format_single_diff_gutter(old_line), left_text, left_style, left_segments);

    if (split_x >= card_x && split_x < card_x + width) {
        tuinator::Style divider = left_style;
        divider.background_rgb = ui_main_bg();
        divider.foreground_rgb = ui_secondary_text();
        divider.dim = true;
        paint_text_with_background(ctx, split_x, y, "│", divider);
    }

    paint_diff_pane(ctx, right_x, y, right_w, format_single_diff_gutter(new_line), right_text, right_style,
                    right_segments);
}

std::string skill_highlight_token(const std::string& skill_name, const std::string& tool_name,
                                  const std::string& args, bool skill_scope) {
    if (!skill_name.empty() && !is_invoke_skill_tool_name(skill_name)) {
        return skill_name;
    }
    if (skill_scope || tool_name == "skill_search") {
        const std::string from_args = skill_name_from_args(args);
        if (!from_args.empty() && !is_invoke_skill_tool_name(from_args)) {
            return from_args;
        }
    }
    return {};
}

void paint_activity_header_text(tuinator::PaintContext& ctx, int x, int y, const std::string& text,
                                tuinator::Style header_style, const std::string& skill_token) {
    if (text.empty()) {
        return;
    }
    if (skill_token.empty()) {
        paint_text_with_background(ctx, x, y, text, header_style);
        return;
    }
    const std::size_t pos = text.find(skill_token);
    if (pos == std::string::npos) {
        paint_text_with_background(ctx, x, y, text, header_style);
        return;
    }

    int cursor = x;
    if (pos > 0) {
        const std::string before = text.substr(0, pos);
        paint_text_with_background(ctx, cursor, y, before, header_style);
        cursor += tuinator::text_display_width(before);
    }

    tuinator::Style badge = header_style;
    badge.background_rgb = ui_badge_bg();
    badge.foreground_rgb = ui_section_heading();
    badge.bold = true;
    paint_text_with_background(ctx, cursor, y, skill_token, badge);
    cursor += tuinator::text_display_width(skill_token);

    const std::size_t after_pos = pos + skill_token.size();
    if (after_pos < text.size()) {
        paint_text_with_background(ctx, cursor, y, text.substr(after_pos), header_style);
    }
}

void paint_activity_tree_prefix(tuinator::PaintContext& ctx, int stream_x, int y, const std::string& branch,
                                const std::string& status, tuinator::Style branch_style,
                                tuinator::Style status_style, tuinator::Rgb row_bg) {
    tuinator::Style row_fill;
    row_fill.background_rgb = row_bg;
    ctx.canvas.fill_rect({stream_x, y, kTreeTextCol, 1}, ' ', row_fill);
    paint_text_with_background(ctx, stream_x, y, branch, branch_style);

    if (status.empty()) {
        return;
    }
    const int status_x = stream_x + 3;
    paint_text_with_background(ctx, status_x, y, status, status_style);

    const int gap_x = status_x + tuinator::text_display_width(status);
    const int gap_w = (stream_x + kTreeTextCol) - gap_x;
    if (gap_w > 0) {
        tuinator::Style gap = branch_style;
        gap.background_rgb = row_bg;
        ctx.canvas.fill_rect({gap_x, y, gap_w, 1}, ' ', activity_draw_style(gap));
    }
}

tuinator::Style make_done_style() {
    tuinator::Style style;
    style.background_rgb = ui_main_bg();
    style.foreground_rgb = ui_section_heading();
    style.bold = false;
    return style;
}

tuinator::Style make_failed_style() {
    tuinator::Style style;
    style.background_rgb = ui_main_bg();
    style.foreground_rgb = tuinator::rgb(250, 200, 200);
    style.bold = true;
    return style;
}

tuinator::Style make_failed_body_style() {
    tuinator::Style style;
    style.background_rgb = ui_main_bg();
    style.foreground_rgb = tuinator::rgb(240, 190, 190);
    return style;
}

bool tool_observation_failed(const std::string& text) {
    const std::string trimmed = trim_copy(text);
    constexpr char kPrefix[] = "error:";
    if (trimmed.size() < sizeof(kPrefix) - 1) {
        return false;
    }
    for (std::size_t i = 0; i < sizeof(kPrefix) - 1; ++i) {
        const unsigned char c = static_cast<unsigned char>(trimmed[i]);
        const unsigned char expected = static_cast<unsigned char>(kPrefix[i]);
        const unsigned char lower = c >= 'A' && c <= 'Z' ? static_cast<unsigned char>(c - 'A' + 'a') : c;
        if (lower != expected) {
            return false;
        }
    }
    return true;
}

tuinator::Style make_add_style() {
    tuinator::Style style;
    style.background_rgb = kActivityAddLineBg;
    style.foreground_rgb = tuinator::rgb(200, 235, 205);
    return style;
}

tuinator::Style make_del_style() {
    tuinator::Style style;
    style.background_rgb = kActivityDelLineBg;
    style.foreground_rgb = tuinator::rgb(250, 200, 200);
    return style;
}

std::vector<std::string> wrap_paragraph(const std::string& paragraph, int max_cols) {
    if (max_cols <= 0 || paragraph.empty()) {
        return paragraph.empty() ? std::vector<std::string>{} : std::vector<std::string>{paragraph};
    }

    std::vector<std::string> lines;
    std::istringstream words(paragraph);
    std::string word;
    std::string current;

    while (words >> word) {
        const std::string candidate = current.empty() ? word : current + " " + word;
        if (tuinator::text_display_width(candidate) <= max_cols) {
            current = candidate;
            continue;
        }

        if (!current.empty()) {
            lines.push_back(current);
            current = word;
            continue;
        }

        std::size_t offset = 0;
        while (offset < word.size()) {
            const std::size_t bytes =
                tuinator::text_byte_length_for_width(std::string_view(word).substr(offset), max_cols);
            if (bytes == 0) {
                break;
            }
            lines.push_back(word.substr(offset, bytes));
            offset += bytes;
        }
        current.clear();
    }

    if (!current.empty()) {
        lines.push_back(current);
    }
    return lines;
}

std::vector<std::string> wrap_text(const std::string& text, int max_cols) {
    std::vector<std::string> lines;
    std::size_t start = 0;
    while (start <= text.size()) {
        const std::size_t end = text.find('\n', start);
        const std::string paragraph =
            end == std::string::npos ? text.substr(start) : text.substr(start, end - start);
        const auto wrapped = wrap_paragraph(paragraph, max_cols);
        lines.insert(lines.end(), wrapped.begin(), wrapped.end());
        if (end == std::string::npos) {
            break;
        }
        start = end + 1;
    }
    if (lines.empty() && !text.empty()) {
        lines.push_back(text);
    }
    return lines;
}

/// Minimal JSON reader used to render tool arguments as readable
/// "key: value" lines instead of raw JSON.
class JsonPrinter {
  public:
    explicit JsonPrinter(std::string_view text) : text_(text) {}

    /// Parses the input into formatted lines. Returns false when the input is
    /// not valid JSON (caller should fall back to raw text).
    bool print(std::vector<std::string>& out) {
        lines_ = &out;
        pos_ = 0;
        if (!value(0, "")) {
            return false;
        }
        skip_ws();
        return pos_ == text_.size();
    }

  private:
    void skip_ws() {
        while (pos_ < text_.size() &&
               (text_[pos_] == ' ' || text_[pos_] == '\t' || text_[pos_] == '\n' || text_[pos_] == '\r')) {
            ++pos_;
        }
    }

    char peek() const { return pos_ < text_.size() ? text_[pos_] : '\0'; }

    bool consume(char expected) {
        if (peek() != expected) {
            return false;
        }
        ++pos_;
        return true;
    }

    void line(int indent, const std::string& text) {
        lines_->push_back(std::string(static_cast<std::size_t>(indent), ' ') + text);
    }

    bool string_value(std::string& out) {
        if (!consume('"')) {
            return false;
        }
        out.clear();
        while (pos_ < text_.size()) {
            const char ch = text_[pos_++];
            if (ch == '"') {
                return true;
            }
            if (ch == '\\' && pos_ < text_.size()) {
                const char esc = text_[pos_++];
                switch (esc) {
                case 'n': out.push_back('\n'); break;
                case 'r': out.push_back('\r'); break;
                case 't': out.push_back('\t'); break;
                case 'b': out.push_back('\b'); break;
                case 'f': out.push_back('\f'); break;
                case 'u': {
                    // Keep unicode escapes verbatim; good enough for display.
                    if (pos_ + 4 <= text_.size()) {
                        out.append("\\u").append(text_.substr(pos_, 4));
                        pos_ += 4;
                    }
                    break;
                }
                default: out.push_back(esc); break;
                }
                continue;
            }
            out.push_back(ch);
        }
        return false;
    }

    bool scalar(std::string& out) {
        if (peek() == '"') {
            return string_value(out);
        }
        const std::size_t begin = pos_;
        while (pos_ < text_.size()) {
            const char ch = text_[pos_];
            if (ch == ',' || ch == '}' || ch == ']' || ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r') {
                break;
            }
            ++pos_;
        }
        out = std::string(text_.substr(begin, pos_ - begin));
        return !out.empty();
    }

    /// Renders a scalar line; multi-line strings (e.g. shell commands) are
    /// broken into one display line per source line below the key.
    void scalar_line(int indent, const std::string& lead, const std::string& value_text) {
        if (value_text.find('\n') == std::string::npos) {
            line(indent, lead.empty() ? value_text : lead + " " + value_text);
            return;
        }
        if (!lead.empty()) {
            line(indent, lead);
        }
        const int cont = lead.empty() ? indent : indent + 2;
        std::size_t start = 0;
        while (start <= value_text.size()) {
            const std::size_t end = value_text.find('\n', start);
            line(cont, end == std::string::npos ? value_text.substr(start)
                                                : value_text.substr(start, end - start));
            if (end == std::string::npos) {
                break;
            }
            start = end + 1;
        }
    }

    bool value(int indent, const std::string& lead) {
        skip_ws();
        const char ch = peek();
        if (ch == '{') {
            return object_value(indent, lead);
        }
        if (ch == '[') {
            return array_value(indent, lead);
        }
        std::string scalar_text;
        if (!scalar(scalar_text)) {
            return false;
        }
        scalar_line(indent, lead, scalar_text);
        return true;
    }

    bool object_value(int indent, const std::string& lead) {
        if (!consume('{')) {
            return false;
        }
        skip_ws();
        if (consume('}')) {
            line(indent, lead.empty() ? "{}" : lead + " {}");
            return true;
        }
        if (!lead.empty()) {
            line(indent, lead);
        }
        const int child = lead.empty() ? indent : indent + 2;
        while (true) {
            skip_ws();
            std::string key;
            if (!string_value(key)) {
                return false;
            }
            skip_ws();
            if (!consume(':')) {
                return false;
            }
            if (!value(child, key + ":")) {
                return false;
            }
            skip_ws();
            if (consume(',')) {
                continue;
            }
            return consume('}');
        }
    }

    bool array_value(int indent, const std::string& lead) {
        if (!consume('[')) {
            return false;
        }
        skip_ws();
        if (consume(']')) {
            line(indent, lead.empty() ? "[]" : lead + " []");
            return true;
        }
        if (!lead.empty()) {
            line(indent, lead);
        }
        const int child = lead.empty() ? indent : indent + 2;
        while (true) {
            if (!value(child, "-")) {
                return false;
            }
            skip_ws();
            if (consume(',')) {
                continue;
            }
            return consume(']');
        }
    }

    std::string_view text_;
    std::size_t pos_ = 0;
    std::vector<std::string>* lines_ = nullptr;
};

bool has_non_blank_text(const std::string& text) {
    return text.find_first_not_of(" \t\n\r") != std::string::npos;
}

std::string truncate_tool_expand_source(const std::string& text) {
    if (text.size() <= kMaxToolExpandSourceChars) {
        return text;
    }
    const std::size_t omitted = text.size() - kMaxToolExpandSourceChars;
    return text.substr(0, kMaxToolExpandSourceChars) + "\n… (" + std::to_string(omitted) + " bytes truncated for display)";
}

std::vector<std::string> pretty_args_lines(const std::string& args) {
    std::vector<std::string> lines;
    const std::string limited = truncate_tool_expand_source(args);
    const auto begin = limited.find_first_not_of(" \t\n\r");
    if (begin == std::string::npos) {
        return lines;
    }
    JsonPrinter printer(std::string_view(limited).substr(begin));
    if (printer.print(lines)) {
        return lines;
    }
    // Not JSON: show the raw text.
    return wrap_text(limited, 68);
}

std::vector<std::string> tool_args_lines_for_display(const std::string& args, int wrap_cols) {
    if (ui_tool_expand_args_format() == "json") {
        const auto begin = args.find_first_not_of(" \t\n\r");
        if (begin == std::string::npos) {
            return {};
        }
        return wrap_text(args.substr(begin), wrap_cols);
    }
    return pretty_args_lines(args);
}

std::vector<std::string> tool_result_lines_for_display(const std::string& body, int wrap_cols) {
    std::vector<std::string> prefix;
    if (body.size() > kMaxToolExpandSourceChars) {
        prefix.push_back("… (" + std::to_string(body.size() - kMaxToolExpandSourceChars) +
                         " bytes truncated for display)");
    }
    const std::string limited = truncate_tool_expand_source(body);
    std::vector<std::string> lines;
    if (ui_tool_expand_result_format() == "json") {
        lines = pretty_args_lines(limited);
    }
    if (lines.empty()) {
        lines = wrap_text(limited, wrap_cols);
    }
    if (!prefix.empty()) {
        lines.insert(lines.begin(), prefix.begin(), prefix.end());
    }
    return lines;
}

} // namespace

ActivityLog::ActivityLog(tuinator::Style active_style, tuinator::Style muted_style)
    : active_style_(active_style), done_style_(make_done_style()), failed_style_(make_failed_style()),
      failed_body_style_(make_failed_body_style()), muted_style_(muted_style),
      body_style_(muted_style), thinking_body_style_(muted_style), heading_style_(muted_style),
      add_style_(make_add_style()), del_style_(make_del_style()), bar_style_(muted_style) {
    set_test_id(taiji::ui::test_id::kActivityLog);
    thinking_body_style_.italic = true;
    heading_style_.bold = true;
    bar_style_.dim = true;
}

void ActivityLog::invalidate_body_cache(Entry& entry) {
    entry.body_cache_valid = false;
    entry.body_cache_wrap_cols = 0;
    entry.body_cache.clear();
}

void ActivityLog::invalidate_all_body_caches() {
    for (Entry& entry : entries_) {
        invalidate_body_cache(entry);
    }
}

void ActivityLog::clear() {
    entries_.clear();
    received_think_deltas_ = false;
    spinner_ms_ = 0;
    pipeline_title_.clear();
    generating_tail_ = false;
    hovered_entry_index_ = -1;
    mark_dirty();
    mark_layout_dirty();
}

void ActivityLog::set_pipeline_title(std::string title) {
    if (pipeline_title_ == title) {
        return;
    }
    pipeline_title_ = std::move(title);
    mark_dirty();
    mark_layout_dirty();
}

void ActivityLog::set_generating_tail(bool show, std::string label) {
    if (generating_tail_ == show && generating_label_ == label) {
        return;
    }
    generating_tail_ = show;
    generating_label_ = std::move(label);
    mark_dirty();
    mark_layout_dirty();
}

std::string ActivityLog::scope_label(Scope scope) {
    const char* key = "scopes.tool";
    switch (scope) {
    case Scope::Thinking:
        key = "scopes.thinking";
        break;
    case Scope::Tool:
        key = "scopes.tool";
        break;
    case Scope::MCP:
        key = "scopes.mcp";
        break;
    case Scope::Skill:
        key = "scopes.skill";
        break;
    case Scope::Decision:
        key = "scopes.decision";
        break;
    case Scope::Question:
        key = "scopes.question";
        break;
    }
    const std::string translated = ui_tr("tools", key);
    if (!translated.empty()) {
        return translated;
    }
    switch (scope) {
    case Scope::Thinking:
        return "Thinking";
    case Scope::Tool:
        return "Tool";
    case Scope::MCP:
        return "MCP";
    case Scope::Skill:
        return "Skill";
    case Scope::Decision:
        return "Decision";
    case Scope::Question:
        return "Question";
    }
    return "Tool";
}

std::string ActivityLog::format_duration(int64_t ms) {
    char buf[32];
    if (ms < 1000) {
        std::snprintf(buf, sizeof(buf), "%dms", static_cast<int>(ms));
    } else {
        std::snprintf(buf, sizeof(buf), "%.1fs", static_cast<double>(ms) / 1000.0);
    }
    return buf;
}

int64_t ActivityLog::elapsed_ms(const Entry& entry) const {
    if (entry.has_duration) {
        return entry.duration_ms;
    }
    if (!entry.running) {
        return 0;
    }
    const auto now = std::chrono::steady_clock::now();
    return std::chrono::duration_cast<std::chrono::milliseconds>(now - entry.started_at).count();
}

ActivityLog::Entry* ActivityLog::running_thinking_entry() {
    if (!entries_.empty() && entries_.back().scope == Scope::Thinking && entries_.back().running) {
        return &entries_.back();
    }
    return nullptr;
}

void ActivityLog::begin_thinking() {
    if (running_thinking_entry() != nullptr) {
        return;
    }
    Entry entry;
    entry.scope = Scope::Thinking;
    entry.running = true;
    entry.started_at = std::chrono::steady_clock::now();
    entries_.push_back(std::move(entry));
    mark_dirty();
    mark_layout_dirty();
}

void ActivityLog::append_thinking(const std::string& delta) {
    begin_thinking();
    received_think_deltas_ = true;
    Entry* entry = running_thinking_entry();
    if (entry == nullptr) {
        return;
    }
    entry->body += delta;
    entry->streamed = true;
    if (entry->expanded) {
        mark_layout_dirty();
    }
    mark_dirty();
}

void ActivityLog::add_thought(const std::string& text) {
    // When thinking was streamed live the entries already hold the text;
    // completed thought events would duplicate it.
    if (received_think_deltas_) {
        end_thinking();
        return;
    }
    if (text.empty()) {
        return;
    }
    Entry entry;
    entry.scope = Scope::Thinking;
    entry.running = false;
    entry.body = text;
    entries_.push_back(std::move(entry));
    mark_dirty();
    mark_layout_dirty();
}

void ActivityLog::end_thinking() {
    Entry* entry = running_thinking_entry();
    if (entry == nullptr) {
        return;
    }
    finalize_entry(*entry);
}

void ActivityLog::finalize_entry(Entry& entry) {
    if (!entry.running) {
        return;
    }
    entry.running = false;
    if (!entry.has_duration) {
        entry.duration_ms = elapsed_ms(entry);
        entry.has_duration = true;
    }
    entry.settle_ms = kUiAnimFadeMs;
    entry.flash_ms = kUiAnimFlashMs;
    mark_dirty();
    mark_layout_dirty();
}

void ActivityLog::finish_all() {
    bool changed = false;
    for (Entry& entry : entries_) {
        if (entry.running) {
            finalize_entry(entry);
            changed = true;
        }
    }
    if (changed) {
        mark_dirty();
        mark_layout_dirty();
    }
}

void ActivityLog::start_tool(const std::string& tool_name, const std::string& args, bool is_mcp, bool is_skill,
                             const std::string& skill_name, const std::string& mcp_server, bool is_decision,
                             const std::string& decision_driver, bool is_ask_user) {
    // A tool call ends any ongoing thinking phase.
    end_thinking();
    if (is_ask_user || tool_name == "ask_user") {
        return;
    }

    Entry entry;
    const bool skill_call = is_skill || tool_name == "invoke_skill";
    const bool decision_call = is_decision || tool_name == "decide";
    if (decision_call) {
        entry.scope = Scope::Decision;
    } else if (skill_call) {
        entry.scope = Scope::Skill;
    } else if (is_mcp) {
        entry.scope = Scope::MCP;
    } else {
        entry.scope = Scope::Tool;
    }
    entry.running = true;
    entry.name = tool_name;
    entry.server = mcp_server;
    entry.skill_name = skill_name;
    if (skill_call && entry.skill_name.empty()) {
        entry.skill_name = skill_name_from_args(args);
    }
    if (skill_call && is_invoke_skill_tool_name(entry.skill_name)) {
        entry.skill_name.clear();
    }
    entry.decision_driver = decision_driver;
    entry.args = args;
    entry.started_at = std::chrono::steady_clock::now();
    if (decision_call) {
        entry.expanded = true;
    }
    entries_.push_back(std::move(entry));
    mark_dirty();
    mark_layout_dirty();
}

std::vector<ActivityLog::BodyLine> ActivityLog::parse_diff(const std::string& diff, tuinator::Style muted,
                                                           tuinator::Style add, tuinator::Style del) {
    std::vector<BodyLine> lines;
    if (diff.empty()) {
        return lines;
    }

    std::size_t start = 0;
    while (start <= diff.size()) {
        std::size_t end = diff.find('\n', start);
        if (end == std::string::npos) {
            end = diff.size();
        }
        std::string line = diff.substr(start, end - start);
        tuinator::Style style = muted;
        if (!line.empty() && line[0] == '+') {
            style = add;
        } else if (!line.empty() && line[0] == '-') {
            style = del;
        }
        lines.push_back(BodyLine{.text = std::move(line), .style = style, .bar = true});
        if (end >= diff.size()) {
            break;
        }
        start = end + 1;
    }
    return lines;
}

void ActivityLog::append_file_diff_side_by_side_body(const std::string& diff, std::vector<BodyLine>& lines,
                                                    tuinator::Style muted, tuinator::Style add, tuinator::Style del,
                                                    bool new_file_hunk) {
    struct SideCell {
        std::string text;
        tuinator::Style style{};
        int line_no = 0;
        std::vector<ActivityDiffTextSegment> segments;
    };

    std::vector<SideCell> old_pending;
    std::vector<SideCell> new_pending;
    std::size_t code_lines = 0;

    auto push_split = [&](const SideCell& left, const SideCell& right) {
        BodyLine row;
        row.kind = BodyLineKind::DiffSplit;
        row.diff_fill_row = true;
        row.text = left.text;
        row.style = left.style;
        row.diff_old_line = left.line_no;
        row.diff_new_line = right.line_no;
        row.diff_right_text = right.text;
        row.diff_right_style = right.style;
        row.diff_segments = left.segments;
        row.diff_right_segments = right.segments;
        lines.push_back(std::move(row));
    };

    auto flush_pending = [&]() {
        const std::size_t rows = std::max(old_pending.size(), new_pending.size());
        for (std::size_t i = 0; i < rows; ++i) {
            SideCell left;
            SideCell right;
            if (i < old_pending.size()) {
                left = old_pending[i];
            } else {
                left.style = muted;
                left.style.background_rgb = ui_main_bg();
            }
            if (i < new_pending.size()) {
                right = new_pending[i];
            } else {
                right.style = muted;
                right.style.background_rgb = ui_main_bg();
            }
            if (!left.text.empty() && !right.text.empty() && left.style.background_rgb.has_value() &&
                right.style.background_rgb.has_value() && *left.style.background_rgb == kActivityDelLineBg &&
                *right.style.background_rgb == kActivityAddLineBg) {
                build_inline_diff_segments(left.text, right.text, left.segments, right.segments);
            } else if (!left.text.empty() && left.segments.empty() &&
                       left.style.background_rgb.has_value() && *left.style.background_rgb == kActivityDelLineBg) {
                append_diff_segment(left.segments, left.text, true);
            } else if (!right.text.empty() && right.segments.empty() &&
                       right.style.background_rgb.has_value() && *right.style.background_rgb == kActivityAddLineBg) {
                append_diff_segment(right.segments, right.text, true);
            }
            push_split(left, right);
        }
        old_pending.clear();
        new_pending.clear();
    };

    int old_line = 0;
    int new_line = 0;
    bool in_hunk = false;

    std::size_t start = 0;
    while (start <= diff.size()) {
        std::size_t end = diff.find('\n', start);
        if (end == std::string::npos) {
            end = diff.size();
        }
        std::string line = diff.substr(start, end - start);

        if (line.rfind("--- ", 0) == 0 || line.rfind("+++ ", 0) == 0) {
            if (end >= diff.size()) {
                break;
            }
            start = end + 1;
            continue;
        }
        if (line.rfind("@@", 0) == 0) {
            flush_pending();
            int parsed_old = 0;
            int parsed_new = 0;
            if (parse_hunk_header(line, parsed_old, parsed_new)) {
                old_line = parsed_old > 0 ? parsed_old : 0;
                new_line = parsed_new;
                in_hunk = true;
            }
            if (end >= diff.size()) {
                break;
            }
            start = end + 1;
            continue;
        }
        if (line.rfind("...", 0) == 0) {
            flush_pending();
            BodyLine more;
            more.bar = false;
            more.text = line;
            more.style = muted;
            more.style.italic = true;
            lines.push_back(std::move(more));
            break;
        }

        if (!in_hunk || line.empty()) {
            if (end >= diff.size()) {
                break;
            }
            start = end + 1;
            continue;
        }

        char prefix = line[0];
        std::string content = line.substr(1);

        if (prefix == ' ') {
            flush_pending();
            tuinator::Style context = muted;
            context.dim = true;
            context.background_rgb = ui_main_bg();
            SideCell left{content, context, old_line};
            SideCell right{content, context, new_line};
            push_split(left, right);
            if (old_line > 0) {
                ++old_line;
            }
            if (new_line > 0) {
                ++new_line;
            }
        } else if (prefix == '-') {
            const std::size_t max_code_lines = new_file_hunk ? 200 : kMaxDiffCodeLines;
            if (code_lines >= max_code_lines) {
                flush_pending();
                BodyLine more;
                more.bar = false;
                more.text = "… (diff truncated in UI)";
                more.style = muted;
                more.style.italic = true;
                lines.push_back(std::move(more));
                break;
            }
            SideCell cell{content, del, old_line};
            old_pending.push_back(std::move(cell));
            if (old_line > 0) {
                ++old_line;
            }
            ++code_lines;
        } else if (prefix == '+') {
            const std::size_t max_code_lines = new_file_hunk ? 200 : kMaxDiffCodeLines;
            if (code_lines >= max_code_lines) {
                flush_pending();
                BodyLine more;
                more.bar = false;
                more.text = "… (diff truncated in UI)";
                more.style = muted;
                more.style.italic = true;
                lines.push_back(std::move(more));
                break;
            }
            SideCell cell{content, add, new_line};
            new_pending.push_back(std::move(cell));
            if (new_line > 0) {
                ++new_line;
            }
            ++code_lines;
        }

        if (end >= diff.size()) {
            break;
        }
        start = end + 1;
    }
    flush_pending();
}

std::vector<ActivityLog::BodyLine> ActivityLog::parse_file_diff_view(const std::string& diff,
                                                                      const std::string& path_hint,
                                                                      tuinator::Style muted, tuinator::Style add,
                                                                      tuinator::Style del, bool* file_diff_view) {
    std::vector<BodyLine> lines;
    if (file_diff_view != nullptr) {
        *file_diff_view = false;
    }
    if (diff.empty()) {
        return lines;
    }

    std::string file_path = path_hint;
    int adds = 0;
    int dels = 0;
    std::size_t code_lines = 0;

    std::size_t start = 0;
    while (start <= diff.size()) {
        std::size_t end = diff.find('\n', start);
        if (end == std::string::npos) {
            end = diff.size();
        }
        std::string line = diff.substr(start, end - start);

        if (line.rfind("+++ b/", 0) == 0) {
            file_path = line.substr(6);
        } else if (line.rfind("--- a/", 0) == 0 && file_path.empty()) {
            file_path = line.substr(6);
        } else if (line.rfind("+++ ", 0) == 0 || line.rfind("--- ", 0) == 0) {
            // file headers, not diff stats
        } else if (!line.empty() && line[0] == '+') {
            ++adds;
        } else if (!line.empty() && line[0] == '-') {
            ++dels;
        }

        if (end >= diff.size()) {
            break;
        }
        start = end + 1;
    }

    if (adds == 0 && dels == 0) {
        return parse_diff(diff, muted, add, del);
    }

    const FileTypeBadge badge = file_type_badge(file_path);
    BodyLine header;
    header.kind = BodyLineKind::DiffHeader;
    header.bar = false;
    header.text = path_basename(file_path.empty() ? path_hint : file_path);
    if (header.text.empty()) {
        header.text = "(file)";
    }
    header.diff_icon = badge.label;
    header.diff_icon_rgb = badge.color;
    header.diff_adds = adds;
    header.diff_dels = dels;
    lines.push_back(std::move(header));

    const bool new_file_hunk = dels == 0 && adds > 0;
    if (ui_file_diff_layout() == FileDiffLayout::SideBySide) {
        ActivityLog::append_file_diff_side_by_side_body(diff, lines, muted, add, del, new_file_hunk);
        if (file_diff_view != nullptr) {
            *file_diff_view = true;
        }
        return lines;
    }

    int old_line = 0;
    int new_line = 0;
    bool in_hunk = false;

    start = 0;
    while (start <= diff.size()) {
        std::size_t end = diff.find('\n', start);
        if (end == std::string::npos) {
            end = diff.size();
        }
        std::string line = diff.substr(start, end - start);

        if (line.rfind("--- ", 0) == 0 || line.rfind("+++ ", 0) == 0) {
            if (end >= diff.size()) {
                break;
            }
            start = end + 1;
            continue;
        }
        if (line.rfind("@@", 0) == 0) {
            int parsed_old = 0;
            int parsed_new = 0;
            if (parse_hunk_header(line, parsed_old, parsed_new)) {
                old_line = parsed_old > 0 ? parsed_old : 0;
                new_line = parsed_new;
                in_hunk = true;
            }
            if (end >= diff.size()) {
                break;
            }
            start = end + 1;
            continue;
        }
        if (line.rfind("...", 0) == 0) {
            BodyLine more;
            more.bar = false;
            more.text = line;
            more.style = muted;
            more.style.italic = true;
            lines.push_back(std::move(more));
            break;
        }

        if (!in_hunk || line.empty()) {
            if (end >= diff.size()) {
                break;
            }
            start = end + 1;
            continue;
        }

        tuinator::Style style = muted;
        style.background_rgb = ui_main_bg();
        bool fill = false;
        char prefix = line[0];
        std::string content = line.substr(1);
        int show_old = 0;
        int show_new = 0;

        if (prefix == '+') {
            style = add;
            fill = true;
            show_new = new_line;
            if (new_line > 0) {
                ++new_line;
            }
        } else if (prefix == '-') {
            style = del;
            fill = true;
            show_old = old_line;
            if (old_line > 0) {
                ++old_line;
            }
        } else if (prefix == ' ') {
            style = muted;
            style.dim = true;
            fill = true;
            show_old = old_line;
            show_new = new_line;
            if (old_line > 0) {
                ++old_line;
            }
            if (new_line > 0) {
                ++new_line;
            }
        } else {
            if (end >= diff.size()) {
                break;
            }
            start = end + 1;
            continue;
        }

        if (fill) {
            const std::size_t max_code_lines = new_file_hunk ? 200 : kMaxDiffCodeLines;
            if (code_lines >= max_code_lines && (prefix == '+' || prefix == '-')) {
                BodyLine more;
                more.bar = false;
                more.text = "… (diff truncated in UI)";
                more.style = muted;
                more.style.italic = true;
                lines.push_back(std::move(more));
                break;
            }

            std::string next_content;
            bool paired_replace = false;
            if (prefix == '-' && end < diff.size()) {
                const std::size_t next_start = end + 1;
                std::size_t next_end = diff.find('\n', next_start);
                if (next_end == std::string::npos) {
                    next_end = diff.size();
                }
                const std::string next_line = diff.substr(next_start, next_end - next_start);
                if (!next_line.empty() && next_line[0] == '+') {
                    next_content = next_line.substr(1);
                    paired_replace = true;
                    end = next_end;
                }
            }

            BodyLine row;
            row.text = format_diff_gutter(show_old, show_new) + content;
            row.style = style;
            row.bar = false;
            row.diff_fill_row = true;

            if (prefix == '-' && paired_replace) {
                std::vector<ActivityDiffTextSegment> old_segments;
                std::vector<ActivityDiffTextSegment> new_segments;
                build_inline_diff_segments(content, next_content, old_segments, new_segments);
                row.text = format_diff_gutter(show_old, 0) + content;
                row.diff_segments = std::move(old_segments);
                lines.push_back(std::move(row));

                const int add_line_no = new_line;
                if (new_line > 0) {
                    ++new_line;
                }
                BodyLine add_row;
                add_row.text = format_diff_gutter(0, add_line_no) + next_content;
                add_row.style = add;
                add_row.bar = false;
                add_row.diff_fill_row = true;
                add_row.diff_segments = std::move(new_segments);
                lines.push_back(std::move(add_row));
                code_lines += 2;
            } else {
                if (prefix == '-' || prefix == '+') {
                    append_diff_segment(row.diff_segments, content, true);
                }
                lines.push_back(std::move(row));
                if (prefix == '+' || prefix == '-') {
                    ++code_lines;
                }
            }
        }

        if (end >= diff.size()) {
            break;
        }
        start = end + 1;
    }

    if (file_diff_view != nullptr) {
        *file_diff_view = true;
    }
    return lines;
}

void ActivityLog::finish_question(const std::string& args, const std::string& text, int64_t duration_ms) {
    end_thinking();
    Entry entry;
    entry.scope = Scope::Question;
    entry.question_header = question_header_from_args(args);
    entry.body = question_answer_summary(text);
    entry.args = args;
    entry.expanded = false;
    entry.spacer_after = true;
    entry.failed = tool_observation_failed(text);
    if (duration_ms >= 0) {
        entry.duration_ms = duration_ms;
        entry.has_duration = true;
    }
    entries_.push_back(std::move(entry));
    mark_dirty();
    mark_layout_dirty();
}

void ActivityLog::finish_tool(const std::string& tool_name, const std::string& text, const std::string& diff,
                              int64_t duration_ms, const std::string& skill_name, bool is_decision,
                              const std::string& decision_driver, bool is_ask_user) {
    std::string diff_payload = diff;
    if (is_ask_user || tool_name == "ask_user") {
        return;
    }
    std::size_t entry_index = entries_.size();
    bool have_entry = false;
    for (std::size_t i = entries_.size(); i-- > 0;) {
        Entry& candidate = entries_[i];
        if (candidate.running && candidate.scope != Scope::Thinking && candidate.name == tool_name) {
            entry_index = i;
            have_entry = true;
            break;
        }
    }
    if (!have_entry) {
        for (std::size_t i = entries_.size(); i-- > 0;) {
            Entry& candidate = entries_[i];
            if (candidate.running && candidate.scope != Scope::Thinking) {
                entry_index = i;
                have_entry = true;
                break;
            }
        }
    }
    if (diff_payload.empty() && is_file_edit_tool(tool_name)) {
        diff_payload = unified_diff_embedded_in_text(text);
    }
    if (diff_payload.empty() && tool_name == "write_file" && have_entry) {
        diff_payload = build_unified_diff_new_file(json_string_field(entries_[entry_index].args, "path"),
                                                   json_string_field(entries_[entry_index].args, "content"));
    }

    if (!have_entry) {
        Entry fallback;
        if (is_decision || tool_name == "decide") {
            fallback.scope = Scope::Decision;
            fallback.expanded = true;
        } else if (tool_name == "invoke_skill") {
            fallback.scope = Scope::Skill;
        } else {
            fallback.scope = Scope::Tool;
        }
        fallback.name = tool_name;
        fallback.started_at = std::chrono::steady_clock::now();
        entries_.push_back(std::move(fallback));
        entry_index = entries_.size() - 1;
    }

    Entry& entry = entries_[entry_index];
    invalidate_body_cache(entry);

    if (entry.scope == Scope::Decision || is_decision || tool_name == "decide") {
        entry.scope = Scope::Decision;
        entry.expanded = true;
    }
    if (!decision_driver.empty()) {
        entry.decision_driver = decision_driver;
    }
    if (!skill_name.empty() && !is_invoke_skill_tool_name(skill_name)) {
        entry.skill_name = skill_name;
    }
    if (tool_name.rfind("graph_", 0) == 0) {
        std::string body = "╭─ ⚡ STATE MUTATION ─────────────────\n";
        for (const std::string& line : split_lines(text)) {
            if (line.empty()) {
                continue;
            }
            body += "│ " + line + "\n";
        }
        body += "╰────────────────────────────────────";
        entry.body = body;
        entry.expanded = true;
    } else {
        entry.body = text;
    }
    if (!diff_payload.empty() && is_file_edit_tool(tool_name)) {
        entry.body = strip_embedded_unified_diff(entry.body);
    }
    entry.failed = tool_observation_failed(text);
    if (entry.scope == Scope::Skill && entry.skill_name.empty()) {
        entry.skill_name = skill_name_from_result_text(text);
    }
    if (entry.scope == Scope::Skill && entry.skill_name.empty()) {
        entry.skill_name = skill_name_from_args(entry.args);
    }
    if (entry.scope == Scope::Skill && is_invoke_skill_tool_name(entry.skill_name)) {
        entry.skill_name.clear();
    }
    if (entry.scope == Scope::Skill && !entry.skill_name.empty()) {
        mark_dirty();
        mark_layout_dirty();
    }

    Entry diff_entry;
    bool push_diff_entry = false;
    if (!diff_payload.empty()) {
        const std::string path_hint = json_string_field(entry.args, "path");
        if (is_file_edit_tool(tool_name)) {
            bool file_view = false;
            std::vector<BodyLine> card =
                parse_file_diff_view(diff_payload, path_hint, muted_style_, add_style_, del_style_, &file_view);
            if (file_view && !card.empty()) {
                entry.body.clear();
                entry.expanded = false;
                entry.diff.clear();
                entry.file_diff_view = false;
                diff_entry.scope = Scope::FileDiff;
                diff_entry.diff = std::move(card);
                diff_entry.diff_source = diff_payload;
                diff_entry.diff_path_hint = path_hint;
                diff_entry.file_diff_view = true;
                diff_entry.failed = entry.failed;
                diff_entry.spacer_after = true;
                push_diff_entry = true;
            } else {
                entry.diff = std::move(card);
                entry.expanded = true;
            }
        } else {
            entry.diff = parse_diff(diff_payload, muted_style_, add_style_, del_style_);
        }
    }
    if (duration_ms >= 0) {
        entry.duration_ms = duration_ms;
        entry.has_duration = true;
    }
    finalize_entry(entry);
    if (push_diff_entry) {
        entries_.push_back(std::move(diff_entry));
        mark_dirty();
        mark_layout_dirty();
    }
}

bool ActivityLog::has_running() const {
    for (const Entry& entry : entries_) {
        if (entry.running) {
            return true;
        }
    }
    return false;
}

bool ActivityLog::has_live_spinner() const {
    return has_running() || generating_tail_;
}

std::string ActivityLog::running_label() const {
    for (const Entry& entry : entries_) {
        if (!entry.running) {
            continue;
        }
        if (entry.scope == Scope::Thinking) {
            return "Thinking…";
        }
        return entry_title(entry);
    }
    return "Loading model…";
}

bool ActivityLog::needs_animation_tick() const {
    if (has_running() || generating_tail_) {
        return true;
    }
    for (const Entry& entry : entries_) {
        if (entry.settle_ms > 0 || entry.flash_ms > 0) {
            return true;
        }
    }
    return false;
}

void ActivityLog::tick(int dt_ms) {
    if (!needs_animation_tick()) {
        return;
    }
    spinner_ms_ = (spinner_ms_ + dt_ms) % 60000;
    bool changed = has_running() || generating_tail_;
    for (Entry& entry : entries_) {
        if (entry.settle_ms > 0) {
            entry.settle_ms = std::max(0, entry.settle_ms - dt_ms);
            changed = true;
        }
        if (entry.flash_ms > 0) {
            entry.flash_ms = std::max(0, entry.flash_ms - dt_ms);
            changed = true;
        }
    }
    if (changed) {
        mark_dirty();
    }
}

const char* ActivityLog::spinner_frame() const {
    const int idx = (spinner_ms_ / kSpinnerPeriodMs) % kSpinnerFrameCount;
    return kSpinnerFrames[idx];
}

void ActivityLog::set_wrap_width(int width) {
    width = std::max(0, width);
    if (wrap_width_hint_ == width) {
        return;
    }
    wrap_width_hint_ = width;
    invalidate_all_body_caches();
    mark_layout_dirty();
}

void ActivityLog::set_taiji_handle(uint64_t handle) {
    taiji_handle_ = handle;
}

std::string ActivityLog::tool_display_title(const std::string& tool_name) const {
    if (tool_name.rfind("graph_", 0) == 0) {
        return "⚡ " + tool_name;
    }
    if (taiji_handle_ == 0 || tool_name.empty()) {
        return {};
    }
    char* raw = taiji_tool_display_title(taiji_handle_, const_cast<char*>(tool_name.c_str()));
    if (raw == nullptr) {
        return {};
    }
    std::string title(raw);
    std::free(raw);
    return title;
}

int ActivityLog::wrap_cols() const {
    if (bounds_.width > 0) {
        return std::max(8, bounds_.width);
    }
    if (wrap_width_hint_ > 0) {
        return std::max(8, wrap_width_hint_);
    }
    return 72;
}

int ActivityLog::body_wrap_cols() const {
    return std::max(8, wrap_cols() - kBodyIndent);
}

std::string ActivityLog::entry_title(const Entry& entry) const {
    if (entry.scope == Scope::MCP) {
        std::string text = scope_label(entry.scope);
        if (!entry.server.empty()) {
            text += " · " + entry.server;
        }
        if (!entry.name.empty()) {
            if (const std::string label = tool_display_title(entry.name); !label.empty()) {
                text += " · " + label;
            } else {
                text += " · " + entry.name;
            }
        }
        return text;
    }
    if (entry.scope == Scope::Skill) {
        std::string text = scope_label(entry.scope);
        std::string skill = entry.skill_name;
        if (skill.empty()) {
            skill = skill_name_from_args(entry.args);
        }
        if (!skill.empty() && !is_invoke_skill_tool_name(skill)) {
            text += " · " + skill;
        }
        return text;
    }
    if (entry.scope == Scope::Decision) {
        std::string text = scope_label(entry.scope);
        if (!entry.decision_driver.empty()) {
            text += " · " + entry.decision_driver;
        } else if (!entry.name.empty() && entry.name != "decide") {
            text += " · " + entry.name;
        }
        return text;
    }
    if (entry.scope == Scope::Question) {
        const std::string q = scope_label(entry.scope);
        if (!entry.question_header.empty()) {
            return q + " · " + entry.question_header;
        }
        return q;
    }
    if (entry.scope == Scope::Tool && !entry.name.empty()) {
        if (const std::string label = tool_display_title(entry.name); !label.empty()) {
            if (entry.name == "skill_search" && !entry.skill_name.empty()) {
                return label + " · " + entry.skill_name;
            }
            return label;
        }
    }
    std::string text = scope_label(entry.scope);
    if (!entry.name.empty()) {
        text += " · " + entry.name;
    }
    return text;
}

bool ActivityLog::entry_has_expandable_body(const Entry& entry) const {
    if (entry.scope == Scope::Thinking || entry.scope == Scope::Decision || entry.scope == Scope::Question) {
        return true;
    }
    if (!entry.diff.empty()) {
        return true;
    }
    const bool compact_file_diff = entry.file_diff_view && !entry.diff.empty();
    if (compact_file_diff && has_non_blank_text(entry.body)) {
        return true;
    }
    if (ui_tool_expand_show_parameters() && has_non_blank_text(entry.args)) {
        return true;
    }
    if (ui_tool_expand_show_result() && has_non_blank_text(entry.body)) {
        return true;
    }
    return false;
}

bool ActivityLog::entry_header_expandable(const Entry& entry) const {
    if (entry.scope == Scope::FileDiff || entry.running) {
        return false;
    }
    return entry_has_expandable_body(entry);
}

int ActivityLog::header_entry_index_at_local_y(int local_y) const {
    int header_index = -1;
    y_for_layout_header_line(local_y, &header_index, nullptr);
    if (header_index < 0 || header_index >= static_cast<int>(entries_.size())) {
        return -1;
    }
    const Entry& entry = entries_[static_cast<std::size_t>(header_index)];
    if (local_y < 0 || entry.scope == Scope::FileDiff || !entry_header_expandable(entry)) {
        return -1;
    }
    return header_index;
}

void ActivityLog::paint_header_expand_affordance(tuinator::PaintContext& ctx, int y, const Entry& entry,
                                                 std::size_t entry_index, tuinator::Rgb row_bg, int chevron_x) const {
    if (!entry_header_expandable(entry)) {
        return;
    }
    const bool hovered = static_cast<int>(entry_index) == hovered_entry_index_;
    if (!entry.expanded && !hovered) {
        return;
    }
    const char* glyph = entry.expanded ? kNfChevronDown : kNfChevronRight;
    const int x = chevron_x;
    tuinator::Style style = activity_on_panel(muted_style_);
    style.background_rgb = row_bg;
    if (entry.expanded) {
        style.foreground_rgb = ui_section_heading();
        style.bold = true;
    } else {
        style.foreground_rgb = ui_secondary_text();
        style.dim = false;
    }
    paint_text_with_background(ctx, x, y, glyph, style);
}

std::string ActivityLog::header_text(const Entry& entry) const {
    if (entry.scope == Scope::Thinking) {
        if (entry.running) {
            return "Thinking… · " + format_duration(elapsed_ms(entry));
        }
        if (entry.has_duration) {
            return "Thought for " + format_duration(entry.duration_ms);
        }
        return "Thought";
    }

    std::string text = entry_title(entry);
    if (entry.scope != Scope::Skill && entry.scope != Scope::Decision && entry.scope != Scope::Question &&
        (entry.running || entry.has_duration)) {
        text += " · " + format_duration(elapsed_ms(entry));
    }
    if ((entry.scope == Scope::Decision || entry.scope == Scope::Question) &&
        (entry.running || entry.has_duration)) {
        text += " · " + format_duration(elapsed_ms(entry));
    }
    return text;
}

std::vector<ActivityLog::BodyLine> ActivityLog::thinking_body_lines(const Entry& entry) const {
    std::vector<BodyLine> lines;
    const std::string trimmed = [&]() {
        const auto begin = entry.body.find_first_not_of(" \t\n\r");
        if (begin == std::string::npos) {
            return std::string{};
        }
        const auto end = entry.body.find_last_not_of(" \t\n\r");
        return entry.body.substr(begin, end - begin + 1);
    }();
    if (trimmed.empty()) {
        lines.push_back(BodyLine{.text = entry.running ? "(thinking…)" : "(no content)",
                                 .style = muted_style_, .bar = true});
        return lines;
    }

    auto wrapped = wrap_text(trimmed, body_wrap_cols());
    std::size_t omitted = 0;
    if (wrapped.size() > kMaxThinkingLines) {
        omitted = wrapped.size() - kMaxThinkingLines;
        wrapped.resize(kMaxThinkingLines);
    }
    for (auto& line : wrapped) {
        lines.push_back(BodyLine{.text = std::move(line), .style = thinking_body_style_, .bar = true});
    }
    if (omitted > 0) {
        lines.push_back(BodyLine{.text = "… (" + std::to_string(omitted) + " more lines)",
                                 .style = muted_style_, .bar = true});
    }
    return lines;
}

std::vector<ActivityLog::BodyLine> ActivityLog::decision_body_lines(const Entry& entry) const {
    std::vector<BodyLine> lines;
    const tuinator::Style& text_style = entry.failed ? failed_body_style_ : body_style_;
    const tuinator::Style& title_style = entry.failed ? failed_style_ : heading_style_;
    const tuinator::Style& hint_style = entry.failed ? failed_body_style_ : muted_style_;

    lines.push_back(BodyLine{.text = "Input", .style = title_style, .bar = true});

    const std::string state = decision_state_from_args(entry.args);
    if (!state.empty()) {
        auto wrapped = wrap_text("state: " + state, body_wrap_cols());
        for (auto& line : wrapped) {
            lines.push_back(BodyLine{.text = std::move(line), .style = text_style, .bar = true});
        }
    }

    auto params = pretty_args_lines(entry.args);
    std::size_t omitted_params = 0;
    if (params.size() > kMaxParamLines) {
        omitted_params = params.size() - kMaxParamLines;
        params.resize(kMaxParamLines);
    }
    for (auto& line : params) {
        if (!state.empty() && line.find("\"state\"") != std::string::npos) {
            continue;
        }
        if (!state.empty() && line.find("\"text\"") != std::string::npos) {
            continue;
        }
        lines.push_back(BodyLine{.text = std::move(line), .style = text_style, .bar = true});
    }
    if (omitted_params > 0) {
        lines.push_back(BodyLine{.text = "… (" + std::to_string(omitted_params) + " more lines)",
                                 .style = hint_style, .bar = true});
    }

    if (!entry.body.empty()) {
        auto result = wrap_text(entry.body, body_wrap_cols());
        std::size_t omitted = 0;
        if (result.size() > kMaxResultLines) {
            omitted = result.size() - kMaxResultLines;
            result.resize(kMaxResultLines);
        }
        lines.push_back(BodyLine{.text = "Result", .style = title_style, .bar = true});
        for (auto& line : result) {
            lines.push_back(BodyLine{.text = std::move(line), .style = text_style, .bar = true});
        }
        if (omitted > 0) {
            lines.push_back(BodyLine{.text = "… (" + std::to_string(omitted) + " more lines)",
                                     .style = hint_style, .bar = true});
        }
    }

    if (lines.size() <= 1) {
        lines.push_back(BodyLine{.text = entry.running ? "(deciding…)" : "(no input)",
                                 .style = hint_style, .bar = true});
    }
    return lines;
}

std::vector<ActivityLog::BodyLine> ActivityLog::tool_body_lines(const Entry& entry) const {
    std::vector<BodyLine> lines;
    const tuinator::Style& text_style = entry.failed ? failed_body_style_ : body_style_;
    const tuinator::Style& title_style = entry.failed ? failed_style_ : heading_style_;
    const tuinator::Style& hint_style = entry.failed ? failed_body_style_ : muted_style_;

    const bool compact_file_diff = entry.file_diff_view && !entry.diff.empty();
    if (!compact_file_diff) {
        if (ui_tool_expand_show_parameters() && has_non_blank_text(entry.args)) {
            auto params = tool_args_lines_for_display(entry.args, body_wrap_cols());
            std::size_t omitted_params = 0;
            if (params.size() > kMaxParamLines) {
                omitted_params = params.size() - kMaxParamLines;
                params.resize(kMaxParamLines);
            }
            if (!params.empty()) {
                lines.push_back(BodyLine{.text = "Parameters", .style = title_style, .bar = true});
                for (auto& line : params) {
                    lines.push_back(BodyLine{.text = std::move(line), .style = text_style, .bar = true});
                }
                if (omitted_params > 0) {
                    lines.push_back(BodyLine{.text = "… (" + std::to_string(omitted_params) + " more lines)",
                                             .style = hint_style, .bar = true});
                }
            }
        }

        if (ui_tool_expand_show_result() && has_non_blank_text(entry.body)) {
            auto result = tool_result_lines_for_display(entry.body, body_wrap_cols());
            std::size_t omitted = 0;
            if (result.size() > kMaxResultLines) {
                omitted = result.size() - kMaxResultLines;
                result.resize(kMaxResultLines);
            }
            lines.push_back(BodyLine{.text = "Result", .style = title_style, .bar = true});
            for (auto& line : result) {
                lines.push_back(BodyLine{.text = std::move(line), .style = text_style, .bar = true});
            }
            if (omitted > 0) {
                lines.push_back(BodyLine{.text = "… (" + std::to_string(omitted) + " more lines)",
                                         .style = hint_style, .bar = true});
            }
        }
    } else if (!entry.body.empty()) {
        for (auto& line : wrap_text(entry.body, body_wrap_cols())) {
            lines.push_back(BodyLine{.text = std::move(line), .style = hint_style, .bar = true});
        }
    }

    for (auto line : entry.diff) {
        if (entry.failed) {
            line.style = failed_body_style_;
        }
        lines.push_back(std::move(line));
    }

    if (lines.empty()) {
        lines.push_back(BodyLine{.text = entry.running ? "(running…)" : "(no output)",
                                 .style = hint_style, .bar = true});
    }
    return lines;
}

std::vector<ActivityLog::BodyLine> ActivityLog::question_body_lines(const Entry& entry) const {
    std::vector<BodyLine> lines;
    const tuinator::Style& text_style = entry.failed ? failed_body_style_ : body_style_;
    const tuinator::Style& hint_style = entry.failed ? failed_body_style_ : muted_style_;
    if (!entry.body.empty()) {
        for (auto& line : wrap_text(entry.body, body_wrap_cols())) {
            lines.push_back(BodyLine{.text = std::move(line), .style = text_style, .bar = true});
        }
    }
    if (lines.empty()) {
        lines.push_back(BodyLine{.text = entry.failed ? "(dismissed)" : "(answered)",
                                 .style = hint_style, .bar = true});
    }
    return lines;
}

std::vector<ActivityLog::BodyLine> ActivityLog::file_diff_body_lines(const Entry& entry) const {
    return entry.diff;
}

std::vector<ActivityLog::BodyLine> ActivityLog::body_lines(const Entry& entry) const {
    const int wrap = body_wrap_cols();
    if (entry.body_cache_valid && entry.body_cache_wrap_cols == wrap) {
        return entry.body_cache;
    }
    Entry& cache_entry = const_cast<Entry&>(entry);
    std::vector<BodyLine> lines;
    if (entry.scope == Scope::Thinking) {
        lines = thinking_body_lines(entry);
    } else if (entry.scope == Scope::Decision) {
        lines = decision_body_lines(entry);
    } else if (entry.scope == Scope::Question) {
        lines = question_body_lines(entry);
    } else if (entry.scope == Scope::FileDiff) {
        lines = file_diff_body_lines(entry);
    } else {
        lines = tool_body_lines(entry);
    }
    cache_entry.body_cache = std::move(lines);
    cache_entry.body_cache_valid = true;
    cache_entry.body_cache_wrap_cols = wrap;
    return cache_entry.body_cache;
}

int ActivityLog::entry_height(const Entry& entry) const {
    if (entry.scope == Scope::FileDiff) {
        return static_cast<int>(entry.diff.size());
    }
    int height = 1;
    if (entry.expanded) {
        height += static_cast<int>(body_lines(entry).size());
    }
    if (entry.spacer_after) {
        ++height;
    }
    return height;
}

bool ActivityLog::has_visible_layout_rows() const { return !build_layout().empty(); }

bool ActivityLog::should_show_pipeline_title() const {
    if (trace_mode_ == ActivityTraceMode::Ignore) {
        return false;
    }
    if (!ui_activity_show_pipeline_title()) {
        return false;
    }
    return !pipeline_title_.empty() && !entries_.empty() && has_visible_layout_rows();
}

bool ActivityLog::should_show_generating_tail() const {
    if (!generating_tail_ || trace_mode_ == ActivityTraceMode::Ignore) {
        return false;
    }
    return has_visible_layout_rows();
}

int ActivityLog::total_lines() const {
    int total = 0;
    if (should_show_pipeline_title()) {
        ++total;
    }
    const std::vector<LayoutRow> layout = build_layout();
    for (std::size_t i = 0; i < layout.size(); ++i) {
        total += layout_row_height(layout[i], layout, i);
        total += layout_row_gap_;
    }
    if (should_show_generating_tail()) {
        ++total;
    }
    return total;
}

tuinator::Size ActivityLog::preferred_size() const {
    if (total_lines() <= 0) {
        return {0, 0};
    }
    const int width =
        std::max({wrap_cols(), chat_stream_content_width(bounds_.width), chat_stream_row_preferred_width(bounds_.width)});
    int max_line_width = 0;
    if (!pipeline_title_.empty()) {
        max_line_width = std::max(max_line_width, tuinator::text_display_width(pipeline_title_) + kTreeTextCol + 2);
    }
    for (const LayoutRow& row : build_layout()) {
        if (row.kind == LayoutRowKind::MundaneCluster) {
            max_line_width = std::max(
                max_line_width,
                tuinator::text_display_width(mundane_cluster_summary(row.mundane_begin, row.mundane_end)) + kTreeTextCol);
            if (!mundane_cluster_is_open(row.mundane_begin)) {
                continue;
            }
            for (std::size_t i = row.mundane_begin; i < row.mundane_end; ++i) {
                const Entry& entry = entries_[i];
                max_line_width =
                    std::max(max_line_width, tuinator::text_display_width(header_text(entry)) + kTreeTextCol + 2);
            }
            continue;
        }
        const Entry& entry = entries_[row.entry_index];
        if (entry.scope == Scope::FileDiff) {
            const ChatStreamLayout stream = chat_stream_layout_in_widget(bounds_.width);
            max_line_width =
                std::max(max_line_width, stream.offset_x + diff_card_width_cols(stream) + kChatStreamPadX);
            continue;
        }
        max_line_width = std::max(max_line_width, tuinator::text_display_width(header_text(entry)) + kTreeTextCol);
        if (entry.expanded) {
            for (const auto& line : body_lines(entry)) {
                max_line_width = std::max(max_line_width, tuinator::text_display_width(line.text) + kBodyIndent);
            }
        }
    }
    return {width, total_lines()};
}

bool ActivityLog::update_header_hover_at_content_point(const tuinator::Point& content_point) {
    int next_hover = -1;
    if (bounds_.width > 0 && bounds_.height > 0 && bounds_.contains(content_point)) {
        const int local_y = content_point.y - bounds_.y;
        next_hover = header_entry_index_at_local_y(local_y);
    }
    if (next_hover == hovered_entry_index_) {
        return false;
    }
    hovered_entry_index_ = next_hover;
    mark_dirty();
    return true;
}

bool ActivityLog::clear_header_hover() {
    if (hovered_entry_index_ == -1) {
        return false;
    }
    hovered_entry_index_ = -1;
    mark_dirty();
    return true;
}

void ActivityLog::set_trace_mode(ActivityTraceMode mode) {
    if (trace_mode_ == mode) {
        return;
    }
    trace_mode_ = mode;
    mundane_cluster_open_.clear();
    mark_dirty();
    mark_layout_dirty();
}

void ActivityLog::set_file_diff_layout(FileDiffLayout layout) {
    if (file_diff_layout_ == layout) {
        return;
    }
    file_diff_layout_ = layout;
    rebuild_file_diff_entries();
}

void ActivityLog::set_layout_row_gap(int lines) {
    const int clamped = std::clamp(lines, 0, 4);
    if (layout_row_gap_ == clamped) {
        return;
    }
    layout_row_gap_ = clamped;
    mark_dirty();
    mark_layout_dirty();
}

void ActivityLog::paint_layout_row_gap(tuinator::PaintContext& ctx, int& y, int stream_x, bool continue_trunk) const {
    for (int i = 0; i < layout_row_gap_ && y < bounds_.height; ++i) {
        paint_activity_row_bg(ctx, y, bounds_.width);
        if (continue_trunk) {
            paint_activity_tree_trunk_continuation(ctx, stream_x, y);
        }
        ++y;
    }
}

void ActivityLog::rebuild_file_diff_entries() {
    const FileDiffLayout previous = ui_file_diff_layout();
    ui_set_file_diff_layout(file_diff_layout_);
    bool changed = false;
    for (Entry& entry : entries_) {
        if (entry.scope != Scope::FileDiff || entry.diff_source.empty()) {
            continue;
        }
        bool file_view = false;
        entry.diff =
            parse_file_diff_view(entry.diff_source, entry.diff_path_hint, muted_style_, add_style_, del_style_, &file_view);
        entry.file_diff_view = file_view;
        changed = true;
    }
    ui_set_file_diff_layout(previous);
    if (changed) {
        mark_dirty();
        mark_layout_dirty();
    }
}

bool ActivityLog::entry_followed_by_file_diff(std::size_t entry_index) const {
    if (entry_index + 1 >= entries_.size()) {
        return false;
    }
    return entries_[entry_index + 1].scope == Scope::FileDiff;
}

bool ActivityLog::entry_visible_when_ignoring_tool_rows(std::size_t entry_index) const {
    if (entry_index >= entries_.size()) {
        return false;
    }
    const Entry& entry = entries_[entry_index];
    if (entry.scope == Scope::FileDiff) {
        return true;
    }
    switch (entry.scope) {
    case Scope::Thinking:
    case Scope::Question:
    case Scope::Decision:
    case Scope::Skill:
        return true;
    case Scope::Tool:
    case Scope::MCP:
        break;
    }
    if (is_file_edit_tool(entry.name) && entry_followed_by_file_diff(entry_index)) {
        return false;
    }
    if (entry.scope == Scope::Tool || entry.scope == Scope::MCP) {
        return false;
    }
    return true;
}

bool ActivityLog::entry_is_prominent(const Entry& entry) const {
    switch (entry.scope) {
    case Scope::FileDiff:
    case Scope::Question:
    case Scope::Decision:
    case Scope::Skill:
    case Scope::Thinking:
        return true;
    case Scope::Tool:
    case Scope::MCP:
        break;
    }
    if (entry.file_diff_view) {
        return true;
    }
    if (is_file_edit_tool(entry.name)) {
        return true;
    }
    return false;
}

bool ActivityLog::mundane_cluster_is_open(std::size_t run_begin) const {
    const auto it = mundane_cluster_open_.find(static_cast<int>(run_begin));
    if (it != mundane_cluster_open_.end()) {
        return it->second;
    }
    return trace_mode_ == ActivityTraceMode::Subtree;
}

void ActivityLog::set_mundane_cluster_open(std::size_t run_begin, bool open) {
    mundane_cluster_open_[static_cast<int>(run_begin)] = open;
    mark_dirty();
    mark_layout_dirty();
}

std::vector<ActivityLog::LayoutRow> ActivityLog::build_layout() const {
    std::vector<LayoutRow> rows;
    std::size_t i = 0;
    while (i < entries_.size()) {
        if (entry_is_prominent(entries_[i])) {
            if (trace_mode_ == ActivityTraceMode::Ignore && !entry_visible_when_ignoring_tool_rows(i)) {
                ++i;
                continue;
            }
            rows.push_back(LayoutRow{.kind = LayoutRowKind::Entry, .entry_index = i});
            ++i;
            continue;
        }
        if (trace_mode_ == ActivityTraceMode::Inline || trace_mode_ == ActivityTraceMode::Ignore) {
            ++i;
            continue;
        }
        if (trace_mode_ == ActivityTraceMode::Subtree) {
            rows.push_back(LayoutRow{.kind = LayoutRowKind::Entry, .entry_index = i});
            ++i;
            continue;
        }
        std::size_t j = i;
        while (j < entries_.size() && !entry_is_prominent(entries_[j])) {
            ++j;
        }
        rows.push_back(
            LayoutRow{.kind = LayoutRowKind::MundaneCluster, .mundane_begin = i, .mundane_end = j});
        i = j;
    }
    return rows;
}

int ActivityLog::layout_row_height(const LayoutRow& row, const std::vector<LayoutRow>& layout,
                                   std::size_t layout_index) const {
    if (row.kind == LayoutRowKind::Entry) {
        return entry_height(entries_[row.entry_index]);
    }
    const bool detached_root = layout_follows_file_diff(layout, layout_index);
    const bool open = mundane_cluster_is_open(row.mundane_begin);
    const std::size_t mundane_count = row.mundane_end - row.mundane_begin;
    const bool skip_header_for_single_open = detached_root && open && mundane_count == 1;

    int height = skip_header_for_single_open ? 0 : 1;
    if (open) {
        for (std::size_t i = row.mundane_begin; i < row.mundane_end; ++i) {
            height += entry_height(entries_[i]);
        }
    }
    return height;
}

std::string ActivityLog::mundane_cluster_summary(std::size_t begin, std::size_t end) const {
    int files = 0;
    int searches = 0;
    int lists = 0;
    int other = 0;
    for (std::size_t i = begin; i < end; ++i) {
        const std::string& name = entries_[i].name;
        if (name == "read_file" || name == "read_mmap") {
            ++files;
        } else if (name == "grep" || name == "workspace_find" || name == "rag_search" || name == "find") {
            ++searches;
        } else if (name == "glob" || name == "list_dir" || name == "outline" || name == "globwalk") {
            ++lists;
        } else {
            ++other;
        }
    }
    std::vector<std::string> parts;
    if (files > 0) {
        parts.push_back(std::to_string(files) + " " + (files == 1 ? "file" : "files"));
    }
    if (searches > 0) {
        parts.push_back(std::to_string(searches) + (searches == 1 ? " search" : " searches"));
    }
    if (lists > 0) {
        parts.push_back(std::to_string(lists) + (lists == 1 ? " listing" : " listings"));
    }
    if (other > 0) {
        parts.push_back(std::to_string(other) + (other == 1 ? " step" : " steps"));
    }
    if (parts.empty()) {
        return std::to_string(end - begin) + " steps";
    }
    std::string joined;
    for (std::size_t p = 0; p < parts.size(); ++p) {
        if (p > 0) {
            joined += ", ";
        }
        joined += parts[p];
    }
    if (trace_mode_ == ActivityTraceMode::Clustered) {
        const std::string prefix = ui_shell("activity.trace.explored_prefix");
        if (!prefix.empty()) {
            return prefix + joined;
        }
        return "Explored " + joined;
    }
    return joined;
}

bool ActivityLog::layout_follows_file_diff(const std::vector<LayoutRow>& layout, std::size_t index) const {
    if (index == 0) {
        return false;
    }
    const LayoutRow& prev = layout[index - 1];
    if (prev.kind != LayoutRowKind::Entry) {
        return false;
    }
    return entries_[prev.entry_index].scope == Scope::FileDiff;
}

void ActivityLog::paint_mundane_cluster_header(tuinator::PaintContext& ctx, int y, std::size_t run_begin,
                                               std::size_t run_end, const ChatStreamLayout& stream,
                                               tuinator::Rgb row_bg, bool open, bool detached_root) const {
    const int stream_x = stream.offset_x;
    paint_activity_row_bg(ctx, y, bounds_.width, row_bg);
    if (detached_root) {
        paint_activity_tree_trunk_clear(ctx, stream_x, y);
    }
    tuinator::Style branch = muted_style_;
    branch.background_rgb = row_bg;
    tuinator::Style chevron_style = activity_on_panel(muted_style_);
    chevron_style.background_rgb = row_bg;
    chevron_style.foreground_rgb = ui_secondary_text();
    const char* chevron = open ? kNfChevronDown : kNfChevronRight;
    const char* branch_glyph = detached_root ? "╭─" : "├─";
    paint_activity_tree_prefix(ctx, stream_x, y, branch_glyph, chevron, branch, chevron_style, row_bg);
    const std::string label = mundane_cluster_summary(run_begin, run_end);
    tuinator::Style label_style = activity_on_panel(muted_style_);
    label_style.background_rgb = row_bg;
    label_style.foreground_rgb = ui_secondary_text();
    paint_text_with_background(ctx, stream_x + kTreeTextCol, y, label, label_style);
}

void ActivityLog::paint_entry_block(tuinator::PaintContext& ctx, int y, std::size_t entry_index,
                                    const ChatStreamLayout& stream, tuinator::Rgb row_bg, int extra_tree_indent,
                                    const char* branch_override, bool more_layout_rows_after) const {
    const Entry& entry = entries_[entry_index];
    const int stream_x = stream.offset_x + extra_tree_indent;
    int row = y;

    if (entry.scope == Scope::FileDiff) {
        const int card_x = stream.offset_x;
        const int card_w = diff_card_width_cols(stream);
        const int trunk_x = stream.offset_x;
        for (const auto& line : entry.diff) {
            if (row >= bounds_.height) {
                break;
            }
            paint_activity_row_bg(ctx, row, bounds_.width);
            paint_activity_tree_trunk_clear(ctx, trunk_x, row);
            if (line.kind == BodyLineKind::DiffHeader) {
                paint_diff_header_line(ctx, card_x, row, card_w, line.text, line.diff_icon, line.diff_icon_rgb,
                                       line.diff_adds, line.diff_dels);
            } else if (line.kind == BodyLineKind::DiffSplit) {
                const std::vector<ActivityDiffTextSegment>* left_segments =
                    line.diff_segments.empty() ? nullptr : &line.diff_segments;
                const std::vector<ActivityDiffTextSegment>* right_segments =
                    line.diff_right_segments.empty() ? nullptr : &line.diff_right_segments;
                paint_diff_split_row(ctx, card_x, row, card_w, line.diff_old_line, line.diff_new_line, line.text,
                                     line.diff_right_text, line.style, line.diff_right_style, left_segments,
                                     right_segments);
            } else if (line.diff_fill_row) {
                const std::vector<ActivityDiffTextSegment>* segments =
                    line.diff_segments.empty() ? nullptr : &line.diff_segments;
                paint_diff_code_line(ctx, card_x, row, card_w, line.text, line.style, segments);
            } else if (!line.text.empty()) {
                const int body_columns = std::max(0, card_w - 1);
                const std::size_t body_bytes = tuinator::text_byte_length_for_width(line.text, body_columns);
                if (body_bytes > 0) {
                    paint_text_with_background(ctx, card_x + 1, row, line.text.substr(0, body_bytes), line.style);
                }
            }
            ++row;
        }
        if (row < bounds_.height) {
            paint_activity_row_bg(ctx, row, bounds_.width);
        }
        return;
    }

    paint_activity_row_bg(ctx, row, bounds_.width, row_bg);
    if (entry.running) {
        paint_activity_running_prefix_bg(ctx, stream_x, row, bounds_.width);
    }

    tuinator::Style chrome_draw = activity_on_panel(muted_style_);
    chrome_draw.background_rgb = row_bg;
    if (entry.running) {
        chrome_draw.foreground_rgb = ui_pulse_rgb(ui_secondary_text(), ui_session_active_bar(), spinner_ms_);
    } else {
        chrome_draw.foreground_rgb = ui_secondary_text();
        chrome_draw.dim = true;
    }

    const std::string status =
        entry.running ? ui_braille_spinner_frame(spinner_ms_, static_cast<int>(entry_index))
                      : (entry.failed ? "✗" : "✓");
    tuinator::Style status_style = entry.running ? active_style_ : (entry.failed ? failed_style_ : done_style_);
    if (entry.running) {
        status_style.background_rgb = ui_badge_bg();
        status_style.foreground_rgb = ui_session_active_bar();
        status_style.bold = false;
    } else if (entry.failed) {
        status_style.background_rgb = row_bg;
        status_style.foreground_rgb = failed_style_.foreground_rgb;
        status_style.bold = true;
    } else {
        status_style.background_rgb = row_bg;
        status_style.foreground_rgb = ui_section_heading();
        status_style.bold = false;
        if (entry.flash_ms > 0) {
            const double t = 1.0 - static_cast<double>(entry.flash_ms) / static_cast<double>(kUiAnimFadeMs);
            status_style.foreground_rgb = ui_fade_toward(tuinator::rgb(255, 255, 255), ui_section_heading(), t);
        }
    }
    const char* branch = branch_override;
    if (branch == nullptr) {
        if (entry_followed_by_file_diff(entry_index)) {
            branch = "╰─";
        } else if (more_layout_rows_after) {
            branch = "├─";
        } else {
            branch = "╰─";
        }
    }
    paint_activity_tree_prefix(ctx, stream_x, row, branch, status, chrome_draw, status_style, row_bg);

    const std::string header = header_text(entry);
    tuinator::Style header_style = entry.running ? active_style_ : (entry.failed ? failed_style_ : body_style_);
    if (entry.running) {
        header_style = activity_on_panel(header_style);
        header_style.background_rgb = ui_main_bg();
        header_style.foreground_rgb = ui_section_heading();
        header_style.bold = false;
    } else if (entry.failed) {
        header_style = activity_on_panel(body_style_);
        header_style.background_rgb = row_bg;
        header_style.foreground_rgb = failed_style_.foreground_rgb;
        header_style.bold = true;
    } else {
        header_style = activity_on_panel(header_style);
        header_style.background_rgb = row_bg;
        if (entry.settle_ms > 0) {
            const double t = 1.0 - static_cast<double>(entry.settle_ms) / static_cast<double>(kUiAnimFadeMs);
            const tuinator::Rgb bright = tuinator::rgb(245, 242, 255);
            const tuinator::Rgb target = tuinator::rgb(220, 225, 240);
            header_style.foreground_rgb = ui_fade_toward(bright, target, t);
        }
    }
    const int header_left = stream_x + kTreeTextCol;
    const int header_right = stream.offset_x + diff_card_width_cols(stream);
    const bool show_affordance =
        entry_header_expandable(entry) && (entry.expanded || static_cast<int>(entry_index) == hovered_entry_index_);
    const int affordance_reserve = show_affordance ? tuinator::text_display_width(kNfChevronRight) + 1 : 0;
    const int max_columns = std::max(0, header_right - header_left - affordance_reserve);
    const std::size_t bytes = tuinator::text_byte_length_for_width(header, max_columns);
    int chevron_x = header_left;
    if (bytes > 0) {
        const std::string clipped = header.substr(0, bytes);
        paint_text_with_background(ctx, header_left, row, clipped, header_style);
        if (show_affordance) {
            chevron_x = header_left + tuinator::text_display_width(clipped) + 1;
        }
    }
    paint_header_expand_affordance(ctx, row, entry, entry_index, row_bg, chevron_x);

    if (!entry.expanded) {
        return;
    }

    ++row;
    for (const auto& line : body_lines(entry)) {
        if (row >= bounds_.height) {
            break;
        }
        paint_activity_row_bg(ctx, row, bounds_.width);
        if (line.kind == BodyLineKind::DiffHeader) {
            const int card_x = stream.offset_x;
            const int card_w = diff_card_width_cols(stream);
            paint_diff_header_line(ctx, card_x, row, card_w, line.text, line.diff_icon, line.diff_icon_rgb,
                                   line.diff_adds, line.diff_dels);
            ++row;
            continue;
        }
        if (line.bar) {
            tuinator::Style bar_style = bar_style_;
            bar_style.background_rgb = ui_main_bg();
            bar_style.foreground_rgb = ui_secondary_text();
            bar_style.dim = true;
            if (entry.running) {
                bar_style.foreground_rgb = ui_pulse_rgb(ui_secondary_text(), ui_session_active_bar(), spinner_ms_);
            }
            paint_text_with_background(ctx, stream_x + kBodyBarCol, row, "│", bar_style);
        }
        if (line.diff_fill_row) {
            const int card_x = stream.offset_x;
            const int card_w = diff_card_width_cols(stream);
            const std::vector<ActivityDiffTextSegment>* segments =
                line.diff_segments.empty() ? nullptr : &line.diff_segments;
            paint_diff_code_line(ctx, card_x, row, card_w, line.text, line.style, segments);
            ++row;
            continue;
        }
        const int body_columns = std::max(0, bounds_.width - stream_x - kBodyIndent);
        const std::size_t body_bytes = tuinator::text_byte_length_for_width(line.text, body_columns);
        if (body_bytes > 0) {
            paint_text_with_background(ctx, stream_x + kBodyIndent, row, line.text.substr(0, body_bytes), line.style);
        }
        ++row;
    }
    if (entry.spacer_after && row < bounds_.height) {
        paint_activity_row_bg(ctx, row, bounds_.width);
    }
}

int ActivityLog::y_for_layout_header_line(int local_y, int* header_entry_index, std::size_t* cluster_run_begin) const {
    if (header_entry_index != nullptr) {
        *header_entry_index = -1;
    }
    if (cluster_run_begin != nullptr) {
        *cluster_run_begin = entries_.size();
    }
    int y = 0;
    if (should_show_pipeline_title()) {
        if (local_y == y) {
            return y;
        }
        ++y;
    }
    const std::vector<LayoutRow> layout = build_layout();
    for (std::size_t layout_index = 0; layout_index < layout.size(); ++layout_index) {
        const LayoutRow& row = layout[layout_index];
        if (row.kind == LayoutRowKind::Entry) {
            if (local_y == y) {
                if (header_entry_index != nullptr) {
                    *header_entry_index = static_cast<int>(row.entry_index);
                }
                return y;
            }
            const int height = entry_height(entries_[row.entry_index]);
            if (local_y > y && local_y < y + height) {
                return -1;
            }
            y += height;
            y += layout_row_gap_;
            continue;
        }
        const bool detached_root = layout_follows_file_diff(layout, layout_index);
        const bool open = mundane_cluster_is_open(row.mundane_begin);
        const std::size_t mundane_count = row.mundane_end - row.mundane_begin;
        const bool skip_header_for_single_open = detached_root && open && mundane_count == 1;
        if (!skip_header_for_single_open && local_y == y) {
            if (cluster_run_begin != nullptr) {
                *cluster_run_begin = row.mundane_begin;
            }
            return y;
        }
        const int height = layout_row_height(row, layout, layout_index);
        if (local_y > y && local_y < y + height) {
            if (open) {
                int inner_y = y + (skip_header_for_single_open ? 0 : 1);
                for (std::size_t i = row.mundane_begin; i < row.mundane_end; ++i) {
                    if (local_y == inner_y) {
                        if (header_entry_index != nullptr) {
                            *header_entry_index = static_cast<int>(i);
                        }
                        return inner_y;
                    }
                    const int entry_h = entry_height(entries_[i]);
                    if (local_y > inner_y && local_y < inner_y + entry_h) {
                        return -1;
                    }
                    inner_y += entry_h;
                }
            }
            return -1;
        }
        y += height;
        y += layout_row_gap_;
    }
    return -1;
}

void ActivityLog::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void ActivityLog::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }
    if (total_lines() <= 0) {
        return;
    }

    const ChatStreamLayout stream = chat_stream_layout_in_widget(bounds_.width);
    const int stream_x = stream.offset_x;

    int y = 0;
    if (should_show_pipeline_title() && y < bounds_.height) {
        paint_activity_row_bg(ctx, y, bounds_.width);
        tuinator::Style branch = muted_style_;
        branch.background_rgb = ui_main_bg();
        paint_activity_tree_prefix(ctx, stream_x, y, "╭─", "", branch, branch, ui_main_bg());
        const int title_x = stream_x + kTreeTextCol - 1;
        const int title_fill_w = std::max(0, bounds_.width - title_x);
        if (title_fill_w > 0) {
            tuinator::Style row_fill = activity_on_panel(heading_style_);
            row_fill.bold = true;
            ctx.canvas.fill_rect({title_x, y, title_fill_w, 1}, ' ', row_fill);
        }
        tuinator::Style title = activity_on_panel(heading_style_);
        title.bold = true;
        ctx.canvas.draw_text({title_x, y}, pipeline_title_, title);
        ++y;
    }

    constexpr int kMundaneTreeIndent = 2;
    const tuinator::Rgb row_bg = ui_main_bg();
    const std::vector<LayoutRow> layout = build_layout();
    for (std::size_t layout_index = 0; layout_index < layout.size(); ++layout_index) {
        const LayoutRow& row = layout[layout_index];
        const bool more_layout_after = layout_index + 1 < layout.size() || should_show_generating_tail();
        if (y >= bounds_.height) {
            break;
        }
        if (row.kind == LayoutRowKind::Entry) {
            paint_entry_block(ctx, y, row.entry_index, stream, row_bg, 0, nullptr, more_layout_after);
            y += entry_height(entries_[row.entry_index]);
            paint_layout_row_gap(ctx, y, stream_x, more_layout_after);
            continue;
        }
        const bool detached_root = layout_follows_file_diff(layout, layout_index);
        const bool open = mundane_cluster_is_open(row.mundane_begin);
        const std::size_t mundane_count = row.mundane_end - row.mundane_begin;
        const bool skip_header_for_single_open = detached_root && open && mundane_count == 1;

        if (!skip_header_for_single_open) {
            paint_mundane_cluster_header(ctx, y, row.mundane_begin, row.mundane_end, stream, row_bg, open,
                                         detached_root);
            ++y;
        }
        if (!open) {
            paint_layout_row_gap(ctx, y, stream_x, more_layout_after);
            continue;
        }
        for (std::size_t i = row.mundane_begin; i < row.mundane_end; ++i) {
            if (y >= bounds_.height) {
                break;
            }
            const bool last_mundane = i + 1 == row.mundane_end;
            const char* branch = (last_mundane && !more_layout_after) ? "╰─" : "├─";
            const int indent = skip_header_for_single_open ? 0 : kMundaneTreeIndent;
            if (skip_header_for_single_open) {
                paint_activity_tree_trunk_clear(ctx, stream.offset_x, y);
            }
            paint_entry_block(ctx, y, i, stream, row_bg, indent, branch, more_layout_after);
            y += entry_height(entries_[i]);
        }
        paint_layout_row_gap(ctx, y, stream_x, more_layout_after);
    }

    if (should_show_generating_tail() && y < bounds_.height) {
        paint_activity_row_bg(ctx, y, bounds_.width, ui_main_bg());
        tuinator::Style branch = muted_style_;
        branch.background_rgb = ui_main_bg();
        tuinator::Style spin = muted_style_;
        spin.background_rgb = ui_main_bg();
        spin.foreground_rgb = ui_session_active_bar();
        spin.bold = false;
        paint_activity_tree_prefix(ctx, stream_x, y, "╰─", ui_braille_spinner_frame(spinner_ms_, 3), branch, spin,
                                   ui_main_bg());
        tuinator::Style label = activity_on_panel(body_style_);
        label.background_rgb = ui_main_bg();
        label.foreground_rgb = ui_section_heading();
        paint_text_with_background(ctx, stream_x + kTreeTextCol, y, generating_label_, label);
    }
}

bool ActivityLog::handle_event(const tuinator::Event& event) {
    const auto* mouse = std::get_if<tuinator::MouseEvent>(&event);
    if (mouse == nullptr) {
        return false;
    }
    const int local_y = mouse->position.y - bounds_.y;
    if (mouse->action == tuinator::MouseAction::Move) {
        if (!contains_point(mouse->position)) {
            if (hovered_entry_index_ != -1) {
                hovered_entry_index_ = -1;
                mark_dirty();
            }
            return false;
        }
        const int next_hover = header_entry_index_at_local_y(local_y);
        if (next_hover != hovered_entry_index_) {
            hovered_entry_index_ = next_hover;
            mark_dirty();
        }
        return true;
    }
    if (mouse->action != tuinator::MouseAction::Release) {
        return false;
    }
    if (!contains_point(mouse->position)) {
        return false;
    }
    int y = 0;
    // Keep in sync with paint(): pipeline title row is not an entry but consumes one line.
    if (should_show_pipeline_title()) {
        if (local_y == y) {
            return true;
        }
        ++y;
    }
    const std::vector<LayoutRow> layout = build_layout();
    for (std::size_t layout_index = 0; layout_index < layout.size(); ++layout_index) {
        const LayoutRow& row = layout[layout_index];
        const int height = layout_row_height(row, layout, layout_index);
        if (local_y < y || local_y >= y + height) {
            y += height;
            y += layout_row_gap_;
            continue;
        }
        if (row.kind == LayoutRowKind::MundaneCluster) {
            const bool detached_root = layout_follows_file_diff(layout, layout_index);
            const bool open = mundane_cluster_is_open(row.mundane_begin);
            const std::size_t mundane_count = row.mundane_end - row.mundane_begin;
            const bool skip_header_for_single_open = detached_root && open && mundane_count == 1;
            if (!skip_header_for_single_open && local_y == y) {
                set_mundane_cluster_open(row.mundane_begin, !open);
                return true;
            }
            if (open) {
                int inner_y = y + (skip_header_for_single_open ? 0 : 1);
                for (std::size_t i = row.mundane_begin; i < row.mundane_end; ++i) {
                    if (local_y == inner_y && entry_header_expandable(entries_[i])) {
                        entries_[i].expanded = !entries_[i].expanded;
                        mark_dirty();
                        mark_layout_dirty();
                        return true;
                    }
                    inner_y += entry_height(entries_[i]);
                }
            }
            return true;
        }
        if (local_y == y && entry_header_expandable(entries_[row.entry_index])) {
            entries_[row.entry_index].expanded = !entries_[row.entry_index].expanded;
            mark_dirty();
            mark_layout_dirty();
        }
        return true;
    }
    if (generating_tail_ && local_y == y) {
        return true;
    }
    return true;
}
