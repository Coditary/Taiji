#include "mcp_sidebar.hpp"

#include "left_label.hpp"

tuinator::Style mcp_dot_style() {
    tuinator::Style style;
    style.foreground_rgb = tuinator::rgb(74, 222, 128);
    return style;
}

tuinator::Style mcp_connected_style() {
    tuinator::Style style;
    style.foreground_rgb = tuinator::rgb(140, 133, 155);
    return style;
}

std::unique_ptr<tuinator::HBox> make_mcp_server_row(const std::string& name, const tuinator::Theme& theme) {
    auto row = std::make_unique<tuinator::HBox>(tuinator::BoxOptions{.gap = 1, .padding = 0});
    row->add_child(std::make_unique<LeftLabel>("●", mcp_dot_style()));
    row->add_child(std::make_unique<LeftLabel>(name, theme.label));
    auto spacer = std::make_unique<LeftLabel>("", theme.label);
    spacer->set_flex(1);
    row->add_child(std::move(spacer));
    row->add_child(std::make_unique<LeftLabel>("Connected", mcp_connected_style()));
    return row;
}
