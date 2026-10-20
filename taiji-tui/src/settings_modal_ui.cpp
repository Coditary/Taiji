#include "settings_modal.hpp"

#include "ui_theme.hpp"

#include <libtaiji.h>

#include <algorithm>
#include <vector>
#include <cmath>
#include <iomanip>
#include <sstream>

namespace {

std::string clip(std::string text, int max_cols) {
    if (max_cols <= 0) {
        return "";
    }
    if (static_cast<int>(text.size()) <= max_cols) {
        return text;
    }
    if (max_cols <= 1) {
        return text.substr(0, static_cast<std::size_t>(max_cols));
    }
    return text.substr(0, static_cast<std::size_t>(max_cols - 1)) + "…";
}

std::string format_float(double value) {
    std::ostringstream out;
    out << std::fixed << std::setprecision(1) << value;
    std::string s = out.str();
    while (s.size() > 1 && s.back() == '0' && s[s.size() - 2] != '.') {
        s.pop_back();
    }
    if (!s.empty() && s.back() == '.') {
        s.pop_back();
    }
    return s;
}

std::string cot_label(const std::string& mode) {
    if (mode == "full") {
        return "native";
    }
    return mode.empty() ? "off" : mode;
}

struct PercentBarLayout {
    int bar_x = 0;
    int bar_w = 0;
    int pct_x = 0;
};

PercentBarLayout percent_bar_layout(int value_x, int value_w, int pct) {
    PercentBarLayout out;
    const std::string pct_text = std::to_string(pct) + "%";
    const int pct_w = tuinator::text_display_width(pct_text);
    const int gap = 1;
    out.bar_x = value_x;
    out.bar_w = std::max(12, value_w - pct_w - gap);
    out.pct_x = value_x + out.bar_w + gap;
    return out;
}

int percent_bar_value_from_x(int click_x, const PercentBarLayout& layout, int int_min, int int_max) {
    if (layout.bar_w <= 0) {
        return int_min;
    }
    const int rel = std::clamp(click_x - layout.bar_x, 0, layout.bar_w - 1);
    const int span = std::max(1, int_max - int_min);
    const int raw = int_min + rel * span / std::max(1, layout.bar_w - 1);
    const int step = 5;
    const int snapped = int_min + ((raw - int_min + step / 2) / step) * step;
    return std::clamp(snapped, int_min, int_max);
}

std::string normalize_metrics(const std::string& preset) {
    if (preset == "minimal" || preset == "tokens") {
        return "compact";
    }
    if (preset == "verbose") {
        return "dev";
    }
    if (preset == "compact" || preset == "dev" || preset == "off") {
        return preset;
    }
    return "dev";
}

} // namespace

void SettingsModal::capture_saved_snapshot() {
    saved_snapshot_ = build_apply_json();
    dirty_ = false;
}

void SettingsModal::note_dirty() {
    dirty_ = !saved_snapshot_.empty() && build_apply_json() != saved_snapshot_;
    mark_dirty();
}

bool SettingsModal::is_dirty() const {
    return dirty_;
}

void SettingsModal::reload_from_backend() {
    if (handle_ == 0) {
        return;
    }
    char scope_buf[] = "chat";
    char* raw = taiji_settings_get(handle_, scope_buf);
    if (raw == nullptr) {
        return;
    }
    const std::string json = raw;
    std::free(raw);
    settings_parse_draft(json, draft_);
    draft_.turn_metrics_preset = normalize_metrics(draft_.turn_metrics_preset);
    if (chat_only_) {
        draft_.scope = "chat";
    }
    refresh_model_catalog_hints();
    rebuild_rows();
}

void SettingsModal::apply_draft_to_backend() {
    if (handle_ == 0) {
        return;
    }
    const std::string json = build_apply_json();
    std::vector<char> buf(json.begin(), json.end());
    buf.push_back('\0');
    taiji_settings_apply(handle_, buf.data());
    capture_saved_snapshot();
}

void SettingsModal::persist_chat_ui_theme(const std::string& theme_id) {
    if (handle_ == 0 || theme_id.empty()) {
        return;
    }
    reload_from_backend();
    draft_.ui_theme = theme_id;
    apply_draft_to_backend();
}

void SettingsModal::open() {
    open_ = true;
    chat_only_ = false;
    focus_nav_ = true;
    selected_row_ = 0;
    content_scroll_ = 0;
    percent_bar_drag_row_ = -1;
    hover_row_ = -1;
    pointer_valid_ = false;
    editing_text_ = false;
    category_ = Category::General;
    reload_from_backend();
    rebuild_rows();
    selected_row_ = first_interactive_row();
    sync_content_scroll();
    capture_saved_snapshot();
    mark_layout_dirty();
}

void SettingsModal::close(bool applied) {
    if (!open_) {
        return;
    }
    open_ = false;
    percent_bar_drag_row_ = -1;
    hover_row_ = -1;
    pointer_valid_ = false;
    fuzzy_open_ = false;
    editing_text_ = false;
    dirty_ = false;
    if (on_closed_) {
        on_closed_(applied);
    }
    mark_layout_dirty();
}

void SettingsModal::set_category(Category cat) {
    category_ = cat;
    selected_row_ = 0;
    content_scroll_ = 0;
    hover_row_ = -1;
    editing_text_ = false;
    rebuild_rows();
    selected_row_ = first_interactive_row();
    sync_content_scroll();
    note_dirty();
}

std::string SettingsModal::category_title(Category cat, bool short_nav) const {
    switch (cat) {
        case Category::General:
            return short_nav ? "General" : "General & Chat";
        case Category::Model:
            return "Model";
        case Category::AgentBehaviour:
            return short_nav ? "Agent" : "Agent & Behaviour";
        case Category::ToolsSkills:
            return short_nav ? "Tools" : "Tools & Skills";
        case Category::Audio:
            return short_nav ? "Audio" : "Audio & Speech";
        case Category::Appearance:
            return "Appearance";
        case Category::Shortcuts:
            return short_nav ? "Keys" : "Keyboard Shortcuts";
        default:
            return "";
    }
}

std::string SettingsModal::category_subtitle() const {
    return category_title(category_, false);
}

void SettingsModal::rebuild_rows() {
    sync_context_from_catalog();
    rows_ = rows_for_category();
    if (selected_row_ >= static_cast<int>(rows_.size())) {
        selected_row_ = std::max(0, static_cast<int>(rows_.size()) - 1);
    }
    sync_content_scroll();
}

bool SettingsModal::row_is_interactive(int row) const {
    if (row < 0 || row >= static_cast<int>(rows_.size())) {
        return false;
    }
    const Row::Kind kind = rows_[row].kind;
    return kind != Row::Kind::Section && kind != Row::Kind::Gap;
}

int SettingsModal::first_interactive_row() const {
    for (int i = 0; i < static_cast<int>(rows_.size()); ++i) {
        if (row_is_interactive(i)) {
            return i;
        }
    }
    return 0;
}

int SettingsModal::content_visible_row_count() const {
    const PanelRect panel = panel_rect();
    const int sep_y = footer_sep_y(panel);
    const int hint_reserve = 1;
    return std::max(0, sep_y - content_row_start_y() - hint_reserve);
}

void SettingsModal::sync_content_scroll() {
    const int visible = content_visible_row_count();
    if (visible <= 0) {
        content_scroll_ = 0;
        return;
    }
    const int total = static_cast<int>(rows_.size());
    const int max_scroll = std::max(0, total - visible);
    if (selected_row_ < content_scroll_) {
        content_scroll_ = selected_row_;
    }
    if (selected_row_ >= content_scroll_ + visible) {
        content_scroll_ = selected_row_ - visible + 1;
    }
    if (content_scroll_ > max_scroll) {
        content_scroll_ = max_scroll;
    }
    if (content_scroll_ < 0) {
        content_scroll_ = 0;
    }
}

void SettingsModal::sync_pointer_from_screen(tuinator::Point root_point) {
    if (!open_) {
        return;
    }
    const tuinator::Point local{root_point.x - bounds_.x, root_point.y - bounds_.y};
    const bool moved =
        !pointer_valid_ || pointer_local_.x != local.x || pointer_local_.y != local.y;
    pointer_local_ = local;
    pointer_valid_ = true;
    int row = -1;
    const int next_hover = hit_content_row_for_hint(local, row) ? row : -1;
    if (moved || next_hover != hover_row_) {
        hover_row_ = next_hover;
        mark_dirty();
    }
}

void SettingsModal::update_percent_bar_from_local(tuinator::Point local, int row) {
    if (row < 0 || row >= static_cast<int>(rows_.size())) {
        return;
    }
    const Row& r = rows_[static_cast<std::size_t>(row)];
    if (r.kind != Row::Kind::PercentBar) {
        return;
    }
    const PanelRect panel = panel_rect();
    const int value_x = nav_width() + 2 + key_col_width();
    const int value_w = std::max(0, panel.w - value_x - 2);
    int pct = 80;
    if (r.enum_key == "user_message_max_width_pct") {
        pct = draft_.user_message_max_width_pct;
    } else if (r.enum_key == "assistant_message_max_width_pct") {
        pct = draft_.assistant_message_max_width_pct;
    }
    const PercentBarLayout layout = percent_bar_layout(value_x, value_w, pct);
    const int click_x = local.x - panel.x;
    const int next = percent_bar_value_from_x(click_x, layout, r.int_min, r.int_max);
    if (r.enum_key == "user_message_max_width_pct") {
        if (draft_.user_message_max_width_pct == next) {
            note_dirty();
            return;
        }
        draft_.user_message_max_width_pct = next;
    } else if (r.enum_key == "assistant_message_max_width_pct") {
        if (draft_.assistant_message_max_width_pct == next) {
            note_dirty();
            return;
        }
        draft_.assistant_message_max_width_pct = next;
    } else {
        return;
    }
    if (on_preview_) {
        on_preview_(draft_);
    }
    note_dirty();
}

void SettingsModal::move_selected_row(int delta) {
    if (rows_.empty()) {
        return;
    }
    int row = selected_row_;
    for (int guard = 0; guard < static_cast<int>(rows_.size()) + 2; ++guard) {
        row += delta;
        if (row < 0) {
            row = static_cast<int>(rows_.size()) - 1;
        }
        if (row >= static_cast<int>(rows_.size())) {
            row = 0;
        }
        if (row_is_interactive(row)) {
            selected_row_ = row;
            sync_content_scroll();
            return;
        }
    }
}

