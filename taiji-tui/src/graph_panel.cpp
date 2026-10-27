#include "graph_panel.hpp"

#include "ui_test_ids.hpp"

#include "ui_theme.hpp"

#include <cctype>

#include <tuinator/render/text.hpp>

#include <algorithm>

namespace {

std::string truncate_to_width(const std::string& text, int max_cols) {
    if (max_cols <= 0) {
        return "";
    }
    if (tuinator::text_display_width(text) <= max_cols) {
        return text;
    }
    std::string out;
    for (char ch : text) {
        const std::string next = out + ch;
        if (tuinator::text_display_width(next) > max_cols - 1) {
            break;
        }
        out = next;
    }
    return out + "…";
}

std::string json_extract_value(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return "";
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    if (i >= json.size()) {
        return "";
    }
    if (json[i] != '{' && json[i] != '[') {
        return "";
    }
    const char open = json[i];
    const char close = open == '{' ? '}' : ']';
    int depth = 0;
    const std::size_t start = i;
    for (; i < json.size(); ++i) {
        const char ch = json[i];
        if (ch == '"') {
            ++i;
            while (i < json.size()) {
                if (json[i] == '\\' && i + 1 < json.size()) {
                    i += 2;
                    continue;
                }
                if (json[i] == '"') {
                    break;
                }
                ++i;
            }
            continue;
        }
        if (ch == open) {
            ++depth;
        } else if (ch == close) {
            --depth;
            if (depth == 0) {
                return json.substr(start, i - start + 1);
            }
        }
    }
    return "";
}

std::string json_extract_string(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return "";
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    if (i >= json.size() || json[i] != '"') {
        return "";
    }
    ++i;
    std::string raw;
    while (i < json.size()) {
        if (json[i] == '"') {
            return raw;
        }
        if (json[i] == '\\' && i + 1 < json.size()) {
            raw.push_back(json[i + 1]);
            i += 2;
            continue;
        }
        raw.push_back(json[i]);
        ++i;
    }
    return "";
}

bool json_extract_bool(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return false;
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    return json.compare(i, 4, "true") == 0;
}

int64_t json_extract_int(const std::string& json, const std::string& key, int64_t fallback = -1) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return fallback;
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    const std::size_t begin = i;
    if (i < json.size() && (json[i] == '-' || json[i] == '+')) {
        ++i;
    }
    while (i < json.size() && std::isdigit(static_cast<unsigned char>(json[i]))) {
        ++i;
    }
    if (i == begin) {
        return fallback;
    }
    try {
        return std::stoll(json.substr(begin, i - begin));
    } catch (...) {
        return fallback;
    }
}

std::vector<std::string> json_extract_string_array_simple(const std::string& json, const std::string& key) {
    std::vector<std::string> out;
    const std::string arr = json_extract_value(json, key);
    if (arr.empty() || arr.front() != '[') {
        return out;
    }
    std::size_t i = 0;
    while (i < arr.size()) {
        const std::size_t q = arr.find('"', i);
        if (q == std::string::npos) {
            break;
        }
        std::size_t end = q + 1;
        std::string item;
        while (end < arr.size()) {
            if (arr[end] == '"' && arr[end - 1] != '\\') {
                break;
            }
            item += arr[end];
            ++end;
        }
        if (!item.empty()) {
            out.push_back(item);
        }
        i = end + 1;
    }
    return out;
}

void parse_entities(const std::string& json, const std::string& key, std::vector<std::string>& out) {
    const std::string arr = json_extract_value(json, key);
    if (arr.empty()) {
        return;
    }
    std::size_t i = 0;
    while (i < arr.size()) {
        const std::size_t start = arr.find('{', i);
        if (start == std::string::npos) {
            break;
        }
        int depth = 0;
        std::size_t end = start;
        for (; end < arr.size(); ++end) {
            if (arr[end] == '{') {
                ++depth;
            } else if (arr[end] == '}') {
                --depth;
                if (depth == 0) {
                    break;
                }
            }
        }
        if (depth != 0) {
            break;
        }
        const std::string obj = arr.substr(start, end - start + 1);
        std::string line = json_extract_string(obj, "name");
        const std::string mem = json_extract_string(obj, "memory");
        const std::string pers = json_extract_string(obj, "persistence");
        if (!pers.empty()) {
            line += " [" + pers + "]";
        }
        if (!mem.empty()) {
            line += " — " + mem;
        }
        out.push_back(std::move(line));
        i = end + 1;
    }
}

