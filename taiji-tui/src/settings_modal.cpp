#include "settings_modal.hpp"

#include "ui_test_ids.hpp"
#include "ui_chat_spacing.hpp"
#include "ui_locale.hpp"

#include <libtaiji.h>

#include <algorithm>
#include <cctype>
#include <cmath>
#include <cstring>
#include <sstream>

namespace {

std::string trim(std::string value) {
    while (!value.empty() && std::isspace(static_cast<unsigned char>(value.front()))) {
        value.erase(value.begin());
    }
    while (!value.empty() && std::isspace(static_cast<unsigned char>(value.back()))) {
        value.pop_back();
    }
    return value;
}

std::string json_unescape(const std::string& raw) {
    std::string out;
    for (std::size_t i = 0; i < raw.size(); ++i) {
        if (raw[i] != '\\' || i + 1 >= raw.size()) {
            out.push_back(raw[i]);
            continue;
        }
        const char next = raw[i + 1];
        switch (next) {
            case '"':
            case '\\':
            case '/':
                out.push_back(next);
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
    while (i < json.size() && std::isspace(static_cast<unsigned char>(json[i]))) {
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

bool json_extract_bool(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return false;
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && std::isspace(static_cast<unsigned char>(json[i]))) {
        ++i;
    }
    return json.compare(i, 4, "true") == 0;
}

double json_extract_double(const std::string& json, const std::string& key, double fallback = 0.0) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return fallback;
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && std::isspace(static_cast<unsigned char>(json[i]))) {
        ++i;
    }
    return std::strtod(json.c_str() + i, nullptr);
}

int json_extract_int(const std::string& json, const std::string& key, int fallback = 0) {
    return static_cast<int>(json_extract_double(json, key, fallback));
}

std::string json_extract_value(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return "";
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && std::isspace(static_cast<unsigned char>(json[i]))) {
        ++i;
    }
    if (i >= json.size()) {
        return "";
    }
    if (json[i] == '"') {
        return json_extract_string(json, key);
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

std::vector<std::string> json_extract_object_strings(const std::string& json, const std::string& key) {
    const std::string array_json = json_extract_value(json, key);
    if (array_json.empty() || array_json.front() != '[') {
        return {};
    }
    std::vector<std::string> out;
    int depth = 0;
    std::size_t start = std::string::npos;
    for (std::size_t i = 0; i < array_json.size(); ++i) {
        const char ch = array_json[i];
        if (ch == '{') {
            if (depth == 0) {
                start = i;
            }
            ++depth;
        } else if (ch == '}') {
            --depth;
            if (depth == 0 && start != std::string::npos) {
                out.push_back(array_json.substr(start, i - start + 1));
                start = std::string::npos;
            }
        }
    }
    return out;
}

std::vector<int> json_extract_int_array(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return {};
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && std::isspace(static_cast<unsigned char>(json[i]))) {
        ++i;
    }
    if (i >= json.size() || json[i] != '[') {
        return {};
    }
    ++i;
    std::vector<int> out;
    while (i < json.size()) {
        while (i < json.size() && (std::isspace(static_cast<unsigned char>(json[i])) || json[i] == ',')) {
            ++i;
        }
        if (i >= json.size() || json[i] == ']') {
            break;
        }
        const std::size_t start = i;
        while (i < json.size() && json[i] != ',' && json[i] != ']') {
            ++i;
        }
        const std::string token = json.substr(start, i - start);
        try {
            out.push_back(std::stoi(token));
        } catch (...) {
            out.push_back(0);
        }
    }
    return out;
}

std::vector<std::string> json_extract_string_array(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return {};
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && std::isspace(static_cast<unsigned char>(json[i]))) {
        ++i;
    }
    if (i >= json.size() || json[i] != '[') {
        return {};
    }
    ++i;
    std::vector<std::string> out;
    while (i < json.size()) {
        while (i < json.size() && (std::isspace(static_cast<unsigned char>(json[i])) || json[i] == ',')) {
            ++i;
        }
        if (i >= json.size() || json[i] == ']') {
            break;
        }
        if (json[i] != '"') {
            break;
        }
        ++i;
        std::string raw;
        while (i < json.size()) {
            if (json[i] == '"') {
                out.push_back(json_unescape(raw));
                ++i;
                break;
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
    }
    return out;
}

std::string clip(std::string text, int max_cols) {
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

} // namespace

std::vector<std::string> parse_taiji_model_detail_lines(const std::string& json) {
    const std::string err = json_extract_string(json, "error");
    if (!err.empty()) {
        return {"Error: " + err};
    }
    return json_extract_string_array(json, "lines");
}

std::vector<ProviderModelPick> parse_taiji_provider_models_response(const std::string& json) {
    std::vector<ProviderModelPick> out;
    for (const std::string& obj : json_extract_object_strings(json, "models")) {
        const std::string id = json_extract_string(obj, "id");
        if (id.empty()) {
            continue;
        }
        std::string name = json_extract_string(obj, "name");
        if (name.empty()) {
            name = id;
        }
        out.push_back({id, name});
    }
    return out;
}

bool settings_parse_draft(const std::string& json, SettingsModal::Draft& out) {
    if (json.empty()) {
        return false;
    }
    out.scope = json_extract_string(json, "scope");
    if (out.scope.empty()) {
        out.scope = "chat";
    }
    out.session_name = json_extract_string(json, "session_name");
    out.template_id = json_extract_string(json, "template_id");
    out.preset = json_extract_string(json, "preset");
    out.provider = json_extract_string(json, "provider");
    out.provider_options = json_extract_string_array(json, "provider_options");
    out.provider_labels = json_extract_string_array(json, "provider_labels");
    out.provider_models = json_extract_string_array(json, "provider_models");
    out.provider_is_local = json_extract_bool(json, "provider_is_local");
    out.context_window = json_extract_int(json, "context_window", 32768);
    out.context_window_set = json_extract_bool(json, "context_window_set");
    out.context_window_default = json_extract_int(json, "context_window_default", out.context_window);
    out.model = json_extract_string(json, "model");
    out.model_default = json_extract_string(json, "model_default");
    out.temperature = json_extract_double(json, "temperature", 0.7);
    out.temperature_set = json_extract_bool(json, "temperature_set");
    out.temperature_default = json_extract_double(json, "temperature_default", 0.7);
    out.max_tokens = json_extract_int(json, "max_tokens", 0);
    out.max_tokens_set = json_extract_bool(json, "max_tokens_set");
    out.max_tokens_default = json_extract_int(json, "max_tokens_default", 4096);
    out.max_steps = json_extract_int(json, "max_steps", 25);
    out.chain_of_thought = json_extract_string(json, "chain_of_thought");
    if (out.chain_of_thought.empty()) {
        out.chain_of_thought = "off";
    }
    out.thought_max_chars = json_extract_int(json, "thought_max_chars", 0);
    out.system_prompt_extra = json_extract_string(json, "system_prompt_extra");
    out.presentation = json_extract_string(json, "presentation");
    if (out.presentation.empty()) {
        out.presentation = "markdown";
    }
    out.turn_metrics_preset = json_extract_string(json, "turn_metrics_preset");
    if (out.turn_metrics_preset.empty()) {
        out.turn_metrics_preset = "dev";
    }
    if (out.turn_metrics_preset == "minimal" || out.turn_metrics_preset == "tokens") {
        out.turn_metrics_preset = "compact";
    } else if (out.turn_metrics_preset == "verbose") {
        out.turn_metrics_preset = "dev";
    }
    out.tool_labels_locale = out.ui_locale.empty() ? "en" : out.ui_locale;
    out.mcp_available = json_extract_string_array(json, "mcp_available");
    out.mcp_enabled = json_extract_string_array(json, "mcp_enabled");
    out.mcp_tools_deny = json_extract_string_array(json, "mcp_tools_deny");
    for (const std::string& obj : json_extract_object_strings(json, "mcp_tools")) {
        SettingsModal::Draft::MCPToolEntry entry;
        entry.server = json_extract_string(obj, "server");
        entry.name = json_extract_string(obj, "name");
        entry.description = json_extract_string(obj, "description");
        if (!entry.server.empty() && !entry.name.empty()) {
            out.mcp_tools.push_back(std::move(entry));
        }
    }
    out.skills_available = json_extract_string_array(json, "skills_available");
    out.skills_allow = json_extract_string_array(json, "skills_allow");
    out.packs_available = json_extract_string_array(json, "packs_available");
    out.packs_node_counts = json_extract_int_array(json, "packs_node_counts");
    out.packs_allow = json_extract_string_array(json, "packs_allow");
    out.skills_deny = json_extract_string_array(json, "skills_deny");
    out.builtin_tools_available = json_extract_string_array(json, "builtin_tools_available");
    out.builtin_tools_deny = json_extract_string_array(json, "builtin_tools_deny");
    out.stream_pacing = json_extract_string(json, "stream_pacing");
    out.council_enabled = json_extract_bool(json, "council_enabled");
    out.council_actors = json_extract_string_array(json, "council_actors");
    out.decisions_enabled = json_extract_bool(json, "decisions_enabled");
    out.memory_enabled = json_extract_bool(json, "memory_enabled");
    out.project_default_provider = json_extract_string(json, "project_default_provider");
    out.project_max_steps = json_extract_int(json, "project_max_steps", out.max_steps);
    out.project_chain_of_thought = json_extract_string(json, "project_chain_of_thought");
    if (out.project_chain_of_thought.empty()) {
        out.project_chain_of_thought = out.chain_of_thought;
    }
    for (const std::string& obj : json_extract_object_strings(json, "templates")) {
        SettingsModal::TemplateOption opt;
        opt.id = json_extract_string(obj, "id");
        opt.name = json_extract_string(obj, "name");
        if (opt.name.empty()) {
            opt.name = opt.id;
        }
        if (!opt.id.empty()) {
            out.templates.push_back(std::move(opt));
        }
    }
    out.model_catalog_known = json_extract_bool(json, "model_catalog_known");
    out.model_catalog_temperature = json_extract_bool(json, "model_catalog_temperature");
    out.model_catalog_context = json_extract_int(json, "model_catalog_context", 0);
    out.model_catalog_output = json_extract_int(json, "model_catalog_output", 0);
    out.ui_theme = json_extract_string(json, "ui_theme");
    if (out.ui_theme.empty()) {
        out.ui_theme = "dark";
    }
    out.ui_locale = ui_normalize_locale_id(json_extract_string(json, "ui_locale"));
    out.ui_corners = json_extract_string(json, "ui_corners");
    if (out.ui_corners.empty()) {
        out.ui_corners = "square";
    }
    out.tool_call_density = json_extract_string(json, "tool_call_density");
    if (out.tool_call_density.empty()) {
        out.tool_call_density = "subtree";
    }
    if (out.tool_call_density == "detailed") {
        out.tool_call_density = "subtree";
    }
    if (out.tool_call_density == "compact") {
        out.tool_call_density = "clustered";
    }
    out.file_diff_layout = json_extract_string(json, "file_diff_layout");
    if (out.file_diff_layout.empty()) {
        out.file_diff_layout = "stacked";
    }
    out.activity_spacing_lines =
        ui_clamp_spacing_lines(static_cast<int>(json_extract_int(json, "activity_spacing_lines", 1)));
    out.message_spacing_lines =
        ui_clamp_spacing_lines(static_cast<int>(json_extract_int(json, "message_spacing_lines", 1)));
    out.activity_show_pipeline_title = json_extract_bool(json, "activity_show_pipeline_title");
    if (json.find("\"activity_show_pipeline_title\":") == std::string::npos) {
        out.activity_show_pipeline_title = true;
    }
    out.tool_expand_show_parameters = json_extract_bool(json, "tool_expand_show_parameters");
    if (json.find("\"tool_expand_show_parameters\":") == std::string::npos) {
        out.tool_expand_show_parameters = true;
    }
    out.tool_expand_show_result = json_extract_bool(json, "tool_expand_show_result");
    if (json.find("\"tool_expand_show_result\":") == std::string::npos) {
        out.tool_expand_show_result = true;
    }
    out.tool_expand_args_format = json_extract_string(json, "tool_expand_args_format");
    if (out.tool_expand_args_format.empty()) {
        out.tool_expand_args_format = "pretty";
    }
    out.tool_expand_result_format = json_extract_string(json, "tool_expand_result_format");
    if (out.tool_expand_result_format.empty()) {
        out.tool_expand_result_format = "text";
    }
    out.turn_metrics_show_actions = json_extract_bool(json, "turn_metrics_show_actions");
    if (json.find("\"turn_metrics_show_actions\":") == std::string::npos) {
        out.turn_metrics_show_actions = true;
    }
    out.code_block_wrap = json_extract_bool(json, "code_block_wrap");
    if (json.find("\"code_block_wrap\":") == std::string::npos) {
        out.code_block_wrap = true;
    }
    out.chat_width = json_extract_string(json, "chat_width");
    if (out.chat_width.empty()) {
        out.chat_width = "auto";
    }
    out.reduce_motion = json_extract_bool(json, "reduce_motion");
    out.message_timestamps = json_extract_bool(json, "message_timestamps");
    if (json.find("\"message_timestamps\":") == std::string::npos) {
        out.message_timestamps = true;
    }
    out.message_time_clock = json_extract_string(json, "message_time_clock");
    if (out.message_time_clock != "12h") {
        out.message_time_clock = "24h";
    }
    out.sticky_user_message = json_extract_bool(json, "sticky_user_message");
    if (json.find("\"sticky_user_message\":") == std::string::npos) {
        out.sticky_user_message = true;
    }
    const bool legacy_fill = json_extract_bool(json, "message_fill_background");
    const bool has_legacy_fill = json.find("\"message_fill_background\":") != std::string::npos;
    out.user_message_fill_background = json_extract_bool(json, "user_message_fill_background");
    if (json.find("\"user_message_fill_background\":") == std::string::npos) {
        out.user_message_fill_background = has_legacy_fill ? legacy_fill : true;
    }
    out.assistant_message_fill_background = json_extract_bool(json, "assistant_message_fill_background");
    if (json.find("\"assistant_message_fill_background\":") == std::string::npos) {
        out.assistant_message_fill_background = has_legacy_fill ? legacy_fill : true;
    }
    out.user_message_style = json_extract_string(json, "user_message_style");
    if (out.user_message_style.empty()) {
        out.user_message_style = "messenger";
    }
    out.user_message_align = json_extract_string(json, "user_message_align");
    if (out.user_message_align.empty()) {
        out.user_message_align = "auto";
    }
    out.user_message_max_width_pct = static_cast<int>(json_extract_int(json, "user_message_max_width_pct", 80));
    out.assistant_message_style = json_extract_string(json, "assistant_message_style");
    if (out.assistant_message_style.empty()) {
        out.assistant_message_style = "messenger";
    }
    out.assistant_message_align = json_extract_string(json, "assistant_message_align");
    if (out.assistant_message_align.empty()) {
        out.assistant_message_align = "auto";
    }
    out.assistant_message_max_width_pct =
        static_cast<int>(json_extract_int(json, "assistant_message_max_width_pct", 80));
    out.auto_compact_context_pct = static_cast<int>(json_extract_int(json, "auto_compact_context_pct", 0));
    if (out.auto_compact_context_pct < 0) {
        out.auto_compact_context_pct = 0;
    } else if (out.auto_compact_context_pct > 100) {
        out.auto_compact_context_pct = 100;
    }
    out.key_open_theme_picker = json_extract_string(json, "key_open_theme_picker");
    if (out.key_open_theme_picker.empty()) {
        out.key_open_theme_picker = "Ctrl+T";
    }
    out.keybinding_style = json_extract_string(json, "keybinding_style");
    if (out.keybinding_style.empty()) {
        out.keybinding_style = "standard";
    }
    out.vim_escape_sequence = json_extract_string(json, "vim_escape_sequence");
    if (out.vim_escape_sequence.empty()) {
        out.vim_escape_sequence = "jk";
    }
    out.leader_key = json_extract_string(json, "leader_key");
    if (out.leader_key.empty()) {
        out.leader_key = "Space";
    }
    out.stt_enabled = json_extract_bool(json, "stt_enabled");
    if (json.find("\"stt_enabled\":") == std::string::npos) {
        out.stt_enabled = true;
    }
    out.stt_driver = json_extract_string(json, "stt_driver");
    out.stt_driver_options = json_extract_string_array(json, "stt_driver_options");
    out.stt_driver_labels = json_extract_string_array(json, "stt_driver_labels");
    out.stt_drivers_error = json_extract_string(json, "stt_drivers_error");
    out.stt_model = json_extract_string(json, "stt_model");
    out.stt_language = json_extract_string(json, "stt_language");
    if (out.stt_language.empty()) {
        out.stt_language = "auto";
    }
    out.stt_quality = json_extract_string(json, "stt_quality");
    out.stt_trim_silence = json_extract_bool(json, "stt_trim_silence");
    out.stt_silence_level = static_cast<float>(json_extract_double(json, "stt_silence_level", 0.0));
    out.stt_capture_device = json_extract_string(json, "stt_capture_device");
    if (out.stt_capture_device.empty()) {
        out.stt_capture_device = "default";
    }
    out.stt_sample_rate = json_extract_int(json, "stt_sample_rate", 16000);
    return true;
}

void apply_model_catalog_json(const std::string& json, SettingsModal::Draft& out) {
    out.model_catalog_known = json_extract_bool(json, "catalog_known");
    out.model_catalog_temperature = json_extract_bool(json, "temperature");
    out.model_catalog_context = json_extract_int(json, "context", 0);
    out.model_catalog_output = json_extract_int(json, "output", 0);
}

bool settings_json_has_error(const std::string& json) {
    return !json_extract_string(json, "error").empty();
}

SettingsModal::SettingsModal(tuinator::Theme theme) : theme_(std::move(theme)) {
    set_test_id(taiji::ui::test_id::kSettingsModal);
}

void SettingsModal::set_handle(uint64_t handle) { handle_ = handle; }

void SettingsModal::set_on_closed(std::function<void(bool applied)> callback) { on_closed_ = std::move(callback); }

void SettingsModal::set_on_preview(std::function<void(const Draft&)> callback) { on_preview_ = std::move(callback); }
void SettingsModal::json_escape_append(std::string& out, const std::string& value) const {
    out.push_back('"');
    for (char ch : value) {
        switch (ch) {
            case '"':
            case '\\':
                out.push_back('\\');
                out.push_back(ch);
                break;
            case '\n':
                out += "\\n";
                break;
            case '\r':
                out += "\\r";
                break;
            case '\t':
                out += "\\t";
                break;
            default:
                out.push_back(ch);
                break;
        }
    }
    out.push_back('"');
}

std::string SettingsModal::build_apply_json() const {
    const std::string tool_labels_locale = draft_.ui_locale.empty() ? "en" : draft_.ui_locale;
    std::string body;
    body += '{';
    body += "\"scope\":";
    json_escape_append(body, chat_only_ ? "chat" : draft_.scope);
    body += ",\"session_name\":";
    json_escape_append(body, draft_.session_name);
    body += ",\"template_id\":";
    json_escape_append(body, draft_.template_id);
    body += ",\"preset\":";
    json_escape_append(body, draft_.preset);
    body += ",\"provider\":";
    json_escape_append(body, draft_.provider);
    body += ",\"model\":";
    json_escape_append(body, draft_.model);
    {
        std::ostringstream num;
        num << draft_.temperature;
        body += ",\"temperature\":" + num.str();
    }
    body += std::string(",\"temperature_set\":") + (draft_.temperature_set ? "true" : "false");
    body += ",\"max_tokens\":" + std::to_string(draft_.max_tokens);
    body += std::string(",\"max_tokens_set\":") + (draft_.max_tokens_set ? "true" : "false");
    body += ",\"context_window\":" + std::to_string(draft_.context_window);
    body += std::string(",\"context_window_set\":") + (draft_.context_window_set ? "true" : "false");
    body += ",\"max_steps\":" + std::to_string(draft_.max_steps);
    body += ",\"chain_of_thought\":";
    json_escape_append(body, draft_.chain_of_thought);
    body += ",\"thought_max_chars\":" + std::to_string(draft_.thought_max_chars);
    body += ",\"system_prompt_extra\":";
    json_escape_append(body, draft_.system_prompt_extra);
    body += ",\"presentation\":";
    json_escape_append(body, draft_.presentation);
    body += ",\"turn_metrics_preset\":";
    json_escape_append(body, draft_.turn_metrics_preset);
    body += ",\"tool_labels_locale\":";
    json_escape_append(body, tool_labels_locale);
    body += ",\"ui_theme\":";
    json_escape_append(body, draft_.ui_theme);
    body += ",\"ui_locale\":";
    json_escape_append(body, ui_normalize_locale_id(draft_.ui_locale));
    body += ",\"ui_corners\":";
    json_escape_append(body, draft_.ui_corners);
    body += ",\"tool_call_density\":";
    json_escape_append(body, draft_.tool_call_density);
    body += ",\"file_diff_layout\":";
    json_escape_append(body, draft_.file_diff_layout);
    body += ",\"activity_spacing_lines\":";
    body += std::to_string(ui_clamp_spacing_lines(draft_.activity_spacing_lines));
    body += ",\"message_spacing_lines\":";
    body += std::to_string(ui_clamp_spacing_lines(draft_.message_spacing_lines));
    body += std::string(",\"activity_show_pipeline_title\":") + (draft_.activity_show_pipeline_title ? "true" : "false");
    body += std::string(",\"tool_expand_show_parameters\":") + (draft_.tool_expand_show_parameters ? "true" : "false");
    body += std::string(",\"tool_expand_show_result\":") + (draft_.tool_expand_show_result ? "true" : "false");
    body += ",\"tool_expand_args_format\":";
    json_escape_append(body, draft_.tool_expand_args_format.empty() ? "pretty" : draft_.tool_expand_args_format);
    body += ",\"tool_expand_result_format\":";
    json_escape_append(body, draft_.tool_expand_result_format.empty() ? "text" : draft_.tool_expand_result_format);
    body += std::string(",\"turn_metrics_show_actions\":") + (draft_.turn_metrics_show_actions ? "true" : "false");
    body += std::string(",\"code_block_wrap\":") + (draft_.code_block_wrap ? "true" : "false");
    body += ",\"chat_width\":";
    json_escape_append(body, draft_.chat_width);
    body += std::string(",\"reduce_motion\":") + (draft_.reduce_motion ? "true" : "false");
    body += std::string(",\"message_timestamps\":") + (draft_.message_timestamps ? "true" : "false");
    body += ",\"message_time_clock\":";
    json_escape_append(body, draft_.message_time_clock == "12h" ? "12h" : "24h");
    body += std::string(",\"sticky_user_message\":") + (draft_.sticky_user_message ? "true" : "false");
    body += std::string(",\"user_message_fill_background\":") + (draft_.user_message_fill_background ? "true" : "false");
    body += std::string(",\"assistant_message_fill_background\":") +
             (draft_.assistant_message_fill_background ? "true" : "false");
    body += ",\"user_message_style\":";
    json_escape_append(body, draft_.user_message_style);
    body += ",\"user_message_align\":";
    json_escape_append(body, draft_.user_message_align);
    body += ",\"user_message_max_width_pct\":" + std::to_string(draft_.user_message_max_width_pct);
    body += ",\"assistant_message_style\":";
    json_escape_append(body, draft_.assistant_message_style);
    body += ",\"assistant_message_align\":";
    json_escape_append(body, draft_.assistant_message_align);
    body += ",\"assistant_message_max_width_pct\":" + std::to_string(draft_.assistant_message_max_width_pct);
    body += ",\"auto_compact_context_pct\":" + std::to_string(draft_.auto_compact_context_pct);
    body += ",\"key_open_theme_picker\":";
    json_escape_append(body, draft_.key_open_theme_picker);
    body += ",\"keybinding_style\":";
    json_escape_append(body, draft_.keybinding_style);
    body += ",\"vim_escape_sequence\":";
    json_escape_append(body, draft_.vim_escape_sequence);
    body += ",\"leader_key\":";
    json_escape_append(body, draft_.leader_key);
    body += std::string(",\"decisions_enabled\":") + (draft_.decisions_enabled ? "true" : "false");
    body += std::string(",\"memory_enabled\":") + (draft_.memory_enabled ? "true" : "false");
    body += ",\"project_default_provider\":";
    json_escape_append(body, draft_.project_default_provider);
    body += ",\"project_max_steps\":" + std::to_string(draft_.project_max_steps);
    body += ",\"project_chain_of_thought\":";
    json_escape_append(body, draft_.project_chain_of_thought);
    body += ",\"mcp_enabled\":[";
    for (std::size_t i = 0; i < draft_.mcp_enabled.size(); ++i) {
        if (i > 0) {
            body += ",";
        }
        json_escape_append(body, draft_.mcp_enabled[i]);
    }
    body += "],\"mcp_tools_deny\":[";
    for (std::size_t i = 0; i < draft_.mcp_tools_deny.size(); ++i) {
        if (i > 0) {
            body += ",";
        }
        json_escape_append(body, draft_.mcp_tools_deny[i]);
    }
    body += "],\"skills_allow\":[";
    for (std::size_t i = 0; i < draft_.skills_allow.size(); ++i) {
        if (i > 0) {
            body += ",";
        }
        json_escape_append(body, draft_.skills_allow[i]);
    }
    body += "],\"skills_deny\":[";
    for (std::size_t i = 0; i < draft_.skills_deny.size(); ++i) {
        if (i > 0) {
            body += ",";
        }
        json_escape_append(body, draft_.skills_deny[i]);
    }
    body += "],\"packs_allow\":[";
    for (std::size_t i = 0; i < draft_.packs_allow.size(); ++i) {
        if (i > 0) {
            body += ",";
        }
        json_escape_append(body, draft_.packs_allow[i]);
    }
    body += "],\"builtin_tools_deny\":[";
    for (std::size_t i = 0; i < draft_.builtin_tools_deny.size(); ++i) {
        if (i > 0) {
            body += ",";
        }
        json_escape_append(body, draft_.builtin_tools_deny[i]);
    }
    body += "],\"stream_pacing\":";
    json_escape_append(body, draft_.stream_pacing);
    body += std::string(",\"council_enabled\":") + (draft_.council_enabled ? "true" : "false");
    body += ",\"council_actors\":[";
    for (std::size_t i = 0; i < draft_.council_actors.size(); ++i) {
        if (i > 0) {
            body += ",";
        }
        json_escape_append(body, draft_.council_actors[i]);
    }
    body += "],\"stt_enabled\":";
    body += draft_.stt_enabled ? "true" : "false";
    body += ",\"stt_driver\":";
    json_escape_append(body, draft_.stt_driver);
    body += ",\"stt_model\":";
    json_escape_append(body, draft_.stt_model);
    body += ",\"stt_language\":";
    json_escape_append(body, draft_.stt_language);
    body += ",\"stt_quality\":";
    json_escape_append(body, draft_.stt_quality);
    body += std::string(",\"stt_trim_silence\":") + (draft_.stt_trim_silence ? "true" : "false");
    {
        std::ostringstream num;
        num << draft_.stt_silence_level;
        body += ",\"stt_silence_level\":" + num.str();
    }
    body += ",\"stt_capture_device\":";
    json_escape_append(body, draft_.stt_capture_device);
    body += ",\"stt_sample_rate\":" + std::to_string(draft_.stt_sample_rate);
    body += "}";
    return body;
}
