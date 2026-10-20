#pragma once

#include <tuinator/tuinator.hpp>

#include <string>

class SidebarKvRow : public tuinator::Widget {
  public:
    explicit SidebarKvRow(tuinator::Theme theme, std::string title);

    void set_title(std::string title);
    void set_empty_value(std::string text);
    void set_detail(std::string detail);
    void set_collapse_when_empty(bool collapse);

    std::string_view widget_type_name() const override { return "SidebarKvRow"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;

  private:
    bool is_collapsed() const;

    tuinator::Theme theme_;
    std::string title_;
    std::string empty_value_;
    std::string detail_;
    bool collapse_when_empty_ = true;
};
