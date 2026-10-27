#include "context_panel.hpp"

#include "ui_test_ids.hpp"
#include "ui_anim.hpp"
#include "ui_layout.hpp"
#include "ui_locale.hpp"
#include "ui_navigation.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <cmath>
#include <cstdio>

ContextPanel::ContextPanel(tuinator::Theme theme) : theme_(std::move(theme)) {
    set_test_id(taiji::ui::test_id::kContextPanel);
}

void ContextPanel::set_stats(int tokens, int context_window, double used_pct, double cost_usd) {
    if (tokens_ == tokens && context_window_ == context_window && used_pct_ == used_pct && cost_usd_ == cost_usd) {
        return;
    }
    tokens_ = tokens;
    context_window_ = context_window;
    used_pct_ = used_pct;
    cost_usd_ = cost_usd;
    mark_dirty();
}

void ContextPanel::set_model_label(std::string model) {
    if (model_label_ == model) {
        return;
    }
    model_label_ = std::move(model);
    mark_dirty();
}

void ContextPanel::tick(int dt_ms) {
    const double target = used_pct_;
    const double delta = target - display_used_pct_;
    if (std::abs(delta) > 0.05) {
        display_used_pct_ += delta * 0.18;
        mark_dirty();
    } else if (display_used_pct_ != target) {
        display_used_pct_ = target;
        mark_dirty();
    }

    sample_accum_ms_ += dt_ms;
    if (use_speed_metric_ && sample_accum_ms_ >= 180) {
        sample_accum_ms_ = 0;
        speed_samples_.push_back(speed_tok_per_sec_);
        if (speed_samples_.size() > 8) {
            speed_samples_.erase(speed_samples_.begin());
        }
        mark_dirty();
    }
}

void ContextPanel::set_use_speed_metric(bool use_speed) {
    if (use_speed_metric_ == use_speed) {
        return;
    }
    use_speed_metric_ = use_speed;
    mark_dirty();
}

void ContextPanel::set_speed_tok_per_sec(double tps) {
    if (speed_tok_per_sec_ == tps) {
        return;
    }
    speed_tok_per_sec_ = tps;
    mark_dirty();
}

void ContextPanel::set_show_cost(bool show) {
    if (show_cost_ == show) {
        return;
    }
    show_cost_ = show;
    mark_dirty();
}

tuinator::Size ContextPanel::preferred_size() const {
    return {18, 2};
}

void ContextPanel::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void ContextPanel::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    paint_bounds_background(ctx, ui_sidebar_panel_style());
    ui_paint_vim_focus_accent(ctx, UiFocusPane::Context, bounds_.height);

    const int pad = kUiPadX;

    tuinator::Style heading;
    heading.bold = true;
    heading.foreground_rgb = ui_section_heading();
    const tuinator::Style muted = ui_resolve_sidebar_text(ctx, *this, ui_secondary_style(theme_));

    int y = 0;
    const std::string context_title = ui_shell("context.title");
    const std::string title = context_title.empty() ? "CONTEXT" : context_title;
    ctx.canvas.draw_text({pad, y}, title, ui_resolve_sidebar_text(ctx, *this, heading));

    std::string title_suffix;
    char pct_buf[16];
    std::snprintf(pct_buf, sizeof(pct_buf), "%3.0f%%", display_used_pct_);
    title_suffix = pct_buf;
    if (context_window_ > 0) {
        title_suffix += "  " + format_token_count_short(tokens_) + " / " + format_token_count_short(context_window_);
    }
    const int suffix_x = std::max(pad, bounds_.width - pad - tuinator::text_display_width(title_suffix));
    ctx.canvas.draw_text({suffix_x, y}, title_suffix, muted);
    ++y;

    if (y < bounds_.height) {
        std::string line = format_token_compact(tokens_);
        line += "  ·  ";
        if (use_speed_metric_) {
            char value_buf[24];
            std::snprintf(value_buf, sizeof(value_buf), "%.1f", speed_tok_per_sec_);
            const std::string pattern = ui_shell("metrics.tok_per_sec");
            line += ui_format(pattern.empty() ? "{value} tok/s" : pattern, "value", value_buf);
            const std::string spark = ui_sparkline_glyph(speed_samples_, speed_tok_per_sec_);
            if (!spark.empty()) {
                line += " ";
                line += spark;
            }
        } else if (show_cost_) {
            line += format_cost_compact(cost_usd_);
        }
        if (!model_label_.empty()) {
            line += "  ·  ";
            line += model_label_;
        }
        ctx.canvas.draw_text({pad, y}, line, muted);
    }
}

bool ContextPanel::handle_event(const tuinator::Event& event) {
    const auto* mouse = std::get_if<tuinator::MouseEvent>(&event);
    if (mouse == nullptr) {
        return false;
    }
    return bounds_.contains(mouse->position);
}

void ContextPanel::collect_focusable(std::vector<tuinator::Widget*>& out) {
    out.push_back(this);
}
