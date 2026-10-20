#include "message_composer.hpp"

#include "skim_fuzzy.hpp"
#include "ui_chat_column.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cstdio>
#include <fstream>
#include <sstream>

namespace {

void replace_agent_mode_chip(std::vector<ComposerChip>& chips, ComposerChip chip) {
    chips.erase(std::remove_if(chips.begin(), chips.end(),
                               [](const ComposerChip& c) { return c.kind == ComposerChipKind::AgentMode; }),
                chips.end());
    chips.push_back(std::move(chip));
}

} // namespace

MessageComposer::MessageComposer(tuinator::TextInputOptions input_options, tuinator::Style style,
                                 tuinator::Style focused_style, tuinator::Theme theme)
    : palette_(theme), theme_(std::move(theme)) {
    input_options.prompt = "  ";
    input_ = std::make_unique<ChatTextInput>(input_options, style, focused_style);
    input_->set_on_submit([this](const std::string& value) {
        if (palette_.is_open()) {
            return;
        }
        const std::string composed = composer_build_outgoing_message(value, chips_);
        if (on_submit_ && !composed.empty()) {
            on_submit_(composed);
        }
    });

    palette_.set_on_select([this](const std::string& id) { handle_palette_select(id); });
    palette_.set_on_close([this]() {
        if (on_palette_open_changed_) {
            on_palette_open_changed_(false);
        }
        mark_layout_dirty();
    });
}

void MessageComposer::set_on_submit(SubmitHandler handler) {
    on_submit_ = std::move(handler);
}

void MessageComposer::set_host_context(HostContextProvider provider) {
    host_context_ = std::move(provider);
}

void MessageComposer::set_on_palette_open_changed(std::function<void(bool)> handler) {
    on_palette_open_changed_ = std::move(handler);
}

void MessageComposer::sync_field_styles(tuinator::Style normal, tuinator::Style focused) {
    if (input_ != nullptr) {
        input_->sync_field_styles(std::move(normal), std::move(focused));
        input_->sync_frame_border_style();
    }
}

void MessageComposer::notify_input_changed(const std::string& /*value*/) {
    sync_trigger_from_input();
    mark_dirty();
}

void MessageComposer::clear_after_send() {
    chips_.clear();
    pending_cot_.clear();
    pending_mcp_enabled_.clear();
    if (input_ != nullptr) {
        input_->set_value("");
    }
    mark_dirty();
}

void MessageComposer::tick(int dt_ms) {
    plus_anim_ms_ += dt_ms;
    palette_.tick(dt_ms);
    if (input_ != nullptr) {
        input_->tick_chrome(dt_ms);
    }
}

tuinator::Size MessageComposer::preferred_size() const {
    int h = input_ != nullptr ? input_->preferred_size().height : 3;
    if (!chips_.empty()) {
        h += 1;
    }
    if (palette_.is_open()) {
        h += palette_.preferred_size().height;
    }
    return {0, h};
}

tuinator::Rect MessageComposer::stream_bounds(const tuinator::Rect& bounds) const {
    const int panel_w = ui_chat_stream_panel_width() > 0 ? ui_chat_stream_panel_width() : bounds.width;
    const ChatStreamLayout column = chat_stream_layout_for_panel(panel_w);
    if (bounds.width <= column.content_width + 4) {
        return bounds;
    }
    return {bounds.x + column.offset_x, bounds.y, column.content_width, bounds.height};
}

void MessageComposer::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    const tuinator::Rect stream = stream_bounds(bounds);
    int y = stream.y;
    if (palette_.is_open()) {
        const tuinator::Size pal = palette_.preferred_size();
        palette_.layout({stream.x, y, stream.width, pal.height});
        y += pal.height;
    } else {
        palette_.layout({0, 0, 0, 0});
    }

    chips_bounds_ = {};
    if (!chips_.empty()) {
        chips_bounds_ = {stream.x, y, stream.width, 1};
        y += 1;
    }

    const int input_h = std::max(3, bounds.height - (y - bounds.y));
    input_bounds_ = {stream.x, y, stream.width, input_h};
    if (input_ != nullptr) {
        input_->layout(input_bounds_);
    }
}

tuinator::Rect MessageComposer::plus_bounds() const {
    if (input_bounds_.width <= 0 || input_bounds_.height <= 0) {
        return {};
    }
    return {input_bounds_.x + 1, input_bounds_.y + 1, 3, 1};
}

bool MessageComposer::plus_hit(tuinator::Point local) const {
    const tuinator::Rect plus = plus_bounds();
    return plus.width > 0 && plus.contains(local);
}

