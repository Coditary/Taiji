#pragma once

#include <tuinator/render/terminal_image.hpp>
#include <tuinator/tuinator.hpp>

class ComposerImageModal : public tuinator::Widget {
  public:
    explicit ComposerImageModal(tuinator::Theme theme);

    bool is_open() const { return open_; }
    void open(const tuinator::TerminalImage& image, const std::string& caption);
    void close();

    std::string_view widget_type_name() const override { return "ComposerImageModal"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;
    tuinator::Widget* hit_test_focusable(tuinator::Point point) override;
    bool captures_keyboard() const override { return open_; }

  private:
    static tuinator::Size display_cells_for(const tuinator::TerminalImage& image, int max_cols, int max_rows);
    tuinator::Rect image_bounds() const;

    tuinator::Theme theme_;
    bool open_ = false;
    tuinator::TerminalImage image_;
    std::string caption_;
    tuinator::Rect image_bounds_{};
};
