#include "activity_log.hpp"

#include "ui_anim.hpp"
#include "ui_chat_column.hpp"
#include "ui_theme.hpp"

#include <libtaiji.h>

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cstdio>
#include <cstdlib>
#include <sstream>

namespace {

std::string json_string_field(const std::string& json, const std::string& key);

bool is_invoke_skill_tool_name(const std::string& name) {
    return name == "invoke_skill";
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
    return "Question";
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
            out.push_back(json[i++]);
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
constexpr std::size_t kMaxResultLines = 80;
constexpr std::size_t kMaxParamLines = 60;

const char* kSpinnerFrames[] = {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"};
constexpr int kSpinnerFrameCount = 10;

constexpr tuinator::Rgb kActivityOkBadgeBg{52, 78, 62};
constexpr tuinator::Rgb kActivityFailBadgeBg{78, 48, 48};
constexpr tuinator::Rgb kActivityAddLineBg{45, 68, 52};
constexpr tuinator::Rgb kActivityDelLineBg{72, 48, 48};
constexpr tuinator::Rgb kActivityRunningBadgeBg{58, 48, 72};

tuinator::Style activity_on_panel(tuinator::Style style) {
    return ui_resolve_main_text(style);
}

tuinator::Style activity_draw_style(tuinator::Style style) {
    if (style.background_rgb.has_value() && *style.background_rgb != ui_main_bg()) {
        return style;
    }
    return activity_on_panel(style);
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
    style.background_rgb = kActivityFailBadgeBg;
    style.foreground_rgb = tuinator::rgb(250, 200, 200);
    style.bold = true;
    return style;
}

tuinator::Style make_failed_body_style() {
    tuinator::Style style;
    style.background_rgb = tuinator::rgb(55, 38, 42);
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

std::vector<std::string> pretty_args_lines(const std::string& args) {
    std::vector<std::string> lines;
    const auto begin = args.find_first_not_of(" \t\n\r");
    if (begin == std::string::npos) {
        return lines;
    }
    JsonPrinter printer(std::string_view(args).substr(begin));
    if (printer.print(lines)) {
        return lines;
    }
    // Not JSON: show the raw text.
    return wrap_text(args, 68);
}

} // namespace

ActivityLog::ActivityLog(tuinator::Style active_style, tuinator::Style muted_style)
    : active_style_(active_style), done_style_(make_done_style()), failed_style_(make_failed_style()),
      failed_body_style_(make_failed_body_style()), muted_style_(muted_style),
      body_style_(muted_style), thinking_body_style_(muted_style), heading_style_(muted_style),
      add_style_(make_add_style()), del_style_(make_del_style()), bar_style_(muted_style) {
    thinking_body_style_.italic = true;
    heading_style_.bold = true;
    bar_style_.dim = true;
}

void ActivityLog::clear() {
    entries_.clear();
    received_think_deltas_ = false;
    spinner_ms_ = 0;
    pipeline_title_.clear();
    generating_tail_ = false;
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
    switch (scope) {
    case Scope::Thinking: return "Thinking";
    case Scope::Tool: return "Tool";
    case Scope::MCP: return "MCP";
    case Scope::Skill: return "Skill";
    case Scope::Decision: return "Decision";
    case Scope::Question: return "Question";
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
    if (is_ask_user || tool_name == "ask_user") {
        return;
    }
    Entry* entry = nullptr;
    for (auto it = entries_.rbegin(); it != entries_.rend(); ++it) {
        if (it->running && it->scope != Scope::Thinking && it->name == tool_name) {
            entry = &*it;
            break;
        }
    }
    if (entry == nullptr) {
        for (auto it = entries_.rbegin(); it != entries_.rend(); ++it) {
            if (it->running && it->scope != Scope::Thinking) {
                entry = &*it;
                break;
            }
        }
    }
    if (entry == nullptr) {
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
        entry = &entries_.back();
    }

    if (entry->scope == Scope::Decision || is_decision || tool_name == "decide") {
        entry->scope = Scope::Decision;
        entry->expanded = true;
    }
    if (!decision_driver.empty()) {
        entry->decision_driver = decision_driver;
    }
    if (!skill_name.empty() && !is_invoke_skill_tool_name(skill_name)) {
        entry->skill_name = skill_name;
    }
    entry->body = text;
    entry->failed = tool_observation_failed(text);
    if (entry->scope == Scope::Skill && entry->skill_name.empty()) {
        entry->skill_name = skill_name_from_result_text(text);
    }
    if (entry->scope == Scope::Skill && entry->skill_name.empty()) {
        entry->skill_name = skill_name_from_args(entry->args);
    }
    if (entry->scope == Scope::Skill && is_invoke_skill_tool_name(entry->skill_name)) {
        entry->skill_name.clear();
    }
    if (entry->scope == Scope::Skill && !entry->skill_name.empty()) {
        mark_dirty();
        mark_layout_dirty();
    }
    if (!diff.empty()) {
        entry->diff = parse_diff(diff, muted_style_, add_style_, del_style_);
    }
    if (duration_ms >= 0) {
        entry->duration_ms = duration_ms;
        entry->has_duration = true;
    }
    finalize_entry(*entry);
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
    mark_layout_dirty();
}

void ActivityLog::set_taiji_handle(uint64_t handle) {
    taiji_handle_ = handle;
}

std::string ActivityLog::tool_display_title(const std::string& tool_name) const {
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
        std::string text = "MCP";
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
        std::string text = "Skill";
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
        std::string text = "Decision";
        if (!entry.decision_driver.empty()) {
            text += " · " + entry.decision_driver;
        } else if (!entry.name.empty() && entry.name != "decide") {
            text += " · " + entry.name;
        }
        return text;
    }
    if (entry.scope == Scope::Question) {
        if (!entry.question_header.empty()) {
            return "Question · " + entry.question_header;
        }
        return "Question";
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

    auto params = pretty_args_lines(entry.args);
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

std::vector<ActivityLog::BodyLine> ActivityLog::body_lines(const Entry& entry) const {
    if (entry.scope == Scope::Thinking) {
        return thinking_body_lines(entry);
    }
    if (entry.scope == Scope::Decision) {
        return decision_body_lines(entry);
    }
    if (entry.scope == Scope::Question) {
        return question_body_lines(entry);
    }
    return tool_body_lines(entry);
}

int ActivityLog::entry_height(const Entry& entry) const {
    int height = 1;
    if (entry.expanded) {
        height += static_cast<int>(body_lines(entry).size());
    }
    if (entry.spacer_after) {
        ++height;
    }
    return height;
}

int ActivityLog::total_lines() const {
    int total = 0;
    if (!pipeline_title_.empty() && !entries_.empty()) {
        ++total;
    }
    for (const Entry& entry : entries_) {
        total += entry_height(entry);
    }
    if (generating_tail_) {
        ++total;
    }
    return total;
}

tuinator::Size ActivityLog::preferred_size() const {
    if (entries_.empty() && !generating_tail_) {
        return {0, 0};
    }
    const int width =
        std::max({wrap_cols(), chat_stream_content_width(bounds_.width), chat_stream_row_preferred_width(bounds_.width)});
    int max_line_width = 0;
    if (!pipeline_title_.empty()) {
        max_line_width = std::max(max_line_width, tuinator::text_display_width(pipeline_title_) + kTreeTextCol + 2);
    }
    for (const Entry& entry : entries_) {
        max_line_width = std::max(max_line_width, tuinator::text_display_width(header_text(entry)) + kTreeTextCol);
        if (entry.expanded) {
            for (const auto& line : body_lines(entry)) {
                max_line_width = std::max(max_line_width, tuinator::text_display_width(line.text) + kBodyIndent);
            }
        }
    }
    return {width, total_lines()};
}

void ActivityLog::layout(tuinator::Rect bounds) {
    if (bounds_.width != bounds.width) {
        mark_layout_dirty();
    }
    bounds_ = bounds;
}

void ActivityLog::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }
    if (entries_.empty() && !generating_tail_) {
        return;
    }

    const ChatStreamLayout stream = chat_stream_layout_in_widget(bounds_.width);
    const int stream_x = stream.offset_x;

    int y = 0;
    if (!pipeline_title_.empty() && !entries_.empty() && y < bounds_.height) {
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

    for (std::size_t entry_index = 0; entry_index < entries_.size(); ++entry_index) {
        const Entry& entry = entries_[entry_index];
        if (y >= bounds_.height) {
            break;
        }

        const tuinator::Rgb row_bg = ui_main_bg();
        paint_activity_row_bg(ctx, y, bounds_.width, row_bg);
        if (entry.running) {
            paint_activity_running_prefix_bg(ctx, stream_x, y, bounds_.width);
        }

        const tuinator::Style& chrome_style = entry.failed ? failed_body_style_ : muted_style_;
        tuinator::Style chrome_draw = chrome_style;
        chrome_draw.background_rgb = row_bg;
        if (entry.running) {
            chrome_draw.foreground_rgb = ui_pulse_rgb(ui_secondary_text(), ui_session_active_bar(), spinner_ms_);
        }

        const std::string status =
            entry.running ? ui_braille_spinner_frame(spinner_ms_, static_cast<int>(entry_index))
                          : (entry.failed ? "✗" : "✓");
        tuinator::Style status_style =
            entry.running ? active_style_ : (entry.failed ? failed_style_ : done_style_);
        if (entry.running) {
            status_style.background_rgb = ui_badge_bg();
            status_style.foreground_rgb = ui_session_active_bar();
            status_style.bold = false;
        } else if (!entry.failed) {
            status_style.background_rgb = row_bg;
            status_style.foreground_rgb = ui_section_heading();
            status_style.bold = false;
            if (entry.flash_ms > 0) {
                const double t = 1.0 - static_cast<double>(entry.flash_ms) / static_cast<double>(kUiAnimFadeMs);
                status_style.foreground_rgb =
                    ui_fade_toward(tuinator::rgb(255, 255, 255), ui_section_heading(), t);
            }
        }
        paint_activity_tree_prefix(ctx, stream_x, y, "├─", status, chrome_draw, status_style, row_bg);

        const std::string header = header_text(entry);
        tuinator::Style header_style =
            entry.running ? active_style_ : (entry.failed ? failed_style_ : body_style_);
        if (entry.running) {
            header_style = activity_on_panel(header_style);
            header_style.background_rgb = ui_main_bg();
            header_style.foreground_rgb = ui_section_heading();
            header_style.bold = false;
        } else if (entry.failed) {
            header_style = activity_draw_style(header_style);
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
        const int max_columns = std::max(0, bounds_.width - stream_x - kTreeTextCol);
        const std::size_t bytes = tuinator::text_byte_length_for_width(header, max_columns);
        if (bytes > 0) {
            const std::string clipped = header.substr(0, bytes);
            const bool skill_scope = entry.scope == Scope::Skill;
            const std::string skill_token =
                skill_highlight_token(entry.skill_name, entry.name, entry.args, skill_scope);
            paint_activity_header_text(ctx, stream_x + kTreeTextCol, y, clipped, header_style, skill_token);
        }
        ++y;

        if (!entry.expanded) {
            continue;
        }

        for (const auto& line : body_lines(entry)) {
            if (y >= bounds_.height) {
                break;
            }
            paint_activity_row_bg(ctx, y, bounds_.width);
            if (line.bar) {
                tuinator::Style bar_style = entry.failed ? failed_body_style_ : bar_style_;
                if (!entry.failed) {
                    bar_style.background_rgb = ui_main_bg();
                }
                if (entry.running) {
                    bar_style.foreground_rgb = ui_pulse_rgb(ui_secondary_text(), ui_session_active_bar(), spinner_ms_);
                }
                paint_text_with_background(ctx, stream_x + kBodyBarCol, y, "│", bar_style);
            }
            const int body_columns = std::max(0, bounds_.width - stream_x - kBodyIndent);
            const std::size_t body_bytes = tuinator::text_byte_length_for_width(line.text, body_columns);
            if (body_bytes > 0) {
                paint_text_with_background(ctx, stream_x + kBodyIndent, y, line.text.substr(0, body_bytes),
                                           line.style);
            }
            ++y;
        }

        if (entry.spacer_after && y < bounds_.height) {
            ++y;
        }
    }

    if (generating_tail_ && y < bounds_.height) {
        paint_activity_row_bg(ctx, y, bounds_.width, ui_main_bg());
        tuinator::Style branch = muted_style_;
        branch.background_rgb = ui_main_bg();
        tuinator::Style spin = muted_style_;
        spin.background_rgb = ui_main_bg();
        spin.foreground_rgb = ui_session_active_bar();
        spin.bold = false;
        paint_activity_tree_prefix(ctx, stream_x, y, "├─", ui_braille_spinner_frame(spinner_ms_, 3), branch, spin,
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
    if (mouse->action != tuinator::MouseAction::Release) {
        return false;
    }
    if (!contains_point(mouse->position)) {
        return false;
    }

    const int local_y = mouse->position.y - bounds_.y;
    int y = 0;
    for (Entry& entry : entries_) {
        const int height = entry_height(entry);
        if (local_y >= y && local_y < y + height) {
            if (local_y == y) {
                entry.expanded = !entry.expanded;
                mark_dirty();
                mark_layout_dirty();
            }
            return true;
        }
        y += height;
    }
    return true;
}
