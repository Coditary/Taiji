#include "speech_mic_button.hpp"

#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

SpeechMicButton::SpeechMicButton(tuinator::Style normal_style, tuinator::Style recording_style,
                                 tuinator::Style hover_style, std::function<void()> on_click)
    : normal_style_(std::move(normal_style)), recording_style_(std::move(recording_style)),
      hover_style_(std::move(hover_style)), on_click_(std::move(on_click)) {}

void SpeechMicButton::set_recording(bool recording) {
    if (recording_ == recording) {
        return;
    }
    recording_ = recording;
    mark_dirty();
    mark_layout_dirty();
}

void SpeechMicButton::set_enabled(bool enabled) {
    if (enabled_ == enabled) {
        return;
    }
    enabled_ = enabled;
    mark_dirty();
    mark_layout_dirty();
}

std::string SpeechMicButton::label_text() const {
    return std::string(" ") + kNfMicrophone + " ";
}

tuinator::Size SpeechMicButton::preferred_size() const {
    const int w = tuinator::text_display_width(label_text());
    return {w, 1};
}

void SpeechMicButton::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void SpeechMicButton::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    tuinator::Style base = normal_style_;
    base.background_rgb = ui_input_chrome_fill();
    if (!enabled_) {
        base.foreground_rgb = ui_secondary_text();
    } else if (recording_) {
        base = recording_style_;
        base.background_rgb = ui_input_chrome_fill();
        base.foreground_rgb = ui_danger();
    } else if (hover_) {
        base = hover_style_;
        base.background_rgb = ui_input_chrome_fill();
        base.foreground_rgb = ui_accent();
    } else {
        base.foreground_rgb = ui_accent();
    }

    const std::string label = label_text();
    ctx.canvas.fill_rect({0, 0, bounds_.width, 1}, ' ', base);
    const tuinator::Style style = ctx.styles().text(*this, base);
    ctx.canvas.draw_text({0, 0}, label, style);
}

bool SpeechMicButton::handle_event(const tuinator::Event& event) {
    if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
        if (mouse->action == tuinator::MouseAction::Move) {
            const bool inside = bounds_.contains(mouse->position);
            if (enabled_ && inside != hover_) {
                hover_ = inside;
                mark_dirty();
            }
            return inside;
        }
        if (mouse->button != tuinator::MouseButton::Left) {
            return bounds_.contains(mouse->position);
        }
        if ((mouse->action == tuinator::MouseAction::Press || mouse->action == tuinator::MouseAction::Click ||
             mouse->action == tuinator::MouseAction::Release) &&
            bounds_.contains(mouse->position)) {
            if (mouse->action == tuinator::MouseAction::Release && on_click_) {
                on_click_();
            }
            return true;
        }
    }
    return false;
}

tuinator::Widget* SpeechMicButton::hit_test(tuinator::Point point) {
    if (!bounds_.contains(point)) {
        return nullptr;
    }
    return this;
}
