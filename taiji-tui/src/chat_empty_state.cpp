#include "chat_empty_state.hpp"

#include "taiji_block_logo.hpp"
#include "ui_chat_column.hpp"
#include "ui_layout.hpp"
#include "ui_locale.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cctype>

namespace {

struct QuickAction {
    const char* icon;
    const char* label_key;
    const char* key_digit;
    char key_letter;
};

constexpr int kQuickActionPillWidth = 32;
constexpr int kQuickActionRowGap = 1;
constexpr int kQuickActionPadX = 2;

constexpr QuickAction kQuickActions[] = {
    {kNfFile, "empty.quick_index_codebase", "1", 'c'},
    {kNfTest, "empty.quick_generate_tests", "2", 't'},
    {kNfGit, "empty.quick_analyze_git", "3", 'g'},
};

tuinator::Rgb quick_action_icon_color(std::size_t index) {
    switch (index) {
    case 0:
        return ui_quick_action_code();
    case 1:
        return ui_quick_action_test();
    case 2:
        return ui_quick_action_git();
    default:
        return ui_quick_action_code();
    }
}

void paint_quick_action_pill(tuinator::PaintContext& ctx, const tuinator::Widget& widget, int x, int y, int width,
                             const QuickAction& action, std::size_t action_index, const tuinator::Theme& theme) {
    if (width <= 0) {
        return;
    }

    tuinator::Style pill_bg;
    pill_bg.background_rgb = ui_quick_action_pill_bg();
    ctx.canvas.fill_rect({x, y, width, 1}, ' ', pill_bg);

    tuinator::Style body = ui_resolve_main_text(theme.label);
    body.background_rgb = pill_bg.background_rgb;
    body.bold = false;
    tuinator::Style icon = body;
    icon.foreground_rgb = quick_action_icon_color(action_index);
    tuinator::Style key_hint = ui_resolve_main_text(ui_secondary_style(theme));
    key_hint.background_rgb = pill_bg.background_rgb;

    std::string pad_left(kQuickActionPadX, ' ');
    const std::string icon_text = pad_left + action.icon + " ";
    const int icon_w = tuinator::text_display_width(icon_text);
    ctx.canvas.draw_text({x, y}, icon_text, icon);

    const std::string localized = ui_shell(action.label_key);
    const std::string label_text = localized.empty() ? action.label_key : localized;
    const std::string pad_right(kQuickActionPadX, ' ');
    const int key_w = tuinator::text_display_width(pad_right + action.key_digit);
    const int label_max = std::max(0, width - icon_w - key_w);
    const std::size_t label_bytes = tuinator::text_byte_length_for_width(label_text, label_max);
    ctx.canvas.draw_text({x + icon_w, y}, label_text.substr(0, label_bytes), body);

    const int key_x = x + width - key_w;
    ctx.canvas.draw_text({key_x, y}, pad_right + action.key_digit, key_hint);
}

void paint_git_status(tuinator::PaintContext& ctx, const tuinator::Widget& widget, int x, int y,
                      const GitWorkspaceStatus& git, const tuinator::Theme& theme) {
    if (!git.valid) {
        return;
    }

    tuinator::Style icon = ui_resolve_main_text(theme.label);
    icon.foreground_rgb = ui_git_branch_accent();
    tuinator::Style branch = icon;
    branch.bold = false;
    branch.foreground_rgb = ui_git_branch_accent();
    tuinator::Style sep = ui_resolve_main_text(ui_secondary_style(theme));
    tuinator::Style changes = ui_resolve_main_text(theme.label);
    changes.foreground_rgb = ui_status_warn();

    int cursor = x;
    ctx.canvas.draw_text({cursor, y}, kNfGitBranch, icon);
    cursor += tuinator::text_display_width(kNfGitBranch);
    ctx.canvas.draw_text({cursor, y}, " ", sep);
    cursor += 1;
    ctx.canvas.draw_text({cursor, y}, git.branch, branch);
    cursor += tuinator::text_display_width(git.branch);

    if (git.uncommitted > 0) {
        const std::string mid = " • ";
        ctx.canvas.draw_text({cursor, y}, mid, sep);
        cursor += tuinator::text_display_width(mid);
        char buf[48];
        const std::string pattern =
            git.uncommitted == 1 ? ui_shell("empty.uncommitted_one") : ui_shell("empty.uncommitted_many");
        const std::string formatted =
            ui_format(pattern.empty() ? "{count} uncommitted change(s)" : pattern, "count", std::to_string(git.uncommitted));
        std::snprintf(buf, sizeof(buf), "%s", formatted.c_str());
        ctx.canvas.draw_text({cursor, y}, buf, changes);
    }
}

bool matches_quick_action(const tuinator::KeyPress& key, const QuickAction& action) {
    if (key.ctrl || key.alt || key.shift) {
        return false;
    }
    if (key.character == action.key_digit[0]) {
        return true;
    }
    if (key.character != '\0') {
        return static_cast<char>(std::tolower(static_cast<unsigned char>(key.character))) == action.key_letter;
    }
    return false;
}

} // namespace

ChatEmptyState::ChatEmptyState(tuinator::Theme theme) : theme_(std::move(theme)) {}

