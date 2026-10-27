#include "activity_trace.hpp"

namespace {

ActivityTraceMode g_activity_trace_mode = ActivityTraceMode::Subtree;

} // namespace

ActivityTraceMode activity_trace_mode_from_pref(const std::string& pref) {
    if (pref == "inline") {
        return ActivityTraceMode::Inline;
    }
    if (pref == "clustered" || pref == "compact") {
        return ActivityTraceMode::Clustered;
    }
    if (pref == "ignore") {
        return ActivityTraceMode::Ignore;
    }
    return ActivityTraceMode::Subtree;
}

const char* activity_trace_mode_to_pref(ActivityTraceMode mode) {
    switch (mode) {
    case ActivityTraceMode::Inline:
        return "inline";
    case ActivityTraceMode::Clustered:
        return "clustered";
    case ActivityTraceMode::Ignore:
        return "ignore";
    case ActivityTraceMode::Subtree:
    default:
        return "subtree";
    }
}

void ui_set_activity_trace_mode(ActivityTraceMode mode) { g_activity_trace_mode = mode; }

ActivityTraceMode ui_activity_trace_mode() { return g_activity_trace_mode; }