void MessageComposer::paint_plus_button(tuinator::PaintContext& ctx, tuinator::Rect area) const {
    if (area.width <= 0) {
        return;
    }
    tuinator::Style style;
    style.background_rgb = ui_input_chrome_fill();
    style.foreground_rgb = palette_.is_open() ? ui_danger() : ui_accent();
    style.bold = true;
    const std::string glyph = palette_.is_open() ? "×" : std::string(kNfAdd);
    ctx.canvas.fill_rect(area, ' ', style);
    ctx.canvas.draw_text({area.x, area.y}, glyph, style);
}

void MessageComposer::paint_chips(tuinator::PaintContext& ctx, tuinator::Rect area) const {
    if (area.width <= 0 || chips_.empty()) {
        return;
    }
    tuinator::Style bg;
    bg.background_rgb = ui_main_bg();
    ctx.canvas.fill_rect(area, ' ', bg);

    int x = area.x + 1;
    for (std::size_t i = 0; i < chips_.size(); ++i) {
        const ComposerChip& chip = chips_[i];
        tuinator::Style pill = ui_resolve_main_text(theme_.label);
        pill.background_rgb = ui_badge_bg();
        pill.foreground_rgb = chip.color;
        std::string text = " " + chip.label + " × ";
        const int w = tuinator::text_display_width(text);
        if (x + w > area.x + area.width) {
            break;
        }
        ctx.canvas.draw_text({x, area.y}, text, pill);
        x += w;
    }
}

void MessageComposer::paint(tuinator::PaintContext& ctx) const {
    if (palette_.is_open()) {
        palette_.paint(ctx);
    }
    if (chips_bounds_.height > 0) {
        paint_chips(ctx, chips_bounds_);
    }
    if (input_bounds_.width > 0 && input_bounds_.height > 0) {
        tuinator::Style chrome_fill;
        chrome_fill.background_rgb = ui_input_chrome_fill();
        ctx.canvas.fill_rect(input_bounds_, ' ', chrome_fill);
    }
    if (input_ != nullptr) {
        input_->paint(ctx);
        paint_plus_button(ctx, plus_bounds());
    }
}

bool MessageComposer::handle_event(const tuinator::Event& event) {
    if (palette_.is_open() && palette_.handle_event(event)) {
        mark_dirty();
        return true;
    }

    if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
        const tuinator::Point local{mouse->position.x - bounds_.x, mouse->position.y - bounds_.y};
        if (mouse->button == tuinator::MouseButton::Left && mouse->action == tuinator::MouseAction::Release) {
            if (plus_hit(local)) {
                if (palette_.is_open()) {
                    close_palette();
                } else {
                    open_palette_with_trigger("");
                }
                return true;
            }
            if (chips_bounds_.contains(local) && !chips_.empty()) {
                int x = chips_bounds_.x + 1;
                for (std::size_t i = 0; i < chips_.size(); ++i) {
                    std::string text = " " + chips_[i].label + " × ";
                    const int w = tuinator::text_display_width(text);
                    if (local.x >= x && local.x < x + w) {
                        remove_chip_at(i);
                        return true;
                    }
                    x += w;
                }
            }
        }
    }

    if (input_ != nullptr && input_->handle_event(event)) {
        return true;
    }
    return false;
}

tuinator::Widget* MessageComposer::hit_test(tuinator::Point point) {
    if (!bounds_.contains(point)) {
        return nullptr;
    }
    if (palette_.is_open()) {
        if (palette_.hit_test(point)) {
            return this;
        }
    }
    if (input_ != nullptr) {
        return input_->hit_test(point);
    }
    return this;
}

tuinator::Widget* MessageComposer::hit_test_focusable(tuinator::Point point) {
    if (palette_.is_open()) {
        return this;
    }
    if (input_ != nullptr) {
        return input_->hit_test_focusable(point);
    }
    return nullptr;
}

void MessageComposer::collect_focusable(std::vector<tuinator::Widget*>& out) {
    if (input_ != nullptr) {
        input_->collect_focusable(out);
    }
}

void MessageComposer::sync_trigger_from_input() {
    if (input_ == nullptr) {
        return;
    }
    const std::string& value = input_->value();
    if (value.empty()) {
        return;
    }
    if (value[0] == '/' || value[0] == '@') {
        std::string filter = value.substr(1);
        if (!palette_.is_open()) {
            open_palette_with_trigger(filter);
        } else {
            palette_.set_filter(filter);
        }
    }
}

void MessageComposer::open_palette_with_trigger(const std::string& filter) {
    if (host_context_) {
        ctx_cache_ = host_context_();
    }
    if (!file_index_built_ && !ctx_cache_.work_dir.empty()) {
        file_index_ = list_workspace_files(ctx_cache_.work_dir);
        file_index_built_ = true;
    }
    rebuild_palette_rows(filter);
    palette_.set_filter(filter);
    palette_.open();
    if (on_palette_open_changed_) {
        on_palette_open_changed_(true);
    }
    mark_layout_dirty();
    mark_dirty();
}

