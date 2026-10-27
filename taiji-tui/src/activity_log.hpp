#pragma once

#include "activity_diff_layout.hpp"
#include "activity_trace.hpp"
#include "ui_chat_column.hpp"

#include <tuinator/tuinator.hpp>

#include <chrono>
#include <cstdint>
#include <string>
#include <unordered_map>
#include <vector>

struct ActivityDiffTextSegment {
    std::string text;
    bool highlight = false;
};

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
    void set_trace_mode(ActivityTraceMode mode);
    void set_file_diff_layout(FileDiffLayout layout);
    void set_layout_row_gap(int lines);
    ActivityTraceMode trace_mode() const { return trace_mode_; }
    /// Scroll-content coordinates (see ChatApp::terminal_to_chat_content). Returns true if repaint needed.
    bool update_header_hover_at_content_point(const tuinator::Point& content_point);
    bool clear_header_hover();

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;

  private:
    enum class Scope { Thinking, Tool, MCP, Skill, Decision, Question, FileDiff };

    enum class BodyLineKind : std::uint8_t { Normal, DiffHeader, DiffHunk, DiffSplit };

    struct BodyLine {
        std::string text;
        tuinator::Style style;
        bool bar = true;
        BodyLineKind kind = BodyLineKind::Normal;
        bool diff_fill_row = false;
        std::vector<ActivityDiffTextSegment> diff_segments;
        std::vector<ActivityDiffTextSegment> diff_right_segments;
        std::string diff_icon;
        tuinator::Rgb diff_icon_rgb{};
        int diff_adds = 0;
        int diff_dels = 0;
        int diff_old_line = 0;
        int diff_new_line = 0;
        std::string diff_right_text;
        tuinator::Style diff_right_style;
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
        std::string diff_source;
        std::string diff_path_hint;
        bool file_diff_view = false;
        bool has_duration = false;
        int64_t duration_ms = 0;
        std::chrono::steady_clock::time_point started_at{};
        int settle_ms = 0;
        int flash_ms = 0;
        mutable bool body_cache_valid = false;
        mutable int body_cache_wrap_cols = 0;
        mutable std::vector<BodyLine> body_cache;
    };

    void invalidate_body_cache(Entry& entry);
    void invalidate_all_body_caches();

    static std::string scope_label(Scope scope);
    static std::string format_duration(int64_t ms);
    static std::vector<BodyLine> parse_diff(const std::string& diff, tuinator::Style muted, tuinator::Style add,
                                            tuinator::Style del);
    static std::vector<BodyLine> parse_file_diff_view(const std::string& diff, const std::string& path_hint,
                                                      tuinator::Style muted, tuinator::Style add, tuinator::Style del,
                                                      bool* file_diff_view);
    static void append_file_diff_side_by_side_body(const std::string& diff, std::vector<BodyLine>& lines,
                                                   tuinator::Style muted, tuinator::Style add, tuinator::Style del,
                                                   bool new_file_hunk);
    void rebuild_file_diff_entries();

    int wrap_cols() const;
    int body_wrap_cols() const;
    int64_t elapsed_ms(const Entry& entry) const;
    std::string entry_title(const Entry& entry) const;
    std::string header_text(const Entry& entry) const;
    std::vector<BodyLine> body_lines(const Entry& entry) const;
    std::vector<BodyLine> thinking_body_lines(const Entry& entry) const;
    std::vector<BodyLine> tool_body_lines(const Entry& entry) const;
    std::vector<BodyLine> file_diff_body_lines(const Entry& entry) const;
    std::vector<BodyLine> decision_body_lines(const Entry& entry) const;
    std::vector<BodyLine> question_body_lines(const Entry& entry) const;
    int entry_height(const Entry& entry) const;
    int total_lines() const;
    Entry* running_thinking_entry();
    void finalize_entry(Entry& entry);
    const char* spinner_frame() const;
    bool entry_has_expandable_body(const Entry& entry) const;
    bool entry_header_expandable(const Entry& entry) const;
    int header_entry_index_at_local_y(int local_y) const;
    void paint_header_expand_affordance(tuinator::PaintContext& ctx, int y, const Entry& entry,
                                        std::size_t entry_index, tuinator::Rgb row_bg, int chevron_x) const;

    enum class LayoutRowKind : std::uint8_t { Entry, MundaneCluster };

    struct LayoutRow {
        LayoutRowKind kind = LayoutRowKind::Entry;
        std::size_t entry_index = 0;
        std::size_t mundane_begin = 0;
        std::size_t mundane_end = 0;
    };

    bool entry_is_prominent(const Entry& entry) const;
    bool entry_followed_by_file_diff(std::size_t entry_index) const;
    bool entry_visible_when_ignoring_tool_rows(std::size_t entry_index) const;
    bool mundane_cluster_is_open(std::size_t run_begin) const;
    void set_mundane_cluster_open(std::size_t run_begin, bool open);
    std::vector<LayoutRow> build_layout() const;
    bool has_visible_layout_rows() const;
    bool should_show_pipeline_title() const;
    bool should_show_generating_tail() const;
    int layout_row_height(const LayoutRow& row, const std::vector<LayoutRow>& layout, std::size_t layout_index) const;
    int y_for_layout_header_line(int local_y, int* header_entry_index, std::size_t* cluster_run_begin) const;
    std::string mundane_cluster_summary(std::size_t begin, std::size_t end) const;
    void paint_entry_block(tuinator::PaintContext& ctx, int y, std::size_t entry_index, const ChatStreamLayout& stream,
                           tuinator::Rgb row_bg, int extra_tree_indent, const char* branch_override = nullptr,
                           bool more_layout_rows_after = false) const;
    void paint_mundane_cluster_header(tuinator::PaintContext& ctx, int y, std::size_t run_begin, std::size_t run_end,
                                      const ChatStreamLayout& stream, tuinator::Rgb row_bg, bool open,
                                      bool detached_root) const;
    void paint_layout_row_gap(tuinator::PaintContext& ctx, int& y, int stream_x, bool continue_trunk) const;
    bool layout_follows_file_diff(const std::vector<LayoutRow>& layout, std::size_t index) const;

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
    int hovered_entry_index_ = -1;
    ActivityTraceMode trace_mode_ = ActivityTraceMode::Subtree;
    FileDiffLayout file_diff_layout_ = FileDiffLayout::Stacked;
    int layout_row_gap_ = 1;
    std::unordered_map<int, bool> mundane_cluster_open_;
    // bounds_ is inherited from tuinator::Widget (hit-testing relies on it).
};
