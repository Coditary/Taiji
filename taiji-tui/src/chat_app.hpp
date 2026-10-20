#pragma once

#include "activity_log.hpp"
#include "assistant_markdown.hpp"
#include "chat_input.hpp"
#include "message_composer.hpp"
#include "question_composer.hpp"
#include "chat_message.hpp"
#include "left_label.hpp"
#include "queue_status_bar.hpp"
#include "left_anchored_split.hpp"
#include "right_anchored_split.hpp"
#include "session_context_menu.hpp"
#include "session_sidebar.hpp"
#include "chat_empty_state.hpp"
#include "chat_middle_pane.hpp"
#include "chat_session_header.hpp"
#include "context_panel.hpp"
#include "mcp_server_list.hpp"
#include "settings_modal.hpp"
#include "sidebar_footer.hpp"
#include "sidebar_kv_row.hpp"
#include "status_footer.hpp"
#include "thinking_indicator.hpp"
#include "turn_metrics_note.hpp"
#include "ui_message_time.hpp"

#include <tuinator/tuinator.hpp>


#include <atomic>
#include <chrono>
#include <cstdint>
#include <deque>
#include <mutex>
#include <optional>
#include <string>
#include <unordered_map>
#include <thread>
#include <vector>

class SpeechMicButton;

class ChatApp {
  public:
    explicit ChatApp(tuinator::Application& app);
    ~ChatApp();

    void set_root();
    void poll_updates();

  private:
    enum class TurnStatus { Queued, Running, Done, Cancelled };

    struct TurnMetrics {
        bool valid = false;
        int64_t total_duration_ms = 0;
        int64_t wait_duration_ms = 0;
        int64_t thinking_duration_ms = 0;
        int64_t generation_duration_ms = 0;
        int output_tokens = 0;
        int input_tokens = 0;
        double estimated_cost_usd = 0.0;
        double tokens_per_sec = 0.0;
        bool had_thinking = false;
        int tool_calls = 0;
        int64_t tool_duration_ms = 0;
        int steps = 0;
        int context_tokens = 0;
        int context_window = 0;
        double context_used_pct = 0.0;
    };

    struct ChatTurn {
        std::string user;
        std::string user_kind;
        std::string user_sent_at;
        std::string user_time_label;
        std::string assistant_preamble;
        std::string assistant;
        std::vector<std::string> versions;
        std::size_t active_version = 0;
        TurnStatus status = TurnStatus::Queued;
        TurnMetrics metrics;
    };

    struct TurnView {
        tuinator::VBox* block = nullptr;
        ChatMessage* user = nullptr;
        ThinkingIndicator* thinking = nullptr;
        AssistantMarkdown* assistant_lead = nullptr;
        ActivityLog* activity = nullptr;
        AssistantMarkdown* assistant = nullptr;
        TurnMetricsNote* metrics = nullptr;
    };

    struct ToolEventUpdate {
        enum class Kind { ToolCall, Observation, ThinkDelta, ThoughtStart, Thought } kind = Kind::ToolCall;
        std::size_t turn_index = 0;
        std::string tool_name;
        std::string args;
        std::string text;
        std::string diff;
        bool is_mcp = false;
        bool is_skill = false;
        bool is_decision = false;
        bool is_ask_user = false;
        std::string skill_name;
        std::string decision_driver;
        std::string mcp_server;
        int64_t duration_ms = -1;
    };

    struct TerminalUpdate {
        enum class Kind { Done, Error } kind;
        std::size_t turn_index = 0;
        std::string text;
        std::string answer;
    };

    struct QuestionPromptUpdate {
        std::string request_id;
        std::vector<QuestionComposer::QuestionInfo> questions;
    };

