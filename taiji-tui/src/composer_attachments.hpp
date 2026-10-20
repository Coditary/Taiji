#pragma once

#include <tuinator/render/theme.hpp>

#include <string>
#include <vector>

enum class ComposerChipKind {
    AgentMode,
    Skill,
    File,
    GitDiff,
    Clipboard,
    Terminal,
    McpFilter,
    Thinking,
};

struct ComposerChip {
    ComposerChipKind kind = ComposerChipKind::AgentMode;
    std::string id;
    std::string label;
    std::string payload;
    tuinator::Rgb color{169, 177, 214};
};

std::string composer_build_outgoing_message(const std::string& user_text, const std::vector<ComposerChip>& chips);
