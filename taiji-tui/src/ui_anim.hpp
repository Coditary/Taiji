#pragma once

#include <tuinator/render/paint_context.hpp>
#include <tuinator/render/style.hpp>

#include <cstdint>
#include <string>
#include <vector>

constexpr int kUiAnimFadeMs = 300;
constexpr int kUiAnimFlashMs = 120;

tuinator::Rgb ui_lerp_rgb(tuinator::Rgb a, tuinator::Rgb b, double t);
double ui_sine_pulse(double phase_ms, double period_ms = 1600.0);
tuinator::Rgb ui_pulse_rgb(tuinator::Rgb base, tuinator::Rgb accent, double phase_ms, double period_ms = 900.0);

/// 0 = fully accent, 1 = fully base.
tuinator::Rgb ui_fade_toward(tuinator::Rgb accent, tuinator::Rgb base, double t);

std::string ui_braille_spinner_frame(int phase_ms, int variant = 0);
std::string ui_sparkline_glyph(const std::vector<double>& samples, double latest);

/// Eight-step horizontal context bar (partial blocks).
std::string ui_smooth_bar_segment(double fill_ratio);

void ui_paint_rect_border_wave(tuinator::PaintContext& ctx, tuinator::Rect rect, int anim_ms, bool active,
                               tuinator::Rgb base, tuinator::Rgb highlight, tuinator::Rgb fill_bg);
