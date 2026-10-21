#include "settings_modal.hpp"
#include "model_display.hpp"
#include "skim_fuzzy.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <libtaiji.h>

#include <algorithm>
#include <cstdlib>

SettingsModal::FuzzyOverlayLayout SettingsModal::fuzzy_overlay_layout(const PanelRect& panel) const {
    FuzzyOverlayLayout layout;
    layout.w = std::min(panel.w - 4, 72);
    const int max_h = 22;
    layout.h = std::min(max_h, panel.h - 4);
    layout.x = (panel.w - layout.w) / 2;
    layout.y = (panel.h - layout.h) / 2;
    layout.list_y = layout.y + 4;
    layout.max_rows = std::max(1, layout.h - 6);
    return layout;
}

SettingsModal::ModelDetailOverlayLayout SettingsModal::model_detail_overlay_layout(const PanelRect& panel) const {
    ModelDetailOverlayLayout layout;
    layout.w = std::min(panel.w - 4, 84);
    layout.h = std::min(panel.h - 4, 26);
    layout.x = (panel.w - layout.w) / 2;
    layout.y = (panel.h - layout.h) / 2;
    layout.body_rows = std::max(1, layout.h - 4);
    return layout;
}

bool SettingsModal::fuzzy_overlay_panel_point(tuinator::Point local, const PanelRect& panel, int& panel_x,
                                              int& panel_y) const {
    panel_x = local.x - panel.x;
    panel_y = local.y - panel.y;
    return panel_x >= 0 && panel_y >= 0 && panel_x < panel.w && panel_y < panel.h;
}

void SettingsModal::sync_fuzzy_viewport_to_cursor() {
    sync_fuzzy_list_scroll(fuzzy_overlay_layout(panel_rect()).max_rows);
}

bool SettingsModal::fuzzy_models_match(const std::string& a, const std::string& b) const {
    if (a.empty() || b.empty()) {
        return false;
    }
    if (a == b) {
        return true;
    }
    const std::string ad = model_display_text(a);
    const std::string bd = model_display_text(b);
    return ad == bd || ad == b || a == bd;
}

void SettingsModal::sync_fuzzy_list_scroll(int max_visible_rows) {
    if (max_visible_rows <= 0) {
        fuzzy_list_scroll_ = 0;
        return;
    }
    const int n = static_cast<int>(fuzzy_filtered_.size());
    if (n == 0) {
        fuzzy_list_scroll_ = 0;
        fuzzy_cursor_ = 0;
        return;
    }
    if (fuzzy_cursor_ < 0) {
        fuzzy_cursor_ = 0;
    }
    if (fuzzy_cursor_ >= n) {
        fuzzy_cursor_ = n - 1;
    }
    if (fuzzy_cursor_ < fuzzy_list_scroll_) {
        fuzzy_list_scroll_ = fuzzy_cursor_;
    }
    if (fuzzy_cursor_ >= fuzzy_list_scroll_ + max_visible_rows) {
        fuzzy_list_scroll_ = fuzzy_cursor_ - max_visible_rows + 1;
    }
    const int max_scroll = std::max(0, n - max_visible_rows);
    if (fuzzy_list_scroll_ > max_scroll) {
        fuzzy_list_scroll_ = max_scroll;
    }
    if (fuzzy_list_scroll_ < 0) {
        fuzzy_list_scroll_ = 0;
    }
}

void SettingsModal::rebuild_fuzzy_filter() {
    fuzzy_filtered_.clear();
    std::vector<std::string> lines;
    std::vector<std::string> meta;
    std::vector<std::string> providers;
    lines.reserve(fuzzy_entries_.size());
    meta.reserve(fuzzy_entries_.size());
    providers.reserve(fuzzy_entries_.size());
    for (const FuzzyEntry& e : fuzzy_entries_) {
        lines.push_back(e.line);
        meta.push_back(e.meta);
        providers.push_back(e.provider_id);
    }
    fuzzy_filtered_ = skim_fuzzy_filter_indices(fuzzy_query_, lines, meta, providers);
    if (fuzzy_cursor_ >= static_cast<int>(fuzzy_filtered_.size())) {
        fuzzy_cursor_ = std::max(0, static_cast<int>(fuzzy_filtered_.size()) - 1);
    }
    sync_fuzzy_list_scroll(10);
    if (fuzzy_open_ && fuzzy_picker_mode_ == FuzzyPickerMode::Theme) {
        preview_theme_fuzzy_selection();
    }
}

