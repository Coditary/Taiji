#include "chat_app.hpp"

#include "left_label.hpp"
#include "mcp_sidebar.hpp"

#include <libtaiji.h>

#include <tuinator/platform/clipboard.hpp>

#include <cctype>
#include <cstdio>
#include <cstdlib>
#include <sstream>
#include <utility>
#include <vector>

namespace {

class ChatRoot : public tuinator::VBox {
  public:
    explicit ChatRoot(tuinator::BoxOptions options) : tuinator::VBox(options) {}

    bool wants_full_screen() const override { return true; }
};

std::unique_ptr<tuinator::VBox> make_sidebar_section(const std::string& title, const tuinator::Theme& theme) {
    auto box = std::make_unique<tuinator::VBox>(tuinator::BoxOptions{.gap = 0, .padding = 0});
    box->add_child(std::make_unique<LeftLabel>(title, theme.muted));
    return box;
}

std::unique_ptr<LeftLabel> make_sidebar_label(const std::string& text, const tuinator::Style& style) {
    return std::make_unique<LeftLabel>(text, style);
}

std::vector<std::string> fetch_mcp_server_names(uint64_t handle) {
    std::vector<std::string> names;
    if (handle == 0) {
        return names;
    }
    char* raw = taiji_mcp_servers(handle);
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

std::string format_duration_ms(int64_t ms) {
    char buf[32];
    if (ms < 1000) {
        std::snprintf(buf, sizeof(buf), "%dms", static_cast<int>(ms));
    } else {
        std::snprintf(buf, sizeof(buf), "%.1fs", static_cast<double>(ms) / 1000.0);
    }
    return buf;
}

std::string format_turn_metrics_note(bool valid, int64_t total_ms, int64_t thinking_ms, double tokens_per_sec,
                                     bool had_thinking) {
    if (!valid || total_ms <= 0) {
        return "";
    }

    std::string text = format_duration_ms(total_ms) + " total";
    if (had_thinking && thinking_ms > 0) {
        text += " · " + format_duration_ms(thinking_ms) + " thought";
    }
    if (tokens_per_sec > 0.0) {
        char buf[32];
        if (tokens_per_sec >= 100.0) {
            std::snprintf(buf, sizeof(buf), " · %.0f tok/s", tokens_per_sec);
        } else {
            std::snprintf(buf, sizeof(buf), " · %.1f tok/s", tokens_per_sec);
        }
        text += buf;
    }
    return text;
}

tuinator::Style composer_input_style() {
    tuinator::Style style;
    style.foreground_rgb = tuinator::rgb(225, 220, 240);
    style.background_rgb = tuinator::rgb(42, 38, 58);
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

constexpr int kSidebarCols = 28;
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

bool is_cancellation_error(const std::string& error) {
    if (error.empty()) {
        return false;
    }
    std::string lower = error;
    for (char& ch : lower) {
        ch = static_cast<char>(std::tolower(static_cast<unsigned char>(ch)));
    }
    return lower.find("cancel") != std::string::npos;
}

} // namespace

ChatApp::ChatApp(tuinator::Application& app)
    : app_(app), theme_(tuinator::dark_theme({.glyphs = tuinator::GlyphSet::Unicode})) {
    app_.set_theme(theme_);
    thinking_style_ = thinking_style();
    input_style_ = composer_input_style();
    input_focused_style_ = input_style_;
    input_focused_style_.foreground_rgb = tuinator::rgb(255, 255, 255);

    char* init_error = nullptr;
    taiji_handle_ = taiji_init(nullptr, 1, &init_error);
    if (taiji_handle_ == 0) {
        const char* msg = init_error != nullptr ? init_error : taiji_init_error();
        session_label_ = std::string("Init fehlgeschlagen: ") + (msg != nullptr ? msg : "unbekannt");
        footer_text_ = session_label_;
        if (init_error != nullptr) {
            std::free(init_error);
        }
    } else {
        taiji_set_event_callback(taiji_handle_, &ChatApp::on_event, this);
    }
}

ChatApp::~ChatApp() {
    if (worker_busy_) {
        taiji_cancel(taiji_handle_);
    }
    if (worker_.joinable()) {
        worker_.join();
    }
    if (taiji_handle_ != 0) {
        taiji_close(taiji_handle_);
    }
}

void ChatApp::set_root() {
    auto root = std::make_unique<ChatRoot>(tuinator::BoxOptions{.gap = 0, .padding = 0});

    auto main_column = std::make_unique<tuinator::VBox>(tuinator::BoxOptions{.gap = 0, .padding = 1});
    main_column->set_flex(1);

    auto chat_log = std::make_unique<tuinator::VBox>(tuinator::BoxOptions{.gap = 1, .padding = 0});
    chat_log_ = chat_log.get();

    tuinator::ScrollViewOptions scroll_options;
    scroll_options.width = 60;
    scroll_options.height = 20;
    scroll_options.scrollbars = chat_scrollbar_options();
    auto chat_scroll = std::make_unique<tuinator::ScrollView>(std::move(chat_log), scroll_options);
    chat_scroll_ = chat_scroll.get();
    chat_scroll->set_flex(1);
    main_column->add_child(std::move(chat_scroll));

    tuinator::TextInputOptions input_options;
    input_options.min_width = 16;
    input_options.chrome = tuinator::TextInputChrome::Framed;
    input_options.max_content_lines = 5;
    input_options.prompt = "› ";
    input_options.placeholder = "Nachricht schreiben…";
    auto input = std::make_unique<ChatTextInput>(input_options, input_style_, input_focused_style_);
    input_ = input.get();
    input->set_flex(1);
    input->set_on_change([this](const std::string&) { request_chat_layout_refresh(); });
    input->set_on_submit([this](const std::string& value) { submit_message(value); });

    auto bottom_bar = std::make_unique<tuinator::VBox>(tuinator::BoxOptions{.gap = 0, .padding = 0});

    auto input_row = std::make_unique<tuinator::HBox>(tuinator::BoxOptions{.gap = 0, .padding = 0});
    input_row->add_child(std::move(input));

    auto composer = std::make_unique<tuinator::VBox>(tuinator::BoxOptions{.gap = 1, .padding = 0});
    auto queue_bar = std::make_unique<QueueStatusBar>(
        theme_, [this]() { promote_front_queued(); }, [this]() { drop_front_queued(); });
    queue_bar_ = queue_bar.get();
    composer->add_child(std::move(queue_bar));
    composer->add_child(std::move(input_row));

    bottom_bar->add_child(std::move(composer));

    auto status_footer = std::make_unique<StatusFooter>(theme_.muted, theme_.muted);
    status_footer_ = status_footer.get();
    bottom_bar->add_child(std::move(status_footer));
    main_column->add_child(std::move(bottom_bar));

    auto sidebar = std::make_unique<tuinator::VBox>(tuinator::BoxOptions{.gap = 1, .padding = 0});

    auto session_box = std::make_unique<tuinator::VBox>(tuinator::BoxOptions{.gap = 0, .padding = 0});
    auto session_title = make_sidebar_label("Neue Session", theme_.heading);
    session_title_label_ = session_title.get();
    session_box->add_child(std::move(session_title));
    sidebar->add_child(std::move(session_box));
    refresh_session_sidebar();

    auto context_box = make_sidebar_section("Context", theme_);
    auto context_tokens = make_sidebar_label("0 tokens", theme_.label);
    context_tokens_label_ = context_tokens.get();
    context_box->add_child(std::move(context_tokens));
    auto context_used = make_sidebar_label("0% used", theme_.label);
    context_used_label_ = context_used.get();
    context_box->add_child(std::move(context_used));
    auto context_cost = make_sidebar_label("$0.00 spent", theme_.label);
    context_cost_label_ = context_cost.get();
    context_box->add_child(std::move(context_cost));
    sidebar->add_child(std::move(context_box));
    refresh_context_sidebar();

    auto mcp_box = make_sidebar_section("MCP", theme_);
    const std::vector<std::string> mcp_servers = fetch_mcp_server_names(taiji_handle_);
    if (mcp_servers.empty()) {
        auto mcp_status = make_sidebar_label("(nicht verbunden)", theme_.label);
        mcp_status_label_ = mcp_status.get();
        mcp_box->add_child(std::move(mcp_status));
        refresh_mcp_sidebar();
    } else {
        for (const std::string& name : mcp_servers) {
            mcp_box->add_child(make_mcp_server_row(name, theme_));
        }
    }
    sidebar->add_child(std::move(mcp_box));

    auto lsp_box = make_sidebar_section("LSP", theme_);
    lsp_box->add_child(make_sidebar_label("deaktiviert", theme_.muted));
    sidebar->add_child(std::move(lsp_box));

    sidebar->add_child(make_sidebar_label("taiji-tui", theme_.muted));

    auto workspace = std::make_unique<tuinator::SplitPane>(
        std::move(main_column), std::move(sidebar),
        tuinator::SplitPaneOptions{
            .orientation = tuinator::SplitOrientation::Horizontal,
            .first_size = 1,
            .outer_border = false,
        });
    workspace->set_flex(1);
    workspace_ = workspace.get();
    root->add_child(std::move(workspace));

    app_.set_root(std::move(root));
    sync_workspace_layout();
    refresh_input_model_info();
    refresh_status_footer();
    refresh_queue_ui();
    focus_input();
    app_.set_interval(kPollIntervalMs, [this]() { poll_updates(); });
}

void ChatApp::sync_workspace_layout() {
    if (workspace_ == nullptr) {
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

    const int main_cols = std::max(48, term.width - kSidebarCols - 1);
    tuinator::SplitPaneOptions opts = workspace_->options();
    opts.first_size = main_cols;
    workspace_->set_options(opts);
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
    sync_workspace_layout();
    sync_scroll_follow();

    if (chat_layout_dirty_ && chat_scroll_ != nullptr) {
        update_chat_wrap_widths();
    }

    apply_pending_updates();

    if (worker_busy_ && has_active_turn_) {
        refresh_turn_view(active_turn_index_);
    }

    if (chat_layout_dirty_ || worker_busy_) {
        flush_chat_layout(worker_busy_);
    }

    refresh_thinking_indicators();

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
    if (input_ == nullptr) {
        return;
    }
    if (!input_->is_focused()) {
        app_.refresh_focus();
    }
    input_->mark_dirty();
}

void ChatApp::request_chat_layout_refresh() {
    chat_layout_dirty_ = true;
}

void ChatApp::update_chat_wrap_widths() {
    const tuinator::Size term = app_.terminal_size();
    if (term.width <= 0) {
        return;
    }

    const int wrap_width = std::max(8, std::max(48, term.width - kSidebarCols - 1) - 2);
    for (const TurnView& view : turn_views_) {
        if (view.user != nullptr) {
            view.user->set_wrap_width(wrap_width);
        }
        if (view.assistant != nullptr) {
            view.assistant->set_wrap_width(wrap_width);
        }
        if (view.activity != nullptr) {
            view.activity->set_wrap_width(wrap_width);
        }
        if (view.metrics != nullptr) {
            view.metrics->set_wrap_width(wrap_width);
            view.metrics->set_right_inset(2);
        }
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

    if (trimmed.empty() || taiji_handle_ == 0) {
        return;
    }

    if (input_ != nullptr) {
        input_->set_value("");
    }

    enqueue_message(trimmed);
    pump_worker();
}

void ChatApp::enqueue_message(const std::string& text) {
    std::size_t turn_index = 0;
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        turn_index = turns_.size();
        turns_.push_back(ChatTurn{.user = text, .assistant = "", .status = TurnStatus::Queued});
        job_queue_.push_back(turn_index);
    }

    ensure_turn_widgets(turn_index + 1);
    refresh_turn_view(turn_index);
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
    if (turn_index < turn_views_.size() && turn_views_[turn_index].activity != nullptr) {
        turn_views_[turn_index].activity->clear();
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
    input_->set_model_info(model_text, provider_text);
    if (model != nullptr) {
        std::free(model);
    }
    if (provider != nullptr) {
        std::free(provider);
    }
}

void ChatApp::refresh_mcp_sidebar() {
    if (mcp_status_label_ == nullptr) {
        return;
    }

    if (taiji_handle_ == 0) {
        mcp_status_label_->set_text("(nicht verbunden)");
        return;
    }

    char* status = taiji_mcp_status(taiji_handle_);
    if (status != nullptr) {
        mcp_status_label_->set_text(status);
        std::free(status);
    } else {
        mcp_status_label_->set_text("(nicht verbunden)");
    }
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
            out << "Generiert…  ·  " << waiting << " in Warteschlange";
            set_footer_status(out.str());
        } else {
            set_footer_status("Generiert…  ·  Esc unterbricht");
        }
    } else if (waiting > 0) {
        set_footer_status("Enter reiht weitere ein");
    } else {
        set_footer_status("Enter senden");
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
        text = turns_[turn_index].assistant;
    }
    if (text.empty()) {
        return;
    }
    tuinator::clipboard::set(text);
    set_footer_status("Antwort kopiert");
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

        if (turn.status == TurnStatus::Done && !turn.assistant.empty()) {
            if (turn.versions.empty()) {
                turn.versions.push_back(turn.assistant);
                turn.active_version = 0;
            }
        }

        turn.assistant.clear();
        turn.metrics = {};
    }

    if (turn_index < turn_views_.size() && turn_views_[turn_index].activity != nullptr) {
        turn_views_[turn_index].activity->clear();
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

void ChatApp::refresh_thinking_indicators() {
    for (auto& view : turn_views_) {
        if (view.thinking != nullptr && view.thinking->active()) {
            view.thinking->tick(kPollIntervalMs);
        }
        if (view.activity != nullptr && view.activity->has_running()) {
            view.activity->tick(kPollIntervalMs);
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
        auto activity = std::make_unique<ActivityLog>(thinking_style_, theme_.muted);
        tuinator::MarkdownViewOptions assistant_options;
        assistant_options.styles = tuinator::markdown_styles_from_theme(theme_);
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

        TurnView view{
            .user = user.get(),
            .thinking = thinking.get(),
            .activity = activity.get(),
            .assistant = assistant.get(),
            .metrics = metrics.get(),
        };
        turn_views_.push_back(view);

        chat_log_->add_child(std::move(user));
        chat_log_->add_child(std::move(thinking));
        chat_log_->add_child(std::move(activity));
        chat_log_->add_child(std::move(assistant));
        chat_log_->add_child(std::move(metrics));
    }
}

void ChatApp::sync_turn_view(std::size_t index, const ChatTurn& turn, bool worker_busy, bool has_active_turn,
                             std::size_t active_turn) {
    if (index >= turn_views_.size()) {
        return;
    }

    const TurnView& view = turn_views_[index];

    if (turn.status == TurnStatus::Cancelled) {
        view.user->set_text("");
        view.thinking->set_active(false);
        if (view.activity != nullptr) {
            view.activity->clear();
        }
        view.assistant->clear();
        if (view.metrics != nullptr) {
            view.metrics->clear();
        }
        return;
    }

    if (turn.status == TurnStatus::Queued) {
        view.user->set_text(turn.user);
        view.user->set_style(theme_.muted);
    } else {
        view.user->set_text(turn.user);
        view.user->set_style(theme_.accent);
    }

    const bool is_active_turn =
        turn.status == TurnStatus::Running && worker_busy && has_active_turn && active_turn == index;
    const bool activity_running = view.activity != nullptr && view.activity->has_running();
    const bool generating = is_active_turn && !activity_running;
    const bool thinking = generating && turn.assistant.empty();
    const bool showing_indicator = is_active_turn && (activity_running || generating);
    view.thinking->set_active(showing_indicator);
    if (showing_indicator) {
        if (activity_running) {
            view.thinking->set_label(view.activity->running_label());
        } else if (thinking) {
            view.thinking->set_label("Thinking");
        } else {
            view.thinking->set_label("Writing…");
        }
    }

    if (turn.status == TurnStatus::Queued) {
        if (view.activity != nullptr) {
            view.activity->clear();
        }
        view.assistant->clear();
        if (view.metrics != nullptr) {
            view.metrics->clear();
        }
        return;
    }

    if (thinking) {
        view.assistant->clear();
        if (view.metrics != nullptr) {
            view.metrics->clear();
        }
        return;
    }

    view.assistant->sync_from(turn.assistant, false);
    if (view.metrics != nullptr) {
        const std::string note =
            format_turn_metrics_note(turn.metrics.valid, turn.metrics.total_duration_ms,
                                     turn.metrics.thinking_duration_ms, turn.metrics.tokens_per_sec,
                                     turn.metrics.had_thinking);
        const std::size_t version_count =
            turn.versions.empty() ? (turn.assistant.empty() ? 0 : 1) : turn.versions.size();
        const bool show_actions = turn.status == TurnStatus::Done && !turn.assistant.empty();
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

void ChatApp::refresh_context_sidebar() {
    if (context_tokens_label_ == nullptr && context_used_label_ == nullptr && context_cost_label_ == nullptr) {
        return;
    }

    if (taiji_handle_ == 0) {
        if (context_tokens_label_ != nullptr) {
            context_tokens_label_->set_text("0 tokens");
        }
        if (context_used_label_ != nullptr) {
            context_used_label_->set_text("0% used");
        }
        if (context_cost_label_ != nullptr) {
            context_cost_label_->set_text("$0.00 spent");
        }
        return;
    }

    taiji_session_stats_t stats{};
    if (taiji_session_stats(taiji_handle_, &stats) == 0) {
        return;
    }

    if (context_tokens_label_ != nullptr) {
        context_tokens_label_->set_text(format_token_count(stats.context_tokens));
    }
    if (context_used_label_ != nullptr) {
        context_used_label_->set_text(format_context_used(stats.context_used_pct));
    }
    if (context_cost_label_ != nullptr) {
        context_cost_label_->set_text(format_cost_spent(stats.estimated_cost_usd));
    }
}

void ChatApp::refresh_session_sidebar() {
    if (session_title_label_ == nullptr) {
        return;
    }

    if (taiji_handle_ == 0) {
        if (!session_label_.empty()) {
            session_title_label_->set_text(session_label_);
        }
        return;
    }

    char* name = taiji_session_name(taiji_handle_);
    if (name != nullptr && name[0] != '\0') {
        session_title_label_->set_text(name);
    } else {
        session_title_label_->set_text("Neue Session");
    }
    if (name != nullptr) {
        std::free(name);
    }
}

void ChatApp::on_job_finished(std::size_t turn_index, bool ok, const std::string& error, const std::string& answer) {
    {
        std::lock_guard<std::mutex> lock(state_mu_);
        if (turn_index < turns_.size()) {
            ChatTurn& turn = turns_[turn_index];
            if (ok) {
                if (turn.assistant.empty() && !answer.empty()) {
                    turn.assistant = answer;
                } else if (turn.assistant.empty()) {
                    turn.assistant = "(keine Antwort)";
                }
                turn.versions.push_back(turn.assistant);
                turn.active_version = turn.versions.size() - 1;
                turn.status = TurnStatus::Done;
                turn.metrics = {};
                if (taiji_handle_ != 0) {
                    taiji_turn_metrics_t raw{};
                    if (taiji_last_turn_metrics(taiji_handle_, &raw) != 0) {
                        turn.metrics.valid = true;
                        turn.metrics.total_duration_ms = raw.total_duration_ms;
                        turn.metrics.thinking_duration_ms = raw.thinking_duration_ms;
                        turn.metrics.tokens_per_sec = raw.tokens_per_sec;
                        turn.metrics.had_thinking = raw.had_thinking != 0;
                    }
                }
            } else if (is_cancellation_error(error)) {
                if (turn.assistant.empty()) {
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
                turn.assistant = error.empty() ? "Fehler" : error;
                turn.status = TurnStatus::Done;
            }
        }
    }

    if (worker_.joinable()) {
        worker_.join();
    }

    has_active_turn_ = false;
    worker_busy_ = false;

    ensure_turn_widgets(turn_index + 1);
    if (turn_index < turn_views_.size() && turn_views_[turn_index].activity != nullptr) {
        turn_views_[turn_index].activity->finish_all();
    }

    refresh_turn_view(turn_index);
    flush_chat_layout(true);

    refresh_session_sidebar();
    refresh_context_sidebar();

    refresh_queue_ui(true);
    focus_input();

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
        view.activity->start_tool(update.tool_name, update.args, update.is_mcp, update.mcp_server);
        break;
    case ToolEventUpdate::Kind::Observation:
        view.activity->finish_tool(update.tool_name, update.text, update.diff, update.duration_ms);
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

void ChatApp::apply_pending_updates() {
    std::string stream_delta;
    std::size_t stream_turn = 0;
    std::deque<ToolEventUpdate> tool_events;
    TerminalUpdate terminal{};
    bool has_terminal = false;

    {
        std::lock_guard<std::mutex> lock(state_mu_);
        stream_delta.swap(pending_stream_delta_);
        stream_turn = pending_stream_turn_;
        tool_events.swap(pending_tool_events_);
        if (has_pending_terminal_) {
            terminal = pending_terminal_;
            has_terminal = true;
            has_pending_terminal_ = false;
        }
    }

    for (const auto& event : tool_events) {
        apply_tool_event(event);
    }

    if (!stream_delta.empty()) {
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
        return;
    }

    switch (terminal.kind) {
    case TerminalUpdate::Kind::Done:
        on_job_finished(terminal.turn_index, true, "", terminal.answer);
        break;
    case TerminalUpdate::Kind::Error:
        on_job_finished(terminal.turn_index, false, terminal.text, "");
        break;
    }
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
    if (!self->has_active_turn_) {
        return;
    }

    const std::string json = event_json;
    const std::string type = json_extract_string(json, "type");
    ToolEventUpdate update;
    update.turn_index = self->active_turn_index_.load();
    update.tool_name = json_extract_string(json, "tool_name");

    if (type == "tool_call") {
        update.kind = ToolEventUpdate::Kind::ToolCall;
        update.args = json_extract_string(json, "args");
        update.is_mcp = json_extract_bool(json, "is_mcp");
        update.mcp_server = json_extract_string(json, "mcp_server");
    } else if (type == "observation") {
        update.kind = ToolEventUpdate::Kind::Observation;
        update.text = json_extract_string(json, "text");
        update.diff = json_extract_string(json, "diff");
        update.duration_ms = json_extract_int(json, "duration_ms", -1);
        update.is_mcp = json_extract_bool(json, "is_mcp");
        update.mcp_server = json_extract_string(json, "mcp_server");
        if (update.diff.empty()) {
            const std::string data_key = "\"data\":";
            const std::size_t data_pos = json.find(data_key);
            if (data_pos != std::string::npos) {
                update.diff = json_extract_string(json.substr(data_pos), "diff");
            }
        }
    } else if (type == "think_delta") {
        update.kind = ToolEventUpdate::Kind::ThinkDelta;
        update.text = json_extract_string(json, "text");
    } else if (type == "thought_start") {
        update.kind = ToolEventUpdate::Kind::ThoughtStart;
    } else if (type == "thought") {
        update.kind = ToolEventUpdate::Kind::Thought;
        update.text = json_extract_string(json, "text");
    } else {
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
