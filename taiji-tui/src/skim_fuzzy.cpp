#include "skim_fuzzy.hpp"

#include <taiji_fuzzy.h>

#include <cstddef>

std::vector<int> skim_fuzzy_filter_indices(const std::string& query,
                                           const std::vector<std::string>& lines,
                                           const std::vector<std::string>& meta,
                                           const std::vector<std::string>& provider_ids) {
    const std::size_t n = lines.size();
    if (n == 0) {
        return {};
    }

    std::vector<taiji_fuzzy_item> items(n);
    for (std::size_t i = 0; i < n; ++i) {
        items[i].line = lines[i].c_str();
        items[i].meta = i < meta.size() ? meta[i].c_str() : "";
        items[i].provider_id = i < provider_ids.size() ? provider_ids[i].c_str() : "";
    }

    std::vector<int> out(n);
    const int written = taiji_fuzzy_filter(query.c_str(), items.data(), n, out.data(), n);
    if (written < 0) {
        out.resize(n);
        for (std::size_t i = 0; i < n; ++i) {
            out[i] = static_cast<int>(i);
        }
        return out;
    }
    out.resize(static_cast<std::size_t>(written));
    return out;
}