void SettingsModal::preview_theme_fuzzy_selection() {
    if (fuzzy_picker_mode_ != FuzzyPickerMode::Theme || fuzzy_filtered_.empty()) {
        return;
    }
    if (fuzzy_cursor_ < 0 || fuzzy_cursor_ >= static_cast<int>(fuzzy_filtered_.size())) {
        return;
    }
    const FuzzyEntry& entry = fuzzy_entries_[static_cast<std::size_t>(fuzzy_filtered_[fuzzy_cursor_])];
    if (entry.model.empty() || entry.model == draft_.ui_theme) {
        return;
    }
    draft_.ui_theme = entry.model;
    if (on_preview_) {
        on_preview_(draft_);
    }
}

std::string SettingsModal::provider_display_label() const {
    const std::string id = provider_display_id();
    for (std::size_t i = 0; i < draft_.provider_options.size(); ++i) {
        if (draft_.provider_options[i] == id && i < draft_.provider_labels.size() &&
            !draft_.provider_labels[i].empty()) {
            return draft_.provider_labels[i];
        }
    }
    return id;
}

std::string SettingsModal::model_display_text(const std::string& model) const {
    return format_model_display_label(model);
}

std::string SettingsModal::current_model_id() const {
    std::string model = draft_.model;
    if (model.empty()) {
        model = draft_.model_default;
    }
    return model;
}

bool SettingsModal::model_setting_show_temperature() const {
    return !draft_.model_catalog_known || draft_.model_catalog_temperature;
}

bool SettingsModal::model_setting_show_max_tokens() const {
    // models.dev documents limit.output as a ceiling only; there is no "max_tokens ctrl"
    // flag. Models like Claude Opus expose fixed generation params (temperature: false);
    // gate max_tokens the same way as temperature for known catalog entries.
    return !draft_.model_catalog_known || draft_.model_catalog_temperature;
}

bool SettingsModal::model_setting_show_context_limit() const {
    return !draft_.model_catalog_known || draft_.model_catalog_context > 0;
}

bool SettingsModal::model_context_limit_fixed() const {
    return draft_.model_catalog_known && draft_.model_catalog_context > 0;
}

std::string SettingsModal::stt_driver_display() const {
    for (std::size_t i = 0; i < draft_.stt_driver_options.size(); ++i) {
        if (draft_.stt_driver_options[i] == draft_.stt_driver) {
            if (i < draft_.stt_driver_labels.size() && !draft_.stt_driver_labels[i].empty()) {
                return draft_.stt_driver_labels[i];
            }
            break;
        }
    }
    if (draft_.stt_driver.empty()) {
        return "(wuji default)";
    }
    return draft_.stt_driver;
}

void SettingsModal::sync_context_from_catalog() {
    if (model_context_limit_fixed()) {
        draft_.context_window = draft_.model_catalog_context;
        draft_.context_window_set = false;
    }
}

void SettingsModal::refresh_model_catalog_hints() {
    if (handle_ == 0) {
        return;
    }
    const std::string provider = provider_display_id();
    const std::string model = current_model_id();
    if (provider.empty() || model.empty()) {
        draft_.model_catalog_known = false;
        draft_.model_catalog_temperature = false;
        draft_.model_catalog_context = 0;
        draft_.model_catalog_output = 0;
        return;
    }
    char* raw = taiji_provider_model_details(handle_, const_cast<char*>(provider.c_str()),
                                             const_cast<char*>(model.c_str()));
    if (raw == nullptr) {
        return;
    }
    const std::string json = raw;
    std::free(raw);
    if (settings_json_has_error(json)) {
        return;
    }
    apply_model_catalog_json(json, draft_);
    sync_context_from_catalog();
}

std::string SettingsModal::format_context_window(int tokens) const {
    if (tokens >= 1'000'000) {
        return std::to_string(tokens / 1'000'000) + "M";
    }
    if (tokens >= 1000) {
        return std::to_string(tokens / 1000) + "k";
    }
    return std::to_string(tokens);
}

void SettingsModal::toggle_scope() {
    if (chat_only_) {
        return;
    }
    draft_.scope = draft_.scope == "chat" ? "project" : "chat";
    rebuild_rows();
    selected_row_ = first_interactive_row();
    note_dirty();
}

