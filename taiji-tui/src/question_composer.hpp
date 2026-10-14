#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <memory>
#include <string>
#include <vector>

class QuestionComposer : public tuinator::Widget {
  public:
    struct Option {
        std::string label;
        std::string description;
    };

    struct QuestionInfo {
        std::string id;
        std::string header;
        std::string question;
        std::vector<Option> options;
        bool multiple = false;
        bool custom = true;
    };

    QuestionComposer(tuinator::Style style, tuinator::Style focused_style);

    void begin(std::string request_id, std::vector<QuestionInfo> questions);
    void clear();
    bool is_active() const { return active_; }

    void set_wrap_width(int width);
    void set_on_submit(std::function<void(std::string request_id, std::vector<std::vector<std::string>> answers)> callback);
    void set_on_reject(std::function<void(std::string request_id)> callback);

    std::string_view widget_type_name() const override { return "QuestionComposer"; }
    bool is_focusable() const override { return active_; }
    bool wants_initial_focus() const override { return active_; }
    bool captures_keyboard() const override { return active_; }
    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;

  private:
    struct VisibleRow {
        int option_index = -1;
        bool custom = false;
    };

    int option_count() const;
    int custom_index() const;
    const QuestionInfo& current_question() const;
    VisibleRow row_at(int display_row) const;
    int display_row_count() const;
    void rebuild_rows();
    void set_selected_row(int row);
    void toggle_multi_selection(int option_index);
    void pick_single(int option_index, bool custom);
    void advance_or_submit();
    void submit_all();
    void begin_custom_edit();
    void finish_custom_edit(bool keep);
    std::string footer_hint() const;

    tuinator::Style style_;
    tuinator::Style focused_style_;
    std::function<void(std::string, std::vector<std::vector<std::string>>)> on_submit_;
    std::function<void(std::string)> on_reject_;

    std::string request_id_;
    std::vector<QuestionInfo> questions_;
    std::vector<std::vector<std::string>> answers_;
    std::vector<int> multi_selected_;
    std::vector<VisibleRow> visible_rows_;

    std::unique_ptr<tuinator::TextInput> custom_input_;
    int wrap_width_ = 60;
    int current_tab_ = 0;
    int selected_row_ = 0;
    bool active_ = false;
    bool editing_custom_ = false;
};

class ComposerSwitch : public tuinator::Widget {
  public:
    ComposerSwitch(std::unique_ptr<tuinator::Widget> input, std::unique_ptr<tuinator::Widget> question);

    tuinator::Widget* input_widget() const { return input_; }
    tuinator::Widget* question_widget() const { return question_; }

    void show_input();
    void show_question();
    bool showing_question() const { return showing_question_; }

    std::string_view widget_type_name() const override { return "ComposerSwitch"; }
    bool wants_initial_focus() const override;
    bool has_focused_descendant() const override;
    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;
    tuinator::Widget* hit_test_focusable(tuinator::Point point) override;
    void collect_focusable(std::vector<tuinator::Widget*>& out) override;

  private:
    tuinator::Widget* active_child() const;

    tuinator::Widget* input_ = nullptr;
    tuinator::Widget* question_ = nullptr;
    bool showing_question_ = false;
};

bool parse_user_prompt_event(const std::string& json, std::string& request_id,
                             std::vector<QuestionComposer::QuestionInfo>& questions);

std::string encode_question_answers_json(const std::vector<std::vector<std::string>>& answers);