void parse_predecessor_nodes(const std::string& json, const std::string& key, std::vector<std::string>& out) {
    const std::string arr = json_extract_value(json, key);
    if (arr.empty()) {
        return;
    }
    std::size_t i = 0;
    while (i < arr.size()) {
        const std::size_t start = arr.find('{', i);
        if (start == std::string::npos) {
            break;
        }
        int depth = 0;
        std::size_t end = start;
        for (; end < arr.size(); ++end) {
            if (arr[end] == '{') {
                ++depth;
            } else if (arr[end] == '}') {
                --depth;
                if (depth == 0) {
                    break;
                }
            }
        }
        if (depth != 0) {
            break;
        }
        const std::string obj = arr.substr(start, end - start + 1);
        std::string title = json_extract_string(obj, "title");
        const std::string label = json_extract_string(obj, "label");
        const bool visited = json_extract_bool(obj, "visited");
        const bool locked = json_extract_bool(obj, "locked");
        if (visited) {
            title += " (besucht)";
        }
        if (locked) {
            title = "🔒 " + title;
        }
        if (!label.empty()) {
            title += " — " + label;
        }
        out.push_back(title);
        i = end + 1;
    }
}

struct ParsedSuccessor {
    std::string id;
    std::string title_line;
};

std::vector<ParsedSuccessor> parse_successor_nodes(const std::string& json) {
    std::vector<ParsedSuccessor> out;
    const std::string arr = json_extract_value(json, "successors");
    if (arr.empty()) {
        return out;
    }
    std::size_t i = 0;
    while (i < arr.size()) {
        const std::size_t start = arr.find('{', i);
        if (start == std::string::npos) {
            break;
        }
        int depth = 0;
        std::size_t end = start;
        for (; end < arr.size(); ++end) {
            if (arr[end] == '{') {
                ++depth;
            } else if (arr[end] == '}') {
                --depth;
                if (depth == 0) {
                    break;
                }
            }
        }
        if (depth != 0) {
            break;
        }
        const std::string obj = arr.substr(start, end - start + 1);
        ParsedSuccessor item;
        item.id = json_extract_string(obj, "id");
        std::string title = json_extract_string(obj, "title");
        const std::string label = json_extract_string(obj, "label");
        const bool visited = json_extract_bool(obj, "visited");
        const bool locked = json_extract_bool(obj, "locked");
        if (visited) {
            title += " (besucht)";
        }
        if (locked) {
            title = "🔒 " + title;
        }
        if (!label.empty()) {
            title += " — " + label;
        }
        item.title_line = title;
        out.push_back(std::move(item));
        i = end + 1;
    }
    return out;
}

} // namespace

GraphPanel::GraphPanel(tuinator::Theme theme) : theme_(std::move(theme)) {
    set_test_id(taiji::ui::test_id::kGraphPanel);
}

void GraphPanel::set_layout(Layout layout) {
    if (layout_ == layout) {
        return;
    }
    layout_ = layout;
    mark_layout_dirty();
    mark_dirty();
    if (on_layout_changed_) {
        on_layout_changed_();
    }
}

std::string GraphPanel::successor_id(std::size_t index) const {
    if (index >= successor_ids_.size()) {
        return "";
    }
    return successor_ids_[index];
}

