#include "chat_input.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>

ChatTextInput::ChatTextInput(tuinator::TextInputOptions options, tuinator::Style style, tuinator::Style focused_style)
    : tuinator::TextInput(options, style, focused_style) {}

void ChatTextInput::set_model_info(std::string model, std::string provider) {
    if (model_label_ == model && provider_label_ == provider) {
        return;
    }
    model_label_ = std::move(model);
    provider_label_ = std::move(provider);
    mark_dirty();
}

void ChatTextInput::paint(tuinator::PaintContext& ctx) const {
    tuinator::TextInput::paint(ctx);

    if (model_label_.empty() && provider_label_.empty()) {
        return;
    }

    const tuinator::Rect rect = bounds();
    if (rect.width <= 4 || rect.height <= 0) {
        return;
    }

    tuinator::Style bg_style;
    bg_style.background_rgb = tuinator::rgb(42, 38, 58);

    int label_width = tuinator::text_display_width(model_label_);
    if (!provider_label_.empty()) {
        label_width += 1 + tuinator::text_display_width(provider_label_);
    }
    label_width = std::min(label_width, rect.width - 4);

    const int x = 2;
    const int y = 0;

    tuinator::Style pad_style = bg_style;
    ctx.canvas.draw_text({x, y}, std::string(label_width, ' '), pad_style);

    tuinator::Style model_style;
    model_style.foreground_rgb = tuinator::rgb(255, 255, 255);
    model_style.background_rgb = bg_style.background_rgb;
    ctx.canvas.draw_text({x, y}, model_label_, model_style);

    if (!provider_label_.empty()) {
        const int provider_x = x + tuinator::text_display_width(model_label_) + 1;
        tuinator::Style provider_style;
        provider_style.foreground_rgb = tuinator::rgb(120, 115, 140);
        provider_style.dim = true;
        provider_style.background_rgb = bg_style.background_rgb;
        ctx.canvas.draw_text({provider_x, y}, provider_label_, provider_style);
    }
}
