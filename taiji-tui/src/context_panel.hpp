#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <string>
#include <vector>

class ContextPanel : public tuinator::Widget {
  public:
    struct UsageBreakdown {
        int system_tokens = 0;
        int tools_tokens = 0;
        int skills_tokens = 0;
        int conversation_tokens = 0;
    };

    explicit ContextPanel(tuinator::Theme theme);

    void set_stats(int tokens, int context_window, double used_pct, double cost_usd);
    void set_usage_breakdown(UsageBreakdown breakdown);
    void set_skills_summary(std::string summary);
    void set_model_label(std::string model);
    void set_show_cost(bool show);
    void set_use_speed_metric(bool use_speed);
    void set_speed_tok_per_sec(double tps);
    void tick(int dt_ms);

    bool usage_expanded() const { return usage_expanded_; }
    void set_usage_expanded(bool expanded);
    void toggle_usage_expanded();
    void set_on_layout_changed(std::function<void()> callback);

    std::string_view widget_type_name() const override { return "ContextPanel"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    void collect_focusable(std::vector<tuinator::Widget*>& out) override;
    bool wants_hover() const override { return true; }

  private:
    int line_count() const;
    void paint_segmented_bar(tuinator::PaintContext& ctx, int x, int y, int width, tuinator::Style track) const;
    void paint_category_row(tuinator::PaintContext& ctx, int y, tuinator::Rgb dot, const std::string& label, int tokens,
                            const tuinator::Style& text_style) const;

    tuinator::Theme theme_;
    int tokens_ = 0;
    int context_window_ = 0;
    double used_pct_ = 0.0;
    double cost_usd_ = 0.0;
    UsageBreakdown usage_;
    std::string skills_summary_;
    std::string model_label_;
    bool show_cost_ = true;
    bool use_speed_metric_ = false;
    double speed_tok_per_sec_ = 0.0;
    double display_used_pct_ = 0.0;
    int sample_accum_ms_ = 0;
    std::vector<double> speed_samples_;
    bool usage_expanded_ = false;
    bool title_hover_ = false;
    std::function<void()> on_layout_changed_;
};
