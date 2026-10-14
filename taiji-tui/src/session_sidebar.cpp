#include "session_sidebar.hpp"

#include <libtaiji.h>

#include <tuinator/render/text.hpp>

#include <cstdio>
#include <cstdlib>
#include <ctime>

namespace {

std::string json_unescape(const std::string& value) {
    std::string out;
    out.reserve(value.size());
    for (std::size_t i = 0; i < value.size(); ++i) {
        if (value[i] != '\\' || i + 1 >= value.size()) {
            out.push_back(value[i]);
            continue;
        }
        const char next = value[i + 1];
        switch (next) {
        case '"':
            out.push_back('"');
            break;
        case '\\':
            out.push_back('\\');
            break;
        case 'n':
            out.push_back('\n');
            break;
        case 't':
            out.push_back('\t');
            break;
        default:
            out.push_back(next);
            break;
        }
        ++i;
    }
    return out;
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
            return json_unescape(raw);
        }
        if (json[i] == '\\' && i + 1 < json.size()) {
            raw.push_back(json[i]);
            raw.push_back(json[i + 1]);
            i += 2;
            continue;
        }
        raw.push_back(json[i]);
        ++i;
    }
    return "";
}

std::vector<std::string> json_split_objects(const std::string& array_json) {
    std::vector<std::string> out;
    if (array_json.empty() || array_json.front() != '[') {
        return out;
    }
    int depth = 0;
    std::size_t start = std::string::npos;
    for (std::size_t i = 0; i < array_json.size(); ++i) {
        const char ch = array_json[i];
        if (ch == '"') {
            ++i;
            while (i < array_json.size()) {
                if (array_json[i] == '\\' && i + 1 < array_json.size()) {
                    i += 2;
                    continue;
                }
                if (array_json[i] == '"') {
                    break;
                }
                ++i;
            }
            continue;
        }
        if (ch == '{') {
            if (depth == 1 && start == std::string::npos) {
                start = i;
            }
            ++depth;
        } else if (ch == '}') {
            --depth;
            if (depth == 1 && start != std::string::npos) {
                out.push_back(array_json.substr(start, i - start + 1));
                start = std::string::npos;
            }
        } else if (ch == '[') {
            ++depth;
        } else if (ch == ']') {
            --depth;
        }
    }
    return out;
}

constexpr char kProjectNewGlyph[] = "+";

} // namespace

SessionSidebar::SessionSidebar(tuinator::Theme theme, SessionSelectHandler on_select, NewSessionHandler on_new,
                               SessionContextHandler on_context)
    : theme_(theme),
      on_select_(std::move(on_select)),
      on_new_(std::move(on_new)),
      on_context_(std::move(on_context)) {}

void SessionSidebar::set_width_hint(int cols) {
    width_hint_ = std::max(18, cols);
    mark_layout_dirty();
    mark_dirty();
}

