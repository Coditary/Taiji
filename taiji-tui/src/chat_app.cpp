#include "chat_app.hpp"

#include "left_label.hpp"
#include "left_anchored_split.hpp"
#include "right_anchored_split.hpp"
#include "session_context_menu.hpp"
#include "session_sidebar.hpp"
#include "model_display.hpp"
#include "settings_modal.hpp"
#include "chat_input_bar.hpp"
#include "speech_mic_button.hpp"
#include "sidebar_divider.hpp"
#include "ui_layout.hpp"
#include "ui_chat_column.hpp"
#include "ui_panel_box.hpp"
#include "ui_theme.hpp"

#include <tuinator/widgets/menu/menu_bar.hpp>

#include <libtaiji.h>

#include <tuinator/platform/clipboard.hpp>

#include <cctype>
#include <cstdio>
#include <cstdlib>
#include "ui_message_time.hpp"

#include <ctime>
#include <sstream>
#include <utility>
#include <vector>

namespace {

class AppShell : public tuinator::Widget {
  public:
    explicit AppShell(tuinator::Theme theme) {
        context_menu_ = std::make_unique<SessionContextMenu>(theme);
        settings_modal_ = std::make_unique<SettingsModal>(theme);
    }

    SessionContextMenu* context_menu() { return context_menu_.get(); }
    SettingsModal* settings_modal() { return settings_modal_.get(); }

    void set_preprocess_event(std::function<bool(const tuinator::Event&)> handler) {
        preprocess_event_ = std::move(handler);
    }

    bool wants_full_screen() const override { return true; }

    tuinator::Size preferred_size() const override {
        if (bounds_.width > 0 && bounds_.height > 0) {
            return bounds_.size();
        }
        if (!children_.empty()) {
            return children_.front()->preferred_size();
        }
        return {80, 24};
    }

    void layout(tuinator::Rect bounds) override {
        bounds_ = bounds;
        if (!children_.empty()) {
            children_.front()->layout(bounds_);
        }
        if (context_menu_ != nullptr) {
            context_menu_->layout(bounds_);
        }
        if (settings_modal_ != nullptr) {
            settings_modal_->layout(bounds_);
        }
    }

    void set_on_dirty(std::function<void(tuinator::Rect)> callback) override {
        tuinator::Widget::set_on_dirty(std::move(callback));
        if (context_menu_ != nullptr) {
            context_menu_->set_on_dirty(on_dirty_);
        }
        if (settings_modal_ != nullptr) {
            settings_modal_->set_on_dirty(on_dirty_);
        }
    }

    tuinator::Widget* hit_test(tuinator::Point point) override {
        if (!bounds_.contains(point)) {
            return nullptr;
        }
        if (settings_modal_ != nullptr && settings_modal_->is_open()) {
            if (tuinator::Widget* hit = settings_modal_->hit_test(point)) {
                return hit;
            }
        }
        if (context_menu_ != nullptr && context_menu_->is_open()) {
            if (tuinator::Widget* hit = context_menu_->hit_test(point)) {
                return hit;
            }
        }
        if (!children_.empty()) {
            return children_.front()->hit_test(point);
        }
        return this;
    }

    bool captures_keyboard() const override {
        if (settings_modal_ != nullptr && settings_modal_->is_open()) {
            return true;
        }
        return context_menu_ != nullptr && context_menu_->is_open();
    }

    void paint(tuinator::PaintContext& ctx) const override {
        if (!children_.empty()) {
            children_.front()->paint(ctx);
        }
        if (settings_modal_ != nullptr && settings_modal_->is_open()) {
            const tuinator::Rect exclude_abs = settings_modal_->panel_bounds();
            const tuinator::Rect exclude_local{exclude_abs.x - bounds_.x, exclude_abs.y - bounds_.y, exclude_abs.width,
                                               exclude_abs.height};
            ctx.canvas.apply_dimming_scrim(bounds_, exclude_local, 0.78f);
            settings_modal_->paint(ctx);
        }
        if (context_menu_ != nullptr && context_menu_->is_open()) {
            context_menu_->paint(ctx);
        }
    }

    bool handle_event(const tuinator::Event& event) override {
        if (preprocess_event_ && preprocess_event_(event)) {
            mark_dirty();
            return true;
        }
        if (settings_modal_ != nullptr && settings_modal_->is_open()) {
            if (settings_modal_->handle_event(event)) {
                mark_dirty();
                return true;
            }
        }
        if (context_menu_ != nullptr && context_menu_->is_open()) {
            if (context_menu_->handle_event(event)) {
                mark_dirty();
                return true;
            }
        }
        if (!children_.empty() && children_.front()->handle_event(event)) {
            mark_dirty();
            return true;
        }
        return false;
    }

    void collect_focusable(std::vector<tuinator::Widget*>& out) override {
        if (!children_.empty()) {
            children_.front()->collect_focusable(out);
        }
    }

    tuinator::Widget* hit_test_focusable(tuinator::Point point) override {
        if (!bounds_.contains(point)) {
            return nullptr;
        }
        if (settings_modal_ != nullptr && settings_modal_->is_open()) {
            return settings_modal_->hit_test_focusable(point);
        }
        if (context_menu_ != nullptr && context_menu_->is_open()) {
            return context_menu_->hit_test_focusable(point);
        }
        if (!children_.empty()) {
            return children_.front()->hit_test_focusable(point);
        }
        return nullptr;
    }

    std::string_view widget_type_name() const override { return "AppShell"; }

  private:
    std::unique_ptr<SessionContextMenu> context_menu_;
    std::unique_ptr<SettingsModal> settings_modal_;
    std::function<bool(const tuinator::Event&)> preprocess_event_;
};

std::unique_ptr<LeftLabel> make_sidebar_heading(const std::string& title, const tuinator::Theme& theme) {
    tuinator::Style style = theme.heading;
    style.bold = true;
    style.background_rgb = kUiSidebarBackground;
    return std::make_unique<LeftLabel>(title, style);
}

std::unique_ptr<LeftLabel> make_sidebar_label(const std::string& text, const tuinator::Style& style) {
    return std::make_unique<LeftLabel>(text, style);
}

std::vector<std::string> fetch_line_list(uint64_t handle, char* (*fetch)(uint64_t)) {
    std::vector<std::string> names;
    if (handle == 0 || fetch == nullptr) {
        return names;
    }
    char* raw = fetch(handle);
    if (raw == nullptr) {
        return names;
    }
    std::string text(raw);
    std::free(raw);
    std::size_t start = 0;
    while (start <= text.size()) {
        std::size_t end = text.find('\n', start);
        if (end == std::string::npos) {
            end = text.size();
        }
        std::string name = text.substr(start, end - start);
        if (!name.empty()) {
            names.push_back(std::move(name));
        }
        if (end >= text.size()) {
            break;
        }
        start = end + 1;
    }
    return names;
}

std::vector<std::string> fetch_mcp_server_names(uint64_t handle) {
    return fetch_line_list(handle, taiji_mcp_servers);
}

std::vector<std::string> fetch_active_skill_names(uint64_t handle) {
    return fetch_line_list(handle, taiji_active_skills);
}

std::string format_token_count(int tokens) {
    char buf[48];
    if (tokens >= 1'000'000) {
        std::snprintf(buf, sizeof(buf), "%.1fM tokens", static_cast<double>(tokens) / 1'000'000.0);
        return buf;
    }
    if (tokens >= 10'000) {
        std::snprintf(buf, sizeof(buf), "%.1fk tokens", static_cast<double>(tokens) / 1'000.0);
        return buf;
    }
    std::snprintf(buf, sizeof(buf), "%d tokens", tokens);
    return buf;
}

std::string format_context_used(double pct) {
    char buf[32];
    std::snprintf(buf, sizeof(buf), "%.0f%% used", pct);
    return buf;
}

std::string format_cost_spent(double usd) {
    char buf[32];
    if (usd < 0.01) {
        std::snprintf(buf, sizeof(buf), "$%.4f spent", usd);
    } else {
        std::snprintf(buf, sizeof(buf), "$%.2f spent", usd);
    }
    return buf;
}

std::string combined_assistant_text(const std::string& preamble, const std::string& followup) {
    if (preamble.empty()) {
        return followup;
    }
    if (followup.empty()) {
        return preamble;
    }
    return preamble + "\n\n" + followup;
}

struct TurnMetricsInput {
    bool valid = false;
    int64_t total_duration_ms = 0;
    int64_t wait_duration_ms = 0;
    int64_t thinking_duration_ms = 0;
    int64_t generation_duration_ms = 0;
    int output_tokens = 0;
    int input_tokens = 0;
    double estimated_cost_usd = 0.0;
    double tokens_per_sec = 0.0;
    bool had_thinking = false;
    int tool_calls = 0;
    int64_t tool_duration_ms = 0;
    int steps = 0;
    int context_tokens = 0;
    int context_window = 0;
    double context_used_pct = 0.0;
};

void fill_turn_metrics_raw(const TurnMetricsInput& metrics, taiji_turn_metrics_t& raw) {
    raw.total_duration_ms = static_cast<int32_t>(metrics.total_duration_ms);
    raw.wait_duration_ms = static_cast<int32_t>(metrics.wait_duration_ms);
    raw.thinking_duration_ms = static_cast<int32_t>(metrics.thinking_duration_ms);
    raw.generation_duration_ms = static_cast<int32_t>(metrics.generation_duration_ms);
    raw.output_tokens = metrics.output_tokens;
    raw.input_tokens = metrics.input_tokens;
    raw.estimated_cost_usd = metrics.estimated_cost_usd;
    raw.tokens_per_sec = metrics.tokens_per_sec;
    raw.had_thinking = metrics.had_thinking ? 1 : 0;
    raw.tool_calls = metrics.tool_calls;
    raw.tool_duration_ms = static_cast<int32_t>(metrics.tool_duration_ms);
    raw.steps = metrics.steps;
    raw.context_tokens = metrics.context_tokens;
    raw.context_window = metrics.context_window;
    raw.context_used_pct = metrics.context_used_pct;
}

std::string format_turn_metrics_line(uint64_t handle, const TurnMetricsInput& metrics, int max_width) {
    if (!metrics.valid || handle == 0) {
        return "";
    }
    taiji_turn_metrics_t raw{};
    fill_turn_metrics_raw(metrics, raw);
    char* text = taiji_format_turn_metrics(handle, &raw, max_width);
    if (text == nullptr) {
        return "";
    }
    std::string out = text;
    std::free(text);
    return out;
}

int chat_width_cols_from_pref(const std::string& width) {
    if (width == "narrow") {
        return 88;
    }
    if (width == "medium") {
        return 120;
    }
    if (width == "wide") {
        return 160;
    }
    return 0; // auto: use terminal width
}

tuinator::Style composer_input_style() {
    tuinator::Style style;
    style.foreground_rgb = tuinator::rgb(225, 220, 240);
    style.background_rgb = kUiInputChromeFill;
    return style;
}

tuinator::Style thinking_style() {
    tuinator::Style style;
    style.foreground_rgb = tuinator::rgb(255, 140, 64);
    style.bold = true;
    return style;
}

tuinator::ScrollbarOptions chat_scrollbar_options() {
    tuinator::Style track_style;
    track_style.foreground_rgb = tuinator::rgb(58, 52, 74);

    tuinator::Style thumb_style;
    thumb_style.foreground_rgb = tuinator::rgb(188, 168, 245);
    thumb_style.background_rgb = tuinator::rgb(78, 68, 104);

    tuinator::ScrollbarGlyphPatch glyphs;
    glyphs.vertical_track = "┊";
    glyphs.thumb = "┃";

    const auto style = tuinator::ScrollbarStyle::thin()
                           .patch_glyphs(glyphs)
                           .with_track_style(track_style)
                           .with_thumb_style(thumb_style);

    return tuinator::ScrollbarOptions::classic()
        .with_horizontal(false)
        .with_behavior(tuinator::ScrollbarBehavior::minimal())
        .with_style(style);
}

constexpr int kNavSidebarCols = 26;
constexpr int kSidebarCols = 28;
constexpr int kMainMinCols = 48;
constexpr int kPollIntervalMs = 33;

constexpr int kStreamUiRefreshMs = 33;
constexpr int kQueueUiRefreshMs = 200;

std::string truncate_preview(const std::string& text, std::size_t max_len = 42) {
    if (text.size() <= max_len) {
        return text;
    }
    return text.substr(0, max_len - 1) + "…";
}

void append_utf8(std::string& out, unsigned long codepoint) {
    if (codepoint < 0x80) {
        out.push_back(static_cast<char>(codepoint));
    } else if (codepoint < 0x800) {
        out.push_back(static_cast<char>(0xC0 | (codepoint >> 6)));
        out.push_back(static_cast<char>(0x80 | (codepoint & 0x3F)));
    } else {
        out.push_back(static_cast<char>(0xE0 | (codepoint >> 12)));
        out.push_back(static_cast<char>(0x80 | ((codepoint >> 6) & 0x3F)));
        out.push_back(static_cast<char>(0x80 | (codepoint & 0x3F)));
    }
}

std::string json_unescape(const std::string& value) {
    std::string out;
    out.reserve(value.size());
    for (std::size_t i = 0; i < value.size(); ++i) {
        if (value[i] != '\\' || i + 1 >= value.size()) {
            out.push_back(value[i]);
            continue;
        }
        const char next = value[++i];
        switch (next) {
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
        case '\\':
        case '/':
            out.push_back(next);
            break;
        case 'u': {
            // Go's json.Marshal HTML-escapes <, > and & as \u003c etc.
            if (i + 4 >= value.size()) {
                out.push_back(next);
                break;
            }
            unsigned long codepoint = 0;
            bool valid = true;
            for (int digit = 0; digit < 4; ++digit) {
                const char hex = value[i + 1 + digit];
                codepoint <<= 4;
                if (hex >= '0' && hex <= '9') {
                    codepoint |= static_cast<unsigned long>(hex - '0');
                } else if (hex >= 'a' && hex <= 'f') {
                    codepoint |= static_cast<unsigned long>(hex - 'a' + 10);
                } else if (hex >= 'A' && hex <= 'F') {
                    codepoint |= static_cast<unsigned long>(hex - 'A' + 10);
                } else {
                    valid = false;
                    break;
                }
            }
            if (!valid) {
                out.push_back(next);
                break;
            }
            i += 4;
            append_utf8(out, codepoint);
            break;
        }
        default:
            out.push_back(next);
            break;
        }
    }
    return out;
}

std::string json_extract_string(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return "";
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    if (i >= json.size() || json[i] != '"') {
        return "";
    }
    ++i;
    std::string raw;
    while (i < json.size()) {
        if (json[i] == '"') {
            return json_unescape(raw);
        }
        if (json[i] == '\\' && i + 1 < json.size()) {
            raw.push_back(json[i]);
            raw.push_back(json[i + 1]);
            i += 2;
            continue;
        }
        raw.push_back(json[i]);
        ++i;
    }
    return "";
}

std::string json_extract_value(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return "";
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    if (i >= json.size()) {
        return "";
    }
    if (json[i] == '"') {
        return json_extract_string(json, key);
    }
    if (json[i] != '{' && json[i] != '[') {
        return "";
    }
    const char open = json[i];
    const char close = open == '{' ? '}' : ']';
    int depth = 0;
    const std::size_t start = i;
    for (; i < json.size(); ++i) {
        const char ch = json[i];
        if (ch == '"') {
            ++i;
            while (i < json.size()) {
                if (json[i] == '\\' && i + 1 < json.size()) {
                    i += 2;
                    continue;
                }
                if (json[i] == '"') {
                    break;
                }
                ++i;
            }
            continue;
        }
        if (ch == open) {
            ++depth;
        } else if (ch == close) {
            --depth;
            if (depth == 0) {
                return json.substr(start, i - start + 1);
            }
        }
    }
    return "";
}

double json_extract_double(const std::string& json, const std::string& key, double fallback = 0.0) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return fallback;
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    const std::size_t begin = i;
    if (i < json.size() && (json[i] == '-' || json[i] == '+')) {
        ++i;
    }
    while (i < json.size()) {
        const char ch = json[i];
        if (std::isdigit(static_cast<unsigned char>(ch)) || ch == '.') {
            ++i;
            continue;
        }
        break;
    }
    if (i == begin) {
        return fallback;
    }
    try {
        return std::stod(json.substr(begin, i - begin));
    } catch (...) {
        return fallback;
    }
}

int64_t json_extract_int(const std::string& json, const std::string& key, int64_t fallback = -1) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return fallback;
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    const std::size_t begin = i;
    if (i < json.size() && (json[i] == '-' || json[i] == '+')) {
        ++i;
    }
    while (i < json.size() && std::isdigit(static_cast<unsigned char>(json[i]))) {
        ++i;
    }
    if (i == begin) {
        return fallback;
    }
    try {
        return std::stoll(json.substr(begin, i - begin));
    } catch (...) {
        return fallback;
    }
}

