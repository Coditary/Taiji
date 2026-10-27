#include "ui_chat_spacing.hpp"

#include <algorithm>
#include <cctype>
#include <string>

namespace {

int g_activity_spacing_lines = 1;
int g_message_spacing_lines = 1;
bool g_activity_show_pipeline_title = true;
bool g_tool_expand_show_parameters = true;
bool g_tool_expand_show_result = true;
std::string g_tool_expand_args_format = "pretty";
std::string g_tool_expand_result_format = "text";

std::string lowercase_copy(std::string value) {
    for (char& ch : value) {
        ch = static_cast<char>(std::tolower(static_cast<unsigned char>(ch)));
    }
    return value;
}

} // namespace

int ui_clamp_spacing_lines(int lines) { return std::clamp(lines, 0, 4); }

int ui_activity_spacing_lines() { return g_activity_spacing_lines; }

int ui_message_spacing_lines() { return g_message_spacing_lines; }

void ui_set_chat_spacing_lines(int activity_lines, int message_lines) {
    g_activity_spacing_lines = ui_clamp_spacing_lines(activity_lines);
    g_message_spacing_lines = ui_clamp_spacing_lines(message_lines);
}

bool ui_activity_show_pipeline_title() { return g_activity_show_pipeline_title; }

void ui_set_activity_show_pipeline_title(bool show) { g_activity_show_pipeline_title = show; }

bool ui_tool_expand_show_parameters() { return g_tool_expand_show_parameters; }

bool ui_tool_expand_show_result() { return g_tool_expand_show_result; }

std::string ui_tool_expand_args_format() { return g_tool_expand_args_format; }

std::string ui_tool_expand_result_format() { return g_tool_expand_result_format; }

void ui_set_tool_expand_display(bool show_parameters, bool show_result, const std::string& args_format,
                                const std::string& result_format) {
    g_tool_expand_show_parameters = show_parameters;
    g_tool_expand_show_result = show_result;
    const std::string args_fmt = lowercase_copy(args_format);
    g_tool_expand_args_format = (args_fmt == "json" || args_fmt == "raw") ? "json" : "pretty";
    const std::string result_lower = lowercase_copy(result_format);
    g_tool_expand_result_format = result_lower == "json" ? "json" : "text";
}