std::string SettingsModal::skill_state(const std::string& name) const {
    const bool denied =
        std::find(draft_.skills_deny.begin(), draft_.skills_deny.end(), name) != draft_.skills_deny.end();
    const bool pinned =
        std::find(draft_.skills_allow.begin(), draft_.skills_allow.end(), name) != draft_.skills_allow.end();
    if (denied) {
        return "off";
    }
    if (pinned) {
        return "pinned";
    }
    return "auto";
}

namespace {

std::string mcp_tool_ref_key(const std::string& server, const std::string& tool) {
    return server + ":" + tool;
}

} // namespace

bool SettingsModal::mcp_server_expanded(const std::string& server) const {
    return std::find(mcp_expanded_servers_.begin(), mcp_expanded_servers_.end(), server) !=
           mcp_expanded_servers_.end();
}

bool SettingsModal::mcp_server_has_tools(const std::string& server) const {
    for (const Draft::MCPToolEntry& tool : draft_.mcp_tools) {
        if (tool.server == server) {
            return true;
        }
    }
    return false;
}

void SettingsModal::toggle_mcp_server_expanded(const std::string& server) {
    auto it = std::find(mcp_expanded_servers_.begin(), mcp_expanded_servers_.end(), server);
    if (it == mcp_expanded_servers_.end()) {
        mcp_expanded_servers_.push_back(server);
        std::sort(mcp_expanded_servers_.begin(), mcp_expanded_servers_.end());
    } else {
        mcp_expanded_servers_.erase(it);
    }
}

bool SettingsModal::mcp_tool_denied(const std::string& server, const std::string& tool) const {
    const std::string ref = mcp_tool_ref_key(server, tool);
    for (const std::string& deny : draft_.mcp_tools_deny) {
        if (deny == ref) {
            return true;
        }
    }
    return false;
}

void SettingsModal::toggle_mcp_tool_denied(const std::string& server, const std::string& tool) {
    const std::string ref = mcp_tool_ref_key(server, tool);
    auto it = std::find(draft_.mcp_tools_deny.begin(), draft_.mcp_tools_deny.end(), ref);
    if (it == draft_.mcp_tools_deny.end()) {
        draft_.mcp_tools_deny.push_back(ref);
        std::sort(draft_.mcp_tools_deny.begin(), draft_.mcp_tools_deny.end());
    } else {
        draft_.mcp_tools_deny.erase(it);
    }
}

std::string SettingsModal::mcp_tool_description(const std::string& server, const std::string& tool) const {
    for (const Draft::MCPToolEntry& entry : draft_.mcp_tools) {
        if (entry.server == server && entry.name == tool) {
            return entry.description;
        }
    }
    return {};
}

void SettingsModal::cycle_skill_state(const std::string& name) {
    const std::string state = skill_state(name);
    auto erase_name = [&](std::vector<std::string>& list) {
        list.erase(std::remove(list.begin(), list.end(), name), list.end());
    };
    erase_name(draft_.skills_allow);
    erase_name(draft_.skills_deny);
    if (state == "auto") {
        draft_.skills_allow.push_back(name);
        std::sort(draft_.skills_allow.begin(), draft_.skills_allow.end());
    } else if (state == "pinned") {
        draft_.skills_deny.push_back(name);
        std::sort(draft_.skills_deny.begin(), draft_.skills_deny.end());
    }
}

void SettingsModal::apply_fuzzy_pick(const FuzzyEntry& pick) {
    if (fuzzy_picker_mode_ == FuzzyPickerMode::Theme) {
        if (!pick.model.empty()) {
            draft_.ui_theme = pick.model;
            note_dirty();
            if (on_preview_) {
                on_preview_(draft_);
            }
        }
        return;
    }
    if (fuzzy_picker_mode_ == FuzzyPickerMode::Provider) {
        std::size_t idx = draft_.provider_options.size();
        for (std::size_t i = 0; i < draft_.provider_options.size(); ++i) {
            if (draft_.provider_options[i] == pick.provider_id) {
                idx = i;
                break;
            }
        }
        if (fuzzy_provider_project_) {
            draft_.project_default_provider = pick.provider_id;
        } else {
            draft_.provider = pick.provider_id;
            if (idx < draft_.provider_models.size()) {
                draft_.model = model_display_text(draft_.provider_models[idx]);
            }
        }
        refresh_model_catalog_hints();
        rebuild_rows();
        return;
    }
    draft_.provider = pick.provider_id;
    draft_.model = pick.model;
    refresh_model_catalog_hints();
    rebuild_rows();
}