std::vector<SettingsModal::Row> SettingsModal::rows_for_category() const {
    std::vector<Row> rows;

    if (!chat_only_ && draft_.scope == "project") {
        Row provider;
        provider.label = "Default Provider";
        provider.kind = Row::Kind::ProviderCycle;
        provider.enum_key = "project_provider";
        rows.push_back(provider);
        return rows;
    }

    switch (category_) {
        case Category::General: {
            if (draft_.scope == "chat") {
                Row name;
                name.label = "Chat Name";
                name.kind = Row::Kind::Text;
                name.enum_key = "session_name";
                rows.push_back(name);
            }
            Row combo;
            combo.label = "Preset / Template";
            combo.kind = Row::Kind::PresetCombo;
            combo.enum_key = "preset_combo";
            for (const TemplateOption& t : draft_.templates) {
                combo.enum_values.push_back("template:" + t.id);
            }
            combo.enum_values.push_back("preset:");
            combo.enum_values.push_back("preset:markdown");
            combo.enum_values.push_back("preset:whatsapp");
            combo.enum_values.push_back("preset:story");
            combo.enum_values.push_back("preset:minimal");
            for (const char* p : {"markdown", "whatsapp", "bubbles", "story", "narrative", "minimal"}) {
                combo.enum_values.push_back(std::string("presentation:") + p);
            }
            rows.push_back(combo);
            break;
        }
        case Category::Model: {
            Row provider;
            provider.label = "Provider";
            provider.kind = Row::Kind::ProviderCycle;
            provider.enum_key = "provider";
            rows.push_back(provider);
            Row model;
            model.label = "Model";
            model.kind = Row::Kind::Text;
            model.enum_key = "model";
            rows.push_back(model);
            if (model_setting_show_temperature()) {
                Row temp;
                temp.label = "Temperature";
                temp.kind = Row::Kind::FloatValue;
                temp.enum_key = "temperature";
                rows.push_back(temp);
            }
            if (model_setting_show_max_tokens()) {
                Row tokens;
                tokens.label = "Max Tokens";
                tokens.kind = Row::Kind::IntValue;
                tokens.enum_key = "max_tokens";
                tokens.int_min = 0;
                tokens.int_max = 200000;
                rows.push_back(tokens);
            }
            if (model_setting_show_context_limit()) {
                Row ctx;
                ctx.label = "Context Limit";
                ctx.enum_key = "context_window";
                if (model_context_limit_fixed()) {
                    ctx.kind = Row::Kind::FixedValue;
                } else {
                    ctx.kind = Row::Kind::IntValue;
                    ctx.int_min = 1024;
                    ctx.int_max = 2'000'000;
                }
                rows.push_back(ctx);
            }
            break;
        }
        case Category::AgentBehaviour: {
            Row sec1;
            sec1.kind = Row::Kind::Section;
            sec1.label = "Execution";
            rows.push_back(sec1);
            Row steps;
            steps.label = "Max Steps";
            steps.kind = Row::Kind::IntValue;
            steps.enum_key = "max_steps";
            steps.int_min = 1;
            steps.int_max = 200;
            rows.push_back(steps);
            Row dec;
            dec.label = "Decisions";
            dec.kind = Row::Kind::BoolToggle;
            dec.enum_key = "decisions_enabled";
            rows.push_back(dec);
            Row mem;
            mem.label = "Memory";
            mem.kind = Row::Kind::BoolToggle;
            mem.enum_key = "memory_enabled";
            rows.push_back(mem);
            Row gap1;
            gap1.kind = Row::Kind::Gap;
            rows.push_back(gap1);
            Row sec2;
            sec2.kind = Row::Kind::Section;
            sec2.label = "Thinking";
            rows.push_back(sec2);
            Row cot;
            cot.label = "Chain of Thought";
            cot.kind = Row::Kind::EnumCycle;
            cot.enum_key = "chain_of_thought";
            cot.enum_values = {"off", "prompt", "full"};
            rows.push_back(cot);
            if (draft_.chain_of_thought != "off") {
                Row chars;
                chars.label = "Thought Max Chars";
                chars.kind = Row::Kind::IntValue;
                chars.enum_key = "thought_max_chars";
                chars.int_min = 0;
                chars.int_max = 50000;
                rows.push_back(chars);
            }
            Row gap2;
            gap2.kind = Row::Kind::Gap;
            rows.push_back(gap2);
            Row sec3;
            sec3.kind = Row::Kind::Section;
            sec3.label = "Prompt";
            rows.push_back(sec3);
            Row extra;
            extra.label = "System Prompt Extra";
            extra.kind = Row::Kind::Text;
            extra.enum_key = "system_prompt_extra";
            rows.push_back(extra);
            break;
        }
        case Category::ToolsSkills: {
            Row sec_mcp;
            sec_mcp.kind = Row::Kind::Section;
            sec_mcp.label = "MCP Servers";
            rows.push_back(sec_mcp);
            for (std::size_t i = 0; i < draft_.mcp_available.size(); ++i) {
                Row row;
                row.label = draft_.mcp_available[i];
                row.kind = Row::Kind::MCPToggle;
                row.mcp_index = static_cast<int>(i);
                rows.push_back(row);
            }
            if (draft_.mcp_available.empty()) {
                Row hint;
                hint.label = "(none connected)";
                hint.kind = Row::Kind::Text;
                hint.enum_key = "none";
                rows.push_back(hint);
            }
            Row gap_sk;
            gap_sk.kind = Row::Kind::Gap;
            rows.push_back(gap_sk);
            Row sec_sk;
            sec_sk.kind = Row::Kind::Section;
            sec_sk.label = "Skills";
            rows.push_back(sec_sk);
            for (std::size_t i = 0; i < draft_.skills_available.size(); ++i) {
                Row row;
                row.label = draft_.skills_available[i];
                row.kind = Row::Kind::SkillAllowToggle;
                row.skill_index = static_cast<int>(i);
                rows.push_back(row);
            }
            if (draft_.skills_available.empty()) {
                Row hint;
                hint.label = "(none loaded)";
                hint.kind = Row::Kind::Text;
                hint.enum_key = "none";
                rows.push_back(hint);
            }
            break;
        }
        case Category::Audio: {
            Row sec;
            sec.kind = Row::Kind::Section;
            sec.label = "Speech-to-text (Wuji)";
            rows.push_back(sec);
            Row enabled;
            enabled.label = "Voice input enabled";
            enabled.kind = Row::Kind::BoolToggle;
            enabled.enum_key = "stt_enabled";
            rows.push_back(enabled);
            Row driver;
            driver.label = "STT driver";
            driver.kind = Row::Kind::EnumCycle;
            driver.enum_key = "stt_driver";
            driver.enum_values = draft_.stt_driver_options;
            if (driver.enum_values.empty() && !draft_.stt_driver.empty()) {
                driver.enum_values.push_back(draft_.stt_driver);
            }
            rows.push_back(driver);
            Row model;
            model.label = "STT model";
            model.kind = Row::Kind::Text;
            model.enum_key = "stt_model";
            rows.push_back(model);
            Row lang;
            lang.label = "Language";
            lang.kind = Row::Kind::EnumCycle;
            lang.enum_key = "stt_language";
            lang.enum_values = {"auto", "de", "en", "fr", "es", "it", "nl", "pl", "pt", "ru", "ja", "zh"};
            rows.push_back(lang);
            Row quality;
            quality.label = "Transcription quality";
            quality.kind = Row::Kind::EnumCycle;
            quality.enum_key = "stt_quality";
            quality.enum_values = {"", "fast", "balanced", "careful"};
            rows.push_back(quality);
            Row trim;
            trim.label = "Trim silence";
            trim.kind = Row::Kind::BoolToggle;
            trim.enum_key = "stt_trim_silence";
            rows.push_back(trim);
            Row silence;
            silence.label = "Silence level";
            silence.kind = Row::Kind::FloatValue;
            silence.enum_key = "stt_silence_level";
            rows.push_back(silence);
            Row device;
            device.label = "Capture device";
            device.kind = Row::Kind::Text;
            device.enum_key = "stt_capture_device";
            rows.push_back(device);
            Row rate;
            rate.label = "Sample rate (Hz)";
            rate.kind = Row::Kind::IntValue;
            rate.enum_key = "stt_sample_rate";
            rate.int_min = 8000;
            rate.int_max = 48000;
            rows.push_back(rate);
            if (!draft_.stt_drivers_error.empty()) {
                Row err;
                err.label = draft_.stt_drivers_error;
                err.kind = Row::Kind::Text;
                err.enum_key = "none";
                rows.push_back(err);
            }
            break;
        }
        case Category::Appearance: {
            Row sec_theme;
            sec_theme.kind = Row::Kind::Section;
            sec_theme.label = "Theme";
            rows.push_back(sec_theme);
            Row theme;
            theme.label = "Color theme";
            theme.kind = Row::Kind::EnumCycle;
            theme.enum_key = "ui_theme";
            theme.enum_values = ui_list_theme_ids();
            if (!draft_.ui_theme.empty() &&
                std::find(theme.enum_values.begin(), theme.enum_values.end(), draft_.ui_theme) ==
                    theme.enum_values.end()) {
                theme.enum_values.push_back(draft_.ui_theme);
            }
            rows.push_back(theme);
            Row corners;
            corners.label = "Corner style";
            corners.kind = Row::Kind::EnumCycle;
            corners.enum_key = "ui_corners";
            corners.enum_values = {"square", "rounded", "flat"};
            rows.push_back(corners);

            Row gap0;
            gap0.kind = Row::Kind::Gap;
            rows.push_back(gap0);
            Row sec_agent;
            sec_agent.kind = Row::Kind::Section;
            sec_agent.label = "Agent & Chat";
            rows.push_back(sec_agent);
            Row tool_density;
            tool_density.label = "Tool call density";
            tool_density.kind = Row::Kind::EnumCycle;
            tool_density.enum_key = "tool_call_density";
            tool_density.enum_values = {"compact", "detailed"};
            rows.push_back(tool_density);
            Row metrics;
            metrics.label = "Turn metrics";
            metrics.kind = Row::Kind::EnumCycle;
            metrics.enum_key = "turn_metrics_preset";
            metrics.enum_values = {"off", "compact", "dev"};
            rows.push_back(metrics);
            Row turn_actions;
            turn_actions.label = "Turn actions";
            turn_actions.kind = Row::Kind::BoolToggle;
            turn_actions.enum_key = "turn_metrics_show_actions";
            rows.push_back(turn_actions);
            Row presentation;
            presentation.label = "Message style";
            presentation.kind = Row::Kind::EnumCycle;
            presentation.enum_key = "presentation";
            presentation.enum_values = {"markdown", "whatsapp", "story", "minimal", "bubbles"};
            rows.push_back(presentation);
            Row sec_you;
            sec_you.kind = Row::Kind::Section;
            sec_you.label = "Your messages";
            rows.push_back(sec_you);
            Row user_msg_style;
            user_msg_style.label = "Style";
            user_msg_style.kind = Row::Kind::EnumCycle;
            user_msg_style.enum_key = "user_message_style";
            user_msg_style.enum_values = {"chat",      "cornered", "block",   "plain",
                                          "header",    "terminal", "log",     "card",
                                          "section"};
            rows.push_back(user_msg_style);
            Row user_align;
            user_align.label = "Alignment";
            user_align.kind = Row::Kind::EnumCycle;
            user_align.enum_key = "user_message_align";
            user_align.enum_values = {"auto", "left", "right", "center"};
            rows.push_back(user_align);
            Row user_width;
            user_width.label = "Max width";
            user_width.kind = Row::Kind::PercentBar;
            user_width.enum_key = "user_message_max_width_pct";
            user_width.int_min = 40;
            user_width.int_max = 100;
            rows.push_back(user_width);
            Row user_fill;
            user_fill.label = "Background fill";
            user_fill.kind = Row::Kind::BoolToggle;
            user_fill.enum_key = "user_message_fill_background";
            rows.push_back(user_fill);
            Row sec_asst;
            sec_asst.kind = Row::Kind::Section;
            sec_asst.label = "Assistant messages";
            rows.push_back(sec_asst);
            Row asst_style;
            asst_style.label = "Style";
            asst_style.kind = Row::Kind::EnumCycle;
            asst_style.enum_key = "assistant_message_style";
            asst_style.enum_values = {"chat",      "cornered", "block",   "plain",
                                      "header",    "terminal", "log",     "card",
                                      "section"};
            rows.push_back(asst_style);
            Row asst_align;
            asst_align.label = "Alignment";
            asst_align.kind = Row::Kind::EnumCycle;
            asst_align.enum_key = "assistant_message_align";
            asst_align.enum_values = {"auto", "left", "right", "center"};
            rows.push_back(asst_align);
            Row asst_width;
            asst_width.label = "Max width";
            asst_width.kind = Row::Kind::PercentBar;
            asst_width.enum_key = "assistant_message_max_width_pct";
            asst_width.int_min = 40;
            asst_width.int_max = 100;
            rows.push_back(asst_width);
            Row asst_fill;
            asst_fill.label = "Background fill";
            asst_fill.kind = Row::Kind::BoolToggle;
            asst_fill.enum_key = "assistant_message_fill_background";
            rows.push_back(asst_fill);
            Row msg_ts;
            msg_ts.label = "Message timestamps";
            msg_ts.kind = Row::Kind::BoolToggle;
            msg_ts.enum_key = "message_timestamps";
            rows.push_back(msg_ts);
            Row sticky_user;
            sticky_user.label = "Sticky last prompt";
            sticky_user.kind = Row::Kind::BoolToggle;
            sticky_user.enum_key = "sticky_user_message";
            rows.push_back(sticky_user);
            Row auto_compact;
            auto_compact.label = "Auto-compact context";
            auto_compact.kind = Row::Kind::EnumCycle;
            auto_compact.enum_key = "auto_compact_context_pct";
            auto_compact.enum_values = {"off", "75", "80", "85", "90"};
            rows.push_back(auto_compact);
            if (draft_.message_timestamps) {
                Row msg_clock;
                msg_clock.label = "Time format";
                msg_clock.kind = Row::Kind::EnumCycle;
                msg_clock.enum_key = "message_time_clock";
                msg_clock.enum_values = {"24h", "12h"};
                rows.push_back(msg_clock);
            }
            Row chat_width;
            chat_width.label = "Chat width";
            chat_width.kind = Row::Kind::EnumCycle;
            chat_width.enum_key = "chat_width";
            chat_width.enum_values = {"auto", "narrow", "medium", "wide"};
            rows.push_back(chat_width);
            Row code_wrap;
            code_wrap.label = "Code block wrap";
            code_wrap.kind = Row::Kind::BoolToggle;
            code_wrap.enum_key = "code_block_wrap";
            rows.push_back(code_wrap);

            Row gap1;
            gap1.kind = Row::Kind::Gap;
            rows.push_back(gap1);
            Row sec_typo;
            sec_typo.kind = Row::Kind::Section;
            sec_typo.label = "Labels & Motion";
            rows.push_back(sec_typo);
            Row locale;
            locale.label = "Tool labels";
            locale.kind = Row::Kind::EnumCycle;
            locale.enum_key = "tool_labels_locale";
            locale.enum_values = {"en", "de"};
            rows.push_back(locale);
            Row motion;
            motion.label = "Reduce motion";
            motion.kind = Row::Kind::BoolToggle;
            motion.enum_key = "reduce_motion";
            rows.push_back(motion);
            break;
        }
        case Category::Shortcuts: {
            Row sec;
            sec.kind = Row::Kind::Section;
            sec.label = "Global";
            rows.push_back(sec);
            Row theme_key;
            theme_key.label = "Open theme picker";
            theme_key.kind = Row::Kind::Text;
            theme_key.enum_key = "key_open_theme_picker";
            rows.push_back(theme_key);
            break;
        }
        default:
            break;
    }
    return rows;
}