bool json_extract_bool(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return false;
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    return json.compare(i, 4, "true") == 0;
}

std::vector<std::string> json_extract_object_array(const std::string& json, const std::string& key) {
    const std::string array_json = json_extract_value(json, key);
    if (array_json.empty() || array_json.front() != '[') {
        return {};
    }

    std::vector<std::string> out;
    std::size_t i = 1;
    while (i < array_json.size()) {
        while (i < array_json.size() && (array_json[i] == ' ' || array_json[i] == '\t' || array_json[i] == ',')) {
            ++i;
        }
        if (i >= array_json.size() || array_json[i] == ']') {
            break;
        }
        if (array_json[i] != '{') {
            break;
        }
        int depth = 0;
        const std::size_t start = i;
        for (; i < array_json.size(); ++i) {
            const char ch = array_json[i];
            if (ch == '"') {
                ++i;
                while (i < array_json.size()) {
                    if (array_json[i] == '\\' && i + 1 < array_json.size()) {
                        i += 2;
                        continue;
                    }
                    if (array_json[i] == '"') {
                        break;
                    }
                    ++i;
                }
                continue;
            }
            if (ch == '{') {
                ++depth;
            } else if (ch == '}') {
                --depth;
                if (depth == 0) {
                    out.push_back(array_json.substr(start, i - start + 1));
                    ++i;
                    break;
                }
            }
        }
    }
    return out;
}

} // namespace

bool ChatApp::parse_tool_event(const std::string& json, std::size_t turn_index, ToolEventUpdate& update) {
    const std::string type = json_extract_string(json, "type");
    update.turn_index = turn_index;
    update.tool_name = json_extract_string(json, "tool_name");

    if (type == "tool_call") {
        update.kind = ToolEventUpdate::Kind::ToolCall;
        update.args = json_extract_value(json, "args");
        update.is_mcp = json_extract_bool(json, "is_mcp");
        update.is_skill = json_extract_bool(json, "is_skill");
        update.is_decision = json_extract_bool(json, "is_decision");
        update.is_ask_user = json_extract_bool(json, "is_ask_user");
        if (update.tool_name == "invoke_skill") {
            update.is_skill = true;
        }
        if (update.tool_name == "decide") {
            update.is_decision = true;
        }
        if (update.tool_name == "ask_user") {
            update.is_ask_user = true;
        }
        update.skill_name = json_extract_string(json, "skill_name");
        update.mcp_server = json_extract_string(json, "mcp_server");
        return true;
    }
    if (type == "observation") {
        update.kind = ToolEventUpdate::Kind::Observation;
        update.text = json_extract_string(json, "text");
        update.diff = json_extract_string(json, "diff");
        update.duration_ms = json_extract_int(json, "duration_ms", -1);
        update.is_mcp = json_extract_bool(json, "is_mcp");
        update.is_skill = json_extract_bool(json, "is_skill");
        update.is_decision = json_extract_bool(json, "is_decision");
        update.is_ask_user = json_extract_bool(json, "is_ask_user");
        if (update.tool_name == "invoke_skill") {
            update.is_skill = true;
        }
        if (update.tool_name == "decide") {
            update.is_decision = true;
        }
        if (update.tool_name == "ask_user") {
            update.is_ask_user = true;
        }
        update.skill_name = json_extract_string(json, "skill_name");
        const std::string data_json = json_extract_value(json, "data");
        if (update.is_skill && update.skill_name.empty() && !data_json.empty()) {
            update.skill_name = json_extract_string(data_json, "skill");
            if (update.diff.empty()) {
                update.diff = json_extract_string(data_json, "diff");
            }
        }
        if (update.is_decision && !data_json.empty()) {
            update.decision_driver = json_extract_string(data_json, "driver");
        }
        update.mcp_server = json_extract_string(json, "mcp_server");
        if (update.diff.empty()) {
            const std::string data_key = "\"data\":";
            const std::size_t data_pos = json.find(data_key);
            if (data_pos != std::string::npos) {
                update.diff = json_extract_string(json.substr(data_pos), "diff");
            }
        }
        return true;
    }
    if (type == "think_delta") {
        update.kind = ToolEventUpdate::Kind::ThinkDelta;
        update.text = json_extract_string(json, "text");
        return true;
    }
    if (type == "thought_start") {
        update.kind = ToolEventUpdate::Kind::ThoughtStart;
        return true;
    }
    if (type == "thought") {
        update.kind = ToolEventUpdate::Kind::Thought;
        update.text = json_extract_string(json, "text");
        return true;
    }
    return false;
}

namespace {

bool is_cancellation_error(const std::string& error) {
    if (error.empty()) {
        return false;
    }
    std::string lower = error;
    for (char& ch : lower) {
        ch = static_cast<char>(std::tolower(static_cast<unsigned char>(ch)));
    }
    return lower == "cancelled";
}

} // namespace

ChatApp::ChatApp(tuinator::Application& app)
    : app_(app), theme_(tuinator::dark_theme({.glyphs = tuinator::GlyphSet::Unicode})) {
    tune_chat_theme(theme_);
    app_.set_theme(theme_);
    thinking_style_ = thinking_style();
    input_style_ = composer_input_style();
    input_focused_style_ = input_style_;
    input_focused_style_.foreground_rgb = tuinator::rgb(255, 255, 255);

    footer_text_ = "Starting…";
}

ChatApp::~ChatApp() {
    if (init_thread_.joinable()) {
        init_thread_.join();
    }
    if (worker_busy_) {
        taiji_cancel(taiji_handle_);
    }
    if (worker_.joinable()) {
        worker_.join();
    }
    if (stt_thread_.joinable()) {
        stt_thread_.join();
    }
    if (taiji_handle_ != 0) {
        taiji_close(taiji_handle_);
    }
}

