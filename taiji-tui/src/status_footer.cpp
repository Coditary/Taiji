#include "status_footer.hpp"

#include "ui_test_ids.hpp"
#include "ui_layout.hpp"
#include "ui_navigation.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <string>
#include <vector>

namespace {

struct FooterSegment {
    std::string key;
    std::string label;
    bool badge = false;
};

std::vector<FooterSegment> parse_footer_segments(const std::string& text) {
    std::vector<FooterSegment> segments;
    std::size_t start = 0;
    while (start < text.size()) {
        const std::size_t sep = text.find(" · ", start);
        const std::string part = sep == std::string::npos ? text.substr(start) : text.substr(start, sep - start);
        if (!part.empty()) {
            const std::size_t colon = part.find(": ");
            if (colon != std::string::npos && colon > 0) {
                segments.push_back(
                    FooterSegment{.key = part.substr(0, colon), .label = part.substr(colon + 2), .badge = true});
            } else {
                segments.push_back(FooterSegment{.key = part, .label = "", .badge = false});
            }
        }
        if (sep == std::string::npos) {
            break;
        }
        start = sep + 3;
    }
    return segments;
}

int draw_footer_segments(tuinator::PaintContext& ctx, const tuinator::Widget& widget, int x, int y, int max_width,
                         const std::vector<FooterSegment>& segments, const tuinator::Style& plain,
                         const tuinator::Style& badge_key, const tuinator::Style& badge_label) {
    int cursor = x;
    for (std::size_t i = 0; i < segments.size(); ++i) {
        if (i > 0) {
            const char* sep = " │ ";
            if (cursor + tuinator::text_display_width(sep) > max_width) {
                break;
            }
            ctx.canvas.draw_text({cursor, y}, sep, plain);
            cursor += tuinator::text_display_width(sep);
        }

        const FooterSegment& seg = segments[i];
        if (!seg.badge) {
            const int w = tuinator::text_display_width(seg.key);
            if (cursor + w > max_width) {
                break;
            }
            ctx.canvas.draw_text({cursor, y}, seg.key, plain);
            cursor += w;
            continue;
        }

        const std::string key_text = " " + seg.key + " ";
        const int key_w = tuinator::text_display_width(key_text);
        if (cursor + key_w > max_width) {
            break;
        }
        tuinator::Style key_bg = badge_key;
        key_bg.background_rgb = ui_badge_bg();
        ctx.canvas.fill_rect({cursor, y, key_w, 1}, ' ', key_bg);
        ctx.canvas.draw_text({cursor, y}, key_text, key_bg);
        cursor += key_w;

        if (!seg.label.empty()) {
            const std::string label_text = " " + seg.label;
            const int label_w = tuinator::text_display_width(label_text);
            if (cursor + label_w > max_width) {
                break;
            }
            ctx.canvas.draw_text({cursor, y}, label_text, badge_label);
            cursor += label_w;
        }
    }
    return cursor;
}

} // namespace

StatusFooter::StatusFooter(tuinator::Style path_style, tuinator::Style status_style)
    : path_style_(path_style), status_style_(status_style) {
    set_test_id(taiji::ui::test_id::kStatusFooter);
}

void StatusFooter::set_path(std::string text) {
    if (path_ == text) {
        return;
    }
    path_ = std::move(text);
    mark_dirty();
}

void StatusFooter::set_status(std::string text) {
    if (status_ == text) {
        return;
    }
    status_ = std::move(text);
    mark_dirty();
}

void StatusFooter::set_vim_mode_badge(bool show, UiInputMode mode) {
    if (show_mode_badge_ == show && mode_badge_ == mode) {
        return;
    }
    show_mode_badge_ = show;
    mode_badge_ = mode;
    mark_dirty();
}

tuinator::Size StatusFooter::preferred_size() const {
    return {1, 1};
}

void StatusFooter::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void StatusFooter::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    paint_bounds_background(ctx, ui_sidebar_panel_style());

    tuinator::Style path = ui_resolve_footer_text(path_style_);
    path.foreground_rgb = ui_secondary_text();
    tuinator::Style status = ui_resolve_footer_text(status_style_);
    tuinator::Style badge_key = status;
    badge_key.bold = true;
    badge_key.foreground_rgb = ui_text();
    tuinator::Style badge_label = status;
    badge_label.dim = true;
    badge_label.foreground_rgb = ui_secondary_text();

    const int pad = kUiPadX;
    int left_cursor = pad;
    if (show_mode_badge_) {
        const char* label = mode_badge_ == UiInputMode::Normal ? " NORMAL " : " INSERT ";
        tuinator::Rgb accent = mode_badge_ == UiInputMode::Normal ? ui_quick_action_test() : ui_quick_action_code();
        tuinator::Style badge;
        badge.bold = true;
        badge.foreground_rgb = ui_text();
        badge.background_rgb = accent;
        const int badge_w = tuinator::text_display_width(label);
        ctx.canvas.fill_rect({left_cursor, 0, badge_w, 1}, ' ', badge);
        ctx.canvas.draw_text({left_cursor, 0}, label, badge);
        left_cursor += badge_w + 1;
    }

    const std::string display_path = path_.empty() ? "" : shorten_display_path(path_);
    if (!display_path.empty()) {
        const int max_width = std::max(0, bounds_.width / 2 - left_cursor);
        const std::size_t bytes = tuinator::text_byte_length_for_width(display_path, max_width);
        if (bytes > 0) {
            ctx.canvas.draw_text({left_cursor, 0}, display_path.substr(0, bytes), path);
        }
    }

    if (!status_.empty()) {
        const std::vector<FooterSegment> segments = parse_footer_segments(status_);
        const bool use_badges = segments.size() > 1 || (segments.size() == 1 && segments.front().badge);
        const int right_pad = pad;
        const int max_x = bounds_.width - right_pad;
        if (use_badges) {
            int total_w = 0;
            for (const FooterSegment& seg : segments) {
                if (!seg.badge) {
                    total_w += tuinator::text_display_width(seg.key);
                } else {
                    total_w += tuinator::text_display_width(" " + seg.key + " ");
                    if (!seg.label.empty()) {
                        total_w += tuinator::text_display_width(" " + seg.label);
                    }
                }
            }
            total_w += tuinator::text_display_width(" │ ") * static_cast<int>(segments.size() > 1 ? segments.size() - 1 : 0);
            const int start_x = std::max(pad, max_x - total_w);
            draw_footer_segments(ctx, *this, start_x, 0, max_x, segments, status, badge_key, badge_label);
        } else {
            const int status_width = tuinator::text_display_width(status_);
            const int x = std::max(pad, max_x - status_width);
            const std::size_t bytes = tuinator::text_byte_length_for_width(status_, bounds_.width - pad);
            if (bytes > 0) {
                ctx.canvas.draw_text({x, 0}, status_.substr(0, bytes), status);
            }
        }
    }
}
