#pragma once

#include <tuinator/tuinator.hpp>

#include <cstdint>

#include <string>
#include <vector>

struct SessionTodoItem {
    enum class Status : std::uint8_t { Pending, Active, Done };
    std::string task;
    Status status = Status::Pending;
};

std::vector<SessionTodoItem> parse_session_todos_text(const std::string& text);

class SessionTodoPanel : public tuinator::Widget {
  public:
    explicit SessionTodoPanel(tuinator::Theme theme);

    void set_items(std::vector<SessionTodoItem> items);

    std::string_view widget_type_name() const override { return "SessionTodoPanel"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;

  private:
    tuinator::Theme theme_;
    std::vector<SessionTodoItem> items_;
};
