#pragma once

#include <string>

/** Human-readable model name; backend id (e.g. ollama/foo.gguf) stays unchanged elsewhere. */
inline std::string format_model_display_label(const std::string& model) {
    std::string out = model;
    static constexpr char kOllamaPrefix[] = "ollama/";
    if (out.size() > sizeof(kOllamaPrefix) - 1 &&
        out.compare(0, sizeof(kOllamaPrefix) - 1, kOllamaPrefix) == 0) {
        out = out.substr(sizeof(kOllamaPrefix) - 1);
    }
    static constexpr char kGgufSuffix[] = ".gguf";
    if (out.size() > sizeof(kGgufSuffix) - 1 &&
        out.compare(out.size() - (sizeof(kGgufSuffix) - 1), sizeof(kGgufSuffix) - 1, kGgufSuffix) == 0) {
        out.resize(out.size() - (sizeof(kGgufSuffix) - 1));
    }
    return out;
}
