#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <string>
#include <vector>

class ComposerThemePicker {
  public:
    explicit ComposerThemePicker(tuinator::Theme theme);

    bool is_open() const { return open_; }
    void open(const std::string& active_theme_id, bool draft_message_saved = false);
    void close();

    void set_filter(const std::string& query);
    const std::string& selected_theme_id() const;

    int panel_height(int max_rows = 10) const;
    tuinator::Size preferred_size(int width, int max_rows = 10) const;

    void layout(tuinator::Rect bounds);
    void paint(tuinator::PaintContext& ctx) const;

    bool move_selection(int delta);
    bool move_page(int rows);
    bool select_visible_row(int visible_index);
    bool hit_test_row(tuinator::Point local, int& visible_index) const;

    tuinator::Rect bounds() const { return bounds_; }

  private:
    struct Entry {
        std::string id;
        std::string line;
        std::string meta;
    };

    void rebuild_entries();
    void rebuild_filter();
    void sync_scroll(int max_visible);
    void preview_selection();

    tuinator::Theme theme_;
    bool open_ = false;
    std::vector<Entry> entries_;
    std::vector<int> filtered_;
    std::string filter_;
    int selected_ = 0;
    int list_scroll_ = 0;
    bool draft_message_saved_ = false;
    tuinator::Rect bounds_{};
    std::function<void(const std::string& theme_id)> on_preview_;
    std::function<void(const std::string& theme_id)> on_commit_;

  public:
    void set_on_preview(std::function<void(const std::string& theme_id)> handler) { on_preview_ = std::move(handler); }
    void set_on_commit(std::function<void(const std::string& theme_id)> handler) { on_commit_ = std::move(handler); }
    void commit_selection();
};