void ChatApp::set_root() {
    auto shell = std::make_unique<AppShell>(theme_);
    session_context_menu_ = shell->context_menu();
    settings_modal_ = shell->settings_modal();
    settings_modal_->set_handle(taiji_handle_);
    settings_modal_->set_on_preview([this](const SettingsModal::Draft& draft) {
        // Live preview while cycling chat width in settings (before Ctrl+S).
        chat_max_width_cols_ = chat_width_cols_from_pref(draft.chat_width);
        update_chat_wrap_widths();
        apply_message_time_prefs(draft.message_timestamps, draft.message_time_clock);
        ui_apply_corners_pref(draft.ui_corners);
        if (input_ != nullptr) {
            input_->sync_frame_border_style();
        }
        refresh_user_message_chrome();
        refresh_user_time_labels_on_screen();
        app_.repaint_all();
    });
    settings_modal_->set_on_closed([this](bool applied) {
        const bool menu_open = session_context_menu_ != nullptr && session_context_menu_->is_open();
        app_.set_pointer_hover_tracking(menu_open);
        // Always restore appearance prefs from the saved settings: reverts live
        // preview changes when the modal was cancelled, and picks up new values
        // after apply.
        if (taiji_handle_ != 0) {
            refresh_appearance_prefs();
        }
        if (applied && taiji_handle_ != 0) {
            refresh_input_model_info();
            refresh_context_sidebar();
            refresh_session_sidebar();
            refresh_session_nav();
            refresh_mcp_sidebar();
            refresh_skills_sidebar();
            refresh_decisions_sidebar();
            refresh_status_footer();
            refresh_speech_input_prefs();
        }
        focus_input();
        app_.repaint_all();
    });
    session_context_menu_->set_on_open_changed([this](bool open) {
        const bool settings_open = settings_modal_ != nullptr && settings_modal_->is_open();
        app_.set_pointer_hover_tracking(open || settings_open);
    });

    auto workspace_root = std::make_unique<MainPanelVBox>(tuinator::BoxOptions{.gap = 0, .padding = 0});

    auto main_column = std::make_unique<MainPanelVBox>(tuinator::BoxOptions{.gap = 1, .padding = 0});
    main_column_ = main_column.get();
    main_column->set_flex(1);

    auto chat_header = std::make_unique<ChatSessionHeader>(theme_);
    chat_header_ = chat_header.get();
    main_column->add_child(std::move(chat_header));

    auto empty_state = std::make_unique<ChatEmptyState>(theme_);
    auto chat_log = std::make_unique<MainPanelVBox>(tuinator::BoxOptions{.gap = 1, .padding = 0});
    chat_log_ = chat_log.get();

    tuinator::ScrollViewOptions scroll_options;
    scroll_options.width = 0;
    scroll_options.height = 0;
    scroll_options.scrollbars = chat_scrollbar_options();
    scroll_options.background = ui_main_panel_style();
    auto chat_scroll = std::make_unique<tuinator::ScrollView>(std::move(chat_log), scroll_options);

    auto middle_pane = std::make_unique<ChatMiddlePane>(std::move(empty_state), std::move(chat_scroll));
    middle_pane_ = middle_pane.get();
    empty_state_ = static_cast<ChatEmptyState*>(middle_pane_->empty_state());
    if (empty_state_ != nullptr) {
        empty_state_->set_on_quick_action([this](int index) { trigger_empty_quick_action(index); });
    }
    chat_scroll_ = static_cast<tuinator::ScrollView*>(middle_pane_->chat_scroll());
    middle_pane->set_flex(1);
    main_column->add_child(std::move(middle_pane));

    tuinator::TextInputOptions input_options;
    input_options.min_width = 16;
    input_options.chrome = tuinator::TextInputChrome::Framed;
    input_options.framed_border_style = ui_frame_border_style();
    input_options.max_content_lines = 5;
    input_options.prompt = "› ";
    input_options.placeholder = "Write a message…";
    auto message_composer =
        std::make_unique<MessageComposer>(input_options, input_style_, input_focused_style_, theme_);
    message_composer_ = message_composer.get();
    input_ = message_composer_->text_input();
    input_->set_on_change([this](const std::string& value) {
        if (input_ != nullptr) {
            input_->mark_dirty();
        }
        if (value.find('\n') != std::string::npos) {
            request_chat_layout_refresh();
        }
        app_.invalidate();
    });
    message_composer_->set_host_context([this]() { return build_message_composer_context(); });
    message_composer_->set_on_palette_open_changed([this](bool) {
        request_chat_layout_refresh();
        app_.repaint_all();
        focus_input();
    });
    message_composer_->set_on_submit([this](const std::string& value) {
        if (message_composer_ != nullptr) {
            apply_composer_session_prefs(*message_composer_);
        }
        submit_message(value);
        if (message_composer_ != nullptr) {
            message_composer_->clear_after_send();
        }
    });

    auto question = std::make_unique<QuestionComposer>(input_style_, input_focused_style_);
    question->set_on_submit([this](const std::string& request_id, std::vector<std::vector<std::string>> answers) {
        submit_question_answers(request_id, std::move(answers));
    });
    question->set_on_reject([this](const std::string& request_id) { reject_question_prompt(request_id); });

    auto composer_switch = std::make_unique<ComposerSwitch>(std::move(message_composer), std::move(question));
    composer_switch_ = composer_switch.get();
    if (input_ != nullptr) {
        input_->set_quick_key_handler([this](const tuinator::KeyPress& key) {
            return empty_state_ != nullptr && empty_state_->is_visible() && empty_state_->try_quick_action(key);
        });
    }
    question_composer_ = static_cast<QuestionComposer*>(composer_switch_->question_widget());
    auto bottom_bar = std::make_unique<MainPanelVBox>(tuinator::BoxOptions{.gap = 0, .padding = 0});

    tuinator::Style mic_style = input_style_;
    tuinator::Style mic_recording = input_style_;
    mic_recording.foreground_rgb = tuinator::rgb(240, 90, 90);
    tuinator::Style mic_hover = input_focused_style_;
    auto speech_mic = std::make_unique<SpeechMicButton>(
        mic_style, mic_recording, mic_hover, [this]() { toggle_speech_input(); });
    speech_mic_ = speech_mic.get();

    auto input_bar = std::make_unique<ChatInputBar>();
    input_bar->set_composer(std::move(composer_switch));
    input_bar->set_mic(std::move(speech_mic));

    auto composer = std::make_unique<MainPanelVBox>(tuinator::BoxOptions{.gap = 1, .padding = 0});
    auto queue_bar = std::make_unique<QueueStatusBar>(
        theme_, [this]() { promote_front_queued(); }, [this]() { drop_front_queued(); });
    queue_bar_ = queue_bar.get();
    composer->add_child(std::move(queue_bar));
    composer->add_child(std::move(input_bar));

    bottom_bar->add_child(std::move(composer));

    tuinator::Style footer_path_style = ui_secondary_style(theme_);
    tuinator::Style footer_status_style = theme_.label;
    auto status_footer = std::make_unique<StatusFooter>(footer_path_style, footer_status_style);
    status_footer_ = status_footer.get();
    bottom_bar->add_child(std::move(status_footer));
    main_column->add_child(std::move(bottom_bar));

    auto sidebar = std::make_unique<SidebarPanelVBox>(tuinator::BoxOptions{.gap = 0, .padding = 0});
    auto sidebar_body = std::make_unique<SidebarPanelVBox>(tuinator::BoxOptions{.gap = 0, .padding = 0});
    sidebar_body->set_flex(1);

    auto context_panel = std::make_unique<ContextPanel>(theme_);
    context_panel_ = context_panel.get();
    context_panel_->set_on_layout_changed([this]() { app_.repaint_all(); });
    shell->set_preprocess_event([this](const tuinator::Event& event) {
        const auto* key = std::get_if<tuinator::KeyPress>(&event);
        if (key == nullptr || !key->alt || key->ctrl) {
            return false;
        }
        if (key->character == 'c' || key->character == 'C') {
            toggle_context_usage_panel();
            return true;
        }
        return false;
    });
    sidebar_body->add_child(std::move(context_panel));
    refresh_context_sidebar();

    sidebar_body->add_child(std::make_unique<SidebarDivider>(theme_));
    sidebar_body->add_child(make_sidebar_heading("AGENT & LAYA", theme_));

    auto skills_row = std::make_unique<SidebarKvRow>(theme_, "Skills");
    skills_row_ = skills_row.get();
    skills_row_->set_empty_value("none");
    sidebar_body->add_child(std::move(skills_row));
    refresh_skills_sidebar();

    auto decisions_row = std::make_unique<SidebarKvRow>(theme_, "Decisions");
    decisions_row_ = decisions_row.get();
    decisions_row_->set_empty_value("none");
    sidebar_body->add_child(std::move(decisions_row));
    refresh_decisions_sidebar();

    auto lsp_row = std::make_unique<SidebarKvRow>(theme_, "LSP");
    lsp_row_ = lsp_row.get();
    lsp_row_->set_empty_value("inactive");
    sidebar_body->add_child(std::move(lsp_row));

    sidebar_body->add_child(std::make_unique<SidebarDivider>(theme_));

    auto mcp_list = std::make_unique<McpServerList>(theme_);
    mcp_list->set_flex(1);
    mcp_list_ = mcp_list.get();
    sidebar_body->add_child(std::move(mcp_list));
    refresh_mcp_sidebar();

    sidebar->add_child(std::move(sidebar_body));

    auto sidebar_footer = std::make_unique<SidebarFooter>(theme_, [this]() { open_settings_modal(); });
    sidebar_footer_ = sidebar_footer.get();
    sidebar->add_child(std::move(sidebar_footer));

    refresh_session_sidebar();
    refresh_chat_header();
    sync_empty_state();

    auto chat_split = std::make_unique<RightAnchoredSplit>(std::move(main_column), std::move(sidebar), kSidebarCols);
    chat_split->set_flex(1);

    auto session_nav = std::make_unique<SessionSidebar>(
        theme_,
        [this](const std::string& project_root, const std::string& session_id) {
            switch_to_session(project_root, session_id);
        },
        [this](const std::string& project_root) { create_new_session(project_root); },
        [this](const std::string& project_root, const std::string& session_id, tuinator::Point position) {
            show_session_context_menu(position, project_root, session_id);
        });
    session_nav_ = session_nav.get();
    session_nav->refresh(taiji_handle_);

    auto outer_split = std::make_unique<LeftAnchoredSplit>(std::move(session_nav), std::move(chat_split), kNavSidebarCols);
    outer_split->set_flex(1);
    workspace_root->add_child(std::move(outer_split));

    shell->add_child(std::move(workspace_root));
    refresh_appearance_prefs();
    app_.set_root(std::move(shell));
    sync_workspace_layout();
    refresh_input_model_info();
    refresh_status_footer();
    refresh_queue_ui();
    focus_input();
    refresh_session_nav();
    app_.set_interval(kPollIntervalMs, [this]() { poll_updates(); });
    start_background_init();
}

void ChatApp::start_background_init() {
    init_thread_ = std::thread([this]() {
        char* init_error = nullptr;
        const uint64_t handle = taiji_init(nullptr, 1, &init_error);
        std::string err;
        if (init_error != nullptr) {
            err = init_error;
            std::free(init_error);
        }
        {
            std::lock_guard<std::mutex> lock(init_mu_);
            init_handle_ = handle;
            init_error_ = std::move(err);
        }
        init_finished_.store(true);
    });
}

void ChatApp::apply_bootstrap_handle(uint64_t handle, const std::string& error) {
    if (handle == 0) {
        const char* msg = !error.empty() ? error.c_str() : taiji_init_error();
        session_label_ = std::string("Init failed: ") + (msg != nullptr ? msg : "unknown");
        footer_text_ = session_label_;
        refresh_status_footer();
        refresh_chat_header();
        app_.repaint_all();
        return;
    }

    taiji_handle_ = handle;
    taiji_set_event_callback(taiji_handle_, &ChatApp::on_event, this);
    sync_activity_handles();
    if (settings_modal_ != nullptr) {
        settings_modal_->set_handle(taiji_handle_);
    }
    footer_text_ = "Starting agent…";
    refresh_appearance_prefs();
    refresh_input_model_info();
    refresh_context_sidebar();
    refresh_session_sidebar();
    refresh_session_nav();
    refresh_mcp_sidebar();
    refresh_skills_sidebar();
    refresh_decisions_sidebar();
    refresh_status_footer();
    refresh_speech_input_prefs();
    refresh_chat_header();
    reload_turns_from_backend();
    focus_input();
    app_.repaint_all();
}

void ChatApp::apply_runtime_ready() {
    if (taiji_handle_ == 0 || runtime_ready_applied_) {
        return;
    }
    char* rt_err = taiji_runtime_error(taiji_handle_);
    if (rt_err != nullptr && rt_err[0] != '\0') {
        footer_text_ = std::string("Agent failed: ") + rt_err;
        std::free(rt_err);
        refresh_status_footer();
        app_.repaint_all();
        runtime_ready_applied_ = true;
        return;
    }
    if (rt_err != nullptr) {
        std::free(rt_err);
    }

    runtime_ready_applied_ = true;
    runtime_ready_at_ = std::chrono::steady_clock::now();
    footer_text_ = kDefaultFooterShortcuts;
    refresh_context_sidebar();
    refresh_mcp_sidebar();
    refresh_skills_sidebar();
    refresh_decisions_sidebar();
    refresh_speech_input_prefs();
    refresh_status_footer();
    app_.repaint_all();
}

void ChatApp::poll_runtime_ready() {
    if (taiji_handle_ == 0 || runtime_ready_applied_) {
        return;
    }
    if (taiji_runtime_ready(taiji_handle_) != 0) {
        apply_runtime_ready();
        return;
    }
    char* rt_err = taiji_runtime_error(taiji_handle_);
    const bool failed = rt_err != nullptr && rt_err[0] != '\0';
    if (rt_err != nullptr) {
        std::free(rt_err);
    }
    if (failed) {
        apply_runtime_ready();
    }
}

void ChatApp::poll_mcp_after_ready() {
    if (!runtime_ready_applied_ || taiji_handle_ == 0) {
        return;
    }
    const auto elapsed = std::chrono::steady_clock::now() - runtime_ready_at_;
    if (elapsed > std::chrono::seconds(8)) {
        return;
    }
    refresh_mcp_sidebar();
}

void ChatApp::poll_background_init() {
    if (init_joined_ || !init_finished_.load()) {
        return;
    }
    if (!init_thread_.joinable()) {
        return;
    }
    init_thread_.join();
    init_joined_ = true;

    uint64_t handle = 0;
    std::string error;
    {
        std::lock_guard<std::mutex> lock(init_mu_);
        handle = init_handle_;
        error = init_error_;
    }
    apply_bootstrap_handle(handle, error);
}

void ChatApp::sync_workspace_layout() {
    if (main_column_ == nullptr) {
        return;
    }

    const tuinator::Size term = app_.terminal_size();
    if (term.width <= 0) {
        return;
    }

    const bool size_changed =
        term.width != last_terminal_width_ || term.height != last_terminal_height_;
    if (!size_changed) {
        return;
    }

    last_terminal_width_ = term.width;
    last_terminal_height_ = term.height;

    if (session_nav_ != nullptr) {
        session_nav_->set_width_hint(kNavSidebarCols);
    }
    request_chat_layout_refresh();
}

void ChatApp::flush_chat_layout(bool scroll_to_bottom) {
    if (chat_scroll_ == nullptr) {
        return;
    }
    update_chat_wrap_widths();
    chat_scroll_->refresh_content();
    chat_layout_dirty_ = false;
    if (scroll_to_bottom && follow_chat_bottom_ && !chat_scroll_->pointer_active()) {
        scroll_chat_to_bottom(true);
    }
}

void ChatApp::poll_updates() {
    poll_background_init();
    poll_runtime_ready();
    poll_mcp_after_ready();
    sync_workspace_layout();
    sync_scroll_follow();

    if (chat_layout_dirty_ && chat_scroll_ != nullptr) {
        update_chat_wrap_widths();
    }

    const bool had_pending_updates = apply_pending_updates();

    if (worker_busy_ && has_active_turn_) {
        refresh_turn_view(active_turn_index_);
    }

    if (chat_layout_dirty_ || worker_busy_) {
        flush_chat_layout(worker_busy_);
    }

    tick_ui_animations();

    // Widget mark_dirty() alone does not always reach the terminal backend while
    // idle. Force frames for chat updates and for continuous UI animations (logo shimmer, etc.).
    if (had_pending_updates || worker_busy_ || chat_layout_dirty_ || ui_needs_animation_frame()) {
        app_.invalidate();
    }

    bool has_waiting = false;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        has_waiting = !job_queue_.empty();
    }

    const auto now = std::chrono::steady_clock::now();
    if (has_waiting || worker_busy_) {
        if (last_queue_ui_refresh_ == std::chrono::steady_clock::time_point{} ||
            std::chrono::duration_cast<std::chrono::milliseconds>(now - last_queue_ui_refresh_).count() >=
                kQueueUiRefreshMs) {
            refresh_queue_ui();
            last_queue_ui_refresh_ = now;
        }
    } else if (!has_waiting) {
        refresh_queue_ui(true);
    }
}

