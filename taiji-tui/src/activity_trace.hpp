#pragma once

#include <cstdint>
#include <string>

enum class ActivityTraceMode : std::uint8_t { Inline, Subtree, Clustered, Ignore };

ActivityTraceMode activity_trace_mode_from_pref(const std::string& pref);
const char* activity_trace_mode_to_pref(ActivityTraceMode mode);

void ui_set_activity_trace_mode(ActivityTraceMode mode);
ActivityTraceMode ui_activity_trace_mode();