std::string SettingsModal::provider_display_id() const {
    const auto in_list = [&](const std::string& id) {
        return std::find(draft_.provider_options.begin(), draft_.provider_options.end(), id) !=
               draft_.provider_options.end();
    };
    if (!draft_.provider.empty() && in_list(draft_.provider)) {
        return draft_.provider;
    }
    for (const std::string& id : draft_.provider_options) {
        if (!id.empty()) {
            return id;
        }
    }
    return draft_.provider;
}

std::string SettingsModal::row_value_text(const Row& row) const {
    switch (row.kind) {
        case Row::Kind::Section:
            return "";
        case Row::Kind::ScopeToggle:
            return draft_.scope == "project" ? "Project" : "Chat";
        case Row::Kind::Text:
            if (editing_text_ && text_edit_target_ == row.enum_key) {
                return text_edit_buffer_ + "|";
            }
            if (row.enum_key == "session_name") {
                return draft_.session_name.empty() ? "(empty)" : draft_.session_name;
            }
            if (row.enum_key == "model") {
                const std::string model = draft_.model.empty() ? draft_.model_default : draft_.model;
                return model.empty() ? "(empty)" : model_display_text(model);
            }
            if (row.enum_key == "stt_model") {
                return draft_.stt_model.empty() ? "(driver default)" : draft_.stt_model;
            }
            if (row.enum_key == "stt_capture_device") {
                return draft_.stt_capture_device.empty() ? "default" : draft_.stt_capture_device;
            }
            if (row.enum_key == "key_open_theme_picker") {
                return draft_.key_open_theme_picker.empty() ? "Ctrl+T" : draft_.key_open_theme_picker;
            }
            if (row.enum_key == "system_prompt_extra") {
                if (draft_.system_prompt_extra.empty()) {
                    return "(empty)";
                }
                const std::size_t n = draft_.system_prompt_extra.size();
                std::string preview = draft_.system_prompt_extra.substr(0, std::min<std::size_t>(n, 28));
                if (n > 28) {
                    preview += "…";
                }
                std::replace(preview.begin(), preview.end(), '\n', ' ');
                return preview + " (" + std::to_string(n) + " chars)";
            }
            return "";
        case Row::Kind::ProviderCycle: {
            std::string id = provider_display_id();
            if (row.enum_key == "project_provider") {
                id = draft_.project_default_provider.empty() ? id : draft_.project_default_provider;
            }
            for (std::size_t i = 0; i < draft_.provider_options.size(); ++i) {
                if (draft_.provider_options[i] == id && i < draft_.provider_labels.size() &&
                    !draft_.provider_labels[i].empty()) {
                    return draft_.provider_labels[i];
                }
            }
            return id;
        }
        case Row::Kind::FixedValue:
            if (row.enum_key == "context_window") {
                if (model_context_limit_fixed()) {
                    return format_context_window(draft_.model_catalog_context);
                }
                return format_context_window(draft_.context_window);
            }
            return "";
        case Row::Kind::IntValue:
            if (editing_text_ && text_edit_target_ == row.enum_key) {
                return text_edit_buffer_ + "|";
            }
            if (row.enum_key == "context_window") {
                if (!draft_.context_window_set) {
                    return format_context_window(draft_.context_window_default) + " (default)";
                }
                return format_context_window(draft_.context_window);
            }
            if (row.enum_key == "max_steps") {
                return std::to_string(draft_.max_steps);
            }
            if (row.enum_key == "max_tokens") {
                if (!draft_.max_tokens_set) {
                    return std::to_string(draft_.max_tokens_default) + " (default)";
                }
                return std::to_string(draft_.max_tokens);
            }
            if (row.enum_key == "stt_sample_rate") {
                return std::to_string(draft_.stt_sample_rate);
            }
            if (row.enum_key == "thought_max_chars") {
                if (draft_.thought_max_chars <= 0) {
                    return "0 (unlimited)";
                }
                return std::to_string(draft_.thought_max_chars);
            }
            return "";
        case Row::Kind::PercentBar:
            if (row.enum_key == "user_message_max_width_pct") {
                return std::to_string(draft_.user_message_max_width_pct) + "%";
            }
            if (row.enum_key == "assistant_message_max_width_pct") {
                return std::to_string(draft_.assistant_message_max_width_pct) + "%";
            }
            return "";
        case Row::Kind::FloatValue:
            if (editing_text_ && text_edit_target_ == row.enum_key) {
                return text_edit_buffer_ + "|";
            }
            if (row.enum_key == "stt_silence_level") {
                if (draft_.stt_silence_level <= 0.f) {
                    return "0 (off)";
                }
                return format_float(draft_.stt_silence_level);
            }
            if (!draft_.temperature_set) {
                return format_float(draft_.temperature_default) + " (default)";
            }
            return format_float(draft_.temperature);
        case Row::Kind::EnumCycle:
            if (row.enum_key == "chain_of_thought") {
                return cot_label(draft_.chain_of_thought);
            }
            if (row.enum_key == "turn_metrics_preset") {
                return normalize_metrics(draft_.turn_metrics_preset);
            }
            if (row.enum_key == "tool_labels_locale") {
                return draft_.tool_labels_locale;
            }
            if (row.enum_key == "ui_theme") {
                return draft_.ui_theme;
            }
            if (row.enum_key == "ui_corners") {
                if (draft_.ui_corners == "rounded" || draft_.ui_corners == "flat") {
                    return draft_.ui_corners;
                }
                return "square";
            }
            if (row.enum_key == "tool_call_density") {
                return draft_.tool_call_density;
            }
            if (row.enum_key == "presentation") {
                return draft_.presentation;
            }
            if (row.enum_key == "user_message_style") {
                if (draft_.user_message_style == "messenger") {
                    return "chat";
                }
                return draft_.user_message_style;
            }
            if (row.enum_key == "assistant_message_style") {
                if (draft_.assistant_message_style == "messenger") {
                    return "chat";
                }
                return draft_.assistant_message_style;
            }
            if (row.enum_key == "user_message_align") {
                return draft_.user_message_align.empty() ? "auto" : draft_.user_message_align;
            }
            if (row.enum_key == "assistant_message_align") {
                return draft_.assistant_message_align.empty() ? "auto" : draft_.assistant_message_align;
            }
            if (row.enum_key == "auto_compact_context_pct") {
                if (draft_.auto_compact_context_pct <= 0) {
                    return "off";
                }
                return std::to_string(draft_.auto_compact_context_pct) + "%";
            }
            if (row.enum_key == "chat_width") {
                return draft_.chat_width;
            }
            if (row.enum_key == "message_time_clock") {
                return draft_.message_time_clock == "12h" ? "12h (AM/PM)" : "24h";
            }
            if (row.enum_key == "stt_driver") {
                return stt_driver_display();
            }
            if (row.enum_key == "stt_language") {
                return draft_.stt_language;
            }
            if (row.enum_key == "stt_quality") {
                return draft_.stt_quality.empty() ? "(default)" : draft_.stt_quality;
            }
            return "";
        case Row::Kind::BoolToggle:
            if (row.enum_key == "decisions_enabled") {
                return draft_.decisions_enabled ? "on" : "off";
            }
            if (row.enum_key == "memory_enabled") {
                return draft_.memory_enabled ? "on" : "off";
            }
            if (row.enum_key == "turn_metrics_show_actions") {
                return draft_.turn_metrics_show_actions ? "on" : "off";
            }
            if (row.enum_key == "code_block_wrap") {
                return draft_.code_block_wrap ? "on" : "off";
            }
            if (row.enum_key == "sticky_user_message") {
                return draft_.sticky_user_message ? "on" : "off";
            }
            if (row.enum_key == "user_message_fill_background") {
                return draft_.user_message_fill_background ? "on" : "off";
            }
            if (row.enum_key == "assistant_message_fill_background") {
                return draft_.assistant_message_fill_background ? "on" : "off";
            }
            if (row.enum_key == "reduce_motion") {
                return draft_.reduce_motion ? "on" : "off";
            }
            if (row.enum_key == "message_timestamps") {
                return draft_.message_timestamps ? "on" : "off";
            }
            if (row.enum_key == "stt_enabled") {
                return draft_.stt_enabled ? "on" : "off";
            }
            if (row.enum_key == "stt_trim_silence") {
                return draft_.stt_trim_silence ? "on" : "off";
            }
            return "off";
        case Row::Kind::MCPToggle: {
            const std::string& name = draft_.mcp_available[row.mcp_index];
            const bool on =
                std::find(draft_.mcp_enabled.begin(), draft_.mcp_enabled.end(), name) != draft_.mcp_enabled.end();
            return on ? "on" : "off";
        }
        case Row::Kind::SkillAllowToggle: {
            const std::string& name = draft_.skills_available[row.skill_index];
            return skill_state(name);
        }
        case Row::Kind::PresetCombo: {
            if (!draft_.preset.empty()) {
                return draft_.preset;
            }
            if (!draft_.template_id.empty()) {
                return draft_.template_id;
            }
            return draft_.presentation;
        }
        default:
            return "";
    }
}