void ChatApp::focus_input() {
    app_.refresh_focus();
    if (question_prompt_active_ && question_composer_ != nullptr) {
        question_composer_->mark_dirty();
        return;
    }
    if (input_ != nullptr) {
        input_->mark_dirty();
    }
}

void ChatApp::show_question_prompt(const QuestionPromptUpdate& prompt) {
    if (question_composer_ == nullptr || composer_switch_ == nullptr) {
        return;
    }
    int wrap_width = 60;
    if (main_column_ != nullptr && main_column_->bounds().width > 0) {
        wrap_width = std::max(16, main_column_->bounds().width - 2);
    }
    question_composer_->set_wrap_width(wrap_width);
    composer_switch_->show_question();
    question_composer_->begin(prompt.request_id, prompt.questions);
    question_prompt_active_ = true;
    set_footer_status("Answer the question  ·  Esc: cancel");
    request_chat_layout_refresh();
    focus_input();
    app_.repaint_all();
}

void ChatApp::hide_question_prompt() {
    if (question_composer_ != nullptr) {
        question_composer_->clear();
    }
    if (composer_switch_ != nullptr) {
        composer_switch_->show_input();
    }
    question_prompt_active_ = false;
    set_footer_status("[ Send ↵ ]  ·  Shift+Enter: newline  ·  Esc: stop");
    request_chat_layout_refresh();
    focus_input();
    app_.repaint_all();
}

void ChatApp::submit_question_answers(const std::string& request_id,
                                      std::vector<std::vector<std::string>> answers) {
    if (taiji_handle_ == 0 || request_id.empty()) {
        hide_question_prompt();
        return;
    }
    const std::string payload = encode_question_answers_json(answers);
    char* request_id_buf = const_cast<char*>(request_id.c_str());
    char* answers_buf = const_cast<char*>(payload.c_str());
    if (taiji_question_reply(taiji_handle_, request_id_buf, answers_buf) == 0) {
        set_footer_status("Could not send answer");
    }
    hide_question_prompt();
}

void ChatApp::reject_question_prompt(const std::string& request_id) {
    if (taiji_handle_ != 0 && !request_id.empty()) {
        char* request_id_buf = const_cast<char*>(request_id.c_str());
        taiji_question_reject(taiji_handle_, request_id_buf);
    }
    hide_question_prompt();
}

void ChatApp::request_chat_layout_refresh() {
    chat_layout_dirty_ = true;
}

void ChatApp::apply_ui_theme(const std::string& theme_id) {
    tuinator::Theme next = tuinator::dark_theme({.glyphs = tuinator::GlyphSet::Unicode});
    if (theme_id == "high_contrast") {
        next = tuinator::dark_theme({.glyphs = tuinator::GlyphSet::Unicode});
        next.muted.foreground_rgb = tuinator::rgb(245, 245, 250);
        next.label.foreground_rgb = tuinator::rgb(255, 255, 255);
        next.accent.foreground_rgb = tuinator::rgb(255, 220, 80);
        next.heading.foreground_rgb = tuinator::rgb(255, 255, 255);
    } else {
        tune_chat_theme(next);
    }
    theme_ = next;
    thinking_style_ = theme_.muted;
    thinking_style_.dim = true;
    input_style_ = composer_input_style();
    input_focused_style_ = input_style_;
    input_focused_style_.foreground_rgb = theme_.accent.foreground_rgb;
    app_.set_theme(theme_);
}

void ChatApp::apply_message_time_prefs(bool enabled, const std::string& clock_pref) {
    message_timestamps_enabled_ = enabled;
    message_time_clock_ = message_time_clock_from_pref(clock_pref);
}

std::string ChatApp::user_time_label_for_turn(const ChatTurn& turn) const {
    if (!message_timestamps_enabled_) {
        return "";
    }
    if (!turn.user_sent_at.empty()) {
        return format_message_time_label(turn.user_sent_at, message_time_clock_);
    }
    return turn.user_time_label;
}

void ChatApp::refresh_user_time_labels_on_screen() {
    std::size_t count = 0;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        count = turns_.size();
    }
    for (std::size_t idx = 0; idx < count && idx < turn_views_.size(); ++idx) {
        ChatTurn turn;
        {
            std::lock_guard<std::mutex> lock(state_mu_);
            turn = turns_[idx];
        }
        if (turn_views_[idx].user != nullptr && turn.status != TurnStatus::Queued) {
            turn_views_[idx].user->set_time_label(user_time_label_for_turn(turn));
        }
    }
}

void ChatApp::refresh_appearance_prefs() {
    if (taiji_handle_ == 0) {
        return;
    }
    char scope_buf[] = "chat";
    char* raw = taiji_settings_get(taiji_handle_, scope_buf);
    if (raw == nullptr) {
        return;
    }
    const std::string json = raw;
    std::free(raw);
    const std::string theme = json_extract_string(json, "ui_theme");
    if (!theme.empty()) {
        apply_ui_theme(theme);
    }
    std::string corners = json_extract_string(json, "ui_corners");
    if (corners.empty()) {
        corners = "square";
    }
    ui_apply_corners_pref(corners);
    if (input_ != nullptr) {
        input_->sync_frame_border_style();
    }
    chat_max_width_cols_ = chat_width_cols_from_pref(json_extract_string(json, "chat_width"));
    update_chat_wrap_widths();
    bool timestamps = json_extract_bool(json, "message_timestamps");
    if (json.find("\"message_timestamps\":") == std::string::npos) {
        timestamps = true;
    }
    apply_message_time_prefs(timestamps, json_extract_string(json, "message_time_clock"));
    refresh_user_time_labels_on_screen();
    refresh_user_message_chrome();
}

void ChatApp::refresh_user_message_chrome() {
    for (const TurnView& view : turn_views_) {
        if (view.user != nullptr) {
            view.user->mark_dirty();
            view.user->mark_layout_dirty();
        }
    }
    request_chat_layout_refresh();
}

void ChatApp::refresh_speech_input_prefs() {
    if (taiji_handle_ == 0) {
        return;
    }
    char scope_buf[] = "chat";
    char* raw = taiji_settings_get(taiji_handle_, scope_buf);
    if (raw == nullptr) {
        return;
    }
    const std::string json = raw;
    std::free(raw);
    stt_enabled_ = json_extract_bool(json, "stt_enabled");
    if (json.find("\"stt_enabled\":") == std::string::npos) {
        stt_enabled_ = true;
    }
    const bool recording = json_extract_bool(json, "stt_recording");
    if (speech_mic_ != nullptr) {
        speech_mic_->set_enabled(stt_enabled_);
        speech_mic_->set_recording(recording);
    }
}

void ChatApp::toggle_speech_input() {
    if (taiji_handle_ == 0) {
        set_footer_status("Agent is still starting…");
        return;
    }
    if (!runtime_ready_applied_ || taiji_runtime_ready(taiji_handle_) == 0) {
        set_footer_status("Agent is still starting…");
        return;
    }
    if (!stt_enabled_) {
        set_footer_status("Voice input is off (Settings → Audio).");
        return;
    }
    if (stt_busy_) {
        return;
    }
    if (input_ != nullptr && speech_mic_ != nullptr && !speech_mic_->recording()) {
        stt_input_base_ = input_->value();
    }
    if (stt_thread_.joinable()) {
        stt_thread_.join();
    }
    stt_busy_ = true;
    if (speech_mic_ != nullptr) {
        speech_mic_->set_enabled(false);
    }
    set_footer_status("Speech input…");
    const uint64_t handle = taiji_handle_;
    stt_thread_ = std::thread([this, handle]() {
        char* raw = taiji_stt_toggle(handle);
        PendingSttUpdate update;
        if (raw != nullptr) {
            const std::string json = raw;
            std::free(raw);
            update.recording = json_extract_bool(json, "recording");
            update.text = json_extract_string(json, "text");
            update.error = json_extract_string(json, "error");
        } else {
            update.error = "speech input failed";
        }
        {
            std::lock_guard<std::mutex> lock(state_mu_);
            pending_stt_ = std::move(update);
            has_pending_stt_ = true;
        }
        stt_busy_ = false;
        app_.invalidate();
    });
}

void ChatApp::update_chat_wrap_widths() {
    int panel_width = kMainMinCols;
    if (chat_scroll_ != nullptr && chat_scroll_->bounds().width > 0) {
        // Match the scroll content viewport (scrollbar sits inside the scroll widget).
        panel_width = std::max(16, chat_scroll_->bounds().width - 1);
    } else if (main_column_ != nullptr) {
        const int main_width = main_column_->bounds().width;
        if (main_width > 0) {
            panel_width = main_width;
        }
    } else {
        const tuinator::Size term = app_.terminal_size();
        if (term.width > 0) {
            panel_width = std::max(8, term.width - kNavSidebarCols - kSidebarCols - 3);
        }
    }
    ui_set_chat_stream_panel_width(panel_width);
    int wrap_width = chat_stream_content_width(panel_width);
    if (chat_max_width_cols_ > 0) {
        wrap_width = std::min(wrap_width, chat_max_width_cols_);
        ui_set_chat_stream_max_cols(chat_max_width_cols_);
    } else {
        ui_set_chat_stream_max_cols(kChatStreamMaxCols);
    }
    for (const TurnView& view : turn_views_) {
        if (view.user != nullptr) {
            view.user->set_wrap_width(wrap_width);
        }
        if (view.assistant_lead != nullptr) {
            view.assistant_lead->set_wrap_width(wrap_width);
        }
        if (view.assistant != nullptr) {
            view.assistant->set_wrap_width(wrap_width);
        }
        if (view.activity != nullptr) {
            view.activity->set_wrap_width(wrap_width);
        }
        if (view.metrics != nullptr) {
            view.metrics->set_wrap_width(wrap_width);
            view.metrics->set_right_inset(0);
        }
    }
    for (std::size_t i = 0; i < turn_views_.size(); ++i) {
        refresh_turn_view(i);
    }
    if (queue_bar_ != nullptr) {
        queue_bar_->set_wrap_width(wrap_width);
    }
}

void ChatApp::submit_message(const std::string& text) {
    const std::string trimmed = [&]() {
        const auto begin = text.find_first_not_of(" \t\n\r");
        if (begin == std::string::npos) {
            return std::string{};
        }
        const auto end = text.find_last_not_of(" \t\n\r");
        return text.substr(begin, end - begin + 1);
    }();

    if (trimmed.empty() || taiji_handle_ == 0 || question_prompt_active_) {
        return;
    }
    if (!runtime_ready_applied_ || taiji_runtime_ready(taiji_handle_) == 0) {
        set_footer_status("Agent is still starting…");
        return;
    }

    if (input_ != nullptr) {
        input_->set_value("");
    }

    if (trimmed == "/skills") {
        handle_skills_command();
        return;
    }
    if (trimmed == "/decisions") {
        handle_decisions_command();
        return;
    }
    if (trimmed == "/settings") {
        open_settings_modal();
        return;
    }

    enqueue_message(trimmed);
    pump_worker();
}

void ChatApp::handle_skills_command() {
    char* raw = taiji_skills_list(taiji_handle_);
    std::string reply = raw != nullptr ? std::string(raw) : "Could not load skills.";
    if (raw != nullptr) {
        std::free(raw);
    }
    append_local_reply("/skills", reply);
}

void ChatApp::handle_decisions_command() {
    char* raw = taiji_decisions_list(taiji_handle_);
    std::string reply = raw != nullptr ? std::string(raw) : "Could not load decisions.";
    if (raw != nullptr) {
        std::free(raw);
    }
    append_local_reply("/decisions", reply);
}

void ChatApp::append_local_reply(const std::string& user, const std::string& assistant) {
    std::size_t turn_index = 0;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        turn_index = turns_.size();
        const std::string sent_at = message_time_now_iso_utc();
        turns_.push_back(ChatTurn{.user = user,
                                  .user_sent_at = sent_at,
                                  .user_time_label = format_message_time_label(sent_at, message_time_clock_),
                                  .assistant = assistant,
                                  .status = TurnStatus::Done});
    }
    ensure_turn_widgets(turn_index + 1);
    refresh_turn_view(turn_index);
    sync_empty_state();
    flush_chat_layout(true);
    focus_input();
}

void ChatApp::enqueue_message(const std::string& text) {
    std::size_t turn_index = 0;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        turn_index = turns_.size();
        const std::string sent_at = message_time_now_iso_utc();
        turns_.push_back(ChatTurn{.user = text,
                                  .user_sent_at = sent_at,
                                  .user_time_label = format_message_time_label(sent_at, message_time_clock_),
                                  .assistant = "",
                                  .status = TurnStatus::Queued});
        job_queue_.push_back(turn_index);
    }

    ensure_turn_widgets(turn_index + 1);
    refresh_turn_view(turn_index);
    sync_empty_state();
    flush_chat_layout(true);
    refresh_queue_ui(true);
    focus_input();
}

