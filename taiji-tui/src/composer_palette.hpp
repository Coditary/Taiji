#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <string>
#include <vector>

struct ComposerPaletteRow {
    std::string id;
    std::string section;
    std::string icon;
    std::string title;
    std::string subtitle;
    std::string hint;
    bool submenu = false;
    bool checked = false;
};

class ComposerPalette : public tuinator::Widget {
  public:
    using SelectHandler = std::function<void(const std::string& row_id)>;

    explicit ComposerPalette(tuinator::Theme theme);

    void set_rows(std::vector<ComposerPaletteRow> rows);
    void set_filter(std::string query);
    const std::string& filter() const { return filter_; }
    void set_on_select(SelectHandler handler);
    void set_on_close(std::function<void()> handler);

    bool is_open() const { return open_; }
    void open();
    void close();

    void tick(int dt_ms);

    std::string_view widget_type_name() const override { return "ComposerPalette"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;
    bool captures_keyboard() const override { return open_; }

  private:
    void rebuild_visible();
    int panel_height() const;
    tuinator::Rect main_panel_rect() const;
    tuinator::Rect flyout_panel_rect() const;

    tuinator::Theme theme_;
    std::vector<ComposerPaletteRow> rows_;
    std::vector<int> visible_;
    std::string filter_;
    int selected_ = 0;
    int flyout_parent_ = -1;
    bool open_ = false;
    int open_anim_ms_ = 0;
    SelectHandler on_select_;
    std::function<void()> on_close_;
};