void SessionSidebar::refresh(uint64_t taiji_handle) {
    projects_.clear();
    if (taiji_handle == 0) {
        rebuild_rows();
        mark_dirty();
        return;
    }

    char* raw = taiji_projects_list(taiji_handle);
    if (raw == nullptr) {
        rebuild_rows();
        mark_dirty();
        return;
    }
    const std::string json(raw);
    std::free(raw);

    const std::string current_session = json_extract_string(json, "current_session");
    const std::string projects_array = [&json]() {
        const std::string needle = "\"projects\":";
        const std::size_t pos = json.find(needle);
        if (pos == std::string::npos) {
            return std::string();
        }
        std::size_t i = pos + needle.size();
        while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
            ++i;
        }
        if (i >= json.size() || json[i] != '[') {
            return std::string();
        }
        int depth = 0;
        const std::size_t start = i;
        for (; i < json.size(); ++i) {
            if (json[i] == '[') {
                ++depth;
            } else if (json[i] == ']') {
                --depth;
                if (depth == 0) {
                    return json.substr(start, i - start + 1);
                }
            }
        }
        return std::string();
    }();

    for (const std::string& project_json : json_split_objects(projects_array)) {
        ProjectGroup group;
        group.root = json_extract_string(project_json, "root");
        group.name = json_extract_string(project_json, "name");
        if (group.name.empty()) {
            group.name = group.root;
        }

        const std::string sessions_array = [&project_json]() {
            const std::string needle = "\"sessions\":";
            const std::size_t pos = project_json.find(needle);
            if (pos == std::string::npos) {
                return std::string();
            }
            std::size_t i = pos + needle.size();
            while (i < project_json.size() && (project_json[i] == ' ' || project_json[i] == '\t')) {
                ++i;
            }
            if (i >= project_json.size() || project_json[i] != '[') {
                return std::string();
            }
            int depth = 0;
            const std::size_t start = i;
            for (; i < project_json.size(); ++i) {
                if (project_json[i] == '[') {
                    ++depth;
                } else if (project_json[i] == ']') {
                    --depth;
                    if (depth == 0) {
                        return project_json.substr(start, i - start + 1);
                    }
                }
            }
            return std::string();
        }();

        for (const std::string& session_json : json_split_objects(sessions_array)) {
            SessionItem item;
            item.project_root = group.root;
            item.session_id = json_extract_string(session_json, "id");
            item.name = json_extract_string(session_json, "name");
            item.created_at = json_extract_string(session_json, "created_at");
            item.active = !current_session.empty() && item.session_id == current_session;
            if (item.name.empty()) {
                item.name = "Neue Session";
            }
            group.sessions.push_back(std::move(item));
        }
        if (!group.sessions.empty()) {
            projects_.push_back(std::move(group));
        }
    }

    rebuild_rows();
    mark_dirty();
}

void SessionSidebar::expand_project(const std::string& project_root) {
    if (project_root.empty()) {
        return;
    }
    collapsed_projects_.erase(project_root);
    rebuild_rows();
    mark_dirty();
}

bool SessionSidebar::is_project_collapsed(const std::string& project_root) const {
    return collapsed_projects_.find(project_root) != collapsed_projects_.end();
}

void SessionSidebar::toggle_project_collapsed(const std::string& project_root) {
    if (project_root.empty()) {
        return;
    }
    if (is_project_collapsed(project_root)) {
        collapsed_projects_.erase(project_root);
    } else {
        collapsed_projects_.insert(project_root);
    }
    rebuild_rows();
    mark_dirty();
}

void SessionSidebar::rebuild_rows() {
    rows_.clear();
    rows_.push_back(Row{.kind = Row::Kind::NewChat, .label = "Neuer Chat"});
    for (const ProjectGroup& project : projects_) {
        rows_.push_back(Row{.kind = Row::Kind::ProjectHeader, .project_root = project.root, .label = project.name});
        if (is_project_collapsed(project.root)) {
            continue;
        }
        for (const SessionItem& session : project.sessions) {
            rows_.push_back(Row{
                .kind = Row::Kind::Session,
                .project_root = project.root,
                .session_id = session.session_id,
                .label = session.name,
                .suffix = format_relative_time(session.created_at),
                .active = session.active,
            });
        }
    }
    if (scroll_offset_ > static_cast<int>(rows_.size())) {
        scroll_offset_ = 0;
    }
}

