#include "wrapped_text_selection.hpp"

#include <tuinator/platform/clipboard.hpp>
#include <tuinator/render/text.hpp>

#include <algorithm>
#include <sstream>

namespace {

std::vector<WrappedLayoutLine> wrap_paragraph_with_map(const std::string& paragraph, std::size_t paragraph_start,
                                                       int max_cols) {
    if (max_cols <= 0) {
        if (paragraph.empty()) {
            return {};
        }
        return {{paragraph, paragraph_start, paragraph_start + paragraph.size()}};
    }

    if (paragraph.empty()) {
        return {};
    }

    std::vector<WrappedLayoutLine> lines;
    std::istringstream words(paragraph);
    std::string word;
    std::string current;
    std::size_t current_start = paragraph_start;

    while (words >> word) {
        const std::string candidate = current.empty() ? word : current + " " + word;
        if (tuinator::text_display_width(candidate) <= max_cols) {
            current = candidate;
            continue;
        }

        if (!current.empty()) {
            lines.push_back({current, current_start, current_start + current.size()});
            current_start += current.size();
            while (current_start < paragraph_start + paragraph.size() &&
                   paragraph[current_start - paragraph_start] == ' ') {
                ++current_start;
            }
            current = word;
            continue;
        }

        std::size_t offset = 0;
        while (offset < word.size()) {
            const std::size_t bytes =
                tuinator::text_byte_length_for_width(std::string_view(word).substr(offset), max_cols);
            if (bytes == 0) {
                break;
            }
            const std::string piece = word.substr(offset, bytes);
            lines.push_back({piece, paragraph_start + offset, paragraph_start + offset + bytes});
            offset += bytes;
        }
        current.clear();
        current_start = paragraph_start + paragraph.size();
    }

    if (!current.empty()) {
        lines.push_back({current, current_start, current_start + current.size()});
    }
    return lines;
}

int display_column_at(const std::string& line, int text_x, int local_x) {
    const int relative = local_x - text_x;
    if (relative <= 0) {
        return 0;
    }
    const int line_width = tuinator::text_display_width(line);
    if (relative >= line_width) {
        return line_width;
    }
    return relative;
}

} // namespace

std::vector<WrappedLayoutLine> build_wrapped_layout(const std::string& text, int max_cols) {
    std::vector<WrappedLayoutLine> lines;
    std::size_t start = 0;
    while (start <= text.size()) {
        const std::size_t end = text.find('\n', start);
        const bool has_newline = end != std::string::npos;
        const std::size_t para_end = has_newline ? end : text.size();
        const std::string paragraph = text.substr(start, para_end - start);
        const auto wrapped = wrap_paragraph_with_map(paragraph, start, max_cols);
        lines.insert(lines.end(), wrapped.begin(), wrapped.end());
        if (!has_newline) {
            break;
        }
        start = end + 1;
    }
    if (lines.empty() && !text.empty()) {
        lines.push_back({text, 0, text.size()});
    }
    return lines;
}

void WrappedTextSelection::reset(std::size_t text_length) {
    selection_anchor_ = text_length;
    cursor_ = text_length;
    selecting_with_mouse_ = false;
}

void WrappedTextSelection::clear_selection(std::size_t text_length) {
    selection_anchor_ = text_length;
    cursor_ = text_length;
    selecting_with_mouse_ = false;
}

void WrappedTextSelection::select_all(std::size_t text_length) {
    selection_anchor_ = 0;
    cursor_ = text_length;
}

bool WrappedTextSelection::has_selection() const { return selection_anchor_ != cursor_; }

std::pair<std::size_t, std::size_t> WrappedTextSelection::selection_range() const {
    const std::size_t start = std::min(selection_anchor_, cursor_);
    const std::size_t end = std::max(selection_anchor_, cursor_);
    return {start, end};
}

std::size_t WrappedTextSelection::byte_index_at(int local_x, int local_y, int text_x, int line_y,
                                              const std::vector<WrappedLayoutLine>& lines,
                                              std::size_t text_length, tuinator::CaretAffinity affinity) const {
    if (lines.empty()) {
        return 0;
    }

    const int line_index = local_y - line_y;
    if (line_index < 0) {
        return 0;
    }
    if (line_index >= static_cast<int>(lines.size())) {
        return text_length;
    }

    const WrappedLayoutLine& line = lines[static_cast<std::size_t>(line_index)];
    const int column = display_column_at(line.text, text_x, local_x);
    if (column >= tuinator::text_display_width(line.text)) {
        return line.source_end;
    }

    const std::size_t offset = tuinator::text_caret_index_at_column(line.text, column, affinity);
    return std::min(line.source_start + offset, text_length);
}

void WrappedTextSelection::set_cursor_at(std::size_t pos, bool extend_selection, std::size_t text_length) {
    const std::size_t next = std::min(pos, text_length);
    if (!extend_selection) {
        selection_anchor_ = next;
    }
    cursor_ = next;
}