std::string GraphPanel::hotkey_hint_line() const {
    if (successors_.empty()) {
        return "";
    }
    std::string line = "Alt+";
    for (std::size_t i = 0; i < successors_.size() && i < 9; ++i) {
        if (i > 0) {
            line += "  ";
        }
        line += "[" + std::to_string(i + 1) + "] " + successors_[i].title_line;
    }
    return line;
}

void GraphPanel::set_from_json(const std::string& json) {
    active_ = json_extract_bool(json, "active");
    graph_name_ = json_extract_string(json, "graph_name");
    current_title_ = json_extract_string(json, "current_title");
    node_state_ = json_extract_string(json, "node_state");
    exit_hint_ = json_extract_string(json, "exit_hint");
    breadcrumb_ = json_extract_string(json, "breadcrumb");
    visible_turns_ = static_cast<int>(json_extract_int(json, "visible_turns", 0));
    entity_count_ = static_cast<int>(json_extract_int(json, "entity_count", 0));
    predecessors_.clear();
    successors_.clear();
    here_.clear();
    party_.clear();
    node_tools_.clear();
    tree_lines_ = json_extract_string_array_simple(json, "tree_lines");
    node_tools_ = json_extract_string_array_simple(json, "node_tools");
    parse_predecessor_nodes(json, "predecessors", predecessors_);
    successors_.clear();
    successor_ids_.clear();
    for (const ParsedSuccessor& item : parse_successor_nodes(json)) {
        SuccessorExit exit;
        exit.id = item.id;
        exit.title_line = item.title_line;
        successors_.push_back(std::move(exit));
        successor_ids_.push_back(item.id);
    }
    parse_entities(json, "here_entities", here_);
    parse_entities(json, "party_entities", party_);
    mark_layout_dirty();
    mark_dirty();
    if (on_layout_changed_) {
        on_layout_changed_();
    }
}

void GraphPanel::set_compact_mode(bool compact) {
    if (compact_ == compact) {
        return;
    }
    compact_ = compact;
    mark_layout_dirty();
    mark_dirty();
    if (on_layout_changed_) {
        on_layout_changed_();
    }
}

void GraphPanel::toggle_compact_mode() { set_compact_mode(!compact_); }

void GraphPanel::set_tree_overlay(bool open) {
    if (tree_overlay_ == open) {
        return;
    }
    tree_overlay_ = open;
    mark_dirty();
}

void GraphPanel::toggle_tree_overlay() { set_tree_overlay(!tree_overlay_); }

void GraphPanel::set_on_layout_changed(std::function<void()> callback) {
    on_layout_changed_ = std::move(callback);
}

int GraphPanel::content_height() const {
    if (!active_) {
        return 0;
    }
    if (compact_) {
        return layout_ == Layout::HorizontalStrip ? 2 : 3;
    }
    if (layout_ == Layout::HorizontalStrip) {
        return 14;
    }
    return 16;
}

tuinator::Size GraphPanel::preferred_size() const {
    if (!active_) {
        return {0, 0};
    }
    return {0, content_height()};
}

void GraphPanel::layout(tuinator::Rect bounds) { bounds_ = bounds; }

void GraphPanel::paint_radar(tuinator::PaintContext& ctx, int x, int y, int w, int max_h, tuinator::Style label,
                             tuinator::Style muted, tuinator::Style accent) const {
    int row = y;
    const auto draw = [&](const std::string& text, const tuinator::Style& style) {
        if (row >= y + max_h) {
            return;
        }
        const std::string clipped = truncate_to_width(text, w);
        ctx.canvas.draw_text({x, row}, clipped, style);
        ++row;
    };
    draw("RADAR", label);
    for (const std::string& p : predecessors_) {
        draw("  " + p, muted);
        draw("    │", muted);
        draw("    ▼", muted);
    }
    draw("  ╭─ " + current_title_ + " (aktiv) ─╮", accent);
    draw("  ╰────────────────────────╯", accent);
    for (std::size_t i = 0; i < successors_.size(); ++i) {
        std::string line = "    ├──▶ ";
        if (i < 9) {
            line += "[" + std::to_string(i + 1) + "] ";
        }
        line += successors_[i].title_line;
        draw(line, label);
    }
    if (row < y + max_h && !exit_hint_.empty()) {
        draw("  ⏳ " + exit_hint_, muted);
    }
}

