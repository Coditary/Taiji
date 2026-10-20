#pragma once

#include "chat_input.hpp"
#include "composer_attachments.hpp"
#include "composer_palette.hpp"
#include "composer_workspace.hpp"
#include "ui_theme.hpp"

#include <tuinator/tuinator.hpp>

#include <cstdint>
#include <functional>
#include <memory>
#include <string>
#include <vector>

struct MessageComposerHostContext {
    uint64_t taiji_handle = 0;
    std::string work_dir;
    GitWorkspaceStatus git{};
    std::vector<std::string> skills;
    std::vector<std::string> mcp_servers;
    std::vector<std::string> mcp_enabled;
    std::string model;
    std::string provider;
};

class MessageComposer : public tuinator::Widget {
  public:
    using SubmitHandler = std::function<void(const std::string& composed_text)>;
    using HostContextProvider = std::function<MessageComposerHostContext()>;

    MessageComposer(tuinator::TextInputOptions input_options, tuinator::Style style, tuinator::Style focused_style,
                    tuinator::Theme theme);

    ChatTextInput* text_input() { return input_.get(); }
    const std::vector<ComposerChip>& chips() const { return chips_; }
    const std::string& pending_chain_of_thought() const { return pending_cot_; }
    const std::vector<std::string>& pending_mcp_enabled() const { return pending_mcp_enabled_; }

    void set_on_submit(SubmitHandler handler);
    void set_host_context(HostContextProvider provider);
    void set_on_palette_open_changed(std::function<void(bool)> handler);

    void tick(int dt_ms);
    void clear_after_send();

    std::string_view widget_type_name() const override { return "MessageComposer"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;
    tuinator::Widget* hit_test_focusable(tuinator::Point point) override;
    void collect_focusable(std::vector<tuinator::Widget*>& out) override;
    bool wants_initial_focus() const override { return true; }
    bool captures_keyboard() const override { return palette_.is_open(); }

  private:
    void open_palette_with_trigger(const std::string& filter);
    void close_palette();
    void rebuild_palette_rows(const std::string& filter);
    void handle_palette_select(const std::string& row_id);
    void add_chip(ComposerChip chip);
    void remove_chip_at(std::size_t index);
    void paint_chips(tuinator::PaintContext& ctx, tuinator::Rect area) const;
    void paint_plus_button(tuinator::PaintContext& ctx, tuinator::Rect area) const;
    bool plus_hit(tuinator::Point local) const;
    tuinator::Rect plus_bounds() const;
    void sync_trigger_from_input();
    std::string read_file_snippet(const std::string& rel_path) const;
    tuinator::Rect stream_bounds(const tuinator::Rect& bounds) const;

    std::unique_ptr<ChatTextInput> input_;
    ComposerPalette palette_;
    std::vector<ComposerChip> chips_;
    HostContextProvider host_context_;
    SubmitHandler on_submit_;
    std::function<void(bool)> on_palette_open_changed_;
    MessageComposerHostContext ctx_cache_;
    std::vector<WorkspaceFileEntry> file_index_;
    bool file_index_built_ = false;
    int plus_anim_ms_ = 0;
    std::string pending_cot_;
    std::vector<std::string> pending_mcp_enabled_;
    tuinator::Rect input_bounds_{};
    tuinator::Rect chips_bounds_{};

    tuinator::Theme theme_;
};