void MessageComposer::close_palette() {
    palette_.close();
    if (input_ != nullptr) {
        std::string value = input_->value();
        if (!value.empty() && (value[0] == '/' || value[0] == '@')) {
            const auto end = value.find_first_not_of(" \t", 1);
            if (end == std::string::npos) {
                input_->set_value("");
            }
        }
    }
    mark_layout_dirty();
    mark_dirty();
}

void MessageComposer::rebuild_palette_rows(const std::string& filter) {
    std::vector<ComposerPaletteRow> rows;
    rows.push_back({"mode.plan", "agent", "⚡", "Plan", "Architecture-first, no edits", "", false, false});
    rows.push_back({"mode.debug", "agent", "🐞", "Debug", "Trace errors and root cause", "", false, false});
    rows.push_back({"mode.ask", "agent", "💬", "Ask", "Read-only answers", "", false, false});
    rows.push_back({"menu.skills", "agent", "⚡", "Skills", "Force a skill", "", true, false});

    for (const std::string& skill : ctx_cache_.skills) {
        rows.push_back({"skill." + skill, "menu.skills", "▪", skill, "", "", false, false});
    }

    rows.push_back({"menu.files", "context", "📎", "Files & Symbols", "Fuzzy workspace search", "[@]", true, false});

    std::vector<std::string> file_lines;
    std::vector<std::string> file_meta;
    for (const WorkspaceFileEntry& entry : file_index_) {
        file_lines.push_back(entry.display);
        file_meta.push_back(entry.path);
    }
    std::vector<int> file_hits = skim_fuzzy_filter_indices(filter, file_lines, file_meta, file_meta);
    int shown = 0;
    for (int idx : file_hits) {
        if (idx < 0 || idx >= static_cast<int>(file_index_.size())) {
            continue;
        }
        if (++shown > 40) {
            break;
        }
        const WorkspaceFileEntry& entry = file_index_[static_cast<std::size_t>(idx)];
        rows.push_back({"file." + entry.path, "menu.files", "📄", entry.display, "", "", false, false});
    }

    char git_sub[64];
    if (ctx_cache_.git.valid && ctx_cache_.git.uncommitted > 0) {
        std::snprintf(git_sub, sizeof(git_sub), "%d uncommitted change%s", ctx_cache_.git.uncommitted,
                      ctx_cache_.git.uncommitted == 1 ? "" : "s");
    } else {
        std::snprintf(git_sub, sizeof(git_sub), "No local changes");
    }
    rows.push_back({"ctx.git", "context", "⑂", "Git diff", git_sub, "", false, false});
    rows.push_back({"ctx.clipboard", "context", "📋", "Clipboard", "Paste as context", "", false, false});
    rows.push_back({"ctx.terminal", "context", ">_", "Last build output", "TAIJI_LAST_BUILD_LOG", "", false, false});

    rows.push_back({"menu.thinking", "runtime", "🧠", "Thinking / reasoning", pending_cot_.empty() ? "default" : pending_cot_,
                    "", true, false});
    rows.push_back({"thinking.off", "menu.thinking", "", "Off", "", "", false, false});
    rows.push_back({"thinking.low", "menu.thinking", "", "Low", "", "", false, false});
    rows.push_back({"thinking.high", "menu.thinking", "", "High", "", "", false, false});

    const int mcp_active = static_cast<int>(ctx_cache_.mcp_enabled.size());
    char mcp_sub[32];
    std::snprintf(mcp_sub, sizeof(mcp_sub), "%d active", mcp_active);
    rows.push_back({"menu.mcp", "runtime", "🔌", "MCP servers", mcp_sub, "", true, false});
    for (const std::string& server : ctx_cache_.mcp_servers) {
        const bool enabled =
            std::find(ctx_cache_.mcp_enabled.begin(), ctx_cache_.mcp_enabled.end(), server) !=
                ctx_cache_.mcp_enabled.end() ||
            std::find(pending_mcp_enabled_.begin(), pending_mcp_enabled_.end(), server) != pending_mcp_enabled_.end();
        rows.push_back({"mcp." + server, "menu.mcp", "", server, "", "", false, enabled});
    }

    std::string model_title = ctx_cache_.model.empty() ? "Model" : ctx_cache_.model;
    rows.push_back({"menu.model", "runtime", "🤖", "Model switcher", model_title, "", true, false});
    rows.push_back({"model.keep", "menu.model", "", "Keep current model", model_title, "", false, false});

    palette_.set_rows(std::move(rows));
    palette_.set_filter(filter);
}