void GraphPanel::paint_entities(tuinator::PaintContext& ctx, int x, int y, int w, int max_h, tuinator::Style label,
                                tuinator::Style muted, tuinator::Style accent) const {
    int row = y;
    const auto draw = [&](const std::string& text, const tuinator::Style& style) {
        if (row >= y + max_h) {
            return;
        }
        ctx.canvas.draw_text({x, row}, truncate_to_width(text, w), style);
        ++row;
    };
    draw("ENTITIES", label);
    draw("📌 Hier:", accent);
    if (here_.empty() && node_tools_.empty()) {
        draw("  (leer)", muted);
    }
    for (const std::string& t : node_tools_) {
        draw("  🛠 " + t, label);
    }
    for (const std::string& e : here_) {
        draw("  • " + e, label);
    }
    if (!node_state_.empty()) {
        draw("  [" + node_state_ + "]", muted);
    }
    draw("🎒 Party:", accent);
    if (party_.empty()) {
        draw("  (niemand)", muted);
    }
    for (const std::string& e : party_) {
        draw("  • " + e, label);
    }
}

void GraphPanel::paint_tree_overlay(tuinator::PaintContext& ctx) const {
    if (!tree_overlay_ || bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }
    tuinator::Style fill = theme_.label;
    fill.background_rgb = ui_main_bg();
    fill.foreground_rgb = ui_main_bg();
    ctx.canvas.fill_rect(bounds_, ' ', fill);
    tuinator::Style frame = theme_.label;
    frame.foreground_rgb = ui_border_chrome();
    frame.background_rgb = ui_main_panel_style().background_rgb;
    const int pad = 1;
    tuinator::Rect inner{bounds_.x + pad, bounds_.y + pad, bounds_.width - pad * 2, bounds_.height - pad * 2};
    ctx.canvas.fill_rect(inner, ' ', frame);
    tuinator::Style title = frame;
    title.foreground_rgb = ui_accent();
    int y = inner.y + 1;
    ctx.canvas.draw_text({inner.x + 1, y}, "🗺️ Graph (Ctrl+G schließen)", title);
    ++y;
    tuinator::Style line_style = frame;
    line_style.dim = true;
    for (const std::string& line : tree_lines_) {
        if (y >= inner.y + inner.height - 1) {
            break;
        }
        ctx.canvas.draw_text({inner.x + 1, y}, truncate_to_width(line, inner.width - 2), line_style);
        ++y;
    }
}

void GraphPanel::paint_horizontal_strip(tuinator::PaintContext& ctx, tuinator::Style label, tuinator::Style muted,
                                      tuinator::Style accent) const {
    const int pad = 1;
    const int x0 = bounds_.x + pad;
    const int y0 = bounds_.y + pad;
    const int inner_w = bounds_.width - pad * 2;
    const int inner_h = bounds_.height - pad * 2;
    if (inner_w <= 4 || inner_h <= 4) {
        return;
    }

    tuinator::Style frame = label;
    frame.foreground_rgb = ui_border_chrome();
    ctx.canvas.draw_text({x0, y0}, "╭─", frame);
    const std::string cap = " 🗺️ GRAPH · " + current_title_ + " ";
    ctx.canvas.draw_text({x0 + 2, y0}, truncate_to_width(cap, inner_w - 4), accent);
    const int split_x = x0 + std::max(20, inner_w * 3 / 5);
    const int body_y = y0 + 1;
    const int body_h = std::max(1, inner_h - 3);
    const int left_w = std::max(8, split_x - x0 - 1);
    const int right_w = std::max(8, x0 + inner_w - split_x - 1);
    paint_radar(ctx, x0 + 1, body_y, left_w, body_h, label, muted, accent);
    tuinator::Style sep = frame;
    sep.dim = true;
    for (int y = body_y; y < body_y + body_h && y < bounds_.y + bounds_.height; ++y) {
        ctx.canvas.draw_text({split_x, y}, "│", sep);
    }
    paint_entities(ctx, split_x + 2, body_y, right_w, body_h, label, muted, accent);
    const int foot_y = bounds_.y + bounds_.height - pad - 1;
    const std::string hint = hotkey_hint_line();
    if (!hint.empty()) {
        ctx.canvas.draw_text({x0 + 1, foot_y}, truncate_to_width("🚪 " + hint, inner_w - 2), muted);
    } else if (!exit_hint_.empty()) {
        ctx.canvas.draw_text({x0 + 1, foot_y}, truncate_to_width("⏳ " + exit_hint_, inner_w - 2), muted);
    }
}

