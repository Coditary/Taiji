#include "activity_diff_layout.hpp"

namespace {

FileDiffLayout g_file_diff_layout = FileDiffLayout::Stacked;

} // namespace

FileDiffLayout file_diff_layout_from_pref(const std::string& pref) {
    if (pref == "side_by_side" || pref == "side-by-side" || pref == "split") {
        return FileDiffLayout::SideBySide;
    }
    return FileDiffLayout::Stacked;
}

const char* file_diff_layout_to_pref(FileDiffLayout layout) {
    switch (layout) {
    case FileDiffLayout::SideBySide:
        return "side_by_side";
    case FileDiffLayout::Stacked:
    default:
        return "stacked";
    }
}

void ui_set_file_diff_layout(FileDiffLayout layout) { g_file_diff_layout = layout; }

FileDiffLayout ui_file_diff_layout() { return g_file_diff_layout; }
