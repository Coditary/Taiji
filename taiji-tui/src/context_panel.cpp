#include "context_panel.hpp"

#include "ui_anim.hpp"
#include "ui_layout.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cmath>
#include <cstdio>
#include <vector>

namespace {

constexpr tuinator::Rgb kUsageSystem{140, 145, 160};
constexpr tuinator::Rgb kUsageTools{187, 154, 247};
constexpr tuinator::Rgb kUsageSkills{235, 200, 120};
constexpr tuinator::Rgb kUsageChat{240, 140, 120};
constexpr tuinator::Rgb kUsageFree{58, 52, 74};

struct UsageSegment {
    int tokens = 0;
    tuinator::Rgb color;
};

std::vector<UsageSegment> usage_segments(const ContextPanel::UsageBreakdown& usage, int context_window, int total_tokens) {
    const int window = std::max({context_window, total_tokens, 1});
    int free_tokens = std::max(0, window - total_tokens);
    std::vector<UsageSegment> segments = {
        {usage.system_tokens, kUsageSystem},
        {usage.tools_tokens, kUsageTools},
        {usage.skills_tokens, kUsageSkills},
        {usage.conversation_tokens, kUsageChat},
        {free_tokens, kUsageFree},
    };
    return segments;
}

} // namespace

ContextPanel::ContextPanel(tuinator::Theme theme) : theme_(std::move(theme)) {}

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

void ContextPanel::set_usage_breakdown(UsageBreakdown breakdown) {
    if (usage_.system_tokens == breakdown.system_tokens && usage_.tools_tokens == breakdown.tools_tokens &&
        usage_.skills_tokens == breakdown.skills_tokens &&
        usage_.conversation_tokens == breakdown.conversation_tokens) {
        return;
    }
    usage_ = breakdown;
    mark_dirty();
}

void ContextPanel::set_skills_summary(std::string summary) {
    if (skills_summary_ == summary) {
        return;
    }
    skills_summary_ = std::move(summary);
    mark_dirty();
}

void ContextPanel::set_model_label(std::string model) {
    if (model_label_ == model) {
        return;
    }
    model_label_ = std::move(model);
    mark_dirty();
}

void ContextPanel::set_usage_expanded(bool expanded) {
    if (usage_expanded_ == expanded) {
        return;
    }
    usage_expanded_ = expanded;
    mark_dirty();
    mark_layout_dirty();
    if (on_layout_changed_) {
        on_layout_changed_();
    }
}

void ContextPanel::toggle_usage_expanded() {
    set_usage_expanded(!usage_expanded_);
}