void ChatApp::pump_worker() {
    if (worker_busy_) {
        return;
    }

    std::size_t turn_index = 0;
    std::string goal;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        while (!job_queue_.empty()) {
            turn_index = job_queue_.front();
            job_queue_.pop_front();
            if (turn_index < turns_.size()) {
                goal = turns_[turn_index].user;
                break;
            }
        }
    }

    if (goal.empty()) {
        refresh_queue_ui(true);
        return;
    }

    run_job(turn_index, goal);
}

void ChatApp::run_job(std::size_t turn_index, const std::string& goal) {
    if (worker_.joinable()) {
        worker_.join();
    }

    ChatTurn running_turn;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        if (turn_index < turns_.size()) {
            turns_[turn_index].status = TurnStatus::Running;
            running_turn = turns_[turn_index];
        }
    }

    has_active_turn_ = true;
    active_turn_index_ = turn_index;
    worker_busy_ = true;
    ensure_turn_widgets(turn_index + 1);
    if (turn_index < turn_views_.size()) {
        TurnView& view = turn_views_[turn_index];
        if (view.activity != nullptr) {
            view.activity->clear();
        }
        if (view.assistant_lead != nullptr) {
            view.assistant_lead->clear();
        }
    }
    sync_turn_view(turn_index, running_turn, true, true, turn_index);
    refresh_queue_ui(true);
    flush_chat_layout(true);

    worker_ = std::thread([this, turn_index, goal]() {
        std::vector<char> goal_buf(goal.begin(), goal.end());
        goal_buf.push_back('\0');
        taiji_chat(taiji_handle_, goal_buf.data(), nullptr, &ChatApp::on_delta, &ChatApp::on_done, this);
    });
}

void ChatApp::append_assistant_delta(std::size_t turn_index, const std::string& delta) {
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        if (turn_index >= turns_.size()) {
            return;
        }
        turns_[turn_index].assistant += delta;
    }
}

void ChatApp::set_footer_status(const std::string& text) {
    if (text == last_footer_status_) {
        return;
    }
    last_footer_status_ = text;
    footer_text_ = text;
    if (status_footer_ != nullptr) {
        status_footer_->set_status(text);
    }
}

void ChatApp::refresh_input_model_info() {
    if (input_ == nullptr || taiji_handle_ == 0) {
        return;
    }

    char* model = taiji_model_name(taiji_handle_);
    char* provider = taiji_provider_label(taiji_handle_);
    const std::string model_text = model != nullptr ? model : "";
    const std::string provider_text = provider != nullptr ? provider : "";
    chat_model_label_ = format_model_display_label(model_text);
    input_->set_model_info(chat_model_label_, provider_text);
    if (model != nullptr) {
        std::free(model);
    }
    if (provider != nullptr) {
        std::free(provider);
    }
}

void ChatApp::refresh_skills_sidebar() {
    if (skills_row_ == nullptr) {
        return;
    }

    const std::vector<std::string> names = fetch_active_skill_names(taiji_handle_);
    if (names.empty()) {
        skills_row_->set_detail("");
        return;
    }

    std::string text;
    for (const std::string& name : names) {
        if (!text.empty()) {
            text.push_back('\n');
        }
        text += name;
    }
    skills_row_->set_detail(text);
}

void ChatApp::refresh_decisions_sidebar() {
    if (decisions_row_ == nullptr) {
        return;
    }
    if (taiji_handle_ == 0) {
        decisions_row_->set_detail("");
        return;
    }
    char* raw = taiji_decisions_list(taiji_handle_);
    if (raw == nullptr) {
        decisions_row_->set_detail("");
        return;
    }
    const std::string text = raw;
    std::free(raw);

    std::string preview;
    std::size_t pos = 0;
    while (pos < text.size()) {
        const std::size_t line_end = text.find('\n', pos);
        const std::string line = text.substr(pos, line_end == std::string::npos ? std::string::npos : line_end - pos);
        pos = line_end == std::string::npos ? text.size() : line_end + 1;
        if (line.rfind("- ", 0) != 0) {
            continue;
        }
        const std::string name = line.substr(2, line.find(':') == std::string::npos ? std::string::npos : line.find(':') - 2);
        if (name.empty()) {
            continue;
        }
        if (!preview.empty()) {
            preview.push_back('\n');
        }
        preview += "○ ";
        preview += name;
    }
    if (preview.empty()) {
        decisions_row_->set_detail("");
        return;
    }
    decisions_row_->set_detail(preview);
}

void ChatApp::sync_activity_handles() {
    for (TurnView& view : turn_views_) {
        if (view.activity != nullptr) {
            view.activity->set_taiji_handle(taiji_handle_);
        }
    }
}

void ChatApp::refresh_mcp_sidebar() {
    if (mcp_list_ == nullptr) {
        return;
    }

    if (taiji_handle_ == 0) {
        mcp_list_->set_servers({});
        mcp_list_->set_status_line("not connected");
        return;
    }

    const std::vector<std::string> servers = fetch_mcp_server_names(taiji_handle_);
    mcp_list_->set_servers(servers);
    if (servers.empty()) {
        mcp_list_->set_status_line("not connected");
    } else {
        mcp_list_->set_status_line("");
    }
}

void ChatApp::open_settings_modal() {
    if (settings_modal_ == nullptr || taiji_handle_ == 0) {
        return;
    }
    settings_modal_->set_handle(taiji_handle_);
    app_.set_pointer_hover_tracking(true);
    settings_modal_->open();
    app_.repaint_all();
}

void ChatApp::refresh_status_footer() {
    if (status_footer_ == nullptr) {
        return;
    }

    if (taiji_handle_ == 0) {
        status_footer_->set_path("");
        status_footer_->set_status(footer_text_);
        return;
    }

    char* work_dir = taiji_work_dir(taiji_handle_);
    const std::string path = work_dir != nullptr ? work_dir : "";
    status_footer_->set_path(path);
    status_footer_->set_status(footer_text_);
    if (work_dir != nullptr) {
        std::free(work_dir);
    }
}

void ChatApp::refresh_queue_ui(bool force) {
    std::vector<ChatTurn> turns;
    std::deque<std::size_t> queue;
    const bool worker_busy = worker_busy_;

    {
        std::lock_guard<std::mutex> lock(state_mu_);
        turns = turns_;
        queue = job_queue_;
    }

    std::string preview;
    std::size_t extra_waiting = 0;
    if (!queue.empty()) {
        const std::size_t front = queue.front();
        if (front < turns.size()) {
            preview = turns[front].user;
        }
        if (queue.size() > 1) {
            extra_waiting = queue.size() - 1;
        }
    }

    if (queue_bar_ != nullptr &&
        (force || preview != last_queue_preview_ || extra_waiting != last_queue_extra_)) {
        last_queue_preview_ = preview;
        last_queue_extra_ = extra_waiting;
        queue_bar_->set_preview(preview, extra_waiting);
        request_chat_layout_refresh();
    }

    const std::size_t waiting = queue.size();
    if (worker_busy) {
        if (waiting > 0) {
            std::ostringstream out;
            out << "Generating…  ·  " << waiting << " queued";
            set_footer_status(out.str());
        } else {
            set_footer_status("Generating…  ·  Esc to stop");
        }
    } else if (waiting > 0) {
        set_footer_status("Enter queues another message");
    } else {
        set_footer_status(kDefaultFooterShortcuts);
    }
}

void ChatApp::drop_front_queued() {
    std::size_t turn_index = 0;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        if (job_queue_.empty()) {
            return;
        }
        turn_index = job_queue_.front();
        job_queue_.pop_front();
        if (turn_index < turns_.size()) {
            turns_[turn_index].status = TurnStatus::Cancelled;
        }
    }

    refresh_turn_view(turn_index);
    request_chat_layout_refresh();
    refresh_queue_ui(true);
    focus_input();
}

void ChatApp::promote_front_queued() {
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        if (job_queue_.empty()) {
            return;
        }
    }

    if (worker_busy_) {
        taiji_cancel(taiji_handle_);
        return;
    }

    pump_worker();
}

void ChatApp::copy_turn_response(std::size_t turn_index) {
    std::string text;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        if (turn_index >= turns_.size()) {
            return;
        }
        text = combined_assistant_text(turns_[turn_index].assistant_preamble, turns_[turn_index].assistant);
    }
    if (text.empty()) {
        return;
    }
    tuinator::clipboard::set(text);
    set_footer_status("Response copied");
}

void ChatApp::step_turn_version(std::size_t turn_index, int delta) {
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        if (turn_index >= turns_.size()) {
            return;
        }
        ChatTurn& turn = turns_[turn_index];
        if (turn.versions.empty()) {
            return;
        }

        const std::size_t next =
            static_cast<std::size_t>(static_cast<int>(turn.active_version) + delta);
        if (next >= turn.versions.size()) {
            return;
        }
        turn.active_version = next;
        turn.assistant_preamble.clear();
        turn.assistant = turn.versions[turn.active_version];
    }

    refresh_turn_view(turn_index);
    flush_chat_layout(false);
}

void ChatApp::start_regenerate(std::size_t turn_index) {
    std::string goal;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        if (turn_index >= turns_.size()) {
            return;
        }
        ChatTurn& turn = turns_[turn_index];
        if (turn.user.empty()) {
            return;
        }
        goal = turn.user;

        if (turn.status == TurnStatus::Done) {
            const std::string full = combined_assistant_text(turn.assistant_preamble, turn.assistant);
            if (!full.empty() && turn.versions.empty()) {
                turn.versions.push_back(full);
                turn.active_version = 0;
            }
        }

        turn.assistant_preamble.clear();
        turn.assistant.clear();
        turn.metrics = {};
    }

    if (turn_index < turn_views_.size()) {
        TurnView& view = turn_views_[turn_index];
        if (view.activity != nullptr) {
            view.activity->clear();
        }
        if (view.assistant_lead != nullptr) {
            view.assistant_lead->clear();
        }
    }

    run_job(turn_index, goal);
}

void ChatApp::redo_turn(std::size_t turn_index) {
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        if (turn_index >= turns_.size()) {
            return;
        }
        const ChatTurn& turn = turns_[turn_index];
        if (turn.user.empty() || turn.status == TurnStatus::Queued) {
            return;
        }
    }

    if (worker_busy_) {
        pending_regenerate_turn_ = turn_index;
        taiji_cancel(taiji_handle_);
        return;
    }

    start_regenerate(turn_index);
}

bool ChatApp::ui_needs_animation_frame() const {
    if (empty_state_ != nullptr && empty_state_->is_visible()) {
        return true;
    }
    if (input_ != nullptr && input_->chrome_animating()) {
        return true;
    }
    if (input_ != nullptr && input_->needs_caret_animation()) {
        return true;
    }
    if (mcp_list_ != nullptr && mcp_list_->has_active_animation()) {
        return true;
    }
    for (const TurnView& view : turn_views_) {
        if (view.thinking != nullptr && view.thinking->active()) {
            return true;
        }
        if (view.activity != nullptr && view.activity->needs_animation_tick()) {
            return true;
        }
        if (view.assistant != nullptr && view.assistant->has_stream_animation()) {
            return true;
        }
        if (view.assistant_lead != nullptr && view.assistant_lead->has_stream_animation()) {
            return true;
        }
    }
    return false;
}

void ChatApp::tick_ui_animations() {
    if (message_composer_ != nullptr) {
        message_composer_->tick(kPollIntervalMs);
    }
    if (input_ != nullptr) {
        input_->set_chrome_busy(worker_busy_.load());
        if (message_composer_ == nullptr) {
            input_->tick_chrome(kPollIntervalMs);
        }
    }
    if (context_panel_ != nullptr) {
        context_panel_->tick(kPollIntervalMs);
    }
    if (mcp_list_ != nullptr) {
        mcp_list_->tick(kPollIntervalMs);
    }
    if (empty_state_ != nullptr && empty_state_->is_visible()) {
        empty_state_->tick(kPollIntervalMs);
    }
    for (auto& view : turn_views_) {
        if (view.thinking != nullptr && view.thinking->active()) {
            view.thinking->tick(kPollIntervalMs);
        }
        if (view.activity != nullptr && view.activity->needs_animation_tick()) {
            view.activity->tick(kPollIntervalMs);
        }
        if (view.assistant != nullptr) {
            view.assistant->tick_stream_glow(kPollIntervalMs);
        }
        if (view.assistant_lead != nullptr) {
            view.assistant_lead->tick_stream_glow(kPollIntervalMs);
        }
    }
}