std::string SettingsModal::row_hint_text(const Row& row) const {
    if (row.kind == Row::Kind::Section) {
        return "";
    }
    if (row.enum_key == "session_name") {
        return "Display name for this chat session.";
    }
    if (row.enum_key == "preset_combo") {
        return "Cycle presets, YAML templates, and presentation styles.";
    }
    if (row.enum_key == "provider" || row.enum_key == "project_provider") {
        return "Enter: fuzzy picker  <-/->: cycle provider.";
    }
    if (row.enum_key == "model") {
        return "Enter/Space: pick model. Click (?): Wuji catalog details.";
    }
    if (row.enum_key == "temperature") {
        return "Sampling temperature; lower is more deterministic.";
    }
    if (row.enum_key == "max_tokens") {
        return "Maximum tokens per model response.";
    }
    if (row.enum_key == "context_window") {
        if (model_context_limit_fixed()) {
            return "Context window is fixed by the model catalog.";
        }
        return "Enter: type token limit. <-/->: adjust in steps.";
    }
    if (row.enum_key == "stt_driver") {
        return "Wuji driver for wuji text --audio (e.g. llama). Saves capability_drivers.text on apply.";
    }
    if (row.enum_key == "stt_enabled") {
        return "Microphone button in the chat input (|O|).";
    }
    if (row.enum_key == "stt_model") {
        return "Model id passed to the STT driver (empty = driver default).";
    }
    if (row.enum_key == "stt_language") {
        return "Hint language for transcription; auto lets the model detect.";
    }
    if (row.enum_key == "stt_quality") {
        return "Speed vs accuracy preset for transcription (driver-dependent).";
    }
    if (row.enum_key == "stt_trim_silence") {
        return "Drop leading/trailing silence from recorded audio before STT.";
    }
    if (row.enum_key == "stt_silence_level") {
        return "Volume threshold for silence detection when trim is on.";
    }
    if (row.enum_key == "stt_capture_device") {
        return "Audio input device name or path (empty = system default).";
    }
    if (row.enum_key == "stt_sample_rate") {
        return "Capture sample rate in Hz (8000–48000).";
    }
    if (row.enum_key == "max_steps") {
        return "Maximum autonomous tool-call steps per request.";
    }
    if (row.enum_key == "decisions_enabled") {
        return "Allow the agent to record project decisions.";
    }
    if (row.enum_key == "memory_enabled") {
        return "Enable long-term memory for this chat.";
    }
    if (row.enum_key == "chain_of_thought") {
        return "How reasoning is exposed (off / prompt / native).";
    }
    if (row.enum_key == "thought_max_chars") {
        return "Cap visible chain-of-thought length (0 = unlimited).";
    }
    if (row.enum_key == "system_prompt_extra") {
        return "Extra system instructions appended for this chat.";
    }
    if (row.enum_key == "turn_metrics_preset") {
        return "Verbosity of per-turn metrics in the chat log.";
    }
    if (row.enum_key == "tool_labels_locale") {
        return "Language for tool status labels in the UI.";
    }
    if (row.enum_key == "ui_theme") {
        return "Enter: fuzzy search themes. <-/->: cycle. User themes: ~/.config/taiji/themes/";
    }
    if (row.enum_key == "ui_corners") {
        return "Square or rounded frame, or flat fill without an outer border.";
    }
    if (row.enum_key == "tool_call_density") {
        return "Compact or detailed tool-call lines in the activity log.";
    }
    if (row.enum_key == "presentation") {
        return "How assistant messages are rendered in this chat.";
    }
    if (row.enum_key == "chat_width") {
        return "Maximum width of the chat column (terminal cells).";
    }
    if (row.enum_key == "turn_metrics_show_actions") {
        return "Show redo/copy controls under finished turns.";
    }
    if (row.enum_key == "code_block_wrap") {
        return "Wrap long lines inside fenced code blocks.";
    }
    if (row.enum_key == "reduce_motion") {
        return "Minimize interface motion (reserved for future animations).";
    }
    if (row.enum_key == "message_timestamps") {
        return "Show send time on your message bubbles.";
    }
    if (row.enum_key == "sticky_user_message") {
        return "Pin the top of your last scrolled prompt while reading long replies.";
    }
    if (row.enum_key == "user_message_fill_background" || row.enum_key == "assistant_message_fill_background") {
        return "On: solid fill behind your message frame (border included). Off: outline only on the chat background.";
    }
    if (row.enum_key == "user_message_style" || row.enum_key == "assistant_message_style") {
        return "Bubble chrome for this role (chat, cornered, block, header, …).";
    }
    if (row.enum_key == "user_message_align" || row.enum_key == "assistant_message_align") {
        return "Horizontal placement; auto follows the role (you: right, assistant: left).";
    }
    if (row.enum_key == "auto_compact_context_pct") {
        return "When context fills this % of the window, summarize older turns automatically.";
    }
    if (row.enum_key == "user_message_max_width_pct" || row.enum_key == "assistant_message_max_width_pct") {
        return "←/→ or click the bar to set max bubble width (40–100%).";
    }
    if (row.enum_key == "message_time_clock") {
        return "24-hour clock or 12-hour with AM/PM.";
    }
    if (row.kind == Row::Kind::MCPToggle) {
        return "Enable or disable this MCP server for the chat.";
    }
    if (row.kind == Row::Kind::SkillAllowToggle) {
        return "Restrict skills to an allowlist, or allow all.";
    }
    if (row.enum_key == "key_open_theme_picker") {
        return "Global shortcut to open the color theme picker (e.g. Ctrl+T).";
    }
    return "";
}

void SettingsModal::begin_value_edit(int row) {
    if (!row_is_interactive(row) || editing_text_) {
        return;
    }
    const Row& r = rows_[row];
    if (r.kind != Row::Kind::IntValue && r.kind != Row::Kind::FloatValue && r.kind != Row::Kind::Text) {
        return;
    }
    if (r.kind == Row::Kind::Text) {
        activate_row(row, 1);
        return;
    }

    editing_text_ = true;
    text_edit_row_ = row;
    text_edit_target_ = r.enum_key;
    if (r.enum_key == "max_steps") {
        text_edit_buffer_ = std::to_string(draft_.max_steps);
    } else if (r.enum_key == "max_tokens") {
        text_edit_buffer_ = draft_.max_tokens_set ? std::to_string(draft_.max_tokens)
                                                 : std::to_string(draft_.max_tokens_default);
    } else if (r.enum_key == "thought_max_chars") {
        text_edit_buffer_ =
            draft_.thought_max_chars <= 0 ? "0" : std::to_string(draft_.thought_max_chars);
    } else if (r.enum_key == "temperature") {
        text_edit_buffer_ = draft_.temperature_set ? format_float(draft_.temperature)
                                                   : format_float(draft_.temperature_default);
    } else if (r.enum_key == "context_window") {
        if (model_context_limit_fixed()) {
            editing_text_ = false;
            text_edit_row_ = -1;
            text_edit_target_.clear();
            text_edit_buffer_.clear();
            return;
        }
        text_edit_buffer_ = draft_.context_window_set
                                ? std::to_string(draft_.context_window)
                                : std::to_string(draft_.context_window_default);
    } else {
        editing_text_ = false;
        text_edit_row_ = -1;
        text_edit_target_.clear();
        text_edit_buffer_.clear();
    }
    note_dirty();
}

