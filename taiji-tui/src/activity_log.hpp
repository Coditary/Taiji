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

    void start_tool(const std::string& tool_name, const std::string& args, bool is_mcp,
                    const std::string& mcp_server);
    void finish_tool(const std::string& tool_name, const std::string& text, const std::string& diff,
                     int64_t duration_ms);

    bool has_entries() const { return !entries_.empty(); }
    bool has_running() const;
    std::string running_label() const;
    void tick(int dt_ms);
    void set_wrap_width(int width);

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;

  private:
    enum class Scope { Thinking, Tool, MCP, Skill };

    struct BodyLine {
        std::string text;
        tuinator::Style style;
        bool bar = true;
    };

    struct Entry {
        Scope scope = Scope::Tool;
        bool running = false;
        bool expanded = false;
        bool streamed = false;
        std::string name;
        std::string server;
        std::string args;
        std::string body;
        std::vector<BodyLine> diff;
        bool has_duration = false;
        int64_t duration_ms = 0;
        std::chrono::steady_clock::time_point started_at{};
    };

    static std::string scope_label(Scope scope);
    static std::string format_duration(int64_t ms);
    static std::vector<BodyLine> parse_diff(const std::string& diff, tuinator::Style muted, tuinator::Style add,
                                            tuinator::Style del);

    int wrap_cols() const;
    int body_wrap_cols() const;
    int64_t elapsed_ms(const Entry& entry) const;
    std::string header_text(const Entry& entry) const;
    std::vector<BodyLine> body_lines(const Entry& entry) const;
    std::vector<BodyLine> thinking_body_lines(const Entry& entry) const;
    std::vector<BodyLine> tool_body_lines(const Entry& entry) const;
    int entry_height(const Entry& entry) const;
    int total_lines() const;
    Entry* running_thinking_entry();
    void finalize_entry(Entry& entry);
    const char* spinner_frame() const;

    tuinator::Style active_style_;
    tuinator::Style done_style_;
    tuinator::Style muted_style_;
    tuinator::Style body_style_;
    tuinator::Style thinking_body_style_;
    tuinator::Style heading_style_;
    tuinator::Style add_style_;
    tuinator::Style del_style_;
    tuinator::Style bar_style_;

    std::vector<Entry> entries_;
    bool received_think_deltas_ = false;
    int spinner_ms_ = 0;
    int wrap_width_hint_ = 0;
    // bounds_ is inherited from tuinator::Widget (hit-testing relies on it).
};
