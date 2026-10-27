#include "chat_session_header.hpp"

#include "ui_test_ids.hpp"

#include "ui_anim.hpp"
#include "ui_layout.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

ChatSessionHeader::ChatSessionHeader(tuinator::Theme theme) : theme_(std::move(theme)) {
    set_test_id(taiji::ui::test_id::kChatSessionHeader);
}

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

void ChatSessionHeader::tick_persona_pulse(int interval_ms) {
    if (persona_pulse_ms_ <= 0 && ready_pulse_ms_ <= 0) {
        return;
    }
    if (persona_pulse_ms_ > 0) {
        persona_pulse_ms_ = std::max(0, persona_pulse_ms_ - interval_ms);
    }
    if (ready_pulse_ms_ > 0) {
        ready_pulse_ms_ = std::max(0, ready_pulse_ms_ - interval_ms);
    }
    mark_dirty();
}

void ChatSessionHeader::trigger_response_ready_pulse() {
    ready_pulse_ms_ = 1600;
    mark_dirty();
}

void ChatSessionHeader::set_template_persona(std::string label, const std::string& theme_color_hex) {
    tuinator::Rgb rgb{};
    const bool has_rgb = ui_parse_hex_rgb(theme_color_hex, rgb);
    if (template_persona_ == label && template_persona_rgb_set_ == has_rgb &&
        (!has_rgb || (template_persona_rgb_.r == rgb.r && template_persona_rgb_.g == rgb.g &&
                      template_persona_rgb_.b == rgb.b))) {
        return;
    }
    template_persona_ = std::move(label);
    template_persona_rgb_set_ = has_rgb;
    if (has_rgb) {
        template_persona_rgb_ = rgb;
    }
    persona_pulse_ms_ = 450;
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

    if (project_.empty() && session_.empty() && template_persona_.empty()) {
        return;
    }

    paint_bounds_background(ctx, ui_main_panel_style());

    const int max_cols = std::max(0, bounds_.width - kUiPadX * 2);
    int x = kUiPadX;

    tuinator::Style project_style = ui_resolve_main_text(ui_secondary_style(theme_));
    tuinator::Style session_style = ui_resolve_main_text(theme_.heading);
    session_style.bold = true;
    session_style.foreground_rgb = ui_list_active_fg();
    if (ready_pulse_ms_ > 0) {
        session_style.foreground_rgb =
            ui_pulse_rgb(ui_status_ok(), ui_list_active_fg(), static_cast<double>(1600 - ready_pulse_ms_), 520.0);
        session_style.background_rgb =
            ui_fade_toward(ui_session_active_bg(), ui_sidebar_bg(), static_cast<double>(ready_pulse_ms_) / 1600.0);
    }
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
            x += tuinator::text_display_width(session_.substr(0, session_bytes));
        }
    } else if (!project_.empty()) {
        // Only project name — show it as the active title.
        // (Already painted in secondary style above; no session segment.)
    }

    if (!template_persona_.empty() && x < kUiPadX + max_cols) {
        tuinator::Style persona_style = ui_resolve_main_text(session_style);
        if (template_persona_rgb_set_) {
            persona_style.foreground_rgb = template_persona_rgb_;
        } else {
            persona_style.foreground_rgb = ui_assistant_chrome_accent_rgb();
        }
        if (persona_pulse_ms_ > 0) {
            persona_style.bold = true;
            persona_style.background_rgb = ui_session_active_bg();
        }
        const std::string sep = " · ";
        const int remaining = std::max(0, kUiPadX + max_cols - x);
        std::string chip = sep + template_persona_;
        const std::size_t chip_bytes = tuinator::text_byte_length_for_width(chip, remaining);
        if (chip_bytes > 0) {
            ctx.canvas.draw_text({x, 0}, chip.substr(0, chip_bytes), persona_style);
        }
    }
}
