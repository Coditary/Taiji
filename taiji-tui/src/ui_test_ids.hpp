#pragma once

#include <string>
#include <string_view>

namespace taiji::ui::test_id {

inline constexpr const char* kAppShell = "app.shell";

inline constexpr const char* kChatInputBar = "chat.input-bar";
inline constexpr const char* kComposerRoot = "composer";
inline constexpr const char* kComposerPromptInput = "composer.prompt.input";
inline constexpr const char* kComposerPalette = "composer.palette";
inline constexpr const char* kComposerThemePicker = "composer.theme-picker";
inline constexpr const char* kComposerActorPicker = "composer.actor-picker";
inline constexpr const char* kComposerPlus = "composer.prompt.plus";
inline constexpr const char* kSpeechMic = "composer.speech-mic";

inline constexpr const char* kSessionSidebar = "session.sidebar";
inline constexpr const char* kSessionSidebarSettings = "session.sidebar.settings";
inline constexpr const char* kSessionSidebarAddFolder = "session.sidebar.add-folder";

inline constexpr const char* kContextPanel = "context.panel";
inline constexpr const char* kStatusFooter = "status.footer";
inline constexpr const char* kChatSessionHeader = "chat.session-header";
inline constexpr const char* kChatMiddlePane = "chat.middle-pane";
inline constexpr const char* kChatEmptyState = "chat.empty-state";
inline constexpr const char* kActivityLog = "activity.log";
inline constexpr const char* kGraphPanel = "graph.panel";
inline constexpr const char* kSettingsModal = "settings.modal";
inline constexpr const char* kSettingsGear = "settings.gear";
inline constexpr const char* kSessionContextMenu = "session.context-menu";
inline constexpr const char* kSessionTodoPanel = "session.todo-panel";
inline constexpr const char* kQueueStatusBar = "queue.status-bar";
inline constexpr const char* kComposerImageModal = "composer.image-modal";

inline std::string selector_token(std::string_view raw) {
    std::string out;
    for (const char ch : raw) {
        if (ch == '/' || ch == '\\' || ch == ' ' || ch == ':') {
            out.push_back('_');
        } else {
            out.push_back(ch);
        }
    }
    return out;
}

inline std::string session_row(std::string_view session_id) {
    return std::string("session.sidebar.session.") + selector_token(session_id);
}

inline std::string project_new_button(std::string_view project_root) {
    return std::string("session.sidebar.project.") + selector_token(project_root) + ".new";
}

inline std::string project_header(std::string_view project_root) {
    return std::string("session.sidebar.project.") + selector_token(project_root);
}

} // namespace taiji::ui::test_id