void SettingsModal::cycle_provider_row(int row, int direction) {
    if (row < 0 || row >= static_cast<int>(rows_.size())) {
        return;
    }
    const Row& r = rows_[row];
    if (r.kind != Row::Kind::ProviderCycle || draft_.provider_options.empty()) {
        return;
    }
    const int dir = direction >= 0 ? 1 : -1;
    const std::string current =
        r.enum_key == "project_provider" ? draft_.project_default_provider : provider_display_id();
    std::size_t idx = 0;
    for (std::size_t i = 0; i < draft_.provider_options.size(); ++i) {
        if (draft_.provider_options[i] == current) {
            idx = i;
            break;
        }
    }
    const int n = static_cast<int>(draft_.provider_options.size());
    const int next = (static_cast<int>(idx) + dir + n) % n;
    const std::string picked = draft_.provider_options[static_cast<std::size_t>(next)];
    if (r.enum_key == "project_provider") {
        draft_.project_default_provider = picked;
    } else {
        draft_.provider = picked;
        if (next < static_cast<int>(draft_.provider_models.size())) {
            draft_.model = model_display_text(draft_.provider_models[static_cast<std::size_t>(next)]);
        }
    }
    refresh_model_catalog_hints();
    rebuild_rows();
    note_dirty();
}

void SettingsModal::open_provider_fuzzy(bool project_scope) {
    fuzzy_picker_mode_ = FuzzyPickerMode::Provider;
    fuzzy_provider_project_ = project_scope;
    fuzzy_entries_.clear();
    const std::string current =
        project_scope ? draft_.project_default_provider : provider_display_id();
    for (std::size_t i = 0; i < draft_.provider_options.size(); ++i) {
        const std::string& id = draft_.provider_options[i];
        std::string label = id;
        if (i < draft_.provider_labels.size() && !draft_.provider_labels[i].empty()) {
            label = draft_.provider_labels[i];
        }
        FuzzyEntry entry;
        entry.provider_id = id;
        entry.line = label;
        entry.meta = id + " " + label;
        fuzzy_entries_.push_back(std::move(entry));
    }
    fuzzy_query_.clear();
    fuzzy_list_scroll_ = 0;
    rebuild_fuzzy_filter();
    fuzzy_cursor_ = 0;
    for (int row = 0; row < static_cast<int>(fuzzy_filtered_.size()); ++row) {
        const FuzzyEntry& entry = fuzzy_entries_[static_cast<std::size_t>(fuzzy_filtered_[row])];
        if (entry.provider_id == current) {
            fuzzy_cursor_ = row;
            break;
        }
    }
    sync_fuzzy_viewport_to_cursor();
    fuzzy_open_ = true;
    mark_dirty();
}

void SettingsModal::open_model_detail(const std::string& model_id) {
    model_detail_lines_.clear();
    model_detail_scroll_ = 0;
    model_detail_open_ = false;
    if (handle_ == 0 || model_id.empty()) {
        return;
    }
    const std::string provider = provider_display_id();
    char* raw = taiji_provider_model_details(handle_, const_cast<char*>(provider.c_str()),
                                             const_cast<char*>(model_id.c_str()));
    if (raw != nullptr) {
        model_detail_lines_ = parse_taiji_model_detail_lines(raw);
        std::free(raw);
    }
    if (model_detail_lines_.empty()) {
        model_detail_lines_.push_back("No details available.");
    }
    model_detail_open_ = true;
    mark_dirty();
}

bool SettingsModal::handle_model_detail_event(const tuinator::Event& event) {
    if (!model_detail_open_) {
        return false;
    }
    const PanelRect panel = panel_rect();
    const int body_rows = model_detail_overlay_layout(panel).body_rows;

    if (const auto* key = std::get_if<tuinator::KeyPress>(&event)) {
        if (key->key == tuinator::Key::Escape) {
            model_detail_open_ = false;
            mark_dirty();
            return true;
        }
        if (key->key == tuinator::Key::Up) {
            model_detail_scroll_ = std::max(0, model_detail_scroll_ - 1);
            mark_dirty();
            return true;
        }
        if (key->key == tuinator::Key::Down) {
            const int max_scroll =
                std::max(0, static_cast<int>(model_detail_lines_.size()) - body_rows);
            model_detail_scroll_ = std::min(max_scroll, model_detail_scroll_ + 1);
            mark_dirty();
            return true;
        }
        return true;
    }
    if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
        if (mouse->action == tuinator::MouseAction::WheelUp || mouse->action == tuinator::MouseAction::WheelDown) {
            const int step = 2;
            const int delta = mouse->action == tuinator::MouseAction::WheelUp ? -step : step;
            const int max_scroll =
                std::max(0, static_cast<int>(model_detail_lines_.size()) - body_rows);
            model_detail_scroll_ = std::max(0, std::min(max_scroll, model_detail_scroll_ + delta));
            mark_dirty();
            return true;
        }
        if (mouse->action == tuinator::MouseAction::Click || mouse->action == tuinator::MouseAction::Release) {
            model_detail_open_ = false;
            mark_dirty();
            return true;
        }
        return true;
    }
    return true;
}

