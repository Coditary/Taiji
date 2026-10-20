#include "thinking_indicator.hpp"

#include "ui_theme.hpp"

ThinkingIndicator::ThinkingIndicator(tuinator::Style throbber_style, tuinator::Style label_style)
    : tuinator::HBox(tuinator::BoxOptions{.gap = 1, .padding = 0}) {
    auto throbber = std::make_unique<tuinator::Throbber>("dots", throbber_style);
    throbber_ = throbber.get();
    add_child(std::move(throbber));
    auto label = std::make_unique<tuinator::Label>("Thinking", label_style);
    label_ = label.get();
    add_child(std::move(label));
}

void ThinkingIndicator::set_label(const std::string& text) {
    if (label_ == nullptr || label_->text() == text) {
        return;
    }
    label_->set_text(text);
    mark_dirty();
    mark_layout_dirty();
}

void ThinkingIndicator::set_active(bool active) {
    if (active_ == active) {
        return;
    }
    active_ = active;
    mark_layout_dirty();
    mark_dirty();
}

void ThinkingIndicator::tick(int dt_ms) {
    if (!active_ || throbber_ == nullptr) {
        return;
    }
    throbber_->tick(dt_ms);
    mark_dirty();
}

tuinator::Size ThinkingIndicator::preferred_size() const {
    if (!active_) {
        return {0, 0};
    }
    return tuinator::HBox::preferred_size();
}

void ThinkingIndicator::paint(tuinator::PaintContext& ctx) const {
    if (!active_) {
        return;
    }
    ui_fill_widget_bounds(ctx, *this, ui_main_bg());
    tuinator::HBox::paint(ctx);
}