void ChatApp::ensure_turn_widgets(std::size_t count) {
    if (chat_log_ == nullptr) {
        return;
    }

    while (turn_views_.size() < count) {
        auto user = std::make_unique<ChatMessage>("", theme_.accent, ChatMessage::Align::Left,
                                                  ChatMessage::Variant::UserPrompt);
        auto thinking = std::make_unique<ThinkingIndicator>(thinking_style_, thinking_style_);
        tuinator::MarkdownViewOptions assistant_options;
        assistant_options.styles = ui_chat_markdown_styles(theme_);
        auto assistant_lead = std::make_unique<AssistantMarkdown>(assistant_options);
        auto activity = std::make_unique<ActivityLog>(thinking_style_, theme_.muted);
        activity->set_taiji_handle(taiji_handle_);
        auto assistant = std::make_unique<AssistantMarkdown>(assistant_options);

        const std::size_t index = turn_views_.size();
        auto metrics = std::make_unique<TurnMetricsNote>(theme_, [this, index](TurnMetricsNote::Action action) {
            switch (action) {
            case TurnMetricsNote::Action::Redo:
                redo_turn(index);
                break;
            case TurnMetricsNote::Action::Copy:
                copy_turn_response(index);
                break;
            case TurnMetricsNote::Action::PrevVersion:
                step_turn_version(index, -1);
                break;
            case TurnMetricsNote::Action::NextVersion:
                step_turn_version(index, 1);
                break;
            }
        });

        auto block = std::make_unique<MainPanelVBox>(tuinator::BoxOptions{.gap = 1, .padding = 0});
        auto response = std::make_unique<MainPanelVBox>(tuinator::BoxOptions{.gap = 1, .padding = 0});

        TurnView view{
            .block = block.get(),
            .user = user.get(),
            .thinking = thinking.get(),
            .assistant_lead = assistant_lead.get(),
            .activity = activity.get(),
            .assistant = assistant.get(),
            .metrics = metrics.get(),
        };
        turn_views_.push_back(view);

        // Activity (route, skill search, tools) before the thinking spinner so
        // deterministic prefetch steps appear above "Thinking…" while the LLM runs.
        response->add_child(std::move(activity));
        response->add_child(std::move(thinking));
        response->add_child(std::move(assistant_lead));
        response->add_child(std::move(assistant));
        response->add_child(std::move(metrics));
        block->add_child(std::move(user));
        block->add_child(std::move(response));
        chat_log_->add_child(std::move(block));
    }
}

void ChatApp::sync_turn_view(std::size_t index, const ChatTurn& turn, bool worker_busy, bool has_active_turn,
                             std::size_t active_turn) {
    if (index >= turn_views_.size()) {
        return;
    }

    const TurnView& view = turn_views_[index];

    if (turn.status == TurnStatus::Cancelled) {
        view.user->set_text(turn.user);
        view.user->set_style(theme_.accent);
        view.thinking->set_active(false);
        if (view.activity != nullptr) {
            view.activity->clear();
        }
        if (view.assistant_lead != nullptr) {
            view.assistant_lead->clear();
        }
        view.assistant->clear();
        if (view.metrics != nullptr) {
            view.metrics->clear();
        }
        return;
    }

    if (turn.status == TurnStatus::Queued) {
        // Queued prompts are shown only in QueueStatusBar (» preview) until they run.
        view.user->set_text("");
    } else {
        view.user->set_text(turn.user);
        view.user->set_style(theme_.accent);
        view.user->set_time_label(user_time_label_for_turn(turn));
    }

    const bool is_active_turn =
        turn.status == TurnStatus::Running && worker_busy && has_active_turn && active_turn == index;
    const bool waiting_for_user = question_prompt_active_ && is_active_turn;
    const bool activity_running =
        view.activity != nullptr && view.activity->has_running() && !waiting_for_user;
    const bool tools_started = view.activity != nullptr && view.activity->has_entries();
    const bool generating = is_active_turn && !activity_running && !waiting_for_user;
    const bool waiting_for_model =
        generating && turn.assistant.empty() && turn.assistant_preamble.empty();
    // After tools finished, hide the spinner until answer tokens arrive — Route/List
    // already show progress; "Thinking" was misleading for nothink models.
    const bool show_wait_spinner = waiting_for_model && !tools_started;
    const bool showing_indicator = is_active_turn && !waiting_for_user &&
                                   (activity_running || show_wait_spinner ||
                                    (generating && !waiting_for_model));
    const bool use_activity_tail = tools_started && generating && is_active_turn && !waiting_for_user;
    view.thinking->set_active(showing_indicator && !tools_started);
    if (showing_indicator && !tools_started) {
        if (activity_running) {
            view.thinking->set_label(view.activity->running_label());
        } else if (waiting_for_model) {
            view.thinking->set_label("Loading model…");
        } else {
            view.thinking->set_label("Generating…");
        }
    }

    if (view.activity != nullptr) {
        if (tools_started || activity_running) {
            std::string title = std::string(kNfModel) + " Taiji";
            if (!chat_model_label_.empty()) {
                title += " (";
                title += chat_model_label_;
                title += ")";
            }
            view.activity->set_pipeline_title(title);
        } else {
            view.activity->set_pipeline_title("");
        }
        view.activity->set_generating_tail(use_activity_tail);
    }

    if (turn.status == TurnStatus::Queued) {
        if (view.activity != nullptr) {
            view.activity->clear();
        }
        if (view.assistant_lead != nullptr) {
            view.assistant_lead->clear();
        }
        view.assistant->clear();
        if (view.metrics != nullptr) {
            view.metrics->clear();
        }
        return;
    }

    const bool split_turn = !turn.assistant_preamble.empty();
    std::string lead_text;
    std::string followup_text;
    if (split_turn) {
        lead_text = turn.assistant_preamble;
        followup_text = turn.assistant;
    } else if (tools_started) {
        // Text that streamed before tool events were shown belongs below tool activity.
        followup_text = turn.assistant;
    } else {
        lead_text = turn.assistant;
    }

    if (view.assistant_lead != nullptr) {
        if (!lead_text.empty()) {
            view.assistant_lead->sync_from(lead_text, false);
        } else if (!view.assistant_lead->is_empty()) {
            view.assistant_lead->clear();
        }
    }

    if (waiting_for_model) {
        if (view.metrics != nullptr) {
            view.metrics->clear();
        }
        return;
    }

    if (view.assistant != nullptr) {
        if (!followup_text.empty()) {
            view.assistant->sync_from(followup_text, false);
        } else if (!view.assistant->is_empty()) {
            view.assistant->clear();
        }
    }
    if (view.metrics != nullptr) {
        const std::string full_text = combined_assistant_text(turn.assistant_preamble, turn.assistant);
        const TurnMetricsInput metrics_input{
            turn.metrics.valid,
            turn.metrics.total_duration_ms,
            turn.metrics.wait_duration_ms,
            turn.metrics.thinking_duration_ms,
            turn.metrics.generation_duration_ms,
            turn.metrics.output_tokens,
            turn.metrics.input_tokens,
            turn.metrics.estimated_cost_usd,
            turn.metrics.tokens_per_sec,
            turn.metrics.had_thinking,
            turn.metrics.tool_calls,
            turn.metrics.tool_duration_ms,
            turn.metrics.steps,
            turn.metrics.context_tokens,
            turn.metrics.context_window,
            turn.metrics.context_used_pct,
        };
        const std::size_t version_count =
            turn.versions.empty() ? (full_text.empty() ? 0 : 1) : turn.versions.size();
        const bool turn_actions = turn.status == TurnStatus::Done && !full_text.empty();
        const bool show_actions = turn_actions && taiji_turn_metrics_show_actions(taiji_handle_) != 0;
        int stats_width = 48;
        if (main_column_ != nullptr && main_column_->bounds().width > 0) {
            const ChatStreamLayout column = chat_stream_layout_for_panel(main_column_->bounds().width);
            stats_width = std::max(16, column.content_width -
                                            TurnMetricsNote::reserved_right_width(show_actions, version_count));
        }
        const std::string note = format_turn_metrics_line(taiji_handle_, metrics_input, stats_width);
        if (!show_actions && note.empty()) {
            view.metrics->clear();
        } else {
            view.metrics->set_content(note, show_actions, turn.active_version, version_count);
        }
    }
}

void ChatApp::refresh_turn_view(std::size_t index) {
    ChatTurn turn;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        if (index >= turns_.size()) {
            return;
        }
        turn = turns_[index];
    }

    ensure_turn_widgets(index + 1);
    sync_turn_view(index, turn, worker_busy_, has_active_turn_, active_turn_index_);
    request_chat_layout_refresh();
}

void ChatApp::refresh_active_turn_stream() {
    const auto now = std::chrono::steady_clock::now();
    if (last_stream_ui_refresh_ != std::chrono::steady_clock::time_point{} &&
        std::chrono::duration_cast<std::chrono::milliseconds>(now - last_stream_ui_refresh_).count() <
            kStreamUiRefreshMs) {
        return;
    }

    last_stream_ui_refresh_ = now;
    refresh_turn_view(active_turn_index_);
}

void ChatApp::refresh_chat_log() {
    if (chat_log_ == nullptr) {
        return;
    }

    std::vector<ChatTurn> turns;
    const bool worker_busy = worker_busy_;
    const bool has_active_turn = has_active_turn_;
    const std::size_t active_turn = active_turn_index_;

    {
        std::lock_guard<std::mutex> lock(state_mu_);
        turns = turns_;
    }

    ensure_turn_widgets(turns.size());
    for (std::size_t i = 0; i < turns.size(); ++i) {
        sync_turn_view(i, turns[i], worker_busy, has_active_turn, active_turn);
    }

    request_chat_layout_refresh();
}

void ChatApp::sync_scroll_follow() {
    if (chat_scroll_ == nullptr) {
        return;
    }

    const int max_y = chat_scroll_->max_scroll_y();
    follow_chat_bottom_ = max_y <= 0 || chat_scroll_->scroll_y() >= max_y - 1;
}

void ChatApp::scroll_chat_to_bottom(bool force) {
    if (chat_scroll_ == nullptr) {
        return;
    }
    if (!force && !follow_chat_bottom_) {
        return;
    }

    chat_scroll_->scroll_to(0, chat_scroll_->max_scroll_y());
    if (force) {
        follow_chat_bottom_ = true;
    }
}

MessageComposerHostContext ChatApp::build_message_composer_context() const {
    MessageComposerHostContext ctx;
    ctx.taiji_handle = taiji_handle_;
    if (taiji_handle_ != 0) {
        char* work = taiji_work_dir(taiji_handle_);
        if (work != nullptr) {
            ctx.work_dir = work;
            std::free(work);
        }
        char* model = taiji_model_name(taiji_handle_);
        if (model != nullptr) {
            ctx.model = model;
            std::free(model);
        }
        char* provider = taiji_provider_label(taiji_handle_);
        if (provider != nullptr) {
            ctx.provider = provider;
            std::free(provider);
        }
    }
    if (ctx.work_dir.empty()) {
        char* cwd = std::getenv("PWD");
        if (cwd != nullptr) {
            ctx.work_dir = cwd;
        }
    }
    ctx.git = read_git_workspace_status(ctx.work_dir);
    ctx.mcp_servers = fetch_mcp_server_names(taiji_handle_);
    ctx.mcp_enabled = ctx.mcp_servers;
    if (taiji_handle_ != 0) {
        ctx.skills = fetch_line_list(taiji_handle_, taiji_skills_list);
    }
    return ctx;
}

void ChatApp::apply_composer_session_prefs(const MessageComposer& composer) {
    if (taiji_handle_ == 0) {
        return;
    }
    const std::string& cot = composer.pending_chain_of_thought();
    const std::vector<std::string>& mcp = composer.pending_mcp_enabled();
    if (cot.empty() && mcp.empty()) {
        return;
    }
    std::ostringstream json;
    json << "{\"scope\":\"chat\"";
    if (!cot.empty()) {
        json << ",\"chain_of_thought\":\"" << cot << "\"";
    }
    if (!mcp.empty()) {
        json << ",\"mcp_enabled\":[";
        for (std::size_t i = 0; i < mcp.size(); ++i) {
            if (i > 0) {
                json << ',';
            }
            json << "\"" << mcp[i] << "\"";
        }
        json << "]";
    }
    json << "}";
    const std::string payload = json.str();
    taiji_settings_apply(taiji_handle_, const_cast<char*>(payload.c_str()));
}

void ChatApp::toggle_context_usage_panel() {
    if (context_panel_ == nullptr) {
        return;
    }
    context_panel_->toggle_usage_expanded();
    app_.repaint_all();
}

void ChatApp::refresh_context_sidebar() {
    if (context_panel_ == nullptr) {
        return;
    }

    context_panel_->set_show_cost(context_show_cost_);
    context_panel_->set_model_label(chat_model_label_);

    if (taiji_handle_ == 0) {
        context_panel_->set_stats(0, 0, 0.0, 0.0);
        context_panel_->set_usage_breakdown({});
        context_panel_->set_skills_summary("");
        context_panel_->set_use_speed_metric(false);
        return;
    }

    bool local_provider = false;
    char* provider_label = taiji_provider_label(taiji_handle_);
    if (provider_label != nullptr) {
        const std::string label(provider_label);
        local_provider = label.find("Local") != std::string::npos;
        std::free(provider_label);
    }
    context_panel_->set_use_speed_metric(local_provider);
    context_panel_->set_speed_tok_per_sec(last_gen_tok_per_sec_);

    taiji_session_stats_t stats{};
    if (taiji_session_stats(taiji_handle_, &stats) == 0) {
        return;
    }

    context_panel_->set_stats(stats.context_tokens, stats.context_window, stats.context_used_pct,
                              stats.estimated_cost_usd);
    ContextPanel::UsageBreakdown usage;
    usage.system_tokens = stats.usage_system_tokens;
    usage.tools_tokens = stats.usage_tools_tokens;
    usage.skills_tokens = stats.usage_skills_tokens;
    usage.conversation_tokens = stats.usage_conversation_tokens;
    context_panel_->set_usage_breakdown(usage);

    std::string skills_summary;
    const std::vector<std::string> active_skills = fetch_active_skill_names(taiji_handle_);
    for (std::size_t i = 0; i < active_skills.size(); ++i) {
        if (i > 0) {
            skills_summary += ", ";
        }
        skills_summary += active_skills[i];
    }
    context_panel_->set_skills_summary(skills_summary);
}