void GraphPanel::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0 || !active_) {
        return;
    }
    paint_bounds_background(ctx, ui_main_panel_style());
    tuinator::Style label = theme_.label;
    label.foreground_rgb = ui_section_heading();
    label.background_rgb = ui_main_panel_style().background_rgb;
    tuinator::Style muted = label;
    muted.dim = true;
    muted.foreground_rgb = ui_secondary_text();
    tuinator::Style accent = label;
    accent.foreground_rgb = ui_session_active_bar();
    accent.bold = true;

    const int pad = 1;
    int y = bounds_.y + pad;
    const int inner_w = bounds_.width - pad * 2;

    std::string header = "🗺️ " + graph_name_;
    if (!current_title_.empty()) {
        header += " · " + current_title_;
    }
    ctx.canvas.draw_text({bounds_.x + pad, y}, truncate_to_width(header, inner_w), label);
    ++y;

    if (visible_turns_ > 0 || entity_count_ > 0) {
        std::string status = "💾 World: " + std::to_string(entity_count_) + " entities";
        if (visible_turns_ > 0) {
            status += " | Turns: " + std::to_string(visible_turns_);
        }
        ctx.canvas.draw_text({bounds_.x + pad, y}, truncate_to_width(status, inner_w), muted);
        ++y;
    }

    if (layout_ == Layout::HorizontalStrip) {
        if (compact_) {
            if (!breadcrumb_.empty()) {
                ctx.canvas.draw_text({bounds_.x + pad, bounds_.y + pad},
                                     truncate_to_width(breadcrumb_, inner_w), accent);
            }
            paint_tree_overlay(ctx);
            return;
        }
        paint_horizontal_strip(ctx, label, muted, accent);
        paint_tree_overlay(ctx);
        return;
    }

    if (compact_) {
        if (!breadcrumb_.empty()) {
            ctx.canvas.draw_text({bounds_.x + pad, y}, truncate_to_width(breadcrumb_, inner_w), accent);
        }
        paint_tree_overlay(ctx);
        return;
    }

    const int col_w = std::max(8, inner_w / 2 - 1);
    const int left_x = bounds_.x + pad;
    const int right_x = left_x + col_w + 2;
    const int body_h = std::max(1, bounds_.height - (y - bounds_.y) - pad);
    paint_radar(ctx, left_x, y, col_w, body_h, label, muted, accent);
    paint_entities(ctx, right_x, y, bounds_.width - (right_x - bounds_.x) - pad, body_h, label, muted, accent);
    paint_tree_overlay(ctx);
}

bool GraphPanel::handle_event(const tuinator::Event& event) {
    if (!active_) {
        return false;
    }
    const auto* mouse = std::get_if<tuinator::MouseEvent>(&event);
    if (mouse != nullptr && mouse->button == tuinator::MouseButton::Left &&
        mouse->action == tuinator::MouseAction::Release && bounds_.contains(mouse->position)) {
        if (tree_overlay_) {
            set_tree_overlay(false);
            return true;
        }
        toggle_compact_mode();
        return true;
    }
    return false;
}

void GraphPanel::collect_focusable(std::vector<tuinator::Widget*>&) {}