bool WrappedTextSelection::handle_mouse(const tuinator::MouseEvent& mouse, const tuinator::Rect& bounds, int text_x,
                                        int line_y, const std::vector<WrappedLayoutLine>& lines,
                                        std::size_t text_length) {
    if (lines.empty()) {
        return false;
    }

    const auto local = tuinator::Point{mouse.position.x - bounds.x, mouse.position.y - bounds.y};

    if (mouse.action == tuinator::MouseAction::Release) {
        if (!selecting_with_mouse_) {
            return bounds.contains(mouse.position);
        }

        const int release_column = display_column_at(
            lines[static_cast<std::size_t>(std::clamp(local.y - line_y, 0, static_cast<int>(lines.size()) - 1))].text,
            text_x, local.x);
        if (release_column != mouse_press_column_) {
            set_cursor_at(byte_index_at(local.x, local.y, text_x, line_y, lines, text_length,
                                        tuinator::CaretAffinity::After),
                          true, text_length);
        } else {
            set_cursor_at(mouse_press_index_, false, text_length);
        }
        selecting_with_mouse_ = false;
        return true;
    }

    if (mouse.action == tuinator::MouseAction::Move) {
        if (!selecting_with_mouse_ || !mouse.left_pressed) {
            return false;
        }

        set_cursor_at(byte_index_at(local.x, local.y, text_x, line_y, lines, text_length,
                                    tuinator::CaretAffinity::After),
                      true, text_length);
        return true;
    }

    if (mouse.action == tuinator::MouseAction::Press || mouse.action == tuinator::MouseAction::Click) {
        if (!bounds.contains(mouse.position)) {
            return false;
        }

        const int clamped_line = std::clamp(local.y - line_y, 0, static_cast<int>(lines.size()) - 1);
        mouse_press_column_ = display_column_at(lines[static_cast<std::size_t>(clamped_line)].text, text_x, local.x);
        mouse_press_index_ =
            byte_index_at(local.x, local.y, text_x, line_y, lines, text_length, tuinator::CaretAffinity::Before);
        set_cursor_at(mouse_press_index_, false, text_length);
        selecting_with_mouse_ = true;
        return true;
    }

    return false;
}

bool WrappedTextSelection::handle_shortcut(const tuinator::KeyPress& key, const std::string& text) {
    char shortcut = '\0';
    if (key.ctrl && key.character >= 'a' && key.character <= 'z') {
        shortcut = key.character;
    } else if (key.character >= 1 && key.character <= 26) {
        shortcut = static_cast<char>('a' + key.character - 1);
    }

    if (shortcut == '\0') {
        return false;
    }

    switch (shortcut) {
    case 'a':
        select_all(text.size());
        return true;
    case 'c':
        copy_selection(text);
        return true;
    default:
        break;
    }

    return false;
}

bool WrappedTextSelection::handle_escape() {
    if (!has_selection()) {
        return false;
    }
    clear_selection(cursor_);
    return true;
}

void WrappedTextSelection::copy_selection(const std::string& text) const {
    if (!has_selection()) {
        return;
    }

    const auto [start, end] = selection_range();
    tuinator::clipboard::set(text.substr(start, end - start));
}

void WrappedTextSelection::paint_highlights(tuinator::PaintContext& ctx,
                                            const std::vector<WrappedLayoutLine>& lines, int text_x, int line_y,
                                            int max_height, const tuinator::Style& style) const {
    if (!has_selection() || lines.empty() || max_height <= 0) {
        return;
    }

    const auto [sel_start, sel_end] = selection_range();
    tuinator::Style selection_style = style;
    selection_style.reverse = true;

    int y = line_y;
    for (const WrappedLayoutLine& line : lines) {
        if (y >= line_y + max_height) {
            break;
        }

        const std::size_t highlight_start = std::max(sel_start, line.source_start);
        const std::size_t highlight_end = std::min(sel_end, line.source_end);
        if (highlight_end > highlight_start) {
            const std::size_t local_start = highlight_start - line.source_start;
            const std::size_t local_end = highlight_end - line.source_start;
            const std::string_view before(line.text.data(), local_start);
            const std::string_view selected(line.text.data() + local_start, local_end - local_start);
            const std::string_view after(line.text.data() + local_end, line.text.size() - local_end);

            int draw_x = text_x;
            if (!before.empty()) {
                ctx.canvas.draw_text({draw_x, y}, before, style);
                draw_x += tuinator::text_display_width(before);
            }
            if (!selected.empty()) {
                ctx.canvas.draw_text({draw_x, y}, selected, selection_style);
                draw_x += tuinator::text_display_width(selected);
            }
            if (!after.empty()) {
                ctx.canvas.draw_text({draw_x, y}, after, style);
            }
        }

        ++y;
    }
}
