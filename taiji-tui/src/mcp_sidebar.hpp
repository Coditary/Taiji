#pragma once

#include <tuinator/layout/box.hpp>
#include <tuinator/render/style.hpp>
#include <tuinator/render/theme.hpp>

#include <memory>
#include <string>

tuinator::Style mcp_dot_style();
tuinator::Style mcp_connected_style();

std::unique_ptr<tuinator::HBox> make_mcp_server_row(const std::string& name, const tuinator::Theme& theme);