    void submit_message(const std::string& text);
    void append_local_reply(const std::string& user, const std::string& assistant);
    void handle_skills_command();
    void handle_decisions_command();
    void handle_compact_command();
    void enqueue_message(const std::string& text);
    void pump_worker();
    void run_job(std::size_t turn_index, const std::string& goal);
    void append_assistant_delta(std::size_t turn_index, const std::string& delta);
    void set_footer_status(const std::string& text);
    void refresh_queue_ui(bool force = false);
    void drop_front_queued();
    void promote_front_queued();
    void redo_turn(std::size_t turn_index);
    void copy_turn_response(std::size_t turn_index);
    void step_turn_version(std::size_t turn_index, int delta);
    void start_regenerate(std::size_t turn_index);
    void ensure_turn_widgets(std::size_t count);
    void sync_turn_view(std::size_t index, const ChatTurn& turn, bool worker_busy, bool has_active_turn,
                        std::size_t active_turn);
    void refresh_turn_view(std::size_t index);
    void refresh_chat_log();
    void refresh_active_turn_stream();
    void request_chat_layout_refresh();
    void flush_chat_layout(bool scroll_to_bottom = false);
    void update_chat_wrap_widths();
    void tick_ui_animations();
    bool ui_needs_animation_frame() const;
    void sync_scroll_follow();
    void scroll_chat_to_bottom(bool force = false);
    void focus_input();
    void show_question_prompt(const QuestionPromptUpdate& prompt);
    void hide_question_prompt();
    void submit_question_answers(const std::string& request_id, std::vector<std::vector<std::string>> answers);
    void reject_question_prompt(const std::string& request_id);
    void sync_workspace_layout();
    bool apply_pending_updates();
    void on_job_finished(std::size_t turn_index, bool ok, const std::string& error, const std::string& answer);
    void refresh_session_sidebar();
    void refresh_session_nav();
    void switch_to_session(const std::string& project_root, const std::string& session_id);
    void delete_session(const std::string& project_root, const std::string& session_id);
    void show_session_context_menu(const tuinator::Point& position, const std::string& project_root,
                                   const std::string& session_id);
    void create_new_session(const std::string& project_root = "");
    void reload_turns_from_backend();
    void clear_turn_view(std::size_t index);
    void refresh_context_sidebar();
    void toggle_context_usage_panel();
    void apply_composer_session_prefs(const MessageComposer& composer);
    MessageComposerHostContext build_message_composer_context() const;
    void refresh_chat_header();
    void sync_empty_state();
    void trigger_empty_quick_action(int index);
    void refresh_status_footer();
    void refresh_input_model_info();
    void refresh_mcp_sidebar();
    void refresh_skills_sidebar();
    void refresh_decisions_sidebar();
    void sync_activity_handles();
    void open_settings_modal();
    void refresh_appearance_prefs();
    void apply_message_time_prefs(bool enabled, const std::string& clock_pref);
    void apply_sticky_user_message_pref(bool enabled);
    const ChatMessage* resolve_sticky_user_message() const;
    const ChatMessage* sticky_user_message_candidate() const;
    int user_message_content_y(const ChatMessage& user) const;
    std::string user_time_label_for_turn(const ChatTurn& turn) const;
    std::string assistant_author_label() const;
    void apply_assistant_author_labels(TurnView& view) const;
    void refresh_user_time_labels_on_screen();
    void refresh_user_message_chrome();
    void refresh_speech_input_prefs();
    void toggle_speech_input();
    void apply_ui_theme(const std::string& theme_id);
    void start_background_init();
    void poll_background_init();
    void poll_runtime_ready();
    void poll_mcp_after_ready();
    void apply_bootstrap_handle(uint64_t handle, const std::string& error);
    void apply_runtime_ready();

    static void on_delta(const char* delta, void* user_data);
    static void on_done(int ok, const char* error, const char* answer, void* user_data);
    static void on_event(const char* event_json, void* user_data);
    static bool parse_tool_event(const std::string& json, std::size_t turn_index, ToolEventUpdate& update);
    void apply_tool_event(const ToolEventUpdate& update);

    tuinator::Application& app_;
    tuinator::Theme theme_;
    int chat_max_width_cols_ = 0;
    bool message_timestamps_enabled_ = true;
    bool sticky_user_message_enabled_ = true;
    std::string key_open_theme_picker_ = "Ctrl+T";
    std::string current_ui_theme_ = "dark";
    int last_chat_scroll_y_ = -1;
    MessageTimeClock message_time_clock_ = MessageTimeClock::Hour24;
    tuinator::Style thinking_style_;
    tuinator::Style input_style_;
    tuinator::Style input_focused_style_;