bool SettingsModal::hit_model_row_help(tuinator::Point local) const {
    int row = 0;
    if (!hit_content_row(local, row) || row < 0 || row >= static_cast<int>(rows_.size()) ||
        rows_[static_cast<std::size_t>(row)].enum_key != "model") {
        return false;
    }
    const PanelRect panel = panel_rect();
    return local.x >= panel.x + panel.w - 6 && local.x < panel.x + panel.w - 1;
}

void SettingsModal::paint_model_detail_overlay(tuinator::Canvas& clipped, const PanelRect& panel) const {
    if (!model_detail_open_) {
        return;
    }
    const ModelDetailOverlayLayout layout = model_detail_overlay_layout(panel);
    const int dx = layout.x;
    const int dy = layout.y;
    const int dw = layout.w;
    const int dh = layout.h;
    const tuinator::Theme& theme = ui_app_theme();
    tuinator::Style border = theme.muted;
    border.foreground_rgb = ui_border_chrome();
    border.bold = true;
    tuinator::Style bg = ui_overlay_panel_style();
    clipped.fill_rect({dx, dy, dw, dh}, ' ', bg);
    clipped.draw_box({dx, dy, dw, dh}, border, theme.glyphs);
    tuinator::Style title = theme.heading;
    title.bold = true;
    title.background_rgb = bg.background_rgb;
    clipped.draw_text({dx + 2, dy}, " Model details ", title);
    const int body_y = dy + 2;
    for (int row = 0; row < layout.body_rows; ++row) {
        const int idx = model_detail_scroll_ + row;
        if (idx >= static_cast<int>(model_detail_lines_.size())) {
            break;
        }
        std::string line = model_detail_lines_[static_cast<std::size_t>(idx)];
        if (static_cast<int>(line.size()) > dw - 4) {
            line = line.substr(0, static_cast<std::size_t>(dw - 5)) + "…";
        }
        tuinator::Style line_style = theme.label;
        line_style.background_rgb = bg.background_rgb;
        if (row == 0) {
            line_style.bold = true;
        }
        clipped.draw_text({dx + 2, body_y + row}, line, line_style);
    }
    tuinator::Style hint = theme.muted;
    hint.dim = true;
    hint.background_rgb = bg.background_rgb;
    clipped.draw_text({dx + 2, dy + dh - 2}, " Esc: close  ↑↓: scroll ", hint);
}

void SettingsModal::open_theme_fuzzy() {
    fuzzy_picker_mode_ = FuzzyPickerMode::Theme;
    fuzzy_theme_revert_id_ = draft_.ui_theme;
    fuzzy_entries_.clear();
    const std::vector<std::string> ids = ui_list_theme_ids();
    for (const std::string& id : ids) {
        FuzzyEntry entry;
        entry.model = id;
        if (ui_is_bundled_theme(id)) {
            entry.line = id + "  ·  " + ui_theme_display_name(id);
        } else {
            entry.line = id + "  ·  user";
        }
        entry.meta = id + " " + ui_theme_display_name(id) + " theme color palette stylesheet";
        fuzzy_entries_.push_back(std::move(entry));
    }
    if (std::find(ids.begin(), ids.end(), draft_.ui_theme) == ids.end() && !draft_.ui_theme.empty()) {
        FuzzyEntry entry;
        entry.model = draft_.ui_theme;
        entry.line = draft_.ui_theme + "  ·  (missing file)";
        entry.meta = draft_.ui_theme;
        fuzzy_entries_.push_back(std::move(entry));
    }
    fuzzy_query_.clear();
    fuzzy_list_scroll_ = 0;
    rebuild_fuzzy_filter();
    fuzzy_cursor_ = 0;
    for (int row = 0; row < static_cast<int>(fuzzy_filtered_.size()); ++row) {
        const FuzzyEntry& entry = fuzzy_entries_[static_cast<std::size_t>(fuzzy_filtered_[row])];
        if (entry.model == draft_.ui_theme) {
            fuzzy_cursor_ = row;
            break;
        }
    }
    sync_fuzzy_viewport_to_cursor();
    fuzzy_open_ = true;
    mark_dirty();
}