void ContextPanel::set_on_layout_changed(std::function<void()> callback) {
    on_layout_changed_ = std::move(callback);
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

int ContextPanel::line_count() const {
    if (!usage_expanded_) {
        return 3;
    }
    return 9;
}

tuinator::Size ContextPanel::preferred_size() const {
    return {18, line_count()};
}

void ContextPanel::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void ContextPanel::paint_segmented_bar(tuinator::PaintContext& ctx, int x, int y, int width,
                                      tuinator::Style track) const {
    if (width <= 0) {
        return;
    }
    const auto segments = usage_segments(usage_, context_window_, tokens_);
    const int window = std::max({context_window_, tokens_, 1});
    int painted = 0;
    for (const UsageSegment& segment : segments) {
        if (segment.tokens <= 0) {
            continue;
        }
        const double share = static_cast<double>(segment.tokens) / static_cast<double>(window);
        int cells = std::clamp(static_cast<int>(std::lround(share * width)), 0, width - painted);
        if (cells <= 0 && segment.tokens > 0 && painted < width) {
            cells = 1;
        }
        tuinator::Style fill = track;
        fill.foreground_rgb = segment.color;
        for (int i = 0; i < cells && painted < width; ++i) {
            ctx.canvas.draw_text({x + painted, y}, "━", fill);
            ++painted;
        }
    }
    for (; painted < width; ++painted) {
        ctx.canvas.draw_text({x + painted, y}, "─", track);
    }
}

void ContextPanel::paint_category_row(tuinator::PaintContext& ctx, int y, tuinator::Rgb dot, const std::string& label,
                                      int tokens, const tuinator::Style& text_style) const {
    const int pad = kUiPadX;
    tuinator::Style bullet = text_style;
    bullet.foreground_rgb = dot;
    ctx.canvas.draw_text({pad, y}, "▪ ", bullet);
    const int label_x = pad + tuinator::text_display_width("▪ ");
    const int inner_w = std::max(0, bounds_.width - pad * 2);
    const int value_w = 6;
    const int label_max = std::max(0, inner_w - value_w - (label_x - pad));
    const std::size_t label_bytes = tuinator::text_byte_length_for_width(label, label_max);
    ctx.canvas.draw_text({label_x, y}, label.substr(0, label_bytes), text_style);
    const std::string value = format_token_count_short(tokens);
    const int value_x = std::max(label_x, bounds_.width - pad - tuinator::text_display_width(value));
    tuinator::Style value_style = text_style;
    value_style.dim = true;
    ctx.canvas.draw_text({value_x, y}, value, value_style);
}

void ContextPanel::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    paint_bounds_background(ctx, ui_sidebar_panel_style());

    const int pad = kUiPadX;
    const int inner_w = std::max(0, bounds_.width - pad * 2);

    tuinator::Style heading = theme_.heading;
    heading.bold = true;
    if (title_hover_) {
        heading.foreground_rgb = theme_.accent.foreground_rgb;
    }
    const tuinator::Style muted = ui_resolve_sidebar_text(ctx, *this, ui_secondary_style(theme_));
    const tuinator::Style label = ui_resolve_sidebar_text(ctx, *this, theme_.label);
    tuinator::Style track = label;
    track.foreground_rgb = kUsageFree;

    int y = 0;
    const std::string chevron = usage_expanded_ ? "▾ " : "▸ ";
    const std::string title = chevron + "CONTEXT";
    ctx.canvas.draw_text({pad, y}, title, ui_resolve_sidebar_text(ctx, *this, heading));

    char pct_buf[16];
    std::snprintf(pct_buf, sizeof(pct_buf), "%3.0f%%", display_used_pct_);
    const int pct_x = std::max(pad, bounds_.width - pad - tuinator::text_display_width(pct_buf));
    ctx.canvas.draw_text({pct_x, y}, pct_buf, muted);
    ++y;

    if (y < bounds_.height) {
        const int bar_w = std::max(4, inner_w - 14);
        paint_segmented_bar(ctx, pad, y, bar_w, track);
        if (context_window_ > 0) {
            const std::string totals =
                format_token_count_short(tokens_) + " / " + format_token_count_short(context_window_);
            const int totals_x = std::max(pad + bar_w + 1, bounds_.width - pad - tuinator::text_display_width(totals));
            ctx.canvas.draw_text({totals_x, y}, totals, muted);
        }
        ++y;
    }

    if (!usage_expanded_) {
        if (y < bounds_.height) {
            std::string line = format_token_compact(tokens_);
            line += "  ·  ";
            if (use_speed_metric_) {
                char buf[24];
                std::snprintf(buf, sizeof(buf), "%.1f tok/s", speed_tok_per_sec_);
                line += buf;
                const std::string spark = ui_sparkline_glyph(speed_samples_, speed_tok_per_sec_);
                if (!spark.empty()) {
                    line += " ";
                    line += spark;
                }
            } else if (show_cost_) {
                line += format_cost_compact(cost_usd_);
            }
            ctx.canvas.draw_text({pad, y}, line, muted);
        }
        return;
    }

    if (y < bounds_.height) {
        std::string skills_label = "Skills";
        if (!skills_summary_.empty()) {
            skills_label += " (" + skills_summary_ + ")";
        }
        paint_category_row(ctx, y++, kUsageSystem, "System prompt", usage_.system_tokens, label);
        if (y < bounds_.height) {
            paint_category_row(ctx, y++, kUsageTools, "Tool definitions & MCP", usage_.tools_tokens, label);
        }
        if (y < bounds_.height) {
            paint_category_row(ctx, y++, kUsageSkills, skills_label, usage_.skills_tokens, label);
        }
        if (y < bounds_.height) {
            paint_category_row(ctx, y++, kUsageChat, "Conversation", usage_.conversation_tokens, label);
        }
    }

    if (y < bounds_.height) {
        tuinator::Style rule = muted;
        rule.foreground_rgb = tuinator::rgb(70, 66, 88);
        std::string rule_text;
        const int rule_w = inner_w;
        for (int i = 0; i < rule_w; ++i) {
            rule_text += "─";
        }
        ctx.canvas.draw_text({pad, y}, rule_text, rule);
        ++y;
    }

    if (y < bounds_.height) {
        std::string footer;
        if (use_speed_metric_) {
            char buf[24];
            std::snprintf(buf, sizeof(buf), "%.1f tok/s", speed_tok_per_sec_);
            footer = buf;
        } else if (show_cost_) {
            footer = format_cost_compact(cost_usd_);
        }
        if (!model_label_.empty()) {
            if (!footer.empty()) {
                footer += " · ";
            }
            footer += model_label_;
        }
        if (!footer.empty()) {
            ctx.canvas.draw_text({pad, y}, footer, muted);
        }
    }
}

bool ContextPanel::handle_event(const tuinator::Event& event) {
    if (const auto* key = std::get_if<tuinator::KeyPress>(&event)) {
        if (key->alt && !key->ctrl && (key->character == 'c' || key->character == 'C')) {
            toggle_usage_expanded();
            return true;
        }
    }

    const auto* mouse = std::get_if<tuinator::MouseEvent>(&event);
    if (mouse == nullptr) {
        return false;
    }
    const tuinator::Point local{mouse->position.x - bounds_.x, mouse->position.y - bounds_.y};
    const bool in_title = local.y == 0 && local.x >= kUiPadX && bounds_.contains(mouse->position);
    if (mouse->action == tuinator::MouseAction::Move) {
        if (in_title != title_hover_) {
            title_hover_ = in_title;
            mark_dirty();
        }
        return bounds_.contains(mouse->position);
    }
    if (mouse->button == tuinator::MouseButton::Left &&
        (mouse->action == tuinator::MouseAction::Click || mouse->action == tuinator::MouseAction::Release) &&
        in_title && mouse->action == tuinator::MouseAction::Release) {
        toggle_usage_expanded();
        return true;
    }
    return bounds_.contains(mouse->position);
}