    tuinator::VBox* chat_log_ = nullptr;
    std::string chat_model_label_;
    tuinator::ScrollView* chat_scroll_ = nullptr;
    tuinator::Widget* main_column_ = nullptr;
    SessionSidebar* session_nav_ = nullptr;
    SessionContextMenu* session_context_menu_ = nullptr;
    SettingsModal* settings_modal_ = nullptr;
    int last_terminal_width_ = 0;
    int last_terminal_height_ = 0;
    ChatTextInput* input_ = nullptr;
    MessageComposer* message_composer_ = nullptr;
    SpeechMicButton* speech_mic_ = nullptr;
    QuestionComposer* question_composer_ = nullptr;
    ComposerSwitch* composer_switch_ = nullptr;
    ChatSessionHeader* chat_header_ = nullptr;
    ChatEmptyState* empty_state_ = nullptr;
    ChatMiddlePane* middle_pane_ = nullptr;
    ContextPanel* context_panel_ = nullptr;
    McpServerList* mcp_list_ = nullptr;
    SidebarKvRow* skills_row_ = nullptr;
    SidebarKvRow* decisions_row_ = nullptr;
    SidebarKvRow* lsp_row_ = nullptr;
    SidebarFooter* sidebar_footer_ = nullptr;
    bool context_show_cost_ = true;
    double last_gen_tok_per_sec_ = 0.0;
    QueueStatusBar* queue_bar_ = nullptr;
    StatusFooter* status_footer_ = nullptr;

    std::vector<TurnView> turn_views_;

    uint64_t taiji_handle_ = 0;
    std::thread init_thread_;
    std::atomic<bool> init_finished_{false};
    bool init_joined_ = false;
    uint64_t init_handle_ = 0;
    std::string init_error_;
    std::mutex init_mu_;
    bool runtime_ready_applied_ = false;
    std::chrono::steady_clock::time_point runtime_ready_at_{};
    std::thread worker_;
    std::thread compact_thread_;
    std::atomic<bool> compact_busy_{false};
    std::thread stt_thread_;
    std::atomic<bool> stt_busy_{false};
    std::atomic<bool> worker_busy_{false};
    std::atomic<std::size_t> active_turn_index_{0};
    std::atomic<bool> has_active_turn_{false};

    std::mutex state_mu_;
    std::vector<ChatTurn> turns_;
    std::deque<std::size_t> job_queue_;
    std::string pending_stream_delta_;
    std::size_t pending_stream_turn_ = 0;
    bool has_pending_stream_reset_ = false;
    std::size_t pending_stream_reset_turn_ = 0;
    std::string pending_stream_reset_text_;
    std::deque<ToolEventUpdate> pending_tool_events_;
    bool has_pending_terminal_ = false;
    TerminalUpdate pending_terminal_{};
    bool has_pending_question_prompt_ = false;
    QuestionPromptUpdate pending_question_prompt_{};
    bool question_prompt_active_ = false;
    std::unordered_map<std::size_t, std::string> pending_ask_user_args_;

    std::string session_label_;
    std::string footer_text_;
    std::string last_queue_preview_;
    std::size_t last_queue_extra_ = 0;
    std::string last_footer_status_;
    bool chat_layout_dirty_ = false;
    bool follow_chat_bottom_ = true;
    std::chrono::steady_clock::time_point last_stream_ui_refresh_{};
    std::chrono::steady_clock::time_point last_queue_ui_refresh_{};
    std::optional<std::size_t> pending_regenerate_turn_;

    struct PendingSttUpdate {
        bool recording = false;
        std::string text;
        std::string error;
    };
    bool has_pending_stt_ = false;
    PendingSttUpdate pending_stt_{};
    bool stt_enabled_ = true;
    std::string stt_input_base_;
    bool has_pending_stt_partial_ = false;
    std::string pending_stt_partial_;

    struct PendingCompactUpdate {
        bool ok = false;
        std::string error;
    };
    bool has_pending_compact_ = false;
    PendingCompactUpdate pending_compact_{};
};