void SettingsModal::open_model_fuzzy() {
    fuzzy_picker_mode_ = FuzzyPickerMode::Model;
    fuzzy_entries_.clear();
    const std::string active_provider = provider_display_id();

    auto push_model = [&](const std::string& model_id, const std::string& label) {
        if (model_id.empty()) {
            return;
        }
        for (const FuzzyEntry& existing : fuzzy_entries_) {
            if (existing.model == model_id) {
                return;
            }
        }
        FuzzyEntry entry;
        const std::string display = model_display_text(model_id);
        entry.provider_id = active_provider;
        entry.model = model_id;
        entry.line = label.empty() ? display : label;
        entry.meta = model_id + " " + label + " " + display;
        fuzzy_entries_.push_back(std::move(entry));
    };

    if (handle_ != 0 && !active_provider.empty()) {
        char* raw = taiji_provider_models(handle_, const_cast<char*>(active_provider.c_str()));
        if (raw != nullptr) {
            const std::vector<ProviderModelPick> picks = parse_taiji_provider_models_response(raw);
            std::free(raw);
            for (const ProviderModelPick& pick : picks) {
                std::string line = pick.name;
                if (line != pick.id && line != model_display_text(pick.id)) {
                    line = pick.name + "  ·  " + model_display_text(pick.id);
                } else {
                    line = model_display_text(pick.id);
                }
                push_model(pick.id, line);
            }
        }
    }

    if (fuzzy_entries_.empty()) {
        if (!draft_.model.empty()) {
            push_model(draft_.model, "");
        }
        for (std::size_t i = 0; i < draft_.provider_options.size(); ++i) {
            if (draft_.provider_options[i] != active_provider) {
                continue;
            }
            std::string model;
            if (i < draft_.provider_models.size()) {
                model = draft_.provider_models[i];
            }
            if (model.empty()) {
                model = draft_.model_default;
            }
            push_model(model, "");
            break;
        }
    } else if (!draft_.model.empty()) {
        push_model(draft_.model, "");
    }
    fuzzy_query_.clear();
    fuzzy_list_scroll_ = 0;
    rebuild_fuzzy_filter();
    std::string current_model = draft_.model;
    if (current_model.empty()) {
        current_model = draft_.model_default;
    }
    fuzzy_cursor_ = 0;
    for (int row = 0; row < static_cast<int>(fuzzy_filtered_.size()); ++row) {
        const FuzzyEntry& entry = fuzzy_entries_[static_cast<std::size_t>(fuzzy_filtered_[row])];
        if (fuzzy_models_match(entry.model, current_model)) {
            fuzzy_cursor_ = row;
            break;
        }
    }
    sync_fuzzy_viewport_to_cursor();
    fuzzy_open_ = true;
    mark_dirty();
}

