#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <string>
#include <vector>

class SessionContextMenu : public tuinator::Widget {
  public:
    struct Item {
        std::string label;
        std::function<void()> action;
    };

    explicit SessionContextMenu(tuinator::Theme theme);

    void set_items(std::vector<Item> items);
    void set_on_open_changed(std::function<void(bool open)> callback);
    void show(tuinator::Point position);
    void close();
    bool is_open() const { return open_; }

    std::string_view widget_type_name() const override { return "SessionContextMenu"; }
    tuinator::Size preferred_size() const override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;
    bool wants_full_screen() const override { return true; }
    bool captures_pointer() const override { return open_; }
    bool captures_keyboard() const override { return open_; }
    bool wants_hover() const override { return open_; }

  private:
    struct PanelLayout {
        int x = 0;
        int y = 0;
        int width = 0;
        int height = 0;
    };

    tuinator::Point to_local(tuinator::Point point) const;
    PanelLayout panel_layout() const;
    tuinator::Rect panel_bounds() const;
    bool hit_panel(tuinator::Point local, int& row) const;
    void set_active_item(int index);
    void mark_panel_dirty();
    void activate_item(int index);

    tuinator::Theme theme_;
    std::vector<Item> items_;
    tuinator::Point anchor_{};
    int active_item_ = 0;
    bool open_ = false;
    std::function<void(bool open)> on_open_changed_;
};
