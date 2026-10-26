#include "message_composer.hpp"

#include "ui_keybindings.hpp"

#include <tuinator/platform/clipboard.hpp>

#include "skim_fuzzy.hpp"
#include "ui_chat_column.hpp"
#include "ui_locale.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cstdio>
#include <filesystem>
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
    : palette_(theme), theme_picker_(theme), actor_picker_(theme), theme_(std::move(theme)) {
    default_input_placeholder_ = input_options.placeholder;
    input_options.prompt = "  ";
    input_ = std::make_unique<ChatTextInput>(input_options, style, focused_style);
    input_->set_on_layout([this]() { mark_layout_dirty(); });
    input_->set_on_submit([this](const std::string& value) {
        if (theme_picker_.is_open()) {
            confirm_theme_picker_selection();
            return;
        }
        if (actor_picker_.is_open()) {
            confirm_actor_picker_selection();
            return;
        }
        if (palette_.is_open()) {
            return;
        }
        std::string work_dir;
        if (host_context_) {
            work_dir = host_context_().work_dir;
        }
        const std::string composed = composer_build_outgoing_message(value, chips_, image_attachments_, work_dir);
        if (on_submit_ && (!composed.empty() || !image_attachments_.empty())) {
            on_submit_(composed);
        }
    });

    input_->set_paste_intercept([this](const std::string& pasted) { return try_consume_paste_event(pasted); });

    theme_picker_.set_on_preview([this](const std::string& theme_id) {
        if (on_theme_preview_) {
            on_theme_preview_(theme_id);
        }
    });
    theme_picker_.set_on_commit([this](const std::string& theme_id) {
        if (on_theme_picked_) {
            on_theme_picked_(theme_id);
        }
    });

    palette_.set_on_select([this](const std::string& id) { handle_palette_select(id); });
    palette_.set_on_close([this]() {
        sync_palette_key_intercept();
        if (on_palette_open_changed_) {
            on_palette_open_changed_(false);
        }
        mark_layout_dirty();
    });
    if (input_ != nullptr) {
        layout_input_height_ = input_->preferred_size().height;
    }
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

void MessageComposer::set_on_theme_preview(std::function<void(const std::string& theme_id)> handler) {
    on_theme_preview_ = std::move(handler);
}

void MessageComposer::set_on_theme_picked(std::function<void(const std::string& theme_id)> handler) {
    on_theme_picked_ = std::move(handler);
}

void MessageComposer::set_on_image_preview(
    std::function<void(const tuinator::TerminalImage&, const std::string&)> handler) {
    on_image_preview_ = std::move(handler);
}

void MessageComposer::sync_field_styles(tuinator::Style normal, tuinator::Style focused) {
    if (input_ != nullptr) {
        input_->sync_field_styles(std::move(normal), std::move(focused));
        input_->sync_frame_border_style();
    }
}

void MessageComposer::notify_input_changed(const std::string& value) {
    if (theme_picker_.is_open()) {
        theme_picker_.set_filter(value);
        mark_dirty();
        return;
    }
    if (actor_picker_.is_open()) {
        if (palette_.is_open()) {
            palette_.close();
            if (on_palette_open_changed_) {
                on_palette_open_changed_(false);
            }
        }
        rebuild_actor_picker_entries();
        actor_picker_.set_filter(value);
        mark_layout_dirty();
        mark_dirty();
        return;
    }
    sync_trigger_from_input();
    if (input_ != nullptr) {
        const int h = input_->preferred_size().height;
        if (h != layout_input_height_) {
            layout_input_height_ = h;
            mark_layout_dirty();
        }
    }
    mark_dirty();
}

void MessageComposer::refresh_locale_strings() {
    if (input_ == nullptr || theme_picker_.is_open() || actor_picker_.is_open()) {
        return;
    }
    const std::string ph = ui_tr("common", "composer.placeholder");
    input_->set_placeholder(ph.empty() ? "Write a message…" : ph);
}

void MessageComposer::clear_after_send() {
    close_actor_picker();
    chips_.clear();
    image_attachments_.clear();
    pending_cot_.clear();
    pending_mcp_enabled_.clear();
    if (input_ != nullptr) {
        input_->set_value("");
        const int h = input_->preferred_size().height;
        if (h != layout_input_height_) {
            layout_input_height_ = h;
            mark_layout_dirty();
        }
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
    h += image_attachment_bar_height();
    if (!chips_.empty()) {
        h += 1;
    }
    if (theme_picker_.is_open()) {
        h += theme_picker_.preferred_size(42).height;
    } else if (actor_picker_.is_open()) {
        h += actor_picker_.preferred_size(42).height;
    } else if (palette_.is_open()) {
        h += palette_.preferred_size().height;
    }
    return {0, h};
}

tuinator::Rect MessageComposer::stream_bounds(const tuinator::Rect& bounds) const {
    const ChatStreamLayout column = chat_stream_layout_in_widget(bounds.width);
    return {bounds.x + column.offset_x, bounds.y, column.content_width, bounds.height};
}

void MessageComposer::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    const tuinator::Rect stream = stream_bounds(bounds);

    const int image_h = image_attachment_bar_height();
    const int chips_h = chips_.empty() ? 0 : 1;
    const int input_h = input_ != nullptr ? std::max(3, input_->preferred_size().height) : 3;
    if (input_h > bounds.height) {
        mark_layout_dirty();
    }

    int overlay_h = 0;
    if (theme_picker_.is_open()) {
        overlay_h = theme_picker_.preferred_size(stream.width).height;
    } else if (actor_picker_.is_open()) {
        overlay_h = actor_picker_.preferred_size(stream.width).height;
    } else if (palette_.is_open()) {
        overlay_h = palette_.preferred_size().height;
    }

    const int stack_h = image_h + chips_h + overlay_h + input_h;
    const int stack_bottom = bounds.y + bounds.height;
    const int stack_top = stack_bottom - stack_h;
    const int input_y = stack_bottom - input_h;
    const int overlay_y = stack_top;
    const int overlay_slot_h = overlay_h;

    theme_picker_bounds_ = {};
    actor_picker_bounds_ = {};
    if (theme_picker_.is_open() && overlay_slot_h > 0) {
        theme_picker_bounds_ = {stream.x, overlay_y, stream.width, overlay_slot_h};
        theme_picker_.layout(theme_picker_bounds_);
    } else {
        theme_picker_.layout({0, 0, 0, 0});
    }
    if (actor_picker_.is_open() && overlay_slot_h > 0) {
        actor_picker_bounds_ = {stream.x, overlay_y, stream.width, overlay_slot_h};
        actor_picker_.layout(actor_picker_bounds_);
    } else {
        actor_picker_.layout({0, 0, 0, 0});
    }
    if (palette_.is_open() && !actor_picker_.is_open() && overlay_slot_h > 0) {
        palette_.layout({stream.x, overlay_y, stream.width, overlay_slot_h});
    } else {
        palette_.layout({0, 0, 0, 0});
    }

    image_attachments_bounds_ = {};
    if (image_h > 0) {
        image_attachments_bounds_ = {stream.x, input_y - chips_h - image_h, stream.width, image_h};
    }

    chips_bounds_ = {};
    if (!chips_.empty()) {
        chips_bounds_ = {stream.x, input_y - chips_h, stream.width, chips_h};
    }

    const int visible_input_h = std::min(input_h, std::max(0, bounds.y + bounds.height - input_y));
    input_bounds_ = {stream.x, input_y, stream.width, visible_input_h};
    if (input_ != nullptr && visible_input_h > 0) {
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
    ctx.canvas.fill_rect({0, 0, area.width, area.height}, ' ', bg);

    int x = 1;
    for (std::size_t i = 0; i < chips_.size(); ++i) {
        const ComposerChip& chip = chips_[i];
        tuinator::Style pill = ui_resolve_main_text(theme_.label);
        pill.background_rgb = ui_badge_bg();
        pill.foreground_rgb = chip.color;
        std::string text = " " + chip.label + " × ";
        const int w = tuinator::text_display_width(text);
        if (x + w > area.width) {
            break;
        }
        ctx.canvas.draw_text({x, 0}, text, pill);
        x += w;
    }
}

void MessageComposer::paint(tuinator::PaintContext& ctx) const {
    const tuinator::Rect stream = stream_bounds(bounds_);
    const tuinator::Rect host{bounds_.x, bounds_.y, bounds_.width, bounds_.height};
    const auto paint_in_bounds = [&](tuinator::Rect area, const auto& draw) {
        const int x0 = std::max(area.x, host.x);
        const int y0 = std::max(area.y, host.y);
        const int x1 = std::min(area.x + area.width, host.x + host.width);
        const int y1 = std::min(area.y + area.height, host.y + host.height);
        if (x1 <= x0 || y1 <= y0) {
            return;
        }
        area = {x0, y0, x1 - x0, y1 - y0};
        const tuinator::Rect local{area.x - stream.x, area.y - bounds_.y, area.width, area.height};
        ctx.with_clip(local, [&](tuinator::PaintContext& child_ctx) { draw(child_ctx); });
    };

    if (theme_picker_.is_open()) {
        paint_in_bounds(theme_picker_bounds_, [&](tuinator::PaintContext& child_ctx) { theme_picker_.paint(child_ctx); });
    }
    if (actor_picker_.is_open()) {
        paint_in_bounds(actor_picker_bounds_,
                        [&](tuinator::PaintContext& child_ctx) { actor_picker_.paint(child_ctx); });
    }
    if (palette_.is_open() && !actor_picker_.is_open()) {
        paint_in_bounds(palette_.bounds(), [&](tuinator::PaintContext& child_ctx) { palette_.paint(child_ctx); });
    }
    if (image_attachments_bounds_.height > 0) {
        paint_in_bounds(image_attachments_bounds_, [&](tuinator::PaintContext& child_ctx) {
            paint_image_attachments(child_ctx, image_attachments_bounds_);
        });
    }
    if (chips_bounds_.height > 0) {
        paint_in_bounds(chips_bounds_,
                        [&](tuinator::PaintContext& child_ctx) { paint_chips(child_ctx, chips_bounds_); });
    }
    if (input_bounds_.width > 0 && input_bounds_.height > 0) {
        paint_in_bounds(input_bounds_, [&](tuinator::PaintContext& child_ctx) {
            if (input_ != nullptr) {
                input_->paint(child_ctx);
            }
            paint_plus_button(child_ctx, {1, 1, 3, 1});
        });
    }
}

bool MessageComposer::theme_picker_navigation_key(const tuinator::KeyPress& key) {
    switch (key.key) {
    case tuinator::Key::Up:
    case tuinator::Key::Down:
    case tuinator::Key::PageUp:
    case tuinator::Key::PageDown:
    case tuinator::Key::Home:
    case tuinator::Key::End:
        return true;
    default:
        return false;
    }
}

bool MessageComposer::has_focused_descendant() const {
    if (input_ != nullptr && (input_->is_focused() || input_->has_focused_descendant())) {
        return true;
    }
    return tuinator::Widget::has_focused_descendant();
}

bool MessageComposer::route_theme_picker_key(const tuinator::KeyPress& key) {
    if (!theme_picker_.is_open()) {
        return false;
    }
    if (key.key == tuinator::Key::Escape) {
        close_theme_picker(false);
        return true;
    }
    if (ui_key_theme_picker_list_prev(key)) {
        theme_picker_.move_selection(-1);
        return true;
    }
    if (ui_key_theme_picker_list_next(key)) {
        theme_picker_.move_selection(1);
        return true;
    }
    if (key.key == tuinator::Key::PageUp) {
        theme_picker_.move_page(-5);
        return true;
    }
    if (key.key == tuinator::Key::PageDown) {
        theme_picker_.move_page(5);
        return true;
    }
    if (key.key == tuinator::Key::Home) {
        theme_picker_.jump_selection(false);
        return true;
    }
    if (key.key == tuinator::Key::End) {
        theme_picker_.jump_selection(true);
        return true;
    }
    if (key.key == tuinator::Key::Enter) {
        confirm_theme_picker_selection();
        return true;
    }
    return false;
}

void MessageComposer::confirm_theme_picker_selection() {
    if (!theme_picker_.is_open()) {
        return;
    }
    theme_picker_.commit_selection();
    close_theme_picker(true);
}

bool MessageComposer::route_palette_key(const tuinator::KeyPress& key) {
    if (!palette_.is_open()) {
        return false;
    }
    const bool slash_mode =
        input_ != nullptr && !input_->value().empty() && input_->value()[0] == '/';
    if (slash_mode) {
        switch (key.key) {
        case tuinator::Key::Up:
        case tuinator::Key::Down:
        case tuinator::Key::Left:
        case tuinator::Key::Right:
        case tuinator::Key::Enter:
        case tuinator::Key::Escape:
        case tuinator::Key::Tab:
            return palette_.handle_keypress(key);
        default:
            return false;
        }
    }
    return palette_.handle_keypress(key);
}

void MessageComposer::set_vim_key_intercept(VimKeyIntercept handler) {
    vim_key_intercept_ = std::move(handler);
    sync_palette_key_intercept();
}

void MessageComposer::sync_palette_key_intercept() {
    if (input_ == nullptr || theme_picker_.is_open() || actor_picker_.is_open()) {
        return;
    }
    if (palette_.is_open()) {
        input_->set_composer_intercept_key(
            [this](const tuinator::KeyPress& key) { return route_palette_key(key); });
    } else if (vim_key_intercept_) {
        input_->set_composer_intercept_key(vim_key_intercept_);
    } else {
        input_->set_composer_intercept_key({});
    }
}

bool MessageComposer::handle_event(const tuinator::Event& event) {
    if (actor_picker_.is_open()) {
        if (const auto* key = std::get_if<tuinator::KeyPress>(&event)) {
            if (route_actor_picker_key(*key)) {
                mark_dirty();
                return true;
            }
        }
        if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
            if (mouse->button == tuinator::MouseButton::Left && mouse->action == tuinator::MouseAction::Release) {
                int visible_row = -1;
                const tuinator::Point picker_local{mouse->position.x - actor_picker_bounds_.x,
                                                   mouse->position.y - actor_picker_bounds_.y};
                if (actor_picker_.hit_test_row(picker_local, visible_row)) {
                    actor_picker_.select_visible_row(visible_row);
                    confirm_actor_picker_selection();
                    return true;
                }
            }
            if (mouse->action == tuinator::MouseAction::WheelUp || mouse->action == tuinator::MouseAction::WheelDown) {
                const tuinator::Point picker_local{mouse->position.x - actor_picker_bounds_.x,
                                                   mouse->position.y - actor_picker_bounds_.y};
                if (picker_local.x >= 0 && picker_local.y >= 0 && picker_local.x < actor_picker_bounds_.width &&
                    picker_local.y < actor_picker_bounds_.height) {
                    const int delta = mouse->action == tuinator::MouseAction::WheelUp ? -1 : 1;
                    actor_picker_.move_selection(delta);
                    mark_dirty();
                    return true;
                }
            }
        }
    }

    if (theme_picker_.is_open()) {
        if (const auto* key = std::get_if<tuinator::KeyPress>(&event)) {
            if (route_theme_picker_key(*key)) {
                mark_dirty();
                return true;
            }
        }
        if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
            const tuinator::Point local{mouse->position.x - bounds_.x, mouse->position.y - bounds_.y};
            if (mouse->button == tuinator::MouseButton::Left && mouse->action == tuinator::MouseAction::Release) {
                int visible_row = -1;
                const tuinator::Point picker_local{mouse->position.x - theme_picker_bounds_.x,
                                                   mouse->position.y - theme_picker_bounds_.y};
                if (theme_picker_.hit_test_row(picker_local, visible_row)) {
                    theme_picker_.select_visible_row(visible_row);
                    confirm_theme_picker_selection();
                    return true;
                }
            }
            if (mouse->action == tuinator::MouseAction::WheelUp || mouse->action == tuinator::MouseAction::WheelDown) {
                const tuinator::Point picker_local{mouse->position.x - theme_picker_bounds_.x,
                                                   mouse->position.y - theme_picker_bounds_.y};
                if (picker_local.x >= 0 && picker_local.y >= 0 && picker_local.x < theme_picker_bounds_.width &&
                    picker_local.y < theme_picker_bounds_.height) {
                    const int delta = mouse->action == tuinator::MouseAction::WheelUp ? -1 : 1;
                    theme_picker_.move_selection(delta);
                    mark_dirty();
                    return true;
                }
            }
        }
    }

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
            if (!image_attachments_.empty()) {
                std::size_t index = 0;
                bool remove_button = false;
                if (image_attachment_hit(mouse->position, &index, &remove_button)) {
                    if (remove_button) {
                        image_attachments_.erase(image_attachments_.begin() + static_cast<std::ptrdiff_t>(index));
                        mark_layout_dirty();
                    } else if (on_image_preview_) {
                        on_image_preview_(image_attachments_[index].image, image_attachments_[index].label);
                    }
                    mark_dirty();
                    return true;
                }
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
    if ((theme_picker_.is_open() || actor_picker_.is_open()) && input_ != nullptr) {
        return input_->hit_test_focusable(point);
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

void MessageComposer::rebuild_actor_picker_entries() {
    if (host_context_) {
        ctx_cache_ = host_context_();
    }
    if (!file_index_built_ && !ctx_cache_.work_dir.empty()) {
        file_index_ = list_workspace_files(ctx_cache_.work_dir);
        file_index_built_ = true;
    }

    std::vector<ComposerFuzzyPickerItem> entries;
    for (const std::string& packed : ctx_cache_.template_mentions) {
        const std::size_t tab = packed.find('\t');
        if (tab == std::string::npos) {
            continue;
        }
        const std::string id = packed.substr(0, tab);
        const std::string label = packed.substr(tab + 1);
        ComposerFuzzyPickerItem item;
        item.id = "actor." + id;
        item.line = id + "  ·  " + label + "  actor template";
        std::string meta = id + " " + label + " actor persona template @mention";
        for (char& ch : meta) {
            if (ch == '-') {
                ch = ' ';
            }
        }
        item.meta = std::move(meta);
        item.label = label + "  @" + id;
        entries.push_back(std::move(item));
    }
    for (const std::string& packed : ctx_cache_.pack_mentions) {
        const std::size_t tab = packed.find('\t');
        if (tab == std::string::npos) {
            continue;
        }
        const std::string id = packed.substr(0, tab);
        const std::string label = packed.substr(tab + 1);
        ComposerFuzzyPickerItem item;
        item.id = "pack." + id;
        item.line = id + "  ·  " + label + "  context pack";
        item.meta = id + " " + label + " lore pack context @mention";
        item.label = "📚 " + label;
        entries.push_back(std::move(item));
    }

    std::string filter;
    if (actor_picker_.is_open() && input_ != nullptr) {
        filter = input_->value();
    }

    // Files are optional in @-mention: only search the workspace index when the user is
    // clearly looking for a path, so actors/templates are not drowned out by ~/Downloads noise.
    constexpr std::size_t kMinFileFilterLen = 3;
    constexpr int kMaxFilePickerRows = 8;
    const bool want_files = filter.size() >= kMinFileFilterLen &&
                            (filter.find('/') != std::string::npos || filter.find('.') != std::string::npos ||
                             filter.find('\\') != std::string::npos);
    if (want_files && !file_index_.empty()) {
        std::vector<std::string> file_lines;
        std::vector<std::string> file_meta;
        file_lines.reserve(file_index_.size());
        file_meta.reserve(file_index_.size());
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
            if (++shown > kMaxFilePickerRows) {
                break;
            }
            const WorkspaceFileEntry& entry = file_index_[static_cast<std::size_t>(idx)];
            ComposerFuzzyPickerItem item;
            item.id = "file." + entry.path;
            item.line = entry.display + "  file";
            item.meta = entry.path + " " + entry.display + " workspace file @mention";
            item.label = "📄 " + entry.display;
            entries.push_back(std::move(item));
        }
    }

    actor_picker_.set_entries(std::move(entries));
}

void MessageComposer::close_actor_picker() {
    if (!actor_picker_.is_open()) {
        actor_mention_at_ = std::string::npos;
        actor_stash_prefix_.clear();
        return;
    }
    actor_picker_.close();
    if (input_ != nullptr && !theme_picker_.is_open()) {
        if (actor_mention_at_ != std::string::npos) {
            input_->set_value(actor_stash_prefix_);
            input_->set_placeholder(default_input_placeholder_);
        }
        input_->set_composer_intercept_key({});
    }
    actor_mention_at_ = std::string::npos;
    actor_stash_prefix_.clear();
    if (!palette_.is_open() && !theme_picker_.is_open() && on_palette_open_changed_) {
        on_palette_open_changed_(false);
    }
    mark_layout_dirty();
    mark_dirty();
}

void MessageComposer::confirm_actor_picker_selection() {
    if (!actor_picker_.is_open() || input_ == nullptr || actor_mention_at_ == std::string::npos) {
        close_actor_picker();
        return;
    }
    const std::string row_id = actor_picker_.selected_id();
    if (row_id.empty()) {
        return;
    }

    const std::string prefix = actor_stash_prefix_;
    actor_stash_prefix_.clear();
    actor_mention_at_ = std::string::npos;
    actor_picker_.close();
    input_->set_placeholder(default_input_placeholder_);
    input_->set_composer_intercept_key({});

    if (row_id.rfind("actor.", 0) == 0) {
        const std::string id = row_id.substr(6);
        input_->set_value(prefix + "@" + id + " ");
    } else if (row_id.rfind("pack.", 0) == 0) {
        const std::string id = row_id.substr(5);
        std::string label = id;
        const std::size_t slash = id.find('/');
        if (slash != std::string::npos && slash + 1 < id.size()) {
            label = id.substr(slash + 1);
        }
        input_->set_value(prefix);
        add_chip({ComposerChipKind::Context, id, label, "", ui_status_warn()});
    } else if (row_id.rfind("file.", 0) == 0) {
        const std::string path = row_id.substr(5);
        input_->set_value(prefix);
        const std::string snippet = read_file_snippet(path);
        add_chip({ComposerChipKind::File, path, path, snippet, ui_quick_action_code()});
    } else {
        input_->set_value(prefix);
    }

    if (!palette_.is_open() && !theme_picker_.is_open() && on_palette_open_changed_) {
        on_palette_open_changed_(false);
    }
    mark_layout_dirty();
    mark_dirty();
}

bool MessageComposer::route_actor_picker_key(const tuinator::KeyPress& key) {
    return actor_picker_.route_key(
        key, [this]() { close_actor_picker(); }, [this]() { confirm_actor_picker_selection(); });
}

void MessageComposer::sync_actor_mention_from_input() {
    if (input_ == nullptr || theme_picker_.is_open() || actor_picker_.is_open()) {
        return;
    }
    const std::string& value = input_->value();
    if (value.empty()) {
        close_actor_picker();
        return;
    }
    if (!value.empty() && value[0] == '/') {
        close_actor_picker();
        return;
    }
    const std::size_t at = value.rfind('@');
    if (at == std::string::npos) {
        close_actor_picker();
        return;
    }
    if (at > 0) {
        const char prev = value[at - 1];
        if (prev != ' ' && prev != '\n' && prev != '\t') {
            close_actor_picker();
            return;
        }
    }
    const std::string token = value.substr(at + 1);
    if (token.find_first_of(" \t\n") != std::string::npos) {
        close_actor_picker();
        return;
    }
    if (palette_.is_open()) {
        palette_.close();
        if (on_palette_open_changed_) {
            on_palette_open_changed_(false);
        }
        mark_layout_dirty();
    }
    actor_mention_at_ = at;
    actor_stash_prefix_ = value.substr(0, at);
    rebuild_actor_picker_entries();
    if (!actor_picker_.is_open()) {
        actor_picker_.open();
        if (input_ != nullptr) {
            const std::string ph = ui_tr("common", "actor_picker.filter_placeholder");
            input_->set_placeholder(ph.empty() ? "Filter actors…" : ph);
            input_->set_composer_intercept_key(
                [this](const tuinator::KeyPress& key) { return route_actor_picker_key(key); });
            input_->set_value(token);
        }
        if (on_palette_open_changed_) {
            on_palette_open_changed_(true);
        }
        mark_layout_dirty();
    }
    actor_picker_.set_filter(token);
    mark_dirty();
}

void MessageComposer::sync_trigger_from_input() {
    if (input_ == nullptr) {
        return;
    }
    const std::string& value = input_->value();
    if (value.empty()) {
        close_actor_picker();
        return;
    }
    if (value[0] == '/') {
        close_actor_picker();
        std::string filter = value.substr(1);
        if (!palette_.is_open()) {
            open_palette_with_trigger(filter);
        } else {
            palette_.set_filter(filter);
        }
        return;
    }
    if (palette_.is_open()) {
        palette_.close();
        if (on_palette_open_changed_) {
            on_palette_open_changed_(false);
        }
        mark_layout_dirty();
    }
    sync_actor_mention_from_input();
}

void MessageComposer::open_palette_with_trigger(const std::string& filter) {
    if (theme_picker_.is_open() || actor_picker_.is_open()) {
        return;
    }
    close_actor_picker();
    if (input_ != nullptr) {
        const std::string& value = input_->value();
        if (!value.empty() && value[0] == '@') {
            return;
        }
        const std::size_t at = value.rfind('@');
        if (at != std::string::npos) {
            const std::string tail = value.substr(at + 1);
            if (tail.find_first_of(" \t\n") == std::string::npos) {
                return;
            }
        }
    }
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
    sync_palette_key_intercept();
    if (on_palette_open_changed_) {
        on_palette_open_changed_(true);
    }
    mark_layout_dirty();
    mark_dirty();
}

void MessageComposer::close_palette() {
    palette_.close();
    sync_palette_key_intercept();
    if (input_ != nullptr) {
        std::string value = input_->value();
        if (!value.empty() && value[0] == '/') {
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
    {
        const std::string skills = ui_shell("composer_menu.skills");
        const std::string hint = ui_shell("composer_menu.skills_hint");
        rows.push_back({"menu.skills", "agent", "⚡", skills.empty() ? "Skills" : skills,
                        hint.empty() ? "Force a skill" : hint, "", true, false});
    }

    for (const std::string& skill : ctx_cache_.skills) {
        rows.push_back({"skill." + skill, "menu.skills", "▪", skill, "", "", false, false});
    }

    rows.push_back({"menu.files", "context", "📎", "Files & Symbols", "Fuzzy workspace search", "[@]", true, false});

    for (const std::string& packed : ctx_cache_.pack_mentions) {
        const std::size_t tab = packed.find('\t');
        if (tab == std::string::npos) {
            continue;
        }
        const std::string id = packed.substr(0, tab);
        const std::string label = packed.substr(tab + 1);
        rows.push_back({"pack." + id, "menu.files", "📚", label, id, "", false, false});
    }

    rows.push_back({"menu.templates", "agent", "🎭", "Templates", "/template", "", true, false});
    rows.push_back({"menu.actors", "agent", "🎭", "Actors", "@", "", true, false});
    for (const std::string& packed : ctx_cache_.template_mentions) {
        const std::size_t tab = packed.find('\t');
        if (tab == std::string::npos) {
            continue;
        }
        const std::string id = packed.substr(0, tab);
        const std::string label = packed.substr(tab + 1);
        rows.push_back({"template." + id, "menu.templates", "🎭", label, id, "", false, false});
        rows.push_back({"actor." + id, "menu.actors", "🎭", label, "@" + id, "", false, false});
    }

    rows.push_back({"menu.graph", "agent", "🗺️", "Guided graph", "/graph", "", true, false});
    rows.push_back({"graph.list", "menu.graph", "", "List graphs", "/graph list", "", false, false});
    rows.push_back({"graph.load.demo", "menu.graph", "", "Load demo-forge", "/graph load demo-forge", "", false, false});
    rows.push_back({"graph.status", "menu.graph", "", "Graph status", "/graph status", "", false, false});
    rows.push_back({"menu.council", "agent", "👥", "Council", "/council", "", true, false});
    rows.push_back({"council.list", "menu.council", "", "List actors", "/council", "", false, false});
    rows.push_back({"council.on", "menu.council", "", "Enable council", "/council on", "", false, false});
    rows.push_back({"council.off", "menu.council", "", "Disable council", "/council off", "", false, false});
    for (const std::string& packed : ctx_cache_.template_mentions) {
        const std::size_t tab = packed.find('\t');
        if (tab == std::string::npos) {
            continue;
        }
        const std::string id = packed.substr(0, tab);
        const std::string label = packed.substr(tab + 1);
        rows.push_back({"council.add." + id, "menu.council", "➕", "Add " + label, id, "", false, false});
    }

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
    {
        const std::string mcp_label = ui_shell("composer_menu.mcp_servers");
        rows.push_back({"menu.mcp", "runtime", "🔌", mcp_label.empty() ? "MCP servers" : mcp_label, mcp_sub, "", true,
                        false});
    }
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
    } else if (row_id.rfind("template.", 0) == 0) {
        const std::string id = row_id.substr(9);
        if (on_submit_) {
            on_submit_("/template " + id);
        }
    } else if (row_id.rfind("actor.", 0) == 0) {
        const std::string id = row_id.substr(6);
        if (input_ != nullptr) {
            input_->set_value("@" + id + " ");
        }
    } else if (row_id.rfind("council.", 0) == 0) {
        if (on_submit_) {
            if (row_id == "council.list") {
                on_submit_("/council");
            } else if (row_id == "council.on") {
                on_submit_("/council on");
            } else if (row_id == "council.off") {
                on_submit_("/council off");
            } else if (row_id.rfind("council.add.", 0) == 0) {
                const std::string id = row_id.substr(12);
                on_submit_("/council add " + id);
            }
        }
    } else if (row_id.rfind("pack.", 0) == 0) {
        const std::string id = row_id.substr(5);
        std::string label = id;
        const std::size_t slash = id.find('/');
        if (slash != std::string::npos && slash + 1 < id.size()) {
            label = id.substr(slash + 1);
        }
        add_chip({ComposerChipKind::Context, id, label, "", ui_status_warn()});
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

void MessageComposer::open_theme_picker(const std::string& active_theme_id) {
    if (theme_picker_.is_open() || palette_.is_open() || input_ == nullptr) {
        return;
    }
    close_actor_picker();
    theme_revert_id_ = active_theme_id;
    stashed_input_ = input_->value();
    input_->set_value("");
    if (stashed_input_.empty()) {
        const std::string ph = ui_tr("common", "theme_picker.filter_placeholder");
        input_->set_placeholder(ph.empty() ? "Filter color themes…" : ph);
    } else {
        const std::string ph = ui_tr("common", "theme_picker.filter_placeholder_draft");
        input_->set_placeholder(ph.empty() ? "Filter themes — Esc restores your message" : ph);
    }
    input_->set_composer_intercept_key([this](const tuinator::KeyPress& key) { return route_theme_picker_key(key); });
    theme_picker_.open(active_theme_id, !stashed_input_.empty());
    if (on_palette_open_changed_) {
        on_palette_open_changed_(true);
    }
    mark_layout_dirty();
    mark_dirty();
}

void MessageComposer::close_theme_picker(bool commit) {
    if (!theme_picker_.is_open()) {
        return;
    }
    if (!commit && on_theme_preview_ && !theme_revert_id_.empty()) {
        on_theme_preview_(theme_revert_id_);
    }
    theme_picker_.close();
    restore_stashed_input();
    theme_revert_id_.clear();
    if (on_palette_open_changed_) {
        on_palette_open_changed_(false);
    }
    mark_layout_dirty();
    mark_dirty();
}

void MessageComposer::restore_stashed_input() {
    if (input_ == nullptr) {
        return;
    }
    input_->set_composer_intercept_key(nullptr);
    input_->set_placeholder(default_input_placeholder_);
    input_->set_value(stashed_input_);
    stashed_input_.clear();
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

int MessageComposer::image_attachment_bar_height() const {
    return image_attachments_.empty() ? 0 : kComposerImageChipHeight;
}

tuinator::Rect MessageComposer::image_attachment_chip_bounds(std::size_t index) const {
    int x = image_attachments_bounds_.x + 1;
    for (std::size_t i = 0; i < index; ++i) {
        x += kComposerImageChipWidth + 1;
    }
    return {x, image_attachments_bounds_.y, kComposerImageChipWidth, kComposerImageChipHeight};
}

bool MessageComposer::image_attachment_hit(tuinator::Point global, std::size_t* index, bool* remove_button) const {
    if (index == nullptr || remove_button == nullptr) {
        return false;
    }
    for (std::size_t i = 0; i < image_attachments_.size(); ++i) {
        const tuinator::Rect chip = image_attachment_chip_bounds(i);
        if (!chip.contains(global)) {
            continue;
        }
        *index = i;
        const tuinator::Point local{global.x - chip.x, global.y - chip.y};
        *remove_button = local.y == 0 && local.x >= chip.width - 2;
        return true;
    }
    return false;
}

void MessageComposer::paint_image_attachments(tuinator::PaintContext& ctx, tuinator::Rect area) const {
    if (area.width <= 0 || area.height <= 0 || image_attachments_.empty()) {
        return;
    }

    tuinator::Style bg;
    bg.background_rgb = ui_main_bg();
    ctx.canvas.fill_rect({0, 0, area.width, area.height}, ' ', bg);

    tuinator::Style frame;
    frame.foreground_rgb = ui_border_chrome();
    frame.background_rgb = ui_main_bg();
    tuinator::Style label = ui_resolve_main_text(theme_.label);
    label.background_rgb = ui_main_bg();
    label.foreground_rgb = ui_secondary_text();
    tuinator::Style close = label;
    close.foreground_rgb = ui_danger();
    close.bold = true;

    int x = 1;
    for (const ComposerImageAttachment& attachment : image_attachments_) {
        if (x + kComposerImageChipWidth > area.width) {
            break;
        }

        const tuinator::Rect chip{x, 0, kComposerImageChipWidth, kComposerImageChipHeight};
        ctx.canvas.draw_text({chip.x + chip.width - 2, chip.y}, "×", close);

        std::string caption = attachment.label;
        const int max_label_w = chip.width - 3;
        while (!caption.empty() && tuinator::text_display_width(caption) > max_label_w) {
            caption.pop_back();
        }
        if (!caption.empty()) {
            ctx.canvas.draw_text({chip.x + 1, chip.y}, caption, label);
        }

        if (!attachment.image.empty()) {
            ctx.canvas.draw_image({chip.x + 1, chip.y + 1}, {kComposerImageThumbCols, kComposerImageThumbRows},
                                  attachment.image);
        }

        x += kComposerImageChipWidth + 1;
    }
}

void MessageComposer::add_image_attachment(tuinator::TerminalImage image, std::string label, std::string source_path) {
    if (image.empty()) {
        return;
    }
    if (label.empty()) {
        label = "image.png";
    }

    ComposerImageAttachment attachment;
    attachment.id = "img-" + std::to_string(next_image_id_++);
    attachment.label = std::move(label);
    attachment.source_path = std::move(source_path);
    attachment.image = std::move(image);
    image_attachments_.push_back(std::move(attachment));
    mark_layout_dirty();
    mark_dirty();
}

bool MessageComposer::try_import_clipboard_image() {
    const std::optional<ClipboardImagePayload> payload = read_clipboard_image();
    if (!payload.has_value()) {
        return false;
    }
    const std::optional<tuinator::TerminalImage> image = decode_image_bytes(*payload);
    if (!image.has_value()) {
        return false;
    }

    std::string label = "clipboard";
    if (payload->mime.find("jpeg") != std::string::npos) {
        label = "clipboard.jpg";
    } else if (payload->mime.find("webp") != std::string::npos) {
        label = "clipboard.webp";
    } else {
        label = "clipboard.png";
    }
    add_image_attachment(*image, std::move(label));
    return true;
}

bool MessageComposer::try_consume_pasted_text(const std::string& pasted) {
    if (const std::optional<std::string> path = image_path_from_pasted_text(pasted)) {
        if (const std::optional<tuinator::TerminalImage> image = load_image_file(*path)) {
            const std::filesystem::path file_path(*path);
            add_image_attachment(*image, file_path.filename().string(), *path);
            return true;
        }
    }
    return false;
}

bool MessageComposer::try_consume_paste_event(const std::string& pasted) {
    if (try_consume_pasted_text(pasted)) {
        mark_layout_dirty();
        mark_dirty();
        return true;
    }

    if (read_clipboard_image().has_value() && try_import_clipboard_image()) {
        mark_layout_dirty();
        mark_dirty();
        return true;
    }

    return false;
}

bool MessageComposer::try_paste_shortcut(const tuinator::KeyPress& key) {
    char shortcut = '\0';
    if (key.ctrl && key.character >= 'a' && key.character <= 'z') {
        shortcut = static_cast<char>(key.character);
    } else if (key.character >= 1 && key.character <= 26) {
        shortcut = static_cast<char>('a' + key.character - 1);
    }
    if (shortcut != 'v') {
        return false;
    }

    if (try_import_clipboard_image()) {
        mark_layout_dirty();
        mark_dirty();
        return true;
    }

    if (input_ != nullptr) {
        const std::string text = tuinator::clipboard::get();
        if (!text.empty()) {
            input_->insert_pasted_text(text);
            mark_dirty();
        }
    }
    return true;
}
