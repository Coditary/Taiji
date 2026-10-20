#include "chat_input.hpp"

#include "ui_anim.hpp"
#include "ui_chat_column.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/glyphs.hpp>
#include <tuinator/render/text.hpp>

#include <optional>

#include <algorithm>

ChatTextInput::ChatTextInput(tuinator::TextInputOptions options, tuinator::Style style, tuinator::Style focused_style)
    : tuinator::TextInput(options, style, focused_style), options_(std::move(options)) {}

void ChatTextInput::sync_field_styles(tuinator::Style normal, tuinator::Style focused) {
    set_field_styles(std::move(normal), std::move(focused));
}

void ChatTextInput::sync_frame_border_style() {
    const tuinator::BorderStyle style = ui_frame_border_style();
    const bool show_border = ui_corners_framed();
    if (options_.framed_border_style == style && options_.framed_show_border == show_border &&
        !options_.framed_border_glyphs.has_value()) {
        return;
    }
    options_.framed_border_style = style;
    options_.framed_show_border = show_border;
    options_.framed_border_glyphs.reset();
    set_options(options_);
    mark_dirty();
}

void ChatTextInput::set_model_info(std::string model, std::string provider) {
    if (model_label_ == model && provider_label_ == provider) {
        return;
    }
    model_label_ = std::move(model);
    provider_label_ = std::move(provider);

    std::string border;
    if (!model_label_.empty()) {
        border = std::string(kNfModel) + " " + model_label_;
        if (!provider_label_.empty()) {
            border += "  ";
            border += provider_label_;
        }
    }
    options_.border_label = std::move(border);
    set_options(options_);
    mark_dirty();
}

void ChatTextInput::set_quick_key_handler(QuickKeyHandler handler) {
    quick_key_handler_ = std::move(handler);
}

void ChatTextInput::set_composer_intercept_key(QuickKeyHandler handler) {
    composer_intercept_key_ = std::move(handler);
}

void ChatTextInput::set_chrome_busy(bool busy) {
    if (chrome_busy_ == busy) {
        return;
    }
    chrome_busy_ = busy;
    mark_dirty();
}

void ChatTextInput::tick_chrome(int dt_ms) {
    chrome_anim_ms_ = (chrome_anim_ms_ + dt_ms) % 600000;
    if (chrome_busy_) {
        mark_dirty();
    }

    if (!is_focused()) {
        caret_visible_ = true;
        caret_blink_ms_ = 0;
        caret_solid_ms_ = 0;
        return;
    }

    if (caret_solid_ms_ > 0) {
        caret_solid_ms_ = std::max(0, caret_solid_ms_ - dt_ms);
        caret_visible_ = true;
        return;
    }

    caret_blink_ms_ += dt_ms;
    if (caret_blink_ms_ >= 530) {
        caret_blink_ms_ = 0;
        caret_visible_ = !caret_visible_;
        mark_dirty();
    }
}

void ChatTextInput::note_caret_activity() {
    caret_solid_ms_ = 450;
    caret_visible_ = true;
    caret_blink_ms_ = 0;
}

void ChatTextInput::layout(tuinator::Rect bounds) {
    tuinator::TextInput::layout(bounds);
}

bool ChatTextInput::handle_event(const tuinator::Event& event) {
    if (composer_intercept_key_) {
        if (const auto* key = std::get_if<tuinator::KeyPress>(&event)) {
            if (is_focused() && composer_intercept_key_(*key)) {
                note_caret_activity();
                return true;
            }
        }
    }
    if (quick_key_handler_) {
        if (const auto* key = std::get_if<tuinator::KeyPress>(&event)) {
            if (value().empty() && !is_focused() && quick_key_handler_(*key)) {
                return true;
            }
        }
    }
    const bool handled = tuinator::TextInput::handle_event(event);
    if (handled && is_focused()) {
        note_caret_activity();
    }
    return handled;
}

void ChatTextInput::paint_border_label(tuinator::PaintContext& ctx) const {
    if (model_label_.empty() || options_.chrome != tuinator::TextInputChrome::Framed || bounds_.width <= 4) {
        return;
    }

    tuinator::Style border_style;
    border_style.background_rgb = ui_input_chrome_fill();
    border_style.foreground_rgb = ui_border_chrome();

    int label_start = 2;
    int erase_reserve = 0;
    if (ui_corners_framed()) {
        const tuinator::BorderGlyphs glyphs = ui_frame_border_glyphs();
        const std::string prefix = glyphs.horizontal + " ";
        const int prefix_width = tuinator::text_display_width(prefix);
        const std::string suffix = " " + glyphs.horizontal;
        const int suffix_width = tuinator::text_display_width(suffix);
        label_start = 1 + prefix_width;
        const int inner_width = bounds_.width - 2;
        erase_reserve = std::max(0, inner_width - prefix_width - suffix_width);
    }

    tuinator::Style model_style = border_style;
    model_style.bold = true;
    model_style.foreground_rgb = ui_text();
    tuinator::Style icon_style = model_style;
    icon_style.foreground_rgb = ui_accent();
    tuinator::Style provider_style = border_style;
    provider_style.bold = false;
    provider_style.foreground_rgb = ui_secondary_text();

    const std::string icon_text = std::string(kNfModel) + " ";
    const std::string model_text = model_label_;
    const std::string spacer = provider_label_.empty() ? "" : "  ";
    const std::string provider_text = provider_label_;

    const int content_w = tuinator::text_display_width(icon_text + model_text + spacer + provider_text);
    const int erase_w = std::min(erase_reserve, content_w + 2);
    if (erase_w > 0) {
        tuinator::Style erase = border_style;
        ctx.canvas.fill_rect({label_start, 0, erase_w, 1}, ' ', erase);
    }

    int x = label_start;
    ctx.canvas.draw_text({x, 0}, icon_text, icon_style);
    x += tuinator::text_display_width(icon_text);
    ctx.canvas.draw_text({x, 0}, model_text, model_style);
    x += tuinator::text_display_width(model_text);
    if (!provider_text.empty()) {
        ctx.canvas.draw_text({x, 0}, spacer, provider_style);
        x += tuinator::text_display_width(spacer);
        ctx.canvas.draw_text({x, 0}, provider_text, provider_style);
    }
}