void SettingsModal::paint_fuzzy_overlay(tuinator::Canvas& clipped, const PanelRect& panel) const {
    if (!fuzzy_open_) {
        return;
    }
    const FuzzyOverlayLayout layout = fuzzy_overlay_layout(panel);
    const int w = layout.w;
    const int h = layout.h;
    const int x = layout.x;
    const int y = layout.y;
    const tuinator::Theme& theme = ui_app_theme();
    tuinator::Style border = theme.muted;
    border.foreground_rgb = ui_border_chrome();
    tuinator::Style bg = ui_overlay_panel_style();
    clipped.fill_rect({x, y, w, h}, ' ', bg);
    clipped.draw_box({x, y, w, h}, border, theme.glyphs);
    tuinator::Style title = theme.heading;
    title.bold = true;
    std::string title_text = " Select Model ";
    if (fuzzy_picker_mode_ == FuzzyPickerMode::Provider) {
        title_text = " Select Provider ";
    } else if (fuzzy_picker_mode_ == FuzzyPickerMode::Theme) {
        title_text = " Select Color Theme ";
    } else if (fuzzy_picker_mode_ == FuzzyPickerMode::Model) {
        title_text = " Model · " + provider_display_label() + " ";
    }
    if (static_cast<int>(title_text.size()) > w - 4) {
        title_text = title_text.substr(0, static_cast<std::size_t>(w - 4));
    }
    clipped.draw_text({x + 2, y}, title_text, title);
    const std::string query_line = "> " + fuzzy_query_ + "_";
    clipped.draw_text({x + 2, y + 2}, query_line.substr(0, static_cast<std::size_t>(w - 4)), theme.label);
    const int list_y = layout.list_y;
    const int max_rows = layout.max_rows;
    if (fuzzy_filtered_.empty()) {
        tuinator::Style empty_style = theme.label;
        empty_style.dim = true;
        const char* msg = " (no matches)";
        if (fuzzy_entries_.empty()) {
            if (fuzzy_picker_mode_ == FuzzyPickerMode::Provider) {
                msg = " (no providers — run: taiji providers refresh)";
            } else if (fuzzy_picker_mode_ == FuzzyPickerMode::Theme) {
                msg = " (no themes — run: taiji theme init)";
            } else {
                msg = " (no models)";
            }
        }
        clipped.draw_text({x + 2, list_y}, msg, empty_style);
    }
    for (int row = 0; row < max_rows; ++row) {
        const int idx = fuzzy_list_scroll_ + row;
        if (idx >= static_cast<int>(fuzzy_filtered_.size())) {
            break;
        }
        const FuzzyEntry& entry = fuzzy_entries_[static_cast<std::size_t>(fuzzy_filtered_[idx])];
        tuinator::Style line_style = theme.label;
        if (idx == fuzzy_cursor_) {
            line_style = ui_list_selection_style(true);
        }
        std::string prefix = idx == fuzzy_cursor_ ? "> " : "  ";
        std::string line = prefix + entry.line;
        if (static_cast<int>(line.size()) > w - 4) {
            line = line.substr(0, static_cast<std::size_t>(w - 5)) + "…";
        }
        clipped.draw_text({x + 2, list_y + row}, line, line_style);
    }
    if (static_cast<int>(fuzzy_filtered_.size()) > max_rows) {
        tuinator::Style scroll_hint = theme.label;
        scroll_hint.dim = true;
        const std::string hint = std::to_string(fuzzy_cursor_ + 1) + "/" +
                                 std::to_string(fuzzy_filtered_.size()) + "  ↑↓";
        clipped.draw_text({x + w - static_cast<int>(hint.size()) - 2, y + h - 2}, hint, scroll_hint);
    }
}

