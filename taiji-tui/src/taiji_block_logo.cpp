#include "taiji_block_logo.hpp"

#include "ui_anim.hpp"
#include "ui_theme.hpp"

namespace {

constexpr int kGlyphW = 5;
constexpr int kGlyphH = 7;
constexpr int kGlyphGap = 1;
constexpr int kLetterCount = 5;

constexpr uint8_t kLetterT[kGlyphH] = {0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04};
constexpr uint8_t kLetterA[kGlyphH] = {0x0E, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11};
constexpr uint8_t kLetterI[kGlyphH] = {0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x1F};
constexpr uint8_t kLetterJ[kGlyphH] = {0x01, 0x01, 0x01, 0x01, 0x11, 0x11, 0x0E};

const uint8_t* kLetters[kLetterCount] = {kLetterT, kLetterA, kLetterI, kLetterJ, kLetterI};

struct LogoTier {
    int pixel_w;
    int pixel_h;
};

LogoTier pick_logo_tier(int max_cols, int max_rows) {
    const int base_w = kLetterCount * kGlyphW + (kLetterCount - 1) * kGlyphGap;
    if (max_cols >= base_w * 2 + 4 && max_rows >= kGlyphH * 2 + 2) {
        return {2, 2};
    }
    if (max_cols >= base_w * 2 + 2 && max_rows >= kGlyphH + 2) {
        return {2, 1};
    }
    if (max_cols >= base_w + 2 && max_rows >= kGlyphH + 2) {
        return {1, 1};
    }
    return {0, 0};
}

bool glyph_pixel(const uint8_t* glyph, int row, int col) {
    if (row < 0 || row >= kGlyphH || col < 0 || col >= kGlyphW) {
        return false;
    }
    const int bit = kGlyphW - 1 - col;
    return ((glyph[row] >> bit) & 1) != 0;
}

void paint_cell(tuinator::PaintContext& ctx, int x, int y, int w, int h, const tuinator::Style& fill,
                const tuinator::Style& shadow) {
    if (w <= 0 || h <= 0) {
        return;
    }
    const tuinator::Rect shadow_rect{x + 1, y + 1, w, h};
    const tuinator::Rect fill_rect{x, y, w, h};
    ctx.canvas.fill_rect(shadow_rect, ' ', shadow);
    ctx.canvas.fill_rect(fill_rect, ' ', fill);
}

} // namespace

TaijiBlockLogoSize paint_taiji_block_logo(tuinator::PaintContext& ctx, const tuinator::Widget& widget, int bounds_width,
                                          int y, int max_cols, int max_rows, const tuinator::Theme& theme,
                                          int offset_x, int anim_ms) {
    TaijiBlockLogoSize out;
    const LogoTier tier = pick_logo_tier(max_cols, max_rows);
    if (tier.pixel_w == 0) {
        const char* fallback = "TAIJI";
        out.width = tuinator::text_display_width(fallback);
        out.height = 1;
        const int x = offset_x + std::max(0, (bounds_width - out.width) / 2);
        const double breathe = ui_sine_pulse(static_cast<double>(anim_ms), 2400.0);
        const int wave = anim_ms > 0 ? (anim_ms / 38) % static_cast<int>(out.width + 6) : -1;
        int cursor = x;
        for (std::size_t i = 0; fallback[i] != '\0'; ++i) {
            const char ch[2] = {fallback[i], '\0'};
            tuinator::Rgb base = ui_lerp_rgb(ui_logo_mid(), ui_logo_top(), breathe);
            tuinator::Rgb fg = base;
            if (static_cast<int>(i) >= wave - 1 && static_cast<int>(i) <= wave + 1) {
                fg = ui_lerp_rgb(base, tuinator::rgb(255, 255, 255), 0.85);
            }
            tuinator::Style letter = ui_resolve_main_text(theme.accent);
            letter.bold = true;
            letter.foreground_rgb = fg;
            letter.background_rgb = ui_main_bg();
            ctx.canvas.draw_text({cursor, y}, ch, letter);
            cursor += tuinator::text_display_width(ch);
        }
        return out;
    }

    const int base_w = kLetterCount * kGlyphW + (kLetterCount - 1) * kGlyphGap;
    out.width = base_w * tier.pixel_w;
    out.height = kGlyphH * tier.pixel_h;
    const int x = offset_x + std::max(0, (bounds_width - out.width) / 2);

    auto logo_fill_for_row = [](int row) -> tuinator::Rgb {
        if (row <= 1) {
            return ui_logo_top();
        }
        if (row >= kGlyphH - 2) {
            return ui_logo_bottom();
        }
        return ui_logo_mid();
    };

    const double breathe = ui_sine_pulse(static_cast<double>(anim_ms), 2600.0);
    const int shimmer_span = out.width + out.height + 16;
    const int shimmer_pos = anim_ms > 0 ? (anim_ms / 32) % shimmer_span : -1;

    tuinator::Style shadow = ui_resolve_main_text(theme.accent);
    shadow.foreground_rgb = ui_logo_shadow();
    shadow.background_rgb = ui_logo_shadow();

    int cursor_x = x;
    for (int letter = 0; letter < kLetterCount; ++letter) {
        const uint8_t* glyph = kLetters[letter];
        for (int row = 0; row < kGlyphH; ++row) {
            tuinator::Rgb row_rgb = logo_fill_for_row(row);
            row_rgb = ui_lerp_rgb(row_rgb, ui_logo_top(), breathe * 0.12);
            for (int col = 0; col < kGlyphW; ++col) {
                if (!glyph_pixel(glyph, row, col)) {
                    continue;
                }
                const int px = cursor_x + col * tier.pixel_w;
                const int py = y + row * tier.pixel_h;
                tuinator::Style fill = shadow;
                fill.foreground_rgb = row_rgb;
                fill.background_rgb = row_rgb;
                const int cell_diag = (px - x) + (py - y);
                if (shimmer_pos >= 0) {
                    const int dist = cell_diag - shimmer_pos;
                    if (dist >= -4 && dist <= 0) {
                        const double t = 1.0 - static_cast<double>(-dist) / 4.0;
                        fill.foreground_rgb = ui_lerp_rgb(row_rgb, tuinator::rgb(255, 255, 255), 0.35 + 0.55 * t);
                        fill.background_rgb = fill.foreground_rgb;
                    }
                }
                paint_cell(ctx, px, py, tier.pixel_w, tier.pixel_h, fill, shadow);
            }
        }
        cursor_x += (kGlyphW + kGlyphGap) * tier.pixel_w;
    }

    return out;
}