void ChatApp::refresh_chat_header() {
    if (chat_header_ == nullptr) {
        return;
    }

    std::string session_name = "New Session";
    std::string project_name;

    if (taiji_handle_ != 0) {
        char* name = taiji_session_name(taiji_handle_);
        if (name != nullptr && name[0] != '\0') {
            session_name = name;
        }
        if (name != nullptr) {
            std::free(name);
        }

        char* work_dir = taiji_work_dir(taiji_handle_);
        if (work_dir != nullptr && work_dir[0] != '\0') {
            const std::string path(work_dir);
            const std::size_t slash = path.find_last_of('/');
            project_name = slash == std::string::npos ? path : path.substr(slash + 1);
            if (empty_state_ != nullptr) {
                empty_state_->set_workspace_path(path);
            }
            std::free(work_dir);
        }
    } else if (!session_label_.empty()) {
        session_name = session_label_;
    }

    chat_header_->set_project_name(project_name);
    chat_header_->set_session_name(session_name);
}

void ChatApp::trigger_empty_quick_action(int index) {
    static constexpr const char* kPrompts[] = {
        "Bitte indexiere die Codebase dieses Projekts.",
        "Generiere Tests für die wichtigsten Module in diesem Repo.",
        "Analysiere den aktuellen Git-Diff und fasse die Änderungen zusammen.",
    };
    if (index < 0 || index >= static_cast<int>(std::size(kPrompts))) {
        return;
    }
    submit_message(kPrompts[index]);
}

void ChatApp::sync_empty_state() {
    if (empty_state_ == nullptr || chat_scroll_ == nullptr) {
        return;
    }

    bool empty = true;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        empty = turns_.empty();
    }

    if (empty_state_ != nullptr) {
        empty_state_->set_visible(empty);
    }
    if (middle_pane_ != nullptr) {
        middle_pane_->set_show_empty(empty);
    }
}

void ChatApp::refresh_session_sidebar() {
    refresh_chat_header();
}

void ChatApp::refresh_session_nav() {
    if (session_nav_ == nullptr) {
        return;
    }
    session_nav_->refresh(taiji_handle_);
}

void ChatApp::clear_turn_view(std::size_t index) {
    if (index >= turn_views_.size()) {
        return;
    }
    const TurnView& view = turn_views_[index];
    if (view.user != nullptr) {
        view.user->set_text("");
    }
    if (view.thinking != nullptr) {
        view.thinking->set_active(false);
    }
    if (view.activity != nullptr) {
        view.activity->clear();
    }
    if (view.assistant_lead != nullptr) {
        view.assistant_lead->clear();
    }
    if (view.assistant != nullptr) {
        view.assistant->clear();
    }
    if (view.metrics != nullptr) {
        view.metrics->clear();
    }
}

void ChatApp::reload_turns_from_backend() {
    if (taiji_handle_ == 0) {
        return;
    }

    char* raw = taiji_session_turns(taiji_handle_);
    if (raw == nullptr) {
        return;
    }
    const std::string json(raw);
    std::free(raw);

    std::vector<ChatTurn> loaded;
    std::vector<std::vector<std::string>> activity_by_turn;
    std::size_t i = 0;
    while (i < json.size()) {
        const std::size_t start = json.find('{', i);
        if (start == std::string::npos) {
            break;
        }
        int depth = 0;
        std::size_t end = start;
        for (; end < json.size(); ++end) {
            if (json[end] == '{') {
                ++depth;
            } else if (json[end] == '}') {
                --depth;
                if (depth == 0) {
                    break;
                }
            }
        }
        if (depth != 0) {
            break;
        }
        const std::string object = json.substr(start, end - start + 1);
        ChatTurn turn;
        turn.user = json_extract_string(object, "user");
        turn.assistant = json_extract_string(object, "assistant");
        turn.user_sent_at = json_extract_string(object, "user_sent_at");
        if (!turn.user_sent_at.empty()) {
            turn.user_time_label = format_message_time_label(turn.user_sent_at, message_time_clock_);
        }
        turn.status = TurnStatus::Done;
        const std::string metrics_json = json_extract_value(object, "metrics");
        if (!metrics_json.empty()) {
            const int64_t total_ms = json_extract_int(metrics_json, "total_duration_ms", 0);
            if (total_ms > 0) {
                turn.metrics.valid = true;
                turn.metrics.total_duration_ms = total_ms;
                turn.metrics.wait_duration_ms = json_extract_int(metrics_json, "wait_duration_ms", 0);
                turn.metrics.thinking_duration_ms = json_extract_int(metrics_json, "thinking_duration_ms", 0);
                turn.metrics.generation_duration_ms =
                    json_extract_int(metrics_json, "generation_duration_ms", 0);
                turn.metrics.output_tokens =
                    static_cast<int>(json_extract_int(metrics_json, "output_tokens", 0));
                turn.metrics.input_tokens = static_cast<int>(json_extract_int(metrics_json, "input_tokens", 0));
                turn.metrics.estimated_cost_usd =
                    json_extract_double(metrics_json, "estimated_cost_usd", 0.0);
                turn.metrics.had_thinking = json_extract_bool(metrics_json, "had_thinking");
                turn.metrics.tokens_per_sec =
                    json_extract_double(metrics_json, "tokens_per_sec", 0.0);
                turn.metrics.tool_calls = static_cast<int>(json_extract_int(metrics_json, "tool_calls", 0));
                turn.metrics.tool_duration_ms = json_extract_int(metrics_json, "tool_duration_ms", 0);
                turn.metrics.steps = static_cast<int>(json_extract_int(metrics_json, "steps", 0));
                turn.metrics.context_tokens =
                    static_cast<int>(json_extract_int(metrics_json, "context_tokens", 0));
                turn.metrics.context_window =
                    static_cast<int>(json_extract_int(metrics_json, "context_window", 0));
                turn.metrics.context_used_pct =
                    json_extract_double(metrics_json, "context_used_pct", 0.0);
            }
        }
        if (!turn.user.empty()) {
            loaded.push_back(std::move(turn));
            activity_by_turn.push_back(json_extract_object_array(object, "activity"));
        }
        i = end + 1;
    }

    {
        std::lock_guard<std::mutex> lock(state_mu_);
        turns_ = std::move(loaded);
        job_queue_.clear();
    }

    std::size_t turn_count = 0;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        turn_count = turns_.size();
    }
    ensure_turn_widgets(turn_count);
    for (std::size_t idx = 0; idx < turn_views_.size(); ++idx) {
        if (idx < turn_count) {
            ChatTurn turn;
            {
                std::lock_guard<std::mutex> lock(state_mu_);
                turn = turns_[idx];
            }
            sync_turn_view(idx, turn, false, false, 0);
            if (idx < activity_by_turn.size()) {
                for (const std::string& event_json : activity_by_turn[idx]) {
                    ToolEventUpdate update;
                    if (ChatApp::parse_tool_event(event_json, idx, update)) {
                        apply_tool_event(update);
                    }
                }
            }
        } else {
            clear_turn_view(idx);
        }
    }
    request_chat_layout_refresh();
    sync_empty_state();
    flush_chat_layout(true);
}

void ChatApp::switch_to_session(const std::string& project_root, const std::string& session_id) {
    if (taiji_handle_ == 0 || session_id.empty()) {
        return;
    }
    if (worker_busy_) {
        taiji_cancel(taiji_handle_);
    }
    if (worker_.joinable()) {
        worker_.join();
    }
    worker_busy_ = false;
    has_active_turn_ = false;

    char* error = nullptr;
    if (taiji_session_switch(taiji_handle_, const_cast<char*>(project_root.c_str()),
                             const_cast<char*>(session_id.c_str()), &error) == 0) {
        if (error != nullptr) {
            set_footer_status(std::string("Failed to switch session: ") + error);
            std::free(error);
        }
        return;
    }
    if (error != nullptr) {
        std::free(error);
    }

    reload_turns_from_backend();
    refresh_session_sidebar();
    refresh_session_nav();
    refresh_context_sidebar();
    refresh_mcp_sidebar();
    refresh_skills_sidebar();
    refresh_status_footer();
    focus_input();
}

void ChatApp::show_session_context_menu(const tuinator::Point& position, const std::string& project_root,
                                        const std::string& session_id) {
    if (session_context_menu_ == nullptr || session_id.empty()) {
        return;
    }

    SessionContextMenu* menu = session_context_menu_;

    const std::string project = project_root;
    const std::string id = session_id;
    menu->set_items({
        SessionContextMenu::Item{"Open", [this, project, id]() { switch_to_session(project, id); }},
        SessionContextMenu::Item{"Delete", [this, project, id]() { delete_session(project, id); }},
    });
    menu->show(position);
    app_.repaint_all();
}

void ChatApp::delete_session(const std::string& project_root, const std::string& session_id) {
    if (taiji_handle_ == 0 || session_id.empty()) {
        return;
    }
    if (worker_busy_) {
        taiji_cancel(taiji_handle_);
    }
    if (worker_.joinable()) {
        worker_.join();
    }
    worker_busy_ = false;
    has_active_turn_ = false;

    char* current_id = taiji_session_id(taiji_handle_);
    const bool deleting_current = current_id != nullptr && session_id == current_id;
    if (current_id != nullptr) {
        std::free(current_id);
    }

    char* error = nullptr;
    if (taiji_session_delete(taiji_handle_, const_cast<char*>(project_root.c_str()),
                             const_cast<char*>(session_id.c_str()), &error) == 0) {
        if (error != nullptr) {
            set_footer_status(std::string("Failed to delete session: ") + error);
            std::free(error);
        }
        return;
    }
    if (error != nullptr) {
        std::free(error);
    }

    if (deleting_current) {
        {
            std::lock_guard<std::mutex> lock(state_mu_);
            turns_.clear();
            job_queue_.clear();
        }
        for (std::size_t idx = 0; idx < turn_views_.size(); ++idx) {
            clear_turn_view(idx);
        }
        request_chat_layout_refresh();
        flush_chat_layout(true);
    }

    refresh_session_sidebar();
    refresh_session_nav();
    refresh_context_sidebar();
    refresh_status_footer();
    focus_input();
}

void ChatApp::create_new_session(const std::string& project_root) {
    if (taiji_handle_ == 0) {
        return;
    }
    if (worker_busy_) {
        taiji_cancel(taiji_handle_);
    }
    if (worker_.joinable()) {
        worker_.join();
    }
    worker_busy_ = false;
    has_active_turn_ = false;

    char* error = nullptr;
    const char* root_arg = project_root.empty() ? nullptr : project_root.c_str();
    if (taiji_session_new(taiji_handle_, const_cast<char*>(root_arg), &error) == 0) {
        if (error != nullptr) {
            set_footer_status(std::string("Failed to create session: ") + error);
            std::free(error);
        }
        return;
    }
    if (error != nullptr) {
        std::free(error);
    }

    {
        std::lock_guard<std::mutex> lock(state_mu_);
        turns_.clear();
        job_queue_.clear();
    }
    for (std::size_t idx = 0; idx < turn_views_.size(); ++idx) {
        clear_turn_view(idx);
    }
    request_chat_layout_refresh();
    flush_chat_layout(true);

    if (!project_root.empty() && session_nav_ != nullptr) {
        session_nav_->expand_project(project_root);
    }

    refresh_session_sidebar();
    refresh_session_nav();
    refresh_context_sidebar();
    refresh_mcp_sidebar();
    refresh_skills_sidebar();
    refresh_status_footer();
    sync_empty_state();
    focus_input();
}