namespace {

tuinator::Style composer_caret_style(const tuinator::Style& field) {
    tuinator::Style style = field;
    style.reverse = false;
    style.bold = false;
    style.dim = false;
    style.foreground_rgb = ui_caret_fg();
    style.background_rgb = ui_caret_bg();
    if (!style.background_rgb.has_value()) {
        style.background_rgb = ui_input_chrome_fill();
    }
    return style;
}

tuinator::Style composer_field_text_style(const tuinator::PaintContext& ctx, const ChatTextInput& input, bool focused) {
    tuinator::Style base;
    base.background_rgb = ui_input_chrome_fill();
    base.foreground_rgb = ui_text();
    return focused ? ctx.styles().focused(input, base) : ctx.styles().text(input, base);
}

} // namespace

void ChatTextInput::paint_caret(tuinator::PaintContext& ctx) const {
    if (!is_focused()) {
        return;
    }

    const std::optional<tuinator::Point> cell = caret_local_cell();
    if (!cell.has_value()) {
        return;
    }

    const tuinator::Point at{cell->x, cell->y};
    const bool focused = true;
    const tuinator::Style field_style = composer_field_text_style(ctx, *this, focused);

    if (!caret_visible_) {
        if (value().empty()) {
            tuinator::Style fill = field_style;
            fill.background_rgb = ui_input_chrome_fill();
            ctx.canvas.fill_rect({at.x, at.y, 1, 1}, ' ', fill);
            return;
        }

        const std::size_t index = cursor_position();
        if (index < value().size()) {
            const int char_len = static_cast<int>(tuinator::utf8_char_length(value(), index));
            tuinator::Style glyph_style = field_style;
            glyph_style.background_rgb = ui_input_chrome_fill();
            ctx.canvas.draw_text({at.x, at.y}, value().substr(index, static_cast<std::size_t>(char_len)), glyph_style);
        } else {
            tuinator::Style fill = field_style;
            fill.background_rgb = ui_input_chrome_fill();
            ctx.canvas.fill_rect({at.x, at.y, 1, 1}, ' ', fill);
        }
        return;
    }

    if (value().empty()) {
        ctx.canvas.draw_char({at.x, at.y}, ' ', composer_caret_style(field_style));
    }
}

void ChatTextInput::paint_chrome_animation(tuinator::PaintContext& ctx) const {
    if (!chrome_busy_ || !ui_corners_framed() || options_.chrome != tuinator::TextInputChrome::Framed ||
        bounds_.width <= 4 || bounds_.height <= 3) {
        return;
    }
    const tuinator::Rect local{0, 0, bounds_.width, bounds_.height};
    ui_paint_rect_border_wave(ctx, local, chrome_anim_ms_, true, ui_border_chrome(), ui_border_chrome_alt(),
                              ui_input_chrome_fill());
}

void ChatTextInput::paint(tuinator::PaintContext& ctx) const {
    const bool focused = is_focused();
    tuinator::TextInput::paint(ctx);
    paint_border_label(ctx);
    paint_chrome_animation(ctx);

    // Focused+empty: ghost placeholder to the right of the block caret (do not erase the caret cell).
    if (focused && value().empty() && !placeholder().empty() &&
        options_.chrome == tuinator::TextInputChrome::Framed) {
        const int row = 1;
        const int col_start = 2 + tuinator::text_display_width(prompt());
        const int inner_w =
            std::max(0, bounds_.width - 4 - static_cast<int>(tuinator::text_display_width(prompt())));

        tuinator::Style placeholder_base;
        placeholder_base.background_rgb = ui_input_chrome_fill();
        placeholder_base.foreground_rgb = ui_secondary_text();
        placeholder_base.dim = true;
        const tuinator::Style placeholder_style = ctx.styles().text(*this, placeholder_base);

        const int placeholder_col = col_start + 1;
        const int placeholder_w = std::max(0, inner_w - 1);
        const std::size_t bytes = tuinator::text_byte_length_for_width(placeholder(), placeholder_w);
        if (bytes > 0) {
            ctx.canvas.draw_text({placeholder_col, row}, placeholder().substr(0, bytes), placeholder_style);
        }
    }

    paint_caret(ctx);
}
