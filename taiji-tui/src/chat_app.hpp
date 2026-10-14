#pragma once

#include "activity_log.hpp"
#include "assistant_markdown.hpp"
#include "chat_input.hpp"
#include "question_composer.hpp"
#include "chat_message.hpp"
#include "left_label.hpp"
#include "queue_status_bar.hpp"
#include "left_anchored_split.hpp"
#include "right_anchored_split.hpp"
#include "session_context_menu.hpp"
#include "session_sidebar.hpp"
#include "status_footer.hpp"
#include "thinking_indicator.hpp"
#include "turn_metrics_note.hpp"

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
    void refresh_thinking_indicators();
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
    void refresh_status_footer();
    void refresh_input_model_info();
    void refresh_mcp_sidebar();
    void refresh_skills_sidebar();
    void refresh_decisions_sidebar();
    void sync_activity_handles();

    static void on_delta(const char* delta, void* user_data);
    static void on_done(int ok, const char* error, const char* answer, void* user_data);
    static void on_event(const char* event_json, void* user_data);
    static bool parse_tool_event(const std::string& json, std::size_t turn_index, ToolEventUpdate& update);
    void apply_tool_event(const ToolEventUpdate& update);

    tuinator::Application& app_;
    tuinator::Theme theme_;
    tuinator::Style thinking_style_;
    tuinator::Style input_style_;
    tuinator::Style input_focused_style_;

    tuinator::VBox* chat_log_ = nullptr;
    tuinator::ScrollView* chat_scroll_ = nullptr;
    tuinator::Widget* main_column_ = nullptr;
    SessionSidebar* session_nav_ = nullptr;
    SessionContextMenu* session_context_menu_ = nullptr;
    int last_terminal_width_ = 0;
    int last_terminal_height_ = 0;
    ChatTextInput* input_ = nullptr;
    QuestionComposer* question_composer_ = nullptr;
    ComposerSwitch* composer_switch_ = nullptr;
    LeftLabel* session_title_label_ = nullptr;
    LeftLabel* context_tokens_label_ = nullptr;
    LeftLabel* context_used_label_ = nullptr;
    LeftLabel* context_cost_label_ = nullptr;
    LeftLabel* mcp_status_label_ = nullptr;
    LeftLabel* skills_status_label_ = nullptr;
    LeftLabel* decisions_status_label_ = nullptr;
    QueueStatusBar* queue_bar_ = nullptr;
    StatusFooter* status_footer_ = nullptr;

    std::vector<TurnView> turn_views_;

    uint64_t taiji_handle_ = 0;
    std::thread worker_;
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
    std::string footer_text_ = "Shift+Enter: neue Zeile  ·  Enter: senden  ·  Esc: unterbrechen";
    std::string last_queue_preview_;
    std::size_t last_queue_extra_ = 0;
    std::string last_footer_status_;
    bool chat_layout_dirty_ = false;
    bool follow_chat_bottom_ = true;
    std::chrono::steady_clock::time_point last_stream_ui_refresh_{};
    std::chrono::steady_clock::time_point last_queue_ui_refresh_{};
    std::optional<std::size_t> pending_regenerate_turn_;
};
