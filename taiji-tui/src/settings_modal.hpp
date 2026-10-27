#pragma once

#include <tuinator/tuinator.hpp>

#include <cstdint>
#include <functional>
#include <string>
#include <vector>

class SettingsModal : public tuinator::Widget {
  public:
    enum class Category {
        General = 0,
        Model,
        AgentBehaviour,
        ToolsSkills,
        Audio,
        Appearance,
        Shortcuts,
        Count
    };

    struct TemplateOption {
        std::string id;
        std::string name;
    };

    struct Draft {
        std::string scope = "chat";
        std::vector<TemplateOption> templates;
        std::string session_name;
        std::string template_id;
        std::string preset;
        std::string provider;
        std::vector<std::string> provider_options;
        std::vector<std::string> provider_labels;
        std::vector<std::string> provider_models;
        bool provider_is_local = false;
        int context_window = 32768;
        bool context_window_set = false;
        int context_window_default = 32768;
        std::string model;
        std::string model_default;
        double temperature = 0.7;
        bool temperature_set = false;
        double temperature_default = 0.7;
        int max_tokens = 0;
        bool max_tokens_set = false;
        int max_tokens_default = 4096;
        int max_steps = 25;
        std::string chain_of_thought = "off";
        int thought_max_chars = 0;
        std::string system_prompt_extra;
        std::string presentation = "markdown";
        std::string turn_metrics_preset = "dev";
        std::string tool_labels_locale = "en";
        struct MCPToolEntry {
            std::string server;
            std::string name;
            std::string description;
        };
        std::vector<std::string> mcp_available;
        std::vector<std::string> mcp_enabled;
        std::vector<MCPToolEntry> mcp_tools;
        std::vector<std::string> mcp_tools_deny;
        std::vector<std::string> skills_available;
        std::vector<std::string> skills_allow;
        std::vector<std::string> skills_deny;
        std::vector<std::string> builtin_tools_available;
        std::vector<std::string> builtin_tools_deny;
        std::string stream_pacing;
        bool council_enabled = false;
        std::vector<std::string> council_actors;
        std::vector<std::string> packs_available;
        std::vector<int> packs_node_counts;
        std::vector<std::string> packs_allow;
        bool decisions_enabled = true;
        bool memory_enabled = true;
        std::string project_default_provider;
        int project_max_steps = 25;
        std::string project_chain_of_thought = "off";
        bool model_catalog_known = false;
        bool model_catalog_temperature = false;
        int model_catalog_context = 0;
        int model_catalog_output = 0;
        std::string ui_theme = "dark";
        std::string ui_locale = "en";
        std::string ui_corners = "square";
        std::string tool_call_density = "subtree";
        std::string file_diff_layout = "stacked";
        int activity_spacing_lines = 1;
        int message_spacing_lines = 1;
        bool activity_show_pipeline_title = true;
        bool tool_expand_show_parameters = true;
        bool tool_expand_show_result = true;
        std::string tool_expand_args_format = "pretty";
        std::string tool_expand_result_format = "text";
        bool turn_metrics_show_actions = true;
        bool code_block_wrap = true;
        std::string chat_width = "auto";
        bool reduce_motion = false;
        bool message_timestamps = true;
        std::string message_time_clock = "24h";
        bool sticky_user_message = true;
        bool user_message_fill_background = true;
        bool assistant_message_fill_background = true;
        std::string user_message_style = "messenger";
        std::string user_message_align = "auto";
        int user_message_max_width_pct = 80;
        std::string assistant_message_style = "messenger";
        std::string assistant_message_align = "auto";
        int assistant_message_max_width_pct = 80;
        int auto_compact_context_pct = 0;
        std::string key_open_theme_picker = "Ctrl+T";
        std::string keybinding_style = "standard";
        std::string vim_escape_sequence = "jk";
        std::string leader_key = "Space";
        bool stt_enabled = true;
        std::string stt_driver;
        std::vector<std::string> stt_driver_options;
        std::vector<std::string> stt_driver_labels;
        std::string stt_drivers_error;
        std::string stt_model;
        std::string stt_language = "auto";
        std::string stt_quality;
        bool stt_trim_silence = false;
        float stt_silence_level = 0.f;
        std::string stt_capture_device = "default";
        int stt_sample_rate = 16000;
    };

    explicit SettingsModal(tuinator::Theme theme);

