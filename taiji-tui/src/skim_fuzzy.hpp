#pragma once

#include <string>
#include <vector>

/** Rank row indices with skim (SkimMatcherV2) for the settings model picker. */
std::vector<int> skim_fuzzy_filter_indices(const std::string& query,
                                           const std::vector<std::string>& lines,
                                           const std::vector<std::string>& meta,
                                           const std::vector<std::string>& provider_ids);