void SettingsModal::cancel_text_edit() {
    if (!editing_text_) {
        return;
    }
    editing_text_ = false;
    text_edit_row_ = -1;
    text_edit_target_.clear();
    text_edit_buffer_.clear();
    note_dirty();
}

void SettingsModal::commit_text_edit() {
    if (!editing_text_) {
        return;
    }
    const std::string target = text_edit_target_;
    const std::string buf = text_edit_buffer_;
    int row = text_edit_row_;
    editing_text_ = false;
    text_edit_row_ = -1;
    text_edit_target_.clear();
    text_edit_buffer_.clear();

    if (target == "session_name") {
        draft_.session_name = buf;
        note_dirty();
        return;
    }
    if (target == "system_prompt_extra") {
        draft_.system_prompt_extra = buf;
        note_dirty();
        return;
    }
    if (target == "stt_model") {
        draft_.stt_model = buf;
        note_dirty();
        return;
    }
    if (target == "stt_capture_device") {
        draft_.stt_capture_device = buf.empty() ? "default" : buf;
        note_dirty();
        return;
    }
    if (target == "key_open_theme_picker") {
        draft_.key_open_theme_picker = buf.empty() ? "Ctrl+T" : buf;
        note_dirty();
        return;
    }

    const Row* r = nullptr;
    if (row >= 0 && row < static_cast<int>(rows_.size())) {
        r = &rows_[row];
    }

    if (target == "max_tokens") {
        try {
            const long v = std::stol(buf);
            int max_v = 200000;
            if (r != nullptr) {
                max_v = r->int_max;
            }
            draft_.max_tokens_set = true;
            draft_.max_tokens = static_cast<int>(std::clamp(v, 0L, static_cast<long>(max_v)));
        } catch (...) {
        }
        note_dirty();
        return;
    }
    if (target == "max_steps") {
        try {
            const long v = std::stol(buf);
            int min_v = 1;
            int max_v = 100;
            if (r != nullptr) {
                min_v = r->int_min;
                max_v = r->int_max;
            }
            draft_.max_steps = static_cast<int>(std::clamp(v, static_cast<long>(min_v), static_cast<long>(max_v)));
        } catch (...) {
        }
        note_dirty();
        return;
    }
    if (target == "thought_max_chars") {
        try {
            const long v = std::stol(buf);
            draft_.thought_max_chars = static_cast<int>(std::max(0L, v));
        } catch (...) {
        }
        note_dirty();
        return;
    }
    if (target == "temperature") {
        try {
            const double v = std::stod(buf);
            draft_.temperature_set = true;
            draft_.temperature = std::clamp(v, 0.0, 2.0);
        } catch (...) {
        }
        note_dirty();
        return;
    }
    if (target == "context_window") {
        try {
            const long v = std::stol(buf);
            int min_v = 1024;
            int max_v = 2'000'000;
            if (r != nullptr) {
                min_v = r->int_min;
                max_v = r->int_max;
            }
            draft_.context_window_set = true;
            draft_.context_window = static_cast<int>(std::clamp(v, static_cast<long>(min_v), static_cast<long>(max_v)));
        } catch (...) {
        }
        note_dirty();
    }
}

void SettingsModal::cycle_row_value(int row, int direction) {
    if (!row_is_interactive(row)) {
        return;
    }
    if (rows_[row].kind == Row::Kind::ProviderCycle) {
        cycle_provider_row(row, direction);
        return;
    }
    activate_row(row, direction);
}

void SettingsModal::activate_row(int row, int direction) {
    if (!row_is_interactive(row)) {
        return;
    }
    Row& r = rows_[row];
    const int dir = direction >= 0 ? 1 : -1;

    switch (r.kind) {
        case Row::Kind::ScopeToggle:
            toggle_scope();
            break;
        case Row::Kind::Text:
            if (r.enum_key == "none") {
                break;
            }
            if (r.enum_key == "model") {
                open_model_fuzzy();
                break;
            }
            if (editing_text_) {
                break;
            }
            editing_text_ = true;
            text_edit_target_ = r.enum_key;
            if (text_edit_target_ == "session_name") {
                text_edit_buffer_ = draft_.session_name;
            } else if (text_edit_target_ == "system_prompt_extra") {
                text_edit_buffer_ = draft_.system_prompt_extra;
            } else if (text_edit_target_ == "stt_model") {
                text_edit_buffer_ = draft_.stt_model;
            } else if (text_edit_target_ == "stt_capture_device") {
                text_edit_buffer_ = draft_.stt_capture_device;
            } else if (text_edit_target_ == "key_open_theme_picker") {
                text_edit_buffer_ = draft_.key_open_theme_picker;
            } else {
                text_edit_buffer_.clear();
            }
            break;
        case Row::Kind::ProviderCycle:
            open_provider_fuzzy(r.enum_key == "project_provider");
            break;
        case Row::Kind::IntValue:
            if (r.enum_key == "max_steps") {
                draft_.max_steps = std::clamp(draft_.max_steps + dir, r.int_min, r.int_max);
            } else if (r.enum_key == "max_tokens") {
                draft_.max_tokens_set = true;
                draft_.max_tokens = std::clamp(draft_.max_tokens + dir * 256, 0, r.int_max);
            } else if (r.enum_key == "context_window") {
                draft_.context_window_set = true;
                const int step = draft_.context_window >= 100000 ? 50000 : (draft_.context_window >= 10000 ? 4096 : 1024);
                draft_.context_window =
                    std::clamp(draft_.context_window + dir * step, r.int_min, r.int_max);
            } else if (r.enum_key == "thought_max_chars") {
                draft_.thought_max_chars = std::max(0, draft_.thought_max_chars + dir * 500);
            } else if (r.enum_key == "stt_sample_rate") {
                draft_.stt_sample_rate = std::clamp(draft_.stt_sample_rate + dir * 1000, r.int_min, r.int_max);
            }
            break;
        case Row::Kind::FloatValue:
            if (r.enum_key == "stt_silence_level") {
                draft_.stt_silence_level =
                    std::clamp(draft_.stt_silence_level + dir * 0.05f, 0.f, 1.f);
            } else {
                draft_.temperature_set = true;
                draft_.temperature = std::clamp(draft_.temperature + dir * 0.1, 0.0, 2.0);
            }
            break;
        case Row::Kind::EnumCycle: {
            std::string* field = nullptr;
            std::string current;
            if (r.enum_key == "chain_of_thought") {
                field = &draft_.chain_of_thought;
                current = draft_.chain_of_thought;
            } else if (r.enum_key == "turn_metrics_preset") {
                field = &draft_.turn_metrics_preset;
                current = normalize_metrics(draft_.turn_metrics_preset);
            } else if (r.enum_key == "tool_labels_locale") {
                field = &draft_.tool_labels_locale;
                current = draft_.tool_labels_locale;
            } else if (r.enum_key == "ui_theme") {
                field = &draft_.ui_theme;
                current = draft_.ui_theme;
            } else if (r.enum_key == "ui_corners") {
                field = &draft_.ui_corners;
                if (draft_.ui_corners == "rounded" || draft_.ui_corners == "flat") {
                    current = draft_.ui_corners;
                } else {
                    current = "square";
                }
            } else if (r.enum_key == "tool_call_density") {
                field = &draft_.tool_call_density;
                current = draft_.tool_call_density;
            } else if (r.enum_key == "presentation") {
                field = &draft_.presentation;
                current = draft_.presentation;
            } else if (r.enum_key == "user_message_style") {
                field = &draft_.user_message_style;
                current = draft_.user_message_style.empty() ? "messenger" : draft_.user_message_style;
                if (current == "messenger") {
                    current = "chat";
                }
            } else if (r.enum_key == "assistant_message_style") {
                field = &draft_.assistant_message_style;
                current = draft_.assistant_message_style.empty() ? "messenger" : draft_.assistant_message_style;
                if (current == "messenger") {
                    current = "chat";
                }
            } else if (r.enum_key == "user_message_align") {
                field = &draft_.user_message_align;
                current = draft_.user_message_align.empty() ? "auto" : draft_.user_message_align;
            } else if (r.enum_key == "assistant_message_align") {
                field = &draft_.assistant_message_align;
                current = draft_.assistant_message_align.empty() ? "auto" : draft_.assistant_message_align;
            } else if (r.enum_key == "auto_compact_context_pct") {
                static const int kAutoCompactLevels[] = {0, 75, 80, 85, 90};
                std::size_t idx = 0;
                for (std::size_t i = 0; i < 5; ++i) {
                    if (draft_.auto_compact_context_pct == kAutoCompactLevels[i]) {
                        idx = i;
                        break;
                    }
                }
                draft_.auto_compact_context_pct = kAutoCompactLevels[(idx + 1) % 5];
                break;
            } else if (r.enum_key == "chat_width") {
                field = &draft_.chat_width;
                current = draft_.chat_width;
            } else if (r.enum_key == "message_time_clock") {
                field = &draft_.message_time_clock;
                current = draft_.message_time_clock == "12h" ? "12h" : "24h";
            } else if (r.enum_key == "stt_driver") {
                field = &draft_.stt_driver;
                current = draft_.stt_driver;
            } else if (r.enum_key == "stt_language") {
                field = &draft_.stt_language;
                current = draft_.stt_language;
            } else if (r.enum_key == "stt_quality") {
                field = &draft_.stt_quality;
                current = draft_.stt_quality;
            }
            if (field != nullptr && !r.enum_values.empty()) {
                std::size_t idx = 0;
                for (std::size_t i = 0; i < r.enum_values.size(); ++i) {
                    if (r.enum_values[i] == current) {
                        idx = i;
                        break;
                    }
                }
                const int n = static_cast<int>(r.enum_values.size());
                const int next = (static_cast<int>(idx) + dir + n) % n;
                *field = r.enum_values[static_cast<std::size_t>(next)];
                if ((r.enum_key == "user_message_style" || r.enum_key == "assistant_message_style") && *field == "chat") {
                    *field = "messenger";
                }
                if ((r.enum_key == "chat_width" || r.enum_key == "message_time_clock" || r.enum_key == "ui_corners" ||
                     r.enum_key == "ui_theme" || r.enum_key == "user_message_style" ||
                     r.enum_key == "assistant_message_style" || r.enum_key == "user_message_align" ||
                     r.enum_key == "assistant_message_align") &&
                    on_preview_) {
                    on_preview_(draft_);
                }
            }
            break;
        }
        case Row::Kind::PercentBar:
            if (r.enum_key == "user_message_max_width_pct") {
                draft_.user_message_max_width_pct =
                    std::clamp(draft_.user_message_max_width_pct + dir * 5, r.int_min, r.int_max);
            } else if (r.enum_key == "assistant_message_max_width_pct") {
                draft_.assistant_message_max_width_pct =
                    std::clamp(draft_.assistant_message_max_width_pct + dir * 5, r.int_min, r.int_max);
            }
            if (on_preview_) {
                on_preview_(draft_);
            }
            break;
        case Row::Kind::BoolToggle:
            if (r.enum_key == "decisions_enabled") {
                draft_.decisions_enabled = !draft_.decisions_enabled;
            } else if (r.enum_key == "memory_enabled") {
                draft_.memory_enabled = !draft_.memory_enabled;
            } else if (r.enum_key == "turn_metrics_show_actions") {
                draft_.turn_metrics_show_actions = !draft_.turn_metrics_show_actions;
            } else if (r.enum_key == "code_block_wrap") {
                draft_.code_block_wrap = !draft_.code_block_wrap;
            } else if (r.enum_key == "reduce_motion") {
                draft_.reduce_motion = !draft_.reduce_motion;
            } else if (r.enum_key == "stt_enabled") {
                draft_.stt_enabled = !draft_.stt_enabled;
            } else if (r.enum_key == "stt_trim_silence") {
                draft_.stt_trim_silence = !draft_.stt_trim_silence;
            } else if (r.enum_key == "message_timestamps") {
                draft_.message_timestamps = !draft_.message_timestamps;
                rebuild_rows();
                if (on_preview_) {
                    on_preview_(draft_);
                }
            } else if (r.enum_key == "sticky_user_message") {
                draft_.sticky_user_message = !draft_.sticky_user_message;
                if (on_preview_) {
                    on_preview_(draft_);
                }
            } else if (r.enum_key == "user_message_fill_background") {
                draft_.user_message_fill_background = !draft_.user_message_fill_background;
                if (on_preview_) {
                    on_preview_(draft_);
                }
            } else if (r.enum_key == "assistant_message_fill_background") {
                draft_.assistant_message_fill_background = !draft_.assistant_message_fill_background;
                if (on_preview_) {
                    on_preview_(draft_);
                }
            }
            break;
        case Row::Kind::PresetCombo: {
            if (r.enum_values.empty()) {
                break;
            }
            std::string current;
            if (!draft_.preset.empty()) {
                current = "preset:" + draft_.preset;
            } else if (!draft_.template_id.empty()) {
                current = "template:" + draft_.template_id;
            } else {
                current = "presentation:" + draft_.presentation;
            }
            std::size_t idx = 0;
            for (std::size_t i = 0; i < r.enum_values.size(); ++i) {
                if (r.enum_values[i] == current) {
                    idx = i;
                    break;
                }
            }
            const int n = static_cast<int>(r.enum_values.size());
            const int next = (static_cast<int>(idx) + dir + n) % n;
            const std::string& pick = r.enum_values[static_cast<std::size_t>(next)];
            if (pick.rfind("template:", 0) == 0) {
                const std::string id = pick.substr(9);
                if (handle_ != 0) {
                    std::vector<char> buf(id.begin(), id.end());
                    buf.push_back('\0');
                    taiji_template_apply(handle_, buf.data());
                    reload_from_backend();
                    note_dirty();
                    return;
                }
                draft_.template_id = id;
                draft_.preset.clear();
            } else if (pick.rfind("preset:", 0) == 0) {
                draft_.preset = pick.substr(7);
                draft_.template_id.clear();
                if (!draft_.preset.empty()) {
                    draft_.presentation = draft_.preset;
                }
            } else if (pick.rfind("presentation:", 0) == 0) {
                draft_.presentation = pick.substr(13);
                draft_.preset.clear();
                draft_.template_id.clear();
            }
            break;
        }
        case Row::Kind::MCPToggle: {
            const std::string& name = draft_.mcp_available[r.mcp_index];
            auto it = std::find(draft_.mcp_enabled.begin(), draft_.mcp_enabled.end(), name);
            if (it == draft_.mcp_enabled.end()) {
                draft_.mcp_enabled.push_back(name);
                std::sort(draft_.mcp_enabled.begin(), draft_.mcp_enabled.end());
            } else {
                draft_.mcp_enabled.erase(it);
            }
            break;
        }
        case Row::Kind::SkillAllowToggle: {
            const std::string& name = draft_.skills_available[r.skill_index];
            cycle_skill_state(name);
            break;
        }
        default:
            break;
    }
    note_dirty();
}

