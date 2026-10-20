#pragma once

#include <tuinator/tuinator.hpp>

#include <chrono>
#include <cstdint>
#include <string>
#include <vector>

/// Chronological, collapsible record of agent activity for one chat turn.
/// Each entry is a header line (chevron + status icon + scope + name +
/// duration) with an expandable body: thought text for thinking blocks,
/// parameters + result for tool/MCP calls. Clicking the header toggles it.
class ActivityLog : public tuinator::Widget {
  public:
    ActivityLog(tuinator::Style active_style, tuinator::Style muted_style);

    void clear();
    /// Stops all running entries (turn finished or was cancelled).
    void finish_all();

    void begin_thinking();
    void append_thinking(const std::string& delta);
    /// Complete thought block from the backend; ignored when deltas streamed.
    void add_thought(const std::string& text);
    void end_thinking();

    void start_tool(const std::string& tool_name, const std::string& args, bool is_mcp, bool is_skill,
                    const std::string& skill_name, const std::string& mcp_server, bool is_decision = false,
                    const std::string& decision_driver = "", bool is_ask_user = false);
    void finish_tool(const std::string& tool_name, const std::string& text, const std::string& diff,
                     int64_t duration_ms, const std::string& skill_name = "", bool is_decision = false,
                     const std::string& decision_driver = "", bool is_ask_user = false);
    void finish_question(const std::string& args, const std::string& text, int64_t duration_ms);

    bool has_entries() const { return !entries_.empty(); }
    bool has_running() const;
    bool has_live_spinner() const;
    std::string running_label() const;
    void tick(int dt_ms);
    bool needs_animation_tick() const;
    void set_wrap_width(int width);
    void set_taiji_handle(uint64_t handle);
    void set_pipeline_title(std::string title);
    void set_generating_tail(bool show, std::string label = "Generating response…");

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;

  private:
    enum class Scope { Thinking, Tool, MCP, Skill, Decision, Question };

    struct BodyLine {
        std::string text;
        tuinator::Style style;
        bool bar = true;
    };

    struct Entry {
        Scope scope = Scope::Tool;
        bool running = false;
        bool failed = false;
        bool expanded = false;
        bool streamed = false;
        std::string name;
        std::string server;
        std::string skill_name;
        std::string decision_driver;
        std::string question_header;
        std::string args;
        std::string body;
        bool spacer_after = false;
        std::vector<BodyLine> diff;
        bool has_duration = false;
        int64_t duration_ms = 0;
        std::chrono::steady_clock::time_point started_at{};
        int settle_ms = 0;
        int flash_ms = 0;
    };

    static std::string scope_label(Scope scope);
    static std::string format_duration(int64_t ms);
    static std::vector<BodyLine> parse_diff(const std::string& diff, tuinator::Style muted, tuinator::Style add,
                                            tuinator::Style del);

    int wrap_cols() const;
    int body_wrap_cols() const;
    int64_t elapsed_ms(const Entry& entry) const;
    std::string entry_title(const Entry& entry) const;
    std::string header_text(const Entry& entry) const;
    std::vector<BodyLine> body_lines(const Entry& entry) const;
    std::vector<BodyLine> thinking_body_lines(const Entry& entry) const;
    std::vector<BodyLine> tool_body_lines(const Entry& entry) const;
    std::vector<BodyLine> decision_body_lines(const Entry& entry) const;
    std::vector<BodyLine> question_body_lines(const Entry& entry) const;
    int entry_height(const Entry& entry) const;
    int total_lines() const;
    Entry* running_thinking_entry();
    void finalize_entry(Entry& entry);
    const char* spinner_frame() const;

    tuinator::Style active_style_;
    tuinator::Style done_style_;
    tuinator::Style failed_style_;
    tuinator::Style failed_body_style_;
    tuinator::Style muted_style_;
    tuinator::Style body_style_;
    tuinator::Style thinking_body_style_;
    tuinator::Style heading_style_;
    tuinator::Style add_style_;
    tuinator::Style del_style_;
    tuinator::Style bar_style_;

    std::string tool_display_title(const std::string& tool_name) const;

    std::vector<Entry> entries_;
    bool received_think_deltas_ = false;
    int spinner_ms_ = 0;
    int wrap_width_hint_ = 0;
    uint64_t taiji_handle_ = 0;
    std::string pipeline_title_;
    bool generating_tail_ = false;
    std::string generating_label_ = "Generating response…";
    // bounds_ is inherited from tuinator::Widget (hit-testing relies on it).
};
