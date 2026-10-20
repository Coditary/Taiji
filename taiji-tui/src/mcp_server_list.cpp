#include "mcp_server_list.hpp"

#include "ui_anim.hpp"
#include "ui_layout.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <cstdio>

namespace {

tuinator::Style mcp_dot_style(bool connected) {
    tuinator::Style style;
    style.foreground_rgb = connected ? ui_status_ok() : ui_status_idle();
    return style;
}

tuinator::Style mcp_row_fill_style() {
    tuinator::Style style;
    style.background_rgb = ui_sidebar_bg();
    return style;
}

} // namespace

McpServerList::McpServerList(tuinator::Theme theme) : theme_(std::move(theme)) {}

void McpServerList::set_servers(std::vector<std::string> names) {
    if (servers_ == names) {
        return;
    }
    servers_ = std::move(names);
    if (selected_row_ >= static_cast<int>(servers_.size())) {
        selected_row_ = servers_.empty() ? -1 : static_cast<int>(servers_.size()) - 1;
    }
    mark_dirty();
    mark_layout_dirty();
}

void McpServerList::set_active_server(const std::string& name) {
    if (active_server_ == name) {
        return;
    }
    active_server_ = name;
    active_pulse_ms_ = 0;
    mark_dirty();
}

void McpServerList::clear_active_server() {
    if (active_server_.empty()) {
        return;
    }
    active_server_.clear();
    mark_dirty();
}

void McpServerList::tick(int dt_ms) {
    if (active_server_.empty()) {
        return;
    }
    active_pulse_ms_ = (active_pulse_ms_ + dt_ms) % 60000;
    mark_dirty();
}

void McpServerList::set_status_line(std::string text) {
    if (status_line_ == text) {
        return;
    }
    status_line_ = std::move(text);
    mark_dirty();
    mark_layout_dirty();
}

tuinator::Size McpServerList::preferred_size() const {
    if (servers_.empty()) {
        return {12, 2};
    }
    return {12, 1 + static_cast<int>(servers_.size())};
}

void McpServerList::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

int McpServerList::row_at_y(int local_y) const {
    if (local_y <= 0) {
        return -1;
    }
    const int row = local_y - 1;
    if (row < 0 || row >= static_cast<int>(servers_.size())) {
        return -1;
    }
    return row;
}

void McpServerList::paint(tuinator::PaintContext& ctx) const {
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

    char title_buf[32];
    if (servers_.empty()) {
        std::snprintf(title_buf, sizeof(title_buf), "MCP");
    } else {
        std::snprintf(title_buf, sizeof(title_buf), "MCP (%zu)", servers_.size());
    }
    ctx.canvas.fill_rect({0, 0, bounds_.width, 1}, ' ', mcp_row_fill_style());
    ctx.canvas.draw_text({pad, 0}, title_buf, heading);

    int y = 1;
    if (servers_.empty()) {
        const std::string state = status_line_.empty() ? "not connected" : status_line_;
        if (y < bounds_.height) {
            ctx.canvas.fill_rect({0, y, bounds_.width, 1}, ' ', mcp_row_fill_style());
            tuinator::Style dot = mcp_dot_style(false);
            dot = ui_resolve_sidebar_text(ctx, *this, dot);
            ctx.canvas.draw_text({pad, y}, "○", dot);
            const int dot_w = tuinator::text_display_width("○");
            ctx.canvas.draw_text({pad + dot_w + 1, y}, state, muted);
        }
        return;
    }

    for (int row = 0; row < static_cast<int>(servers_.size()); ++row) {
        if (y >= bounds_.height) {
            break;
        }
        const std::string& name = servers_[static_cast<std::size_t>(row)];
        tuinator::Style row_bg = mcp_row_fill_style();
        if (row == selected_row_) {
            row_bg.background_rgb = ui_session_active_bg();
        } else if (row == hovered_row_) {
            row_bg.background_rgb = ui_list_active_bg();
        }
        ctx.canvas.fill_rect({0, y, bounds_.width, 1}, ' ', row_bg);

        const tuinator::Rgb row_rgb =
            row_bg.background_rgb.has_value() ? *row_bg.background_rgb : ui_sidebar_bg();
        tuinator::Style dot = ui_resolve_panel_text(mcp_dot_style(true), row_rgb);
        if (!active_server_.empty() && name == active_server_) {
            dot.foreground_rgb = ui_pulse_rgb(ui_status_ok(), ui_git_branch_accent(), active_pulse_ms_, 700.0);
        }
        tuinator::Style name_style = ui_resolve_panel_text(theme_.label, row_rgb);
        if (row == selected_row_) {
            name_style.bold = true;
            name_style.foreground_rgb = ui_list_active_fg();
        }

        ctx.canvas.draw_text({pad, y}, "●", dot);
        const int name_x = pad + tuinator::text_display_width("●") + 1;
        const int max_cols = std::max(0, bounds_.width - name_x - pad);
        const std::size_t bytes = tuinator::text_byte_length_for_width(name, max_cols);
        if (bytes > 0) {
            ctx.canvas.draw_text({name_x, y}, name.substr(0, bytes), name_style);
        }
        ++y;
    }

    if (y < bounds_.height) {
        ctx.canvas.fill_rect({0, y, bounds_.width, bounds_.height - y}, ' ', mcp_row_fill_style());
    }
}

bool McpServerList::handle_event(const tuinator::Event& event) {
    const auto* mouse = std::get_if<tuinator::MouseEvent>(&event);
    if (mouse == nullptr) {
        return false;
    }
    const tuinator::Point local{mouse->position.x - bounds_.x, mouse->position.y - bounds_.y};
    const int row = row_at_y(local.y);

    if (mouse->action == tuinator::MouseAction::Move) {
        if (row != hovered_row_) {
            hovered_row_ = row;
            mark_dirty();
        }
        return bounds_.contains(mouse->position);
    }

    if (mouse->button == tuinator::MouseButton::Left && mouse->action == tuinator::MouseAction::Release && row >= 0) {
        if (selected_row_ != row) {
            selected_row_ = row;
            mark_dirty();
        }
        return true;
    }

    return bounds_.contains(mouse->position);
}