SettingsModal::PanelRect SettingsModal::panel_rect() const {
    PanelRect panel;
    if (!open_) {
        return panel;
    }
    const int max_w = std::max(48, bounds_.width - 4);
    const int max_h = std::max(18, bounds_.height - 4);
    panel.w = std::min(86, max_w);
    panel.h = std::min(26, max_h);
    panel.x = std::max(0, (bounds_.width - panel.w) / 2);
    panel.y = std::max(0, (bounds_.height - panel.h) / 2);
    return panel;
}

tuinator::Rect SettingsModal::panel_bounds() const {
    const PanelRect panel = panel_rect();
    return {bounds_.x + panel.x, bounds_.y + panel.y, panel.w, panel.h};
}

tuinator::Point SettingsModal::to_local(tuinator::Point point) const {
    return {point.x - bounds_.x, point.y - bounds_.y};
}

tuinator::Size SettingsModal::preferred_size() const {
    if (!open_) {
        return {0, 0};
    }
    return bounds_.size();
}

void SettingsModal::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void SettingsModal::paint_frame(tuinator::Canvas& clipped, const PanelRect& panel) const {
    const tuinator::Theme& theme = ui_app_theme();
    tuinator::Style border = theme.muted;
    border.foreground_rgb = ui_border_chrome();
    tuinator::Style line = theme.muted;
    const int nav_w = nav_width();
    const int sep_y = footer_sep_y(panel);
    const tuinator::Rect inner{0, 0, panel.w, panel.h};

    clipped.draw_box(inner, border, theme.glyphs);
    for (int x = 1; x < panel.w - 1; ++x) {
        clipped.draw_text({x, sep_y}, "─", line);
    }
    for (int y = 1; y < sep_y; ++y) {
        clipped.draw_text({nav_w, y}, "│", line);
    }
    clipped.draw_text({nav_w, 0}, "┬", border);
    clipped.draw_text({nav_w, sep_y}, "┴", border);
}

void SettingsModal::paint_content_row(tuinator::Canvas& clipped, const Row& row, int y, int content_x, int value_x,
                                      int value_w, bool selected) const {
    if (row.kind == Row::Kind::Gap) {
        return;
    }
    if (row.kind == Row::Kind::Section) {
        tuinator::Style section = ui_app_theme().heading;
        section.bold = true;
        clipped.draw_text({content_x, y}, clip(row.label, value_x + value_w - content_x), section);
        return;
    }

    const tuinator::Theme& theme = ui_app_theme();
    tuinator::Style key_style = theme.label;
    key_style.foreground_rgb = ui_secondary_text();
    tuinator::Style value_style = theme.label;
    value_style.foreground_rgb = ui_text();

    tuinator::Style sel_bg = ui_overlay_panel_style();
    sel_bg.background_rgb = ui_list_active_bg();
    if (selected) {
        const int row_w = std::max(1, value_x + value_w - content_x + 1);
        clipped.fill_rect({content_x - 1, y, row_w, 1}, ' ', sel_bg);
        tuinator::Style marker = theme.accent;
        marker.background_rgb = sel_bg.background_rgb;
        clipped.draw_text({content_x, y}, "▸", marker);
        key_style.background_rgb = sel_bg.background_rgb;
        value_style.background_rgb = sel_bg.background_rgb;
    }

    clipped.draw_text({content_x + (selected ? 2 : 0), y}, clip(row.label, key_col_width()), key_style);

    const std::string value = row_value_text(row);
    const bool dim_value = value.find("(default)") != std::string::npos || value.find("(empty)") != std::string::npos ||
                           value.find("(none)") != std::string::npos || value == "0 (unlimited)";
    if (dim_value) {
        value_style.dim = true;
        value_style.foreground_rgb = ui_secondary_text();
    }

    std::string rendered;
    if (row.kind == Row::Kind::BoolToggle) {
        const bool on = value == "on";
        tuinator::Style toggle = value_style;
        if (on) {
            toggle.foreground_rgb = ui_status_ok();
        } else {
            toggle.foreground_rgb = ui_toggle_off();
            toggle.dim = true;
        }
        rendered = on ? "● on" : "○ off";
        clipped.draw_text({value_x, y}, clip(rendered, value_w), toggle);
        return;
    }
    if (row.kind == Row::Kind::MCPToggle) {
        const bool on = value == "on";
        tuinator::Style toggle = value_style;
        toggle.foreground_rgb = on ? ui_status_ok() : ui_toggle_off();
        if (!on) {
            toggle.dim = true;
        }
        rendered = on ? "● on" : "○ off";
        clipped.draw_text({value_x, y}, clip(rendered, value_w), toggle);
        return;
    }
    if (row.kind == Row::Kind::SkillAllowToggle) {
        tuinator::Style toggle = value_style;
        if (value == "pinned") {
            toggle.foreground_rgb = ui_status_ok();
            rendered = "● pinned";
        } else if (value == "auto") {
            toggle.foreground_rgb = ui_emphasis();
            rendered = "● auto";
        } else {
            toggle.foreground_rgb = ui_toggle_off();
            toggle.dim = true;
            rendered = "○ off";
        }
        clipped.draw_text({value_x, y}, clip(rendered, value_w), toggle);
        return;
    }
    if (row.kind == Row::Kind::EnumCycle && row.enum_key == "ui_theme") {
        rendered = "[ " + value + " ]";
        clipped.draw_text({value_x, y}, clip(rendered, std::max(0, value_w - 5)), value_style);
        tuinator::Style help = ui_app_theme().accent;
        if (!selected) {
            help.dim = true;
        }
        clipped.draw_text({value_x + std::max(0, value_w - 4), y}, "(⌕)", help);
        return;
    }
    if (row.kind == Row::Kind::EnumCycle || row.kind == Row::Kind::ProviderCycle ||
        row.kind == Row::Kind::PresetCombo) {
        rendered = "◂ " + value + " ▸";
        clipped.draw_text({value_x, y}, clip(rendered, value_w), value_style);
        return;
    }

    if (row.kind == Row::Kind::PercentBar) {
        int pct = 80;
        try {
            if (!value.empty() && value.back() == '%') {
                pct = std::stoi(value.substr(0, value.size() - 1));
            }
        } catch (...) {
        }
        pct = std::clamp(pct, row.int_min, row.int_max);
        const PercentBarLayout layout = percent_bar_layout(value_x, value_w, pct);
        const int span = std::max(1, row.int_max - row.int_min);
        int filled = std::clamp((pct - row.int_min) * layout.bar_w / span, 0, layout.bar_w);
        if (filled == 0 && layout.bar_w > 0 && pct == row.int_min) {
            filled = 1;
        }

        tuinator::Style track = value_style;
        track.dim = false;
        if (!selected) {
            const tuinator::Style panel_bg = ui_overlay_panel_style();
            track.background_rgb = panel_bg.background_rgb;
            track.foreground_rgb = panel_bg.foreground_rgb;
        }
        clipped.fill_rect({layout.bar_x, y, layout.bar_w, 1}, ' ', track);

        if (filled > 0) {
            tuinator::Style fill = value_style;
            fill.dim = false;
            fill.foreground_rgb = selected ? ui_accent() : ui_emphasis();
            fill.background_rgb = fill.foreground_rgb;
            clipped.fill_rect({layout.bar_x, y, filled, 1}, ' ', fill);
            if (selected) {
                tuinator::Style thumb = fill;
                thumb.bold = true;
                thumb.background_rgb = sel_bg.background_rgb;
                const int thumb_x = std::min(layout.bar_x + filled, layout.bar_x + layout.bar_w - 1);
                clipped.draw_text({thumb_x, y}, "▮", thumb);
            }
        }

        tuinator::Style pct_style = value_style;
        pct_style.dim = !selected;
        pct_style.foreground_rgb = selected ? ui_text() : ui_secondary_text();
        clipped.draw_text({layout.pct_x, y}, std::to_string(pct) + "%", pct_style);
        return;
    }

    if (row.kind == Row::Kind::Text || row.kind == Row::Kind::IntValue || row.kind == Row::Kind::FloatValue) {
        if (row.enum_key == "model") {
            rendered = "[ " + value + " ]";
            clipped.draw_text({value_x, y}, clip(rendered, std::max(0, value_w - 5)), value_style);
            tuinator::Style help = ui_app_theme().accent;
            if (!selected) {
                help.dim = true;
            }
            clipped.draw_text({value_x + std::max(0, value_w - 4), y}, "(?)", help);
            return;
        }
        rendered = "[ " + value + " ]";
        if (row.kind == Row::Kind::Text && row.enum_key != "none") {
            rendered += " ✎";
        }
        clipped.draw_text({value_x, y}, clip(rendered, value_w), value_style);
        return;
    }

    clipped.draw_text({value_x, y}, clip(value, value_w), value_style);
}

