#include "activity_log.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cstdio>
#include <sstream>

namespace {

constexpr int kHeaderIndent = 4; // chevron + status icon + spaces
constexpr int kBodyBarCol = 2;
constexpr int kBodyIndent = 4;
constexpr int kSpinnerPeriodMs = 80;
constexpr std::size_t kMaxThinkingLines = 120;
constexpr std::size_t kMaxResultLines = 80;
constexpr std::size_t kMaxParamLines = 60;

const char* kSpinnerFrames[] = {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"};
constexpr int kSpinnerFrameCount = 10;

tuinator::Style make_done_style() {
    tuinator::Style style;
    style.foreground_rgb = tuinator::rgb(150, 200, 170);
    style.bold = true;
    return style;
}

tuinator::Style make_add_style() {
    tuinator::Style style;
    style.foreground_rgb = tuinator::rgb(120, 210, 140);
    return style;
}

tuinator::Style make_del_style() {
    tuinator::Style style;
    style.foreground_rgb = tuinator::rgb(235, 120, 120);
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
    : active_style_(active_style), done_style_(make_done_style()), muted_style_(muted_style),
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
    mark_dirty();
    mark_layout_dirty();
}

std::string ActivityLog::scope_label(Scope scope) {
    switch (scope) {
    case Scope::Thinking: return "Thinking";
    case Scope::Tool: return "Tool";
    case Scope::MCP: return "MCP";
    case Scope::Skill: return "Skill";
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

void ActivityLog::start_tool(const std::string& tool_name, const std::string& args, bool is_mcp,
                             const std::string& mcp_server) {
    // A tool call ends any ongoing thinking phase.
    end_thinking();

    Entry entry;
    entry.scope = is_mcp ? Scope::MCP : Scope::Tool;
    entry.running = true;
    entry.name = tool_name;
    entry.server = mcp_server;
    entry.args = args;
    entry.started_at = std::chrono::steady_clock::now();
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

void ActivityLog::finish_tool(const std::string& tool_name, const std::string& text, const std::string& diff,
                              int64_t duration_ms) {
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
        fallback.scope = Scope::Tool;
        fallback.name = tool_name;
        fallback.started_at = std::chrono::steady_clock::now();
        entries_.push_back(std::move(fallback));
        entry = &entries_.back();
    }

    entry->body = text;
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

std::string ActivityLog::running_label() const {
    for (const Entry& entry : entries_) {
        if (!entry.running) {
            continue;
        }
        if (entry.scope == Scope::Thinking) {
            return "Thinking…";
        }
        return scope_label(entry.scope) + " · " + entry.name;
    }
    return "Working…";
}

void ActivityLog::tick(int dt_ms) {
    if (!has_running()) {
        return;
    }
    spinner_ms_ = (spinner_ms_ + dt_ms) % (kSpinnerPeriodMs * kSpinnerFrameCount);
    mark_dirty();
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

    std::string text = scope_label(entry.scope);
    if (!entry.name.empty()) {
        text += " · " + entry.name;
    }
    if (entry.scope == Scope::MCP && !entry.server.empty()) {
        text += " · " + entry.server;
    }
    if (entry.running || entry.has_duration) {
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

std::vector<ActivityLog::BodyLine> ActivityLog::tool_body_lines(const Entry& entry) const {
    std::vector<BodyLine> lines;

    auto params = pretty_args_lines(entry.args);
    std::size_t omitted_params = 0;
    if (params.size() > kMaxParamLines) {
        omitted_params = params.size() - kMaxParamLines;
        params.resize(kMaxParamLines);
    }
    if (!params.empty()) {
        lines.push_back(BodyLine{.text = "Parameters", .style = heading_style_, .bar = true});
        for (auto& line : params) {
            lines.push_back(BodyLine{.text = std::move(line), .style = body_style_, .bar = true});
        }
        if (omitted_params > 0) {
            lines.push_back(BodyLine{.text = "… (" + std::to_string(omitted_params) + " more lines)",
                                     .style = muted_style_, .bar = true});
        }
    }

    if (!entry.body.empty()) {
        auto result = wrap_text(entry.body, body_wrap_cols());
        std::size_t omitted = 0;
        if (result.size() > kMaxResultLines) {
            omitted = result.size() - kMaxResultLines;
            result.resize(kMaxResultLines);
        }
        lines.push_back(BodyLine{.text = "Result", .style = heading_style_, .bar = true});
        for (auto& line : result) {
            lines.push_back(BodyLine{.text = std::move(line), .style = body_style_, .bar = true});
        }
        if (omitted > 0) {
            lines.push_back(BodyLine{.text = "… (" + std::to_string(omitted) + " more lines)",
                                     .style = muted_style_, .bar = true});
        }
    }

    for (const auto& line : entry.diff) {
        lines.push_back(line);
    }

    if (lines.empty()) {
        lines.push_back(BodyLine{.text = entry.running ? "(running…)" : "(no output)",
                                 .style = muted_style_, .bar = true});
    }
    return lines;
}

std::vector<ActivityLog::BodyLine> ActivityLog::body_lines(const Entry& entry) const {
    if (entry.scope == Scope::Thinking) {
        return thinking_body_lines(entry);
    }
    return tool_body_lines(entry);
}

int ActivityLog::entry_height(const Entry& entry) const {
    int height = 1;
    if (entry.expanded) {
        height += static_cast<int>(body_lines(entry).size());
    }
    return height;
}

int ActivityLog::total_lines() const {
    int total = 0;
    for (const Entry& entry : entries_) {
        total += entry_height(entry);
    }
    return total;
}

tuinator::Size ActivityLog::preferred_size() const {
    if (entries_.empty()) {
        return {0, 0};
    }
    const int width = wrap_cols();
    int max_line_width = 0;
    for (const Entry& entry : entries_) {
        max_line_width = std::max(max_line_width, tuinator::text_display_width(header_text(entry)) + kHeaderIndent);
        if (entry.expanded) {
            for (const auto& line : body_lines(entry)) {
                max_line_width = std::max(max_line_width, tuinator::text_display_width(line.text) + kBodyIndent);
            }
        }
    }
    return {std::min(max_line_width, width), total_lines()};
}

void ActivityLog::layout(tuinator::Rect bounds) {
    if (bounds_.width != bounds.width) {
        mark_layout_dirty();
    }
    bounds_ = bounds;
}

void ActivityLog::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0 || entries_.empty()) {
        return;
    }

    int y = 0;
    for (const Entry& entry : entries_) {
        if (y >= bounds_.height) {
            break;
        }

        const char* chevron = entry.expanded ? "▾" : "▸";
        ctx.canvas.draw_text({0, y}, chevron, muted_style_);

        const char* status = entry.running ? spinner_frame() : "✓";
        const tuinator::Style& status_style = entry.running ? active_style_ : done_style_;
        ctx.canvas.draw_text({2, y}, status, status_style);

        const std::string header = header_text(entry);
        const tuinator::Style& header_style = entry.running ? active_style_ : body_style_;
        const int max_columns = std::max(0, bounds_.width - kHeaderIndent);
        const std::size_t bytes = tuinator::text_byte_length_for_width(header, max_columns);
        if (bytes > 0) {
            ctx.canvas.draw_text({kHeaderIndent, y}, header.substr(0, bytes), header_style);
        }
        ++y;

        if (!entry.expanded) {
            continue;
        }

        for (const auto& line : body_lines(entry)) {
            if (y >= bounds_.height) {
                break;
            }
            if (line.bar) {
                ctx.canvas.draw_text({kBodyBarCol, y}, "│", bar_style_);
            }
            const int body_columns = std::max(0, bounds_.width - kBodyIndent);
            const std::size_t body_bytes = tuinator::text_byte_length_for_width(line.text, body_columns);
            if (body_bytes > 0) {
                ctx.canvas.draw_text({kBodyIndent, y}, line.text.substr(0, body_bytes), line.style);
            }
            ++y;
        }
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
