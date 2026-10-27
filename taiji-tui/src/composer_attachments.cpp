#include "composer_attachments.hpp"

#include <tuinator/render/graphics_encode.hpp>

#include <filesystem>
#include <fstream>
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

std::string composer_persist_image_attachment(const ComposerImageAttachment& image, const std::string& work_dir) {
    if (image.image.empty() || work_dir.empty()) {
        return {};
    }

    const std::filesystem::path dir = std::filesystem::path(work_dir) / ".taiji" / "composer-images";
    std::error_code ec;
    std::filesystem::create_directories(dir, ec);
    const std::filesystem::path file = dir / (image.id + ".png");
    const std::vector<std::uint8_t> png = tuinator::rgba_to_png(image.image);
    if (png.empty()) {
        return {};
    }

    std::ofstream out(file, std::ios::binary);
    if (!out) {
        return {};
    }
    out.write(reinterpret_cast<const char*>(png.data()), static_cast<std::streamsize>(png.size()));
    if (!out.good()) {
        return {};
    }
    return file.string();
}

std::string composer_build_outgoing_message(const std::string& user_text, const std::vector<ComposerChip>& chips,
                                            const std::vector<ComposerImageAttachment>& images,
                                            const std::string& work_dir) {
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
        case ComposerChipKind::Context:
            prefix << "[Context: " << chip.label << "]\n";
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

    std::ostringstream image_blocks;
    for (const ComposerImageAttachment& attachment : images) {
        if (attachment.image.empty()) {
            continue;
        }
        if (!image_blocks.str().empty()) {
            image_blocks << "\n\n";
        }
        const std::string saved =
            attachment.source_path.empty() ? composer_persist_image_attachment(attachment, work_dir)
                                           : attachment.source_path;
        image_blocks << "### Image: " << attachment.label << " (" << attachment.image.width() << "×"
                     << attachment.image.height() << ")\n\n";
        if (!saved.empty()) {
            image_blocks << "![[" << attachment.label << "]](" << saved << ")";
        }
    }

    const std::string pre = prefix.str();
    const std::string images_text = image_blocks.str();
    std::string body;
    if (!pre.empty()) {
        body = pre;
    }
    if (!images_text.empty()) {
        if (!body.empty()) {
            body += "\n\n";
        }
        body += images_text;
    }
    if (!user_text.empty()) {
        if (!body.empty()) {
            body += "\n\n";
        }
        body += user_text;
    }
    return body;
}