void SettingsModal::paint(tuinator::PaintContext& ctx) const {
    if (!open_) {
        return;
    }
    tuinator::Canvas& canvas = ctx.canvas;
    const PanelRect panel = panel_rect();
    const tuinator::Rect box{panel.x, panel.y, panel.w, panel.h};
    const tuinator::Theme& theme = ui_app_theme();
    tuinator::Style panel_bg = ui_overlay_panel_style();

    canvas.with_clip(box, [&](tuinator::Canvas& clipped) {
        const tuinator::Rect inner{0, 0, panel.w, panel.h};
        clipped.fill_rect(inner, ' ', panel_bg);
        paint_frame(clipped, panel);

        tuinator::Style title_style = theme.heading;
        clipped.draw_text({2, 0}, " Settings ", title_style);
        if (!chat_only_) {
            const std::string scope_label = draft_.scope == "project" ? "Project" : "Chat";
            const std::string scope_text = "Save to: ◂ " + scope_label + " ▸";
            const int scope_w = tuinator::text_display_width(scope_text);
            tuinator::Style scope_style = theme.label;
            scope_style.foreground_rgb = ui_text();
            clipped.draw_text({std::max(2, panel.w - scope_w - 2), 0}, scope_text, scope_style);
        }

        const int nav_w = nav_width();
        const int content_x = nav_w + 2;
        const int value_x = content_x + key_col_width();
        const int value_w = std::max(0, panel.w - value_x - 2);
        const int sep_y = footer_sep_y(panel);
        const int footer_row = footer_y(panel);

        tuinator::Style nav_heading = theme.muted;
        nav_heading.dim = !focus_nav_;
        if (focus_nav_) {
            nav_heading.foreground_rgb = ui_section_heading();
            nav_heading.foreground = tuinator::Color::Default;
            nav_heading.bold = true;
        }
        clipped.draw_text({1, 1}, clip("Categories", nav_w - 1), nav_heading);

        tuinator::Style content_heading = theme.muted;
        content_heading.dim = focus_nav_;
        if (!focus_nav_) {
            content_heading.foreground_rgb = ui_section_heading();
            content_heading.foreground = tuinator::Color::Default;
            content_heading.bold = true;
        }
        clipped.draw_text({content_x, 1}, clip(category_subtitle(), value_x + value_w - content_x), content_heading);

        for (int i = 0; i < category_count(); ++i) {
            const Category cat = static_cast<Category>(i);
            const bool active = cat == category_;
            tuinator::Style style = active ? theme.accent : theme.muted;
            if (!focus_nav_) {
                style.dim = true;
            }
            const std::string label = (active ? "› " : "  ") + category_title(cat, true);
            clipped.draw_text({1, 2 + i}, clip(label, nav_w - 1), style);
        }

        const int row_start_y = content_row_start_y();
        const int visible_rows = content_visible_row_count();
        for (int vis = 0; vis < visible_rows; ++vis) {
            const int row = content_scroll_ + vis;
            if (row >= static_cast<int>(rows_.size())) {
                break;
            }
            int pointer_row = -1;
            if (pointer_valid_) {
                hit_content_row_for_hint(pointer_local_, pointer_row);
            }
            const bool selected = !focus_nav_ && row == selected_row_;
            const bool hovered = row == pointer_row || row == hover_row_;
            paint_content_row(clipped, rows_[row], row_start_y + vis, content_x, value_x, value_w,
                              selected || hovered);
        }

        if (static_cast<int>(rows_.size()) > visible_rows && visible_rows > 0) {
            tuinator::Style scroll_mark = theme.muted;
            scroll_mark.dim = true;
            const int mark_x = panel.w - 2;
            if (content_scroll_ > 0) {
                clipped.draw_text({mark_x, row_start_y}, "▲", scroll_mark);
            }
            if (content_scroll_ + visible_rows < static_cast<int>(rows_.size())) {
                clipped.draw_text({mark_x, sep_y - 2}, "▼", scroll_mark);
            }
        }

        std::string hint;
        int hint_row = -1;
        if (pointer_valid_) {
            hit_content_row_for_hint(pointer_local_, hint_row);
        }
        if (hint_row < 0) {
            hint_row = hover_row_;
        }
        if (hint_row < 0 && !focus_nav_ && selected_row_ >= 0 && selected_row_ < static_cast<int>(rows_.size()) &&
            row_is_interactive(selected_row_)) {
            hint_row = selected_row_;
        }
        if (hint_row >= 0 && hint_row < static_cast<int>(rows_.size())) {
            hint = row_hint_text(rows_[static_cast<std::size_t>(hint_row)]);
        }
        if (!hint.empty() && sep_y > row_start_y + 1) {
            tuinator::Style hint_style = theme.muted;
            hint_style.dim = true;
            clipped.draw_text({content_x, sep_y - 1}, clip(hint, panel.w - content_x - 2), hint_style);
        }

        const std::string footer = " Esc: close   Tab: pane   Space/Enter: change   ←/→: cycle";
        clipped.draw_text({2, footer_row}, clip(footer, panel.w - 22), theme.muted);
        if (is_dirty()) {
            tuinator::Style apply_style = theme.accent;
            apply_style.bold = true;
            clipped.draw_text({panel.w - 18, footer_row}, " Ctrl+S: apply ", apply_style);
        }
        paint_fuzzy_overlay(clipped, panel);
        paint_model_detail_overlay(clipped, panel);
    });
}

bool SettingsModal::hit_nav(tuinator::Point local, Category& cat) const {
    const PanelRect panel = panel_rect();
    if (local.x < panel.x + 1 || local.x >= panel.x + nav_width()) {
        return false;
    }
    const int row = local.y - panel.y - 2;
    if (row < 0 || row >= category_count()) {
        return false;
    }
    cat = static_cast<Category>(row);
    return true;
}

bool SettingsModal::content_row_click_on_value(tuinator::Point local) const {
    const PanelRect panel = panel_rect();
    if (local.x < panel.x || local.x >= panel.x + panel.w) {
        return false;
    }
    const int x_in_panel = local.x - panel.x;
    const int value_col = nav_width() + 2 + key_col_width();
    return x_in_panel >= value_col;
}

bool SettingsModal::hit_content_row(tuinator::Point local, int& row) const {
    row = -1;
    const PanelRect panel = panel_rect();
    const int content_x = panel.x + nav_width() + 2;
    if (local.x < content_x || local.x >= panel.x + panel.w - 1) {
        return false;
    }
    const int row_start_y = panel.y + content_row_start_y();
    const int rel = local.y - row_start_y;
    if (rel < 0 || rel >= content_visible_row_count()) {
        return false;
    }
    row = content_scroll_ + rel;
    return row >= 0 && row < static_cast<int>(rows_.size());
}

bool SettingsModal::hit_content_row_for_hint(tuinator::Point local, int& row) const {
    if (!hit_content_row(local, row)) {
        return false;
    }
    if (!row_is_interactive(row)) {
        row = -1;
        return false;
    }
    if (row >= 0 && row < static_cast<int>(rows_.size()) && row_hint_text(rows_[row]).empty()) {
        row = -1;
        return false;
    }
    return true;
}

bool SettingsModal::hit_footer(tuinator::Point local, bool& apply) const {
    const PanelRect panel = panel_rect();
    apply = false;
    if (local.y != panel.y + footer_y(panel)) {
        return false;
    }
    if (is_dirty() && local.x >= panel.x + panel.w - 18) {
        apply = true;
        return true;
    }
    return false;
}

tuinator::Widget* SettingsModal::hit_test(tuinator::Point point) {
    if (!open_ || !bounds_.contains(point)) {
        return nullptr;
    }
    return this;
}

tuinator::Widget* SettingsModal::hit_test_focusable(tuinator::Point point) {
    if (!open_ || !bounds_.contains(point)) {
        return nullptr;
    }
    return this;
}