std::string SessionSidebar::format_relative_time(const std::string& iso_time) const {
    if (iso_time.empty()) {
        return "";
    }

    std::tm tm{};
    int year = 0;
    int month = 0;
    int day = 0;
    int hour = 0;
    int minute = 0;
    int second = 0;
    if (std::sscanf(iso_time.c_str(), "%d-%d-%dT%d:%d:%d", &year, &month, &day, &hour, &minute, &second) < 6) {
        return "";
    }
    tm.tm_year = year - 1900;
    tm.tm_mon = month - 1;
    tm.tm_mday = day;
    tm.tm_hour = hour;
    tm.tm_min = minute;
    tm.tm_sec = second;
    tm.tm_isdst = -1;

    const std::time_t created = timegm(&tm);
    if (created <= 0) {
        return "";
    }
    const std::time_t now = std::time(nullptr);
    const std::time_t diff = now > created ? now - created : 0;

    char buf[16];
    if (diff < 60) {
        std::snprintf(buf, sizeof(buf), "jetzt");
    } else if (diff < 3600) {
        std::snprintf(buf, sizeof(buf), "%dm", static_cast<int>(diff / 60));
    } else if (diff < 86400) {
        std::snprintf(buf, sizeof(buf), "%dh", static_cast<int>(diff / 3600));
    } else {
        std::snprintf(buf, sizeof(buf), "%dd", static_cast<int>(diff / 86400));
    }
    return buf;
}

std::string SessionSidebar::truncate_label(const std::string& text, int max_cols) const {
    if (max_cols <= 0 || text.empty()) {
        return {};
    }
    if (tuinator::text_display_width(text) <= max_cols) {
        return text;
    }
    if (max_cols <= 1) {
        return "…";
    }
    const std::size_t bytes = tuinator::text_byte_length_for_width(text, max_cols - 1);
    if (bytes == 0) {
        return "…";
    }
    return text.substr(0, bytes) + "…";
}

tuinator::Size SessionSidebar::preferred_size() const {
    return {width_hint_, std::max(8, static_cast<int>(rows_.size()))};
}

void SessionSidebar::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    const int visible_rows = std::max(1, bounds_.height);
    if (static_cast<int>(rows_.size()) > visible_rows) {
        scroll_offset_ = std::min(scroll_offset_, static_cast<int>(rows_.size()) - visible_rows);
    } else {
        scroll_offset_ = 0;
    }
}

int SessionSidebar::row_at_y(int local_y) const {
    if (local_y < 0 || local_y >= bounds_.height) {
        return -1;
    }
    return scroll_offset_ + local_y;
}

int SessionSidebar::project_new_button_x() const {
    return std::max(0, bounds_.width - tuinator::text_display_width(kProjectNewGlyph));
}

bool SessionSidebar::is_project_new_click(int local_x, int local_y) const {
    const int row_index = row_at_y(local_y);
    if (row_index < 0 || row_index >= static_cast<int>(rows_.size())) {
        return false;
    }
    const Row& row = rows_[static_cast<std::size_t>(row_index)];
    if (row.kind != Row::Kind::ProjectHeader) {
        return false;
    }
    return local_x >= project_new_button_x();
}

void SessionSidebar::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    paint_bounds_background(ctx, theme_.muted);

    const int visible_rows = std::min(bounds_.height, static_cast<int>(rows_.size()) - scroll_offset_);
    for (int y = 0; y < visible_rows; ++y) {
        const int row_index = scroll_offset_ + y;
        if (row_index < 0 || row_index >= static_cast<int>(rows_.size())) {
            break;
        }
        const Row& row = rows_[static_cast<std::size_t>(row_index)];

        tuinator::Style style = theme_.label;
        if (row.kind == Row::Kind::NewChat) {
            style = theme_.accent;
            style.bold = true;
        } else if (row.kind == Row::Kind::ProjectHeader) {
            style = theme_.heading;
            style.bold = true;
        } else if (row.active) {
            style = theme_.accent;
        } else if (row_index == hovered_row_) {
            style.background_rgb = tuinator::rgb(48, 42, 68);
        }

        std::string text;
        if (row.kind == Row::Kind::ProjectHeader) {
            const char* chevron = is_project_collapsed(row.project_root) ? "▸" : "▾";
            text = std::string(chevron) + " " + row.label;
        } else if (row.kind == Row::Kind::Session) {
            text = "  • " + row.label;
        } else {
            text = "+ " + row.label;
        }

        const int new_btn_cols =
            row.kind == Row::Kind::ProjectHeader ? tuinator::text_display_width(kProjectNewGlyph) + 1 : 0;
        const int suffix_cols = row.suffix.empty() ? 0 : tuinator::text_display_width(row.suffix) + 1;
        const int label_cols = std::max(4, bounds_.width - suffix_cols - new_btn_cols);
        text = truncate_label(text, label_cols);

        ctx.canvas.draw_text({0, y}, text, style);
        if (row.kind == Row::Kind::ProjectHeader) {
            tuinator::Style plus_style = theme_.accent;
            plus_style.bold = true;
            ctx.canvas.draw_text({project_new_button_x(), y}, kProjectNewGlyph, plus_style);
        }
        if (!row.suffix.empty()) {
            const int suffix_x = std::max(0, bounds_.width - new_btn_cols - tuinator::text_display_width(row.suffix));
            tuinator::Style suffix_style = theme_.muted;
            ctx.canvas.draw_text({suffix_x, y}, row.suffix, suffix_style);
        }
    }
}