void ChatApp::on_job_finished(std::size_t turn_index, bool ok, const std::string& error, const std::string& answer) {
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        if (turn_index < turns_.size()) {
            ChatTurn& turn = turns_[turn_index];
            if (ok) {
                const bool has_split = !turn.assistant_preamble.empty();
                if (!has_split && turn.assistant.empty() && !answer.empty()) {
                    turn.assistant = answer;
                } else if (has_split && turn.assistant.empty() && !answer.empty()) {
                    turn.assistant = answer;
                } else if (!has_split && turn.assistant.empty() && turn.assistant_preamble.empty()) {
                    turn.assistant = "(no response)";
                }
                turn.versions.push_back(combined_assistant_text(turn.assistant_preamble, turn.assistant));
                turn.active_version = turn.versions.size() - 1;
                turn.status = TurnStatus::Done;
                turn.metrics = {};
                if (taiji_handle_ != 0) {
                    taiji_turn_metrics_t raw{};
                    if (taiji_last_turn_metrics(taiji_handle_, &raw) != 0) {
                        turn.metrics.valid = true;
                        turn.metrics.total_duration_ms = raw.total_duration_ms;
                        turn.metrics.wait_duration_ms = raw.wait_duration_ms;
                        turn.metrics.thinking_duration_ms = raw.thinking_duration_ms;
                        turn.metrics.generation_duration_ms = raw.generation_duration_ms;
                        turn.metrics.output_tokens = raw.output_tokens;
                        turn.metrics.input_tokens = raw.input_tokens;
                        turn.metrics.estimated_cost_usd = raw.estimated_cost_usd;
                        turn.metrics.tokens_per_sec = raw.tokens_per_sec;
                        turn.metrics.had_thinking = raw.had_thinking != 0;
                        turn.metrics.tool_calls = raw.tool_calls;
                        turn.metrics.tool_duration_ms = raw.tool_duration_ms;
                        turn.metrics.steps = raw.steps;
                        turn.metrics.context_tokens = raw.context_tokens;
                        turn.metrics.context_window = raw.context_window;
                        turn.metrics.context_used_pct = raw.context_used_pct;
                    }
                }
            } else if (is_cancellation_error(error)) {
                if (turn.assistant.empty() && turn.assistant_preamble.empty()) {
                    if (!turn.versions.empty()) {
                        turn.active_version = turn.versions.size() - 1;
                        turn.assistant = turn.versions[turn.active_version];
                        turn.status = TurnStatus::Done;
                    } else {
                        turn.status = TurnStatus::Cancelled;
                        turn.assistant.clear();
                    }
                } else {
                    turn.status = TurnStatus::Done;
                }
            } else {
                turn.assistant = error.empty() ? "Error" : error;
                turn.status = TurnStatus::Done;
            }
        }
    }

    if (worker_.joinable()) {
        worker_.join();
    }

    has_active_turn_ = false;
    worker_busy_ = false;
    hide_question_prompt();
    pending_ask_user_args_.erase(turn_index);

    ensure_turn_widgets(turn_index + 1);
    if (turn_index < turn_views_.size() && turn_views_[turn_index].activity != nullptr) {
        turn_views_[turn_index].activity->finish_all();
    }

    refresh_turn_view(turn_index);
    flush_chat_layout(true);

    refresh_session_sidebar();
    refresh_session_nav();
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        if (turn_index < turns_.size() && turns_[turn_index].metrics.valid &&
            turns_[turn_index].metrics.tokens_per_sec > 0) {
            last_gen_tok_per_sec_ = turns_[turn_index].metrics.tokens_per_sec;
        }
    }
    refresh_context_sidebar();
    refresh_skills_sidebar();

    refresh_queue_ui(true);
    focus_input();

    if (!ok && !is_cancellation_error(error)) {
        set_footer_status(error.empty() ? "Request failed" : error);
    }

    if (pending_regenerate_turn_.has_value()) {
        const std::size_t regenerate_index = *pending_regenerate_turn_;
        pending_regenerate_turn_ = std::nullopt;
        start_regenerate(regenerate_index);
        return;
    }

    pump_worker();
}

void ChatApp::apply_tool_event(const ToolEventUpdate& update) {
    ensure_turn_widgets(update.turn_index + 1);
    if (update.turn_index >= turn_views_.size()) {
        return;
    }
    TurnView& view = turn_views_[update.turn_index];
    if (view.activity == nullptr) {
        return;
    }
    switch (update.kind) {
    case ToolEventUpdate::Kind::ToolCall:
        if (update.is_ask_user || update.tool_name == "ask_user") {
            pending_ask_user_args_[update.turn_index] = update.args;
            break;
        }
        view.activity->start_tool(update.tool_name, update.args, update.is_mcp, update.is_skill,
                                  update.skill_name, update.mcp_server, update.is_decision,
                                  update.decision_driver, update.is_ask_user);
        if (update.is_mcp && mcp_list_ != nullptr) {
            mcp_list_->set_active_server(update.mcp_server.empty() ? update.tool_name : update.mcp_server);
        }
        if (!update.skill_name.empty() && skills_row_ != nullptr) {
            skills_row_->set_detail(update.skill_name);
        }
        if (update.is_skill || update.tool_name == "skill_search") {
            refresh_skills_sidebar();
        }
        if (update.is_decision) {
            refresh_decisions_sidebar();
        }
        break;
    case ToolEventUpdate::Kind::Observation:
        if (update.is_ask_user || update.tool_name == "ask_user") {
            std::string args;
            if (auto it = pending_ask_user_args_.find(update.turn_index); it != pending_ask_user_args_.end()) {
                args = it->second;
                pending_ask_user_args_.erase(it);
            }
            view.activity->finish_question(args, update.text, update.duration_ms);
            break;
        }
        view.activity->finish_tool(update.tool_name, update.text, update.diff, update.duration_ms,
                                   update.skill_name, update.is_decision, update.decision_driver,
                                   update.is_ask_user);
        if (update.is_mcp && mcp_list_ != nullptr) {
            mcp_list_->clear_active_server();
        }
        if (update.is_skill) {
            refresh_skills_sidebar();
        }
        break;
    case ToolEventUpdate::Kind::ThinkDelta:
        view.activity->append_thinking(update.text);
        break;
    case ToolEventUpdate::Kind::ThoughtStart:
        view.activity->end_thinking();
        view.activity->begin_thinking();
        break;
    case ToolEventUpdate::Kind::Thought:
        view.activity->add_thought(update.text);
        break;
    }
    refresh_turn_view(update.turn_index);
    request_chat_layout_refresh();
    scroll_chat_to_bottom(false);
}

bool ChatApp::apply_pending_updates() {
    std::string stream_delta;
    std::size_t stream_turn = 0;
    std::deque<ToolEventUpdate> tool_events;
    TerminalUpdate terminal{};
    bool has_terminal = false;
    QuestionPromptUpdate question_prompt{};
    bool has_question_prompt = false;
    PendingSttUpdate stt_update{};
    bool has_stt = false;
    std::string stt_partial;
    bool has_stt_partial = false;

    bool stream_reset = false;
    std::size_t stream_reset_turn = 0;
    std::string stream_reset_text;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        stream_delta.swap(pending_stream_delta_);
        stream_turn = pending_stream_turn_;
        tool_events.swap(pending_tool_events_);
        if (has_pending_stream_reset_) {
            stream_reset = true;
            stream_reset_turn = pending_stream_reset_turn_;
            stream_reset_text = pending_stream_reset_text_;
            has_pending_stream_reset_ = false;
            pending_stream_reset_text_.clear();
        }
        if (has_pending_terminal_) {
            terminal = pending_terminal_;
            has_terminal = true;
            has_pending_terminal_ = false;
        }
        if (has_pending_question_prompt_) {
            question_prompt = pending_question_prompt_;
            has_question_prompt = true;
            has_pending_question_prompt_ = false;
        }
        if (has_pending_stt_) {
            stt_update = pending_stt_;
            has_stt = true;
            has_pending_stt_ = false;
        }
        if (has_pending_stt_partial_) {
            stt_partial = pending_stt_partial_;
            has_stt_partial = true;
            has_pending_stt_partial_ = false;
        }
    }

    bool changed = false;

    if (has_stt_partial && input_ != nullptr) {
        changed = true;
        std::string merged = stt_input_base_;
        if (!merged.empty() && !stt_partial.empty() && merged.back() != ' ' && merged.back() != '\n') {
            merged.push_back(' ');
        }
        merged += stt_partial;
        input_->set_value(merged);
        request_chat_layout_refresh();
    }

    if (has_stt) {
        changed = true;
        if (speech_mic_ != nullptr) {
            speech_mic_->set_enabled(stt_enabled_ && !stt_busy_);
            speech_mic_->set_recording(stt_update.recording);
        }
        if (speech_mic_ != nullptr) {
            speech_mic_->set_enabled(stt_enabled_);
        }
        if (!stt_update.error.empty()) {
            set_footer_status(stt_update.error);
        } else if (!stt_update.text.empty() && input_ != nullptr) {
            std::string merged = stt_input_base_;
            if (!merged.empty() && merged.back() != ' ' && merged.back() != '\n') {
                merged.push_back(' ');
            }
            merged += stt_update.text;
            input_->set_value(merged);
            stt_input_base_.clear();
            focus_input();
            set_footer_status("Transcription inserted.");
        } else if (stt_update.recording) {
            if (speech_mic_ != nullptr) {
                speech_mic_->set_recording(true);
                speech_mic_->set_enabled(stt_enabled_);
            }
            set_footer_status("Recording… text updates while you speak. Press Stop when done.");
        } else {
            refresh_status_footer();
        }
    }

    if (has_question_prompt) {
        changed = true;
        show_question_prompt(question_prompt);
    }

    if (stream_reset && !stream_reset_text.empty()) {
        changed = true;
        {
            std::lock_guard<std::mutex> lock(state_mu_);
            if (stream_reset_turn < turns_.size()) {
                ChatTurn& turn = turns_[stream_reset_turn];
                const std::string existing = turn.assistant;
                turn.assistant_preamble = stream_reset_text;
                if (!existing.empty() && existing != stream_reset_text) {
                    if (existing.size() > stream_reset_text.size() &&
                        existing.compare(0, stream_reset_text.size(), stream_reset_text) == 0) {
                        std::string suffix = existing.substr(stream_reset_text.size());
                        const std::size_t start = suffix.find_first_not_of(" \t\r\n");
                        turn.assistant = start == std::string::npos ? "" : suffix.substr(start);
                    } else {
                        turn.assistant = existing;
                    }
                } else {
                    turn.assistant.clear();
                }
            }
        }
        ensure_turn_widgets(stream_reset_turn + 1);
        if (stream_reset_turn < turn_views_.size()) {
            TurnView& view = turn_views_[stream_reset_turn];
            if (view.assistant_lead != nullptr) {
                view.assistant_lead->sync_from(stream_reset_text, true);
            }
            if (view.assistant != nullptr) {
                view.assistant->clear();
            }
        }
        refresh_turn_view(stream_reset_turn);
    }

    if (!tool_events.empty()) {
        changed = true;
    }
    for (const auto& event : tool_events) {
        apply_tool_event(event);
    }

    if (!stream_delta.empty()) {
        changed = true;
        append_assistant_delta(stream_turn, stream_delta);
        // Answer text ends any ongoing thinking phase.
        ensure_turn_widgets(stream_turn + 1);
        if (stream_turn < turn_views_.size() && turn_views_[stream_turn].activity != nullptr) {
            turn_views_[stream_turn].activity->end_thinking();
        }
        if (stream_turn == active_turn_index_) {
            refresh_active_turn_stream();
        }
    }

    if (!has_terminal) {
        return changed;
    }

    changed = true;
    switch (terminal.kind) {
    case TerminalUpdate::Kind::Done:
        on_job_finished(terminal.turn_index, true, "", terminal.answer);
        break;
    case TerminalUpdate::Kind::Error:
        on_job_finished(terminal.turn_index, false, terminal.text, "");
        break;
    }
    return changed;
}

void ChatApp::on_delta(const char* delta, void* user_data) {
    if (delta == nullptr || user_data == nullptr || delta[0] == '\0') {
        return;
    }
    auto* self = static_cast<ChatApp*>(user_data);
    if (!self->has_active_turn_) {
        return;
    }
    std::lock_guard<std::mutex> lock(self->state_mu_);
    self->pending_stream_turn_ = self->active_turn_index_.load();
    self->pending_stream_delta_.append(delta);
}

void ChatApp::on_event(const char* event_json, void* user_data) {
    if (event_json == nullptr || user_data == nullptr) {
        return;
    }
    auto* self = static_cast<ChatApp*>(user_data);
    const std::string json = event_json;
    if (json_extract_string(json, "type") == "stt_partial") {
        const std::string partial = json_extract_string(json, "text");
        if (!partial.empty()) {
            std::lock_guard<std::mutex> lock(self->state_mu_);
            self->pending_stt_partial_ = partial;
            self->has_pending_stt_partial_ = true;
        }
        return;
    }
    if (!self->has_active_turn_) {
        return;
    }
    if (json_extract_string(json, "type") == "user_prompt") {
        QuestionPromptUpdate prompt;
        if (parse_user_prompt_event(json, prompt.request_id, prompt.questions)) {
            std::lock_guard<std::mutex> lock(self->state_mu_);
            self->pending_question_prompt_ = std::move(prompt);
            self->has_pending_question_prompt_ = true;
        }
        return;
    }

    if (json_extract_string(json, "type") == "stream_reset") {
        std::lock_guard<std::mutex> lock(self->state_mu_);
        self->pending_stream_reset_turn_ = self->active_turn_index_.load();
        self->pending_stream_reset_text_ = json_extract_string(json, "text");
        self->has_pending_stream_reset_ = true;
        return;
    }

    ToolEventUpdate update;
    if (!ChatApp::parse_tool_event(json, self->active_turn_index_.load(), update)) {
        return;
    }

    std::lock_guard<std::mutex> lock(self->state_mu_);
    self->pending_tool_events_.push_back(std::move(update));
}

void ChatApp::on_done(int ok, const char* error, const char* answer, void* user_data) {
    auto* self = static_cast<ChatApp*>(user_data);
    TerminalUpdate update;
    update.turn_index = self->has_active_turn_ ? self->active_turn_index_.load() : 0;
    if (ok) {
        update.kind = TerminalUpdate::Kind::Done;
        update.answer = answer != nullptr ? answer : "";
    } else {
        update.kind = TerminalUpdate::Kind::Error;
        update.text = error != nullptr ? error : "unknown error";
    }
    std::lock_guard<std::mutex> lock(self->state_mu_);
    self->pending_terminal_ = std::move(update);
    self->has_pending_terminal_ = true;
}
