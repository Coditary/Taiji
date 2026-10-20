#include "chat_session_header.hpp"

#include "ui_layout.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

ChatSessionHeader::ChatSessionHeader(tuinator::Theme theme) : theme_(std::move(theme)) {}

void ChatSessionHeader::set_project_name(std::string name) {
    if (project_ == name) {
        return;
    }
    project_ = std::move(name);
    mark_dirty();
}

void ChatSessionHeader::set_session_name(std::string name) {
    if (session_ == name) {
        return;
    }
    session_ = std::move(name);
    mark_dirty();
}

tuinator::Size ChatSessionHeader::preferred_size() const {
    return {20, 1};
}

void ChatSessionHeader::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void ChatSessionHeader::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    if (project_.empty() && session_.empty()) {
        return;
    }

    paint_bounds_background(ctx, ui_main_panel_style());

    const int max_cols = std::max(0, bounds_.width - kUiPadX * 2);
    int x = kUiPadX;

    tuinator::Style project_style = ui_resolve_main_text(ui_secondary_style(theme_));
    tuinator::Style session_style = ui_resolve_main_text(theme_.heading);
    session_style.bold = true;
    session_style.foreground_rgb = ui_list_active_fg();
    tuinator::Style slash_style = project_style;

    if (project_.empty() && !session_.empty()) {
        const std::size_t session_bytes = tuinator::text_byte_length_for_width(session_, max_cols);
        if (session_bytes > 0) {
            ctx.canvas.draw_text({x, 0}, session_.substr(0, session_bytes), session_style);
        }
        return;
    }

    if (!project_.empty()) {
        const std::string icon_prefix = std::string(kNfFolder) + " ";
        const std::size_t icon_bytes = tuinator::text_byte_length_for_width(icon_prefix, max_cols);
        if (icon_bytes > 0) {
            ctx.canvas.draw_text({x, 0}, icon_prefix.substr(0, icon_bytes), project_style);
            x += tuinator::text_display_width(icon_prefix.substr(0, icon_bytes));
        }
        const std::size_t project_bytes = tuinator::text_byte_length_for_width(project_, max_cols);
        if (project_bytes > 0) {
            ctx.canvas.draw_text({x, 0}, project_.substr(0, project_bytes), project_style);
            x += tuinator::text_display_width(project_.substr(0, project_bytes));
        }
    }

    if (!project_.empty() && !session_.empty()) {
        const char* slash = " / ";
        ctx.canvas.draw_text({x, 0}, slash, slash_style);
        x += tuinator::text_display_width(slash);
    }

    if (!session_.empty()) {
        const std::size_t session_bytes = tuinator::text_byte_length_for_width(session_, max_cols);
        if (session_bytes > 0) {
            ctx.canvas.draw_text({x, 0}, session_.substr(0, session_bytes), session_style);
        }
    } else if (!project_.empty()) {
        // Only project name — show it as the active title.
        // (Already painted in secondary style above; no session segment.)
    }
}