bool SettingsModal::handle_event(const tuinator::Event& event) {
    if (!open_) {
        return false;
    }
    if (model_detail_open_) {
        return handle_model_detail_event(event);
    }
    if (fuzzy_open_) {
        return handle_fuzzy_event(event);
    }

    const auto apply_settings = [this]() { apply_draft_to_backend(); };

    if (editing_text_) {
        if (const auto* key = std::get_if<tuinator::KeyPress>(&event)) {
            if (key->key == tuinator::Key::Escape) {
                cancel_text_edit();
                return true;
            }
            if (key->key == tuinator::Key::Tab) {
                commit_text_edit();
                focus_nav_ = !focus_nav_;
                note_dirty();
                return true;
            }
            if (key->key == tuinator::Key::Enter) {
                commit_text_edit();
                return true;
            }
            if (key->key == tuinator::Key::Backspace && !text_edit_buffer_.empty()) {
                text_edit_buffer_.pop_back();
                note_dirty();
                return true;
            }
            if (key->character >= 32 && key->character < 127) {
                const bool numeric_field = text_edit_target_ == "max_tokens" || text_edit_target_ == "max_steps" ||
                                           text_edit_target_ == "thought_max_chars" ||
                                           text_edit_target_ == "context_window";
                const bool float_field = text_edit_target_ == "temperature";
                if (numeric_field) {
                    if (key->character < '0' || key->character > '9') {
                        return true;
                    }
                } else if (float_field) {
                    if ((key->character < '0' || key->character > '9') && key->character != '.') {
                        return true;
                    }
                    if (key->character == '.' && text_edit_buffer_.find('.') != std::string::npos) {
                        return true;
                    }
                }
                text_edit_buffer_.push_back(key->character);
                note_dirty();
                return true;
            }
            return true;
        }
        if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
            if (mouse->action == tuinator::MouseAction::Click || mouse->action == tuinator::MouseAction::Release) {
                commit_text_edit();
            } else {
                return true;
            }
        } else {
            return true;
        }
    }

    if (const auto* key = std::get_if<tuinator::KeyPress>(&event)) {
        if (key->key == tuinator::Key::Escape) {
            close(false);
            return true;
        }
        if (key->key == tuinator::Key::Tab) {
            focus_nav_ = !focus_nav_;
            note_dirty();
            return true;
        }
        if (key->ctrl && (key->character == 's' || key->character == 'S')) {
            apply_settings();
            close(true);
            return true;
        }
        if (focus_nav_) {
            if (key->key == tuinator::Key::Down || key->character == 'j') {
                const int next = std::min(category_count() - 1, static_cast<int>(category_) + 1);
                set_category(static_cast<Category>(next));
                return true;
            }
            if (key->key == tuinator::Key::Up || key->character == 'k') {
                const int next = std::max(0, static_cast<int>(category_) - 1);
                set_category(static_cast<Category>(next));
                return true;
            }
            if (key->key == tuinator::Key::Right || key->key == tuinator::Key::Enter) {
                focus_nav_ = false;
                selected_row_ = first_interactive_row();
                sync_content_scroll();
                note_dirty();
                return true;
            }
        } else {
            if (key->key == tuinator::Key::Down || key->character == 'j') {
                move_selected_row(1);
                note_dirty();
                return true;
            }
            if (key->key == tuinator::Key::Up || key->character == 'k') {
                move_selected_row(-1);
                note_dirty();
                return true;
            }
            if (key->key == tuinator::Key::Left || key->character == 'h') {
                if (row_is_interactive(selected_row_) &&
                    (rows_[selected_row_].kind == Row::Kind::EnumCycle ||
                     rows_[selected_row_].kind == Row::Kind::ProviderCycle ||
                     rows_[selected_row_].kind == Row::Kind::PresetCombo ||
                     rows_[selected_row_].kind == Row::Kind::IntValue ||
                     rows_[selected_row_].kind == Row::Kind::FloatValue ||
                     rows_[selected_row_].kind == Row::Kind::PercentBar)) {
                    cycle_row_value(selected_row_, -1);
                    note_dirty();
                    return true;
                }
                focus_nav_ = true;
                note_dirty();
                return true;
            }
            if (key->key == tuinator::Key::Right || key->character == 'l') {
                if (row_is_interactive(selected_row_)) {
                    cycle_row_value(selected_row_, 1);
                    note_dirty();
                }
                return true;
            }
            if (!focus_nav_ && row_is_interactive(selected_row_) && rows_[selected_row_].enum_key == "model" &&
                (key->character == '?' || key->character == 'i' || key->character == 'I')) {
                open_model_detail(current_model_id());
                return true;
            }
            if (key->key == tuinator::Key::Enter || key->character == ' ') {
                if (row_is_interactive(selected_row_)) {
                    const Row& r = rows_[selected_row_];
                    if (r.kind == Row::Kind::ProviderCycle) {
                        open_provider_fuzzy(r.enum_key == "project_provider");
                    } else if (r.kind == Row::Kind::EnumCycle && r.enum_key == "ui_theme") {
                        open_theme_fuzzy();
                    } else if (r.kind == Row::Kind::FixedValue) {
                        note_dirty();
                    } else if (r.kind == Row::Kind::Text || r.kind == Row::Kind::IntValue ||
                               r.kind == Row::Kind::FloatValue) {
                        begin_value_edit(selected_row_);
                    } else if (r.kind == Row::Kind::BoolToggle || r.kind == Row::Kind::MCPToggle ||
                               r.kind == Row::Kind::SkillAllowToggle) {
                        activate_row(selected_row_, 1);
                    } else {
                        cycle_row_value(selected_row_, 1);
                    }
                }
                return true;
            }
        }
        return true;
    }

    if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
        const tuinator::Point local = to_local(mouse->position);
        pointer_local_ = local;
        pointer_valid_ = true;
        if (mouse->action == tuinator::MouseAction::Move && percent_bar_drag_row_ < 0) {
            int row = -1;
            const int next_hover = hit_content_row_for_hint(local, row) ? row : -1;
            if (next_hover != hover_row_) {
                hover_row_ = next_hover;
                note_dirty();
            }
            return true;
        }
        if (percent_bar_drag_row_ >= 0) {
            if (mouse->action == tuinator::MouseAction::Move) {
                update_percent_bar_from_local(local, percent_bar_drag_row_);
                return true;
            }
            if (mouse->action == tuinator::MouseAction::Release && mouse->button == tuinator::MouseButton::Left) {
                update_percent_bar_from_local(local, percent_bar_drag_row_);
                percent_bar_drag_row_ = -1;
                return true;
            }
        }
        if (mouse->action == tuinator::MouseAction::Move && mouse->left_pressed && percent_bar_drag_row_ < 0) {
            int row = 0;
            if (hit_content_row(local, row) && row_is_interactive(row) &&
                rows_[static_cast<std::size_t>(row)].kind == Row::Kind::PercentBar && content_row_click_on_value(local)) {
                focus_nav_ = false;
                selected_row_ = row;
                percent_bar_drag_row_ = row;
                update_percent_bar_from_local(local, row);
                return true;
            }
        }
        if (mouse->action == tuinator::MouseAction::WheelUp || mouse->action == tuinator::MouseAction::WheelDown) {
            const PanelRect panel = panel_rect();
            const tuinator::Point local = to_local(mouse->position);
            const int content_left = panel.x + nav_width();
            const int content_top = panel.y + content_row_start_y();
            const int content_bottom = panel.y + footer_sep_y(panel);
            if (local.x >= content_left && local.y >= content_top && local.y < content_bottom) {
                const int delta = mouse->action == tuinator::MouseAction::WheelUp ? -1 : 1;
                const int max_scroll =
                    std::max(0, static_cast<int>(rows_.size()) - content_visible_row_count());
                content_scroll_ = std::clamp(content_scroll_ + delta, 0, max_scroll);
                note_dirty();
                return true;
            }
        }
        if ((mouse->action == tuinator::MouseAction::Press || mouse->action == tuinator::MouseAction::Click) &&
            mouse->button == tuinator::MouseButton::Left) {
            int row = 0;
            if (hit_content_row(local, row) && row_is_interactive(row) && content_row_click_on_value(local) &&
                rows_[static_cast<std::size_t>(row)].kind == Row::Kind::PercentBar) {
                focus_nav_ = false;
                selected_row_ = row;
                percent_bar_drag_row_ = row;
                update_percent_bar_from_local(local, row);
                return true;
            }
        }
        if (mouse->action == tuinator::MouseAction::Click || mouse->action == tuinator::MouseAction::Release) {
            bool apply = false;
            if (hit_footer(local, apply) && apply) {
                apply_settings();
                close(true);
                return true;
            }
            Category cat;
            if (hit_nav(local, cat) && (chat_only_ || draft_.scope == "chat")) {
                focus_nav_ = true;
                set_category(cat);
                return true;
            }
            if (hit_model_row_help(local) && mouse->action == tuinator::MouseAction::Release &&
                mouse->button == tuinator::MouseButton::Left) {
                open_model_detail(current_model_id());
                return true;
            }
            int row = 0;
            if (hit_content_row(local, row)) {
                if (mouse->action == tuinator::MouseAction::Click) {
                    focus_nav_ = false;
                    selected_row_ = row;
                    mark_dirty();
                    return true;
                }
                if (mouse->action == tuinator::MouseAction::Release && mouse->button == tuinator::MouseButton::Left) {
                    focus_nav_ = false;
                    selected_row_ = row;
                    if (row_is_interactive(row) && content_row_click_on_value(local)) {
                        const Row& r = rows_[row];
                        if (r.kind == Row::Kind::ProviderCycle) {
                            open_provider_fuzzy(r.enum_key == "project_provider");
                        } else if (r.kind == Row::Kind::Text && r.enum_key == "model") {
                            open_model_fuzzy();
                        } else if (r.kind == Row::Kind::EnumCycle && r.enum_key == "ui_theme") {
                            open_theme_fuzzy();
                        } else if (r.kind == Row::Kind::IntValue || r.kind == Row::Kind::FloatValue) {
                            begin_value_edit(row);
                        } else if (r.kind == Row::Kind::Text) {
                            begin_value_edit(row);
                        } else if (r.kind == Row::Kind::PercentBar) {
                            update_percent_bar_from_local(local, row);
                            percent_bar_drag_row_ = -1;
                        } else {
                            activate_row(row, 1);
                        }
                    } else {
                        mark_dirty();
                    }
                    return true;
                }
                return true;
            }
            close(false);
            return true;
        }
        return true;
    }
    return true;
}
