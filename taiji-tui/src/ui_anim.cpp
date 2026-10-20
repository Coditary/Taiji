#include "ui_anim.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/glyphs.hpp>
#include <tuinator/render/paint_context.hpp>
#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cmath>
#include <vector>

tuinator::Rgb ui_lerp_rgb(tuinator::Rgb a, tuinator::Rgb b, double t) {
    t = std::clamp(t, 0.0, 1.0);
    return tuinator::rgb(static_cast<int>(a.r + (b.r - a.r) * t), static_cast<int>(a.g + (b.g - a.g) * t),
                         static_cast<int>(a.b + (b.b - a.b) * t));
}

double ui_sine_pulse(double phase_ms, double period_ms) {
    if (period_ms <= 0.0) {
        return 0.0;
    }
    const double angle = (phase_ms / period_ms) * 2.0 * M_PI;
    return 0.5 + 0.5 * std::sin(angle);
}

tuinator::Rgb ui_pulse_rgb(tuinator::Rgb base, tuinator::Rgb accent, double phase_ms, double period_ms) {
    return ui_lerp_rgb(base, accent, ui_sine_pulse(phase_ms, period_ms));
}

tuinator::Rgb ui_fade_toward(tuinator::Rgb accent, tuinator::Rgb base, double t) {
    return ui_lerp_rgb(accent, base, std::clamp(t, 0.0, 1.0));
}

std::string ui_braille_spinner_frame(int phase_ms, int variant) {
    static const char* kWave[] = {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"};
    static const char* kHelix[] = {"⠋", "⠧", "⠹", "⠼", "⠙", "⠇", "⠸", "⠴"};
    const int period = 80;
    const int idx = (phase_ms / period) % 10;
    if (variant % 2 == 0) {
        return kWave[idx];
    }
    const int h = (phase_ms / period) % 8;
    return kHelix[h];
}

std::string ui_sparkline_glyph(const std::vector<double>& samples, double latest) {
    static const char* kBlocks[] = {"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"};
    std::vector<double> values = samples;
    values.push_back(latest);
    if (values.size() > 8) {
        values.erase(values.begin(), values.end() - 8);
    }
    double max_v = 0.01;
    for (double v : values) {
        max_v = std::max(max_v, v);
    }
    std::string out;
    for (double v : values) {
        const double norm = std::clamp(v / max_v, 0.0, 1.0);
        const int idx = std::clamp(static_cast<int>(norm * 7.0), 0, 7);
        out += kBlocks[idx];
    }
    return out;
}

std::string ui_smooth_bar_segment(double fill_ratio) {
    static const char* kParts[] = {" ", "▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"};
    fill_ratio = std::clamp(fill_ratio, 0.0, 1.0);
    const int scaled = static_cast<int>(fill_ratio * 8.0);
    return kParts[scaled];
}

namespace {

int rect_perimeter(const tuinator::Rect& rect) {
    if (rect.width < 2 || rect.height < 2) {
        return 0;
    }
    return 2 * (rect.width + rect.height) - 4;
}

void paint_border_glyph(tuinator::PaintContext& ctx, int x, int y, const std::string& glyph, tuinator::Rgb fg,
                        tuinator::Rgb bg) {
    if (glyph.empty()) {
        return;
    }
    tuinator::Style style;
    style.foreground_rgb = fg;
    style.background_rgb = bg;
    ctx.canvas.draw_text({x, y}, glyph, style);
}

} // namespace

void ui_paint_rect_border_wave(tuinator::PaintContext& ctx, tuinator::Rect rect, int anim_ms, bool active,
                               tuinator::Rgb base, tuinator::Rgb highlight, tuinator::Rgb fill_bg) {
    if (rect.width < 3 || rect.height < 3) {
        return;
    }
    const tuinator::BorderGlyphs g = ui_frame_border_glyphs();
    const int perimeter = rect_perimeter(rect);
    if (perimeter <= 0) {
        return;
    }

    const int head = active ? (perimeter - 1 - (anim_ms / 42) % perimeter) : -1;
    const double breathe = active ? 1.0 : ui_sine_pulse(static_cast<double>(anim_ms), 2200.0);
    const tuinator::Rgb idle_fg = ui_lerp_rgb(base, highlight, breathe * 0.35);

    int step = 0;
    const auto color_at = [&](int index) -> tuinator::Rgb {
        if (!active || head < 0) {
            return idle_fg;
        }
        const int dist = (index - head + perimeter) % perimeter;
        if (dist <= 2) {
            const double t = 1.0 - static_cast<double>(dist) / 3.0;
            return ui_lerp_rgb(base, highlight, t);
        }
        return base;
    };

    for (int x = 0; x < rect.width; ++x) {
        const std::string& glyph = x == 0 ? g.top_left : (x == rect.width - 1 ? g.top_right : g.horizontal);
        paint_border_glyph(ctx, rect.x + x, rect.y, glyph, color_at(step++), fill_bg);
    }
    for (int y = 1; y < rect.height - 1; ++y) {
        paint_border_glyph(ctx, rect.x + rect.width - 1, rect.y + y, g.vertical, color_at(step++), fill_bg);
    }
    if (rect.height > 1) {
        for (int x = rect.width - 1; x >= 0; --x) {
            const std::string& glyph =
                x == rect.width - 1 ? g.bottom_right : (x == 0 ? g.bottom_left : g.horizontal);
            paint_border_glyph(ctx, rect.x + x, rect.y + rect.height - 1, glyph, color_at(step++), fill_bg);
        }
    }
    for (int y = rect.height - 2; y >= 1; --y) {
        paint_border_glyph(ctx, rect.x, rect.y + y, g.vertical, color_at(step++), fill_bg);
    }
}
