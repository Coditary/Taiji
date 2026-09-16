#pragma once

#include <tuinator/tuinator.hpp>

#include <cstddef>
#include <string>
#include <utility>
#include <vector>

struct WrappedLayoutLine {
    std::string text;
    std::size_t source_start = 0;
    std::size_t source_end = 0;
};

std::vector<WrappedLayoutLine> build_wrapped_layout(const std::string& text, int max_cols);

/// Mouse/keyboard text selection over wrapped plain-text lines mapped to source bytes.
class WrappedTextSelection {
  public:
    void reset(std::size_t text_length);
    void clear_selection(std::size_t text_length);
    void select_all(std::size_t text_length);

    bool has_selection() const;
    bool pointer_active() const { return selecting_with_mouse_; }
    std::pair<std::size_t, std::size_t> selection_range() const;

    bool handle_mouse(const tuinator::MouseEvent& mouse, const tuinator::Rect& bounds, int text_x, int line_y,
                      const std::vector<WrappedLayoutLine>& lines, std::size_t text_length);
    bool handle_shortcut(const tuinator::KeyPress& key, const std::string& text);
    bool handle_escape();

    void paint_highlights(tuinator::PaintContext& ctx, const std::vector<WrappedLayoutLine>& lines, int text_x,
                          int line_y, int max_height, const tuinator::Style& style) const;

  private:
    std::size_t byte_index_at(int local_x, int local_y, int text_x, int line_y,
                              const std::vector<WrappedLayoutLine>& lines, std::size_t text_length,
                              tuinator::CaretAffinity affinity) const;
    void set_cursor_at(std::size_t pos, bool extend_selection, std::size_t text_length);
    void copy_selection(const std::string& text) const;

    std::size_t selection_anchor_ = 0;
    std::size_t cursor_ = 0;
    bool selecting_with_mouse_ = false;
    std::size_t mouse_press_index_ = 0;
    int mouse_press_column_ = 0;
};