void MessageComposer::handle_palette_select(const std::string& row_id) {
    if (row_id == "mode.plan") {
        replace_agent_mode_chip(chips_, {ComposerChipKind::AgentMode, "plan", "Plan", "", ui_accent()});
    } else if (row_id == "mode.debug") {
        replace_agent_mode_chip(chips_, {ComposerChipKind::AgentMode, "debug", "Debug", "", ui_accent()});
    } else if (row_id == "mode.ask") {
        replace_agent_mode_chip(chips_, {ComposerChipKind::AgentMode, "ask", "Ask", "", ui_accent()});
    } else if (row_id.rfind("skill.", 0) == 0) {
        const std::string name = row_id.substr(6);
        add_chip({ComposerChipKind::Skill, name, name, "", ui_status_warn()});
    } else if (row_id.rfind("file.", 0) == 0) {
        const std::string path = row_id.substr(5);
        const std::string snippet = read_file_snippet(path);
        add_chip({ComposerChipKind::File, path, path, snippet, ui_quick_action_code()});
    } else if (row_id == "ctx.git") {
        const std::string diff = capture_git_diff(ctx_cache_.work_dir);
        add_chip({ComposerChipKind::GitDiff, "git", "Git diff", diff, ui_quick_action_git()});
    } else if (row_id == "ctx.clipboard") {
        const std::string clip = read_clipboard_text();
        std::string label = "Clipboard";
        if (!clip.empty()) {
            label = "Clipboard text";
        }
        add_chip({ComposerChipKind::Clipboard, "clip", label, clip, ui_quick_action_test()});
    } else if (row_id == "ctx.terminal") {
        const std::string log = read_last_terminal_capture();
        add_chip({ComposerChipKind::Terminal, "build", "Build log", log, ui_quick_action_git()});
    } else if (row_id == "thinking.off") {
        pending_cot_ = "off";
        add_chip({ComposerChipKind::Thinking, "off", "Thinking off", "", ui_accent()});
    } else if (row_id == "thinking.low") {
        pending_cot_ = "low";
        add_chip({ComposerChipKind::Thinking, "low", "Thinking low", "", ui_accent()});
    } else if (row_id == "thinking.high") {
        pending_cot_ = "high";
        add_chip({ComposerChipKind::Thinking, "high", "Thinking high", "", ui_accent()});
    } else if (row_id.rfind("mcp.", 0) == 0) {
        const std::string server = row_id.substr(4);
        auto it = std::find(pending_mcp_enabled_.begin(), pending_mcp_enabled_.end(), server);
        if (it == pending_mcp_enabled_.end()) {
            pending_mcp_enabled_.push_back(server);
            add_chip({ComposerChipKind::McpFilter, server, server, "", ui_status_warn()});
        } else {
            pending_mcp_enabled_.erase(it);
            chips_.erase(std::remove_if(chips_.begin(), chips_.end(),
                                        [&](const ComposerChip& c) {
                                            return c.kind == ComposerChipKind::McpFilter && c.id == server;
                                        }),
                           chips_.end());
        }
        rebuild_palette_rows(palette_.filter());
        mark_dirty();
        return;
    } else if (row_id == "model.keep") {
        close_palette();
        return;
    }

    close_palette();
    mark_layout_dirty();
    mark_dirty();
}

void MessageComposer::add_chip(ComposerChip chip) {
    chips_.push_back(std::move(chip));
    mark_layout_dirty();
    mark_dirty();
}

void MessageComposer::remove_chip_at(std::size_t index) {
    if (index >= chips_.size()) {
        return;
    }
    const ComposerChip removed = chips_[index];
    chips_.erase(chips_.begin() + static_cast<std::ptrdiff_t>(index));
    if (removed.kind == ComposerChipKind::McpFilter) {
        pending_mcp_enabled_.erase(
            std::remove(pending_mcp_enabled_.begin(), pending_mcp_enabled_.end(), removed.id),
            pending_mcp_enabled_.end());
    }
    if (removed.kind == ComposerChipKind::Thinking) {
        pending_cot_.clear();
    }
    mark_layout_dirty();
    mark_dirty();
}

std::string MessageComposer::read_file_snippet(const std::string& rel_path) const {
    if (ctx_cache_.work_dir.empty()) {
        return {};
    }
    const std::string full = ctx_cache_.work_dir + "/" + rel_path;
    std::ifstream in(full);
    if (!in) {
        return {};
    }
    std::ostringstream out;
    out << in.rdbuf();
    std::string content = out.str();
    if (content.size() > 80000) {
        content.resize(80000);
        content += "\n… (file truncated)";
    }
    return content;
}
