#pragma once

#include <tuinator/render/paint_context.hpp>
#include <tuinator/tuinator.hpp>

struct TaijiBlockLogoSize {
    int width = 0;
    int height = 0;
};

/// Claude Code–style block lettering; picks a fixed scale tier from available columns (no stretching).
TaijiBlockLogoSize paint_taiji_block_logo(tuinator::PaintContext& ctx, const tuinator::Widget& widget, int bounds_width,
                                          int y, int max_cols, int max_rows, const tuinator::Theme& theme,
                                          int offset_x = 0, int anim_ms = 0);