bool SettingsModal::handle_fuzzy_event(const tuinator::Event& event) {
    if (!fuzzy_open_) {
        return false;
    }
    const PanelRect panel = panel_rect();
    const FuzzyOverlayLayout layout = fuzzy_overlay_layout(panel);
    const int fuzzy_visible_rows = layout.max_rows;

    if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
        int panel_x = 0;
        int panel_y = 0;
        if (!fuzzy_overlay_panel_point(to_local(mouse->position), panel, panel_x, panel_y)) {
            return true;
        }
        const bool in_overlay = panel_x >= layout.x && panel_x < layout.x + layout.w && panel_y >= layout.y &&
                                panel_y < layout.y + layout.h;
        if (mouse->action == tuinator::MouseAction::WheelUp || mouse->action == tuinator::MouseAction::WheelDown) {
            if (!in_overlay || fuzzy_filtered_.empty()) {
                return true;
            }
            const int step = 3;
            const int delta = mouse->action == tuinator::MouseAction::WheelUp ? -step : step;
            const int max_scroll = std::max(0, static_cast<int>(fuzzy_filtered_.size()) - fuzzy_visible_rows);
            fuzzy_list_scroll_ = std::max(0, std::min(max_scroll, fuzzy_list_scroll_ + delta));
            mark_dirty();
            return true;
        }
        if (in_overlay && panel_y >= layout.list_y && panel_y < layout.list_y + fuzzy_visible_rows &&
            (mouse->action == tuinator::MouseAction::Click || mouse->action == tuinator::MouseAction::Release)) {
            const int row = panel_y - layout.list_y;
            const int idx = fuzzy_list_scroll_ + row;
            if (idx >= 0 && idx < static_cast<int>(fuzzy_filtered_.size())) {
                fuzzy_cursor_ = idx;
                sync_fuzzy_list_scroll(fuzzy_visible_rows);
                preview_theme_fuzzy_selection();
                if (mouse->action == tuinator::MouseAction::Release &&
                    mouse->button == tuinator::MouseButton::Left) {
                    const FuzzyEntry& pick = fuzzy_entries_[static_cast<std::size_t>(fuzzy_filtered_[idx])];
                    apply_fuzzy_pick(pick);
                    note_dirty();
                    fuzzy_open_ = false;
                }
                mark_dirty();
            }
            return true;
        }
        return true;
    }

    if (const auto* paste = std::get_if<tuinator::ClipboardPaste>(&event)) {
        if (!paste->text.empty()) {
            fuzzy_query_ += paste->text;
            rebuild_fuzzy_filter();
            fuzzy_cursor_ = 0;
            fuzzy_list_scroll_ = 0;
            mark_dirty();
        }
        return true;
    }
    const auto* key = std::get_if<tuinator::KeyPress>(&event);
    if (key != nullptr) {
        if (key->key == tuinator::Key::Escape) {
            if (fuzzy_picker_mode_ == FuzzyPickerMode::Theme) {
                draft_.ui_theme = fuzzy_theme_revert_id_;
                if (on_preview_) {
                    on_preview_(draft_);
                }
            }
            fuzzy_open_ = false;
            mark_dirty();
            return true;
        }
        if (key->key == tuinator::Key::Enter) {
            if (!fuzzy_filtered_.empty() && fuzzy_cursor_ < static_cast<int>(fuzzy_filtered_.size())) {
                const FuzzyEntry& pick = fuzzy_entries_[static_cast<std::size_t>(fuzzy_filtered_[fuzzy_cursor_])];
                apply_fuzzy_pick(pick);
                note_dirty();
            }
            fuzzy_open_ = false;
            mark_dirty();
            return true;
        }
        if (key->key == tuinator::Key::Up) {
            fuzzy_cursor_ = std::max(0, fuzzy_cursor_ - 1);
            sync_fuzzy_list_scroll(fuzzy_visible_rows);
            preview_theme_fuzzy_selection();
            mark_dirty();
            return true;
        }
        if (key->key == tuinator::Key::Down) {
            if (!fuzzy_filtered_.empty()) {
                fuzzy_cursor_ = std::min(static_cast<int>(fuzzy_filtered_.size()) - 1, fuzzy_cursor_ + 1);
                sync_fuzzy_list_scroll(fuzzy_visible_rows);
                preview_theme_fuzzy_selection();
                mark_dirty();
            }
            return true;
        }
        if (key->key == tuinator::Key::PageUp) {
            fuzzy_cursor_ = std::max(0, fuzzy_cursor_ - fuzzy_visible_rows);
            sync_fuzzy_list_scroll(fuzzy_visible_rows);
            preview_theme_fuzzy_selection();
            mark_dirty();
            return true;
        }
        if (key->key == tuinator::Key::PageDown) {
            if (!fuzzy_filtered_.empty()) {
                fuzzy_cursor_ =
                    std::min(static_cast<int>(fuzzy_filtered_.size()) - 1, fuzzy_cursor_ + fuzzy_visible_rows);
                sync_fuzzy_list_scroll(fuzzy_visible_rows);
                preview_theme_fuzzy_selection();
                mark_dirty();
            }
            return true;
        }
        if (key->key == tuinator::Key::Home) {
            fuzzy_cursor_ = 0;
            sync_fuzzy_list_scroll(fuzzy_visible_rows);
            preview_theme_fuzzy_selection();
            mark_dirty();
            return true;
        }
        if (key->key == tuinator::Key::End) {
            if (!fuzzy_filtered_.empty()) {
                fuzzy_cursor_ = static_cast<int>(fuzzy_filtered_.size()) - 1;
                sync_fuzzy_list_scroll(fuzzy_visible_rows);
                preview_theme_fuzzy_selection();
                mark_dirty();
            }
            return true;
        }
        if (key->key == tuinator::Key::Backspace) {
            if (!fuzzy_query_.empty()) {
                fuzzy_query_.pop_back();
                rebuild_fuzzy_filter();
                fuzzy_cursor_ = 0;
                fuzzy_list_scroll_ = 0;
                mark_dirty();
            }
            return true;
        }
        if (tuinator::allows_text_insert_modifiers(*key)) {
            const std::string insert = tuinator::keypress_insert_text(*key);
            if (!insert.empty()) {
                fuzzy_query_ += insert;
                rebuild_fuzzy_filter();
                fuzzy_cursor_ = 0;
                fuzzy_list_scroll_ = 0;
                mark_dirty();
                return true;
            }
        }
        return true;
    }
    return true;
}
