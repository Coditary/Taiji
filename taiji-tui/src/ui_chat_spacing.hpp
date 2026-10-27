#pragma once

#include <string>

int ui_activity_spacing_lines();
int ui_message_spacing_lines();
void ui_set_chat_spacing_lines(int activity_lines, int message_lines);

int ui_clamp_spacing_lines(int lines);

bool ui_activity_show_pipeline_title();
void ui_set_activity_show_pipeline_title(bool show);

bool ui_tool_expand_show_parameters();
bool ui_tool_expand_show_result();
std::string ui_tool_expand_args_format();
std::string ui_tool_expand_result_format();
void ui_set_tool_expand_display(bool show_parameters, bool show_result, const std::string& args_format,
                                const std::string& result_format);
