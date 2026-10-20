#pragma once

#include <tuinator/tuinator.hpp>

#include <string>

class ChatSessionHeader : public tuinator::Widget {
  public:
    explicit ChatSessionHeader(tuinator::Theme theme);

    void set_project_name(std::string name);
    void set_session_name(std::string name);

    std::string_view widget_type_name() const override { return "ChatSessionHeader"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;

  private:
    tuinator::Theme theme_;
    std::string project_;
    std::string session_;
};