    void set_handle(uint64_t handle);
    void set_on_closed(std::function<void(bool applied)> callback);
    /// Live preview while editing (e.g. chat width) before Ctrl+S applies.
    void set_on_preview(std::function<void(const Draft&)> callback);
    void open();
    void close(bool applied);
    bool is_open() const { return open_; }
    void persist_chat_ui_theme(const std::string& theme_id);
    void sync_pointer_from_screen(tuinator::Point root_point);
    /// Rebuild row labels after ui_set_locale (live preview while settings are open).
    void refresh_translated_content();

    std::string_view widget_type_name() const override { return "SettingsModal"; }
    bool wants_full_screen() const override { return true; }
    bool captures_pointer() const override { return open_; }
    bool captures_keyboard() const override { return open_; }
    bool wants_hover() const override { return open_; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;
    tuinator::Widget* hit_test_focusable(tuinator::Point point) override;

    tuinator::Rect panel_bounds() const;

  private:
    struct PanelRect {
        int x = 0;
        int y = 0;
        int w = 0;
        int h = 0;
    };

    struct FuzzyOverlayLayout {
        int x = 0;
        int y = 0;
        int w = 0;
        int h = 0;
        int list_y = 0;
        int max_rows = 0;
    };

    struct ModelDetailOverlayLayout {
        int x = 0;
        int y = 0;
        int w = 0;
        int h = 0;
        int body_rows = 0;
    };

    struct Row {
        std::string label;
        enum class Kind {
            Section,
            Gap,
            ScopeToggle,
            Text,
            ProviderCycle,
            IntValue,
            FloatValue,
            EnumCycle,
            BoolToggle,
            MCPToggle,
            MCPToolToggle,
            BuiltinToolToggle,
            SkillAllowToggle,
            PackAllowToggle,
            PresetCombo,
            FixedValue,
            PercentBar,
            Action
        } kind = Kind::Text;
        std::string enum_key;
        std::vector<std::string> enum_values;
        int int_min = 0;
        int int_max = 9999;
        int mcp_index = -1;
        std::string mcp_tool_server;
        std::string mcp_tool_name;
        std::string builtin_tool_name;
        int skill_index = -1;
        int pack_index = -1;
    };

    void reload_from_backend();
    std::string build_apply_json() const;
    PanelRect panel_rect() const;
    tuinator::Point to_local(tuinator::Point point) const;
    void rebuild_rows();
    void localize_settings_row(Row& row);
    std::vector<Row> rows_for_category() const;
    int nav_width() const { return 22; }
    int key_col_width() const { return 24; }
    int category_count() const { return static_cast<int>(Category::Count); }
    int footer_sep_y(const PanelRect& panel) const { return panel.h - 3; }
    int footer_y(const PanelRect& panel) const { return panel.h - 2; }
    int content_row_start_y() const { return 3; }
    void set_category(Category cat);
    void activate_row(int row, int direction = 1);
    void cycle_row_value(int row, int direction);
    void begin_value_edit(int row);
    void commit_text_edit();
    void cancel_text_edit();
    bool row_is_interactive(int row) const;
    void move_selected_row(int delta);
    int content_visible_row_count() const;
    void sync_content_scroll();
    void update_percent_bar_from_local(tuinator::Point local, int row);
    int first_interactive_row() const;
    std::string category_title(Category cat, bool short_nav) const;
    std::string category_subtitle() const;
    std::string row_value_text(const Row& row) const;
    bool row_bool_toggle_on(const Row& row) const;
    std::string row_hint_text(const Row& row) const;
    std::string provider_display_id() const;
    std::string provider_display_label() const;
    std::string format_context_window(int tokens) const;
    /** UI label for a model id (e.g. strips Wuji `ollama/` storage prefix). */
    std::string model_display_text(const std::string& model) const;
    std::string current_model_id() const;
    void toggle_scope();
    enum class FuzzyPickerMode { Model, Provider, Theme };

    void open_model_fuzzy();
    void open_provider_fuzzy(bool project_scope);
    void open_theme_fuzzy();
    void apply_draft_to_backend();
    void preview_theme_fuzzy_selection();
    void cycle_provider_row(int row, int direction);
    void rebuild_fuzzy_filter();
    FuzzyOverlayLayout fuzzy_overlay_layout(const PanelRect& panel) const;
    ModelDetailOverlayLayout model_detail_overlay_layout(const PanelRect& panel) const;
    bool fuzzy_overlay_panel_point(tuinator::Point local, const PanelRect& panel, int& panel_x, int& panel_y) const;
    void paint_fuzzy_overlay(tuinator::Canvas& clipped, const PanelRect& panel) const;
    bool handle_fuzzy_event(const tuinator::Event& event);
    std::string skill_state(const std::string& name) const;
    void cycle_skill_state(const std::string& name);
    bool pack_enabled(const std::string& name) const;
    void toggle_pack_enabled(const std::string& name);
    bool mcp_server_expanded(const std::string& server) const;
    bool mcp_server_has_tools(const std::string& server) const;
    void toggle_mcp_server_expanded(const std::string& server);
    bool mcp_tool_denied(const std::string& server, const std::string& tool) const;
    void toggle_mcp_tool_denied(const std::string& server, const std::string& tool);
    std::string mcp_tool_description(const std::string& server, const std::string& tool) const;
    bool builtin_tool_denied(const std::string& tool) const;
    void toggle_builtin_tool_denied(const std::string& tool);
    void paint_content_row(tuinator::Canvas& clipped, const Row& row, int y, int content_x, int value_x, int value_w,
                           bool selected) const;
    void paint_frame(tuinator::Canvas& clipped, const PanelRect& panel) const;
    bool hit_nav(tuinator::Point local, Category& cat) const;
    bool hit_content_row(tuinator::Point local, int& row) const;
    bool hit_content_row_for_hint(tuinator::Point local, int& row) const;
    bool content_row_click_on_value(tuinator::Point local) const;
    bool hit_footer(tuinator::Point local, bool& apply) const;
    void json_escape_append(std::string& out, const std::string& value) const;
    void note_dirty();
    bool is_dirty() const;
    void capture_saved_snapshot();

    tuinator::Theme theme_;
    bool chat_only_ = true;
    uint64_t handle_ = 0;
    std::function<void(bool applied)> on_closed_;
    std::function<void(const Draft&)> on_preview_;
    bool open_ = false;
    Category category_ = Category::General;
    bool focus_nav_ = true;
    int selected_row_ = 0;
    int content_scroll_ = 0;
    bool editing_text_ = false;
    int text_edit_row_ = -1;
    std::string text_edit_buffer_;
    std::string text_edit_target_;
    Draft draft_;
    std::vector<Row> rows_;
    std::vector<std::string> mcp_expanded_servers_;
    std::string saved_snapshot_;
    bool dirty_ = false;

    struct FuzzyEntry {
        std::string provider_id;
        std::string model;
        std::string line;
        std::string meta;
    };
    void apply_fuzzy_pick(const FuzzyEntry& pick);
    void sync_fuzzy_list_scroll(int max_visible_rows);
    void sync_fuzzy_viewport_to_cursor();
    bool fuzzy_models_match(const std::string& a, const std::string& b) const;
    void refresh_model_catalog_hints();
    bool model_setting_show_temperature() const;
    bool model_setting_show_max_tokens() const;
    bool model_setting_show_context_limit() const;
    bool model_context_limit_fixed() const;
    void sync_context_from_catalog();
    std::string stt_driver_display() const;
    void open_model_detail(const std::string& model_id);
    void paint_model_detail_overlay(tuinator::Canvas& clipped, const PanelRect& panel) const;
    bool handle_model_detail_event(const tuinator::Event& event);
    bool hit_model_row_help(tuinator::Point local) const;
    bool fuzzy_open_ = false;
    FuzzyPickerMode fuzzy_picker_mode_ = FuzzyPickerMode::Model;
    std::string fuzzy_theme_revert_id_;
    bool fuzzy_provider_project_ = false;
    std::vector<FuzzyEntry> fuzzy_entries_;
    std::vector<int> fuzzy_filtered_;
    std::string fuzzy_query_;
    int fuzzy_cursor_ = 0;
    int fuzzy_list_scroll_ = 0;
    bool model_detail_open_ = false;
    std::vector<std::string> model_detail_lines_;
    int model_detail_scroll_ = 0;
    int percent_bar_drag_row_ = -1;
    int hover_row_ = -1;
    bool pointer_valid_ = false;
    tuinator::Point pointer_local_{0, 0};
};

bool settings_parse_draft(const std::string& json, SettingsModal::Draft& out);

struct ProviderModelPick {
    std::string id;
    std::string name;
};
std::vector<ProviderModelPick> parse_taiji_provider_models_response(const std::string& json);
std::vector<std::string> parse_taiji_model_detail_lines(const std::string& json);
void apply_model_catalog_json(const std::string& json, SettingsModal::Draft& out);
bool settings_json_has_error(const std::string& json);