void ChatEmptyState::set_visible(bool visible) {
    if (visible_ == visible) {
        return;
    }
    visible_ = visible;
    if (visible_) {
        reveal_ms_ = 0;
    }
    mark_dirty();
    mark_layout_dirty();
}

void ChatEmptyState::tick(int dt_ms) {
    if (!visible_) {
        return;
    }
    anim_clock_ms_ = (anim_clock_ms_ + dt_ms) % 600000;
    if (reveal_ms_ < 400) {
        reveal_ms_ = std::min(400, reveal_ms_ + dt_ms);
    }
    // Keep repainting so the logo shimmer / idle pulse stays alive.
    mark_dirty();
}

void ChatEmptyState::set_workspace_path(std::string path) {
    if (workspace_path_ == path) {
        return;
    }
    workspace_path_ = std::move(path);
    mark_dirty();
}

void ChatEmptyState::set_on_quick_action(QuickActionHandler handler) {
    on_quick_action_ = std::move(handler);
}

tuinator::Size ChatEmptyState::preferred_size() const {
    if (!visible_) {
        return {0, 0};
    }
    return {kQuickActionPillWidth + 4, 14};
}

void ChatEmptyState::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

bool ChatEmptyState::try_quick_action(const tuinator::KeyPress& key) {
    if (!visible_ || !on_quick_action_) {
        return false;
    }
    for (std::size_t i = 0; i < std::size(kQuickActions); ++i) {
        if (matches_quick_action(key, kQuickActions[i])) {
            on_quick_action_(static_cast<int>(i));
            return true;
        }
    }
    return false;
}

bool ChatEmptyState::handle_event(const tuinator::Event& event) {
    const auto* key = std::get_if<tuinator::KeyPress>(&event);
    if (key == nullptr) {
        return false;
    }
    return try_quick_action(*key);
}

void ChatEmptyState::paint(tuinator::PaintContext& ctx) const {
    if (!visible_ || bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    paint_bounds_background(ctx, ui_main_panel_style());

    const tuinator::Style hint = ui_resolve_main_text(ui_secondary_style(theme_));

    const int quick_rows = static_cast<int>(std::size(kQuickActions));
    const int block_h_est = 10 + quick_rows + (quick_rows - 1) * kQuickActionRowGap + 4;
    const int start_y = std::max(0, (bounds_.height - block_h_est) / 2);

    const ChatStreamLayout column = chat_stream_layout_for_panel(bounds_.width);
    const int stream_x = column.offset_x;
    const int stream_w = column.content_width;

    const int logo_max_w = std::max(0, stream_w - kUiPadX * 2);
    const int logo_max_h = std::max(1, bounds_.height / 3);
    const TaijiBlockLogoSize logo_size =
        paint_taiji_block_logo(ctx, *this, stream_w, start_y, logo_max_w, logo_max_h, theme_, stream_x, anim_clock_ms_);

    const int pill_w = std::min(kQuickActionPillWidth, std::max(12, stream_w - kUiPadX * 2));
    const int pill_x = stream_x + std::max(kUiPadX, (stream_w - pill_w) / 2);
    const int pills_top = start_y + logo_size.height + 2;
    for (std::size_t i = 0; i < std::size(kQuickActions); ++i) {
        const int delay = static_cast<int>(i) * 40;
        if (reveal_ms_ < delay) {
            continue;
        }
        const int base_y = pills_top + static_cast<int>(i) * (1 + kQuickActionRowGap);
        const int slide = std::max(0, (delay + 40 - reveal_ms_) / 10);
        const int pill_y_anim = base_y + slide;
        if (pill_y_anim >= bounds_.height) {
            break;
        }
        paint_quick_action_pill(ctx, *this, pill_x, pill_y_anim, pill_w, kQuickActions[i], i, theme_);
    }
    int pill_y = pills_top + static_cast<int>(std::size(kQuickActions)) * (1 + kQuickActionRowGap);

    const GitWorkspaceStatus git = read_git_workspace_status(workspace_path_);
    const int git_y = pill_y + 1;
    if (git.valid && git_y < bounds_.height) {
        const int git_w = [&]() {
            int w = tuinator::text_display_width(kNfGitBranch) + 1 + tuinator::text_display_width(git.branch);
            if (git.uncommitted > 0) {
                const std::string pattern =
                    git.uncommitted == 1 ? ui_shell("empty.uncommitted_one") : ui_shell("empty.uncommitted_many");
                const std::string changes = ui_format(pattern.empty() ? "{count} changes" : pattern, "count",
                                                      std::to_string(git.uncommitted));
                w += tuinator::text_display_width(" • " + changes);
            }
            return w;
        }();
        const int git_x = stream_x + std::max(kUiPadX, (stream_w - git_w) / 2);
        paint_git_status(ctx, *this, git_x, git_y, git, theme_);
    }

    const std::string hint_line_text = ui_shell("empty.hint_send");
    const std::string hint_line = hint_line_text.empty() ? "Send a message to start" : hint_line_text;
    const int hint_w = tuinator::text_display_width(hint_line);
    const int hint_x = stream_x + std::max(0, (stream_w - hint_w) / 2);
    const int hint_y = git_y + (git.valid ? 2 : 1);
    if (hint_y < bounds_.height) {
        ctx.canvas.draw_text({hint_x, hint_y}, hint_line, hint);
    }
}
