#include "session_todo_panel.hpp"

#include "ui_layout.hpp"
#include "ui_locale.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <cctype>

namespace {

tuinator::Style todo_row_fill_style() {
    tuinator::Style style;
    style.background_rgb = ui_sidebar_bg();
    return style;
}

const char* status_glyph(SessionTodoItem::Status status) {
    switch (status) {
    case SessionTodoItem::Status::Done:
        return "✓";
    case SessionTodoItem::Status::Active:
        return "◐";
    default:
        return "○";
    }
}

tuinator::Style status_glyph_style(SessionTodoItem::Status status) {
    tuinator::Style style;
    switch (status) {
    case SessionTodoItem::Status::Done:
        style.foreground_rgb = ui_status_ok();
        style.dim = true;
        break;
    case SessionTodoItem::Status::Active:
        style.foreground_rgb = ui_git_branch_accent();
        style.bold = true;
        break;
    default:
        style.foreground_rgb = ui_status_idle();
        break;
    }
    return style;
}

} // namespace

std::vector<SessionTodoItem> parse_session_todos_text(const std::string& text) {
    std::vector<SessionTodoItem> items;
    std::size_t pos = 0;
    while (pos < text.size()) {
        const std::size_t line_end = text.find('\n', pos);
        const std::string line =
            text.substr(pos, line_end == std::string::npos ? std::string::npos : line_end - pos);
        pos = line_end == std::string::npos ? text.size() : line_end + 1;
        if (line.size() < 4 || line[0] != '[') {
            continue;
        }
        SessionTodoItem item;
        const char mark = line[1];
        if (mark == 'x') {
            item.status = SessionTodoItem::Status::Done;
        } else if (mark == '>') {
            item.status = SessionTodoItem::Status::Active;
        } else {
            item.status = SessionTodoItem::Status::Pending;
        }
        const std::size_t bracket_end = line.find(']');
        if (bracket_end == std::string::npos || bracket_end + 1 >= line.size()) {
            continue;
        }
        std::string rest = line.substr(bracket_end + 1);
        while (!rest.empty() && (rest.front() == ' ' || rest.front() == '\t')) {
            rest.erase(rest.begin());
        }
        const std::size_t dot = rest.find(". ");
        if (dot != std::string::npos && dot <= 8) {
            bool id_chars = !rest.empty();
            for (std::size_t i = 0; i < dot && id_chars; ++i) {
                const unsigned char ch = static_cast<unsigned char>(rest[i]);
                if (!std::isalnum(ch) && rest[i] != '-' && rest[i] != '_') {
                    id_chars = false;
                }
            }
            if (id_chars) {
                rest = rest.substr(dot + 2);
            }
        }
        item.task = rest;
        if (item.task.empty()) {
            continue;
        }
        items.push_back(std::move(item));
    }
    return items;
}

SessionTodoPanel::SessionTodoPanel(tuinator::Theme theme) : theme_(std::move(theme)) {}

void SessionTodoPanel::set_items(std::vector<SessionTodoItem> items) {
    if (items.size() == items_.size()) {
        bool same = true;
        for (std::size_t i = 0; i < items.size(); ++i) {
            if (items[i].task != items_[i].task || items[i].status != items_[i].status) {
                same = false;
                break;
            }
        }
        if (same) {
            return;
        }
    }
    items_ = std::move(items);
    mark_dirty();
    mark_layout_dirty();
}

tuinator::Size SessionTodoPanel::preferred_size() const {
    if (items_.empty()) {
        return {12, 2};
    }
    return {12, 1 + static_cast<int>(items_.size())};
}

void SessionTodoPanel::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void SessionTodoPanel::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    ui_fill_widget_bounds(ctx, *this, ui_sidebar_bg());

    const int pad = kUiPadX;
    tuinator::Style heading;
    heading.bold = true;
    heading.foreground_rgb = ui_section_heading();
    heading = ui_resolve_sidebar_text(ctx, *this, heading);
    tuinator::Style muted = ui_resolve_sidebar_text(ctx, *this, ui_secondary_style(theme_));
    tuinator::Style label = ui_resolve_sidebar_text(ctx, *this, theme_.label);

    std::string title = ui_shell("sidebar.todos_section");
    if (title.empty()) {
        title = "TODOS";
    }
    if (!items_.empty()) {
        const std::string pattern = ui_shell("sidebar.todos_section_count");
        title = ui_format(pattern.empty() ? "TODOS ({count})" : pattern, "count", std::to_string(items_.size()));
    }
    ctx.canvas.fill_rect({0, 0, bounds_.width, 1}, ' ', todo_row_fill_style());
    ctx.canvas.draw_text({pad, 0}, title, heading);

    int y = 1;
    if (items_.empty()) {
        const std::string empty = ui_shell("sidebar.todos_empty");
        const std::string state = empty.empty() ? "none" : empty;
        if (y < bounds_.height) {
            ctx.canvas.fill_rect({0, y, bounds_.width, 1}, ' ', todo_row_fill_style());
            tuinator::Style dot = muted;
            dot.foreground_rgb = ui_status_idle();
            ctx.canvas.draw_text({pad, y}, "○", dot);
            const int dot_w = tuinator::text_display_width("○");
            ctx.canvas.draw_text({pad + dot_w + 1, y}, state, muted);
        }
        return;
    }

    for (const SessionTodoItem& item : items_) {
        if (y >= bounds_.height) {
            break;
        }
        ctx.canvas.fill_rect({0, y, bounds_.width, 1}, ' ', todo_row_fill_style());

        tuinator::Style glyph_style = status_glyph_style(item.status);
        glyph_style = ui_resolve_sidebar_text(ctx, *this, glyph_style);
        const char* glyph = status_glyph(item.status);
        ctx.canvas.draw_text({pad, y}, glyph, glyph_style);
        const int glyph_w = tuinator::text_display_width(glyph);

        tuinator::Style task_style = label;
        if (item.status == SessionTodoItem::Status::Done) {
            task_style.dim = true;
        } else if (item.status == SessionTodoItem::Status::Active) {
            task_style.bold = true;
        }
        const int name_x = pad + glyph_w + 1;
        const int max_cols = std::max(0, bounds_.width - name_x - pad);
        const std::size_t bytes = tuinator::text_byte_length_for_width(item.task, max_cols);
        if (bytes > 0) {
            ctx.canvas.draw_text({name_x, y}, item.task.substr(0, bytes), task_style);
        }
        ++y;
    }

    if (y < bounds_.height) {
        ctx.canvas.fill_rect({0, y, bounds_.width, bounds_.height - y}, ' ', todo_row_fill_style());
    }
}
