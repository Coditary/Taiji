#pragma once

#include <tuinator/widgets/controls/text_input.hpp>

#include <string>

class ChatTextInput : public tuinator::TextInput {
  public:
    ChatTextInput(tuinator::TextInputOptions options, tuinator::Style style, tuinator::Style focused_style);

    void set_model_info(std::string model, std::string provider);

    bool wants_initial_focus() const override { return true; }
    void paint(tuinator::PaintContext& ctx) const override;

  private:
    std::string model_label_;
    std::string provider_label_;
};
