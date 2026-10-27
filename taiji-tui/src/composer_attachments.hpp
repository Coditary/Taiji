#pragma once

#include <tuinator/render/terminal_image.hpp>
#include <tuinator/render/theme.hpp>

#include <string>
#include <vector>

enum class ComposerChipKind {
    AgentMode,
    Skill,
    Context,
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

struct ComposerImageAttachment {
    std::string id;
    std::string label;
    std::string source_path;
    tuinator::TerminalImage image;
};

constexpr int kComposerImageThumbCols = 10;
constexpr int kComposerImageThumbRows = 4;
constexpr int kComposerImageChipWidth = kComposerImageThumbCols + 2;
constexpr int kComposerImageChipHeight = kComposerImageThumbRows + 1;

std::string composer_build_outgoing_message(const std::string& user_text, const std::vector<ComposerChip>& chips,
                                            const std::vector<ComposerImageAttachment>& images,
                                            const std::string& work_dir);

std::string composer_persist_image_attachment(const ComposerImageAttachment& image, const std::string& work_dir);
