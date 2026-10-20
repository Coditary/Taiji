#include "composer_attachments.hpp"

#include <sstream>

namespace {

void append_block(std::ostringstream& out, const std::string& title, const std::string& body) {
    if (body.empty()) {
        return;
    }
    if (!out.str().empty()) {
        out << "\n\n";
    }
    out << "### " << title << "\n\n" << body;
}

} // namespace

std::string composer_build_outgoing_message(const std::string& user_text, const std::vector<ComposerChip>& chips) {
    std::ostringstream prefix;
    for (const ComposerChip& chip : chips) {
        switch (chip.kind) {
        case ComposerChipKind::AgentMode:
            prefix << "[Mode: " << chip.label << "]\n";
            if (chip.id == "plan") {
                prefix << "Plan-only: outline steps and architecture before making file edits.\n";
            } else if (chip.id == "debug") {
                prefix << "Debug mode: focus on stack traces, build output, and root cause.\n";
            } else if (chip.id == "ask") {
                prefix << "Ask mode: answer questions read-only; avoid file-edit tools unless asked.\n";
            }
            break;
        case ComposerChipKind::Skill:
            prefix << "[Skill: " << chip.label << "]\n";
            break;
        case ComposerChipKind::File:
            prefix << "[File: " << chip.label << "]\n";
            break;
        case ComposerChipKind::GitDiff:
            prefix << "[Git diff attached]\n";
            break;
        case ComposerChipKind::Clipboard:
            prefix << "[Clipboard";
            if (!chip.label.empty()) {
                prefix << ": " << chip.label;
            }
            prefix << "]\n";
            break;
        case ComposerChipKind::Terminal:
            prefix << "[Build output attached]\n";
            break;
        case ComposerChipKind::McpFilter:
            prefix << "[MCP: " << chip.label << "]\n";
            break;
        case ComposerChipKind::Thinking:
            prefix << "[Thinking: " << chip.label << "]\n";
            break;
        }
    }

    for (const ComposerChip& chip : chips) {
        switch (chip.kind) {
        case ComposerChipKind::GitDiff:
        case ComposerChipKind::Clipboard:
        case ComposerChipKind::Terminal:
        case ComposerChipKind::File:
            append_block(prefix, chip.label.empty() ? chip.id : chip.label, chip.payload);
            break;
        default:
            break;
        }
    }

    const std::string pre = prefix.str();
    if (pre.empty()) {
        return user_text;
    }
    if (user_text.empty()) {
        return pre;
    }
    return pre + "\n\n" + user_text;
}