bool SessionSidebar::handle_event(const tuinator::Event& event) {
    const auto* mouse = std::get_if<tuinator::MouseEvent>(&event);
    if (mouse != nullptr) {
        const tuinator::Point local{mouse->position.x - bounds_.x, mouse->position.y - bounds_.y};
        if (mouse->action == tuinator::MouseAction::Move) {
            hovered_row_ = row_at_y(local.y);
            mark_dirty();
            return bounds_.contains(mouse->position);
        }
        if (mouse->action == tuinator::MouseAction::Press || mouse->action == tuinator::MouseAction::Click ||
            mouse->action == tuinator::MouseAction::Release) {
            if (!bounds_.contains(mouse->position)) {
                return false;
            }

            if (mouse->button == tuinator::MouseButton::Right &&
                (mouse->action == tuinator::MouseAction::Click || mouse->action == tuinator::MouseAction::Release)) {
                const int row_index = row_at_y(local.y);
                if (row_index >= 0 && row_index < static_cast<int>(rows_.size())) {
                    const Row& row = rows_[static_cast<std::size_t>(row_index)];
                    if (row.kind == Row::Kind::Session && on_context_) {
                        on_context_(row.project_root, row.session_id, mouse->position);
                    }
                }
                return true;
            }

            if (mouse->button != tuinator::MouseButton::Left) {
                return true;
            }

            if (mouse->action != tuinator::MouseAction::Release) {
                return true;
            }

            const int row_index = row_at_y(local.y);
            if (row_index < 0 || row_index >= static_cast<int>(rows_.size())) {
                return false;
            }
            const Row& row = rows_[static_cast<std::size_t>(row_index)];

            if (row.kind == Row::Kind::NewChat) {
                if (on_new_) {
                    on_new_("");
                }
                return true;
            }
            if (row.kind == Row::Kind::ProjectHeader) {
                if (is_project_new_click(local.x, local.y)) {
                    if (on_new_) {
                        on_new_(row.project_root);
                    }
                } else {
                    toggle_project_collapsed(row.project_root);
                }
                return true;
            }
            if (row.kind == Row::Kind::Session && on_select_) {
                on_select_(row.project_root, row.session_id);
                return true;
            }
            return true;
        }
    }

    const auto* key = std::get_if<tuinator::KeyPress>(&event);
    if (key != nullptr) {
        if (key->key == tuinator::Key::Up) {
            scroll_offset_ = std::max(0, scroll_offset_ - 1);
            mark_dirty();
            return true;
        }
        if (key->key == tuinator::Key::Down) {
            const int max_offset = std::max(0, static_cast<int>(rows_.size()) - bounds_.height);
            scroll_offset_ = std::min(max_offset, scroll_offset_ + 1);
            mark_dirty();
            return true;
        }
    }

    return false;
}
