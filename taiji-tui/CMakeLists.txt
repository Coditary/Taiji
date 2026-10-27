cmake_minimum_required(VERSION 3.16)
project(taiji-tui VERSION 0.1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

option(TAIJI_TUI_FETCH_TUINATOR "Fetch Tuinator from GitHub when no local tree is set" ON)
option(TAIJI_TUI_BUILD_TESTS "Build taiji-tui component tests and JSON test-agent" OFF)
set(TAIJI_TUI_TUINATOR_DIR "" CACHE PATH "Local Tuinator source tree (overrides fetch)")
if(TAIJI_TUI_TUINATOR_DIR STREQUAL "" AND EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/../../../Coditary/core/Tuinator/CMakeLists.txt")
    set(TAIJI_TUI_TUINATOR_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../../../Coditary/core/Tuinator")
endif()

set(TAIJI_AI_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../taiji-ai")
set(TAIJI_EMBED_DIR "${TAIJI_AI_DIR}/embed")
set(TAIJI_EMBED_ARCHIVE "${CMAKE_CURRENT_BINARY_DIR}/libtaiji.a")
set(TAIJI_EMBED_HEADER "${CMAKE_CURRENT_BINARY_DIR}/libtaiji.h")

find_program(GO_EXECUTABLE go REQUIRED)
find_program(CARGO_EXECUTABLE cargo REQUIRED)

set(TAIJI_FUZZY_CRATE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/fuzzy")
set(TAIJI_FUZZY_ARCHIVE "${TAIJI_FUZZY_CRATE_DIR}/target/release/libtaiji_fuzzy.a")

add_custom_command(
    OUTPUT "${TAIJI_FUZZY_ARCHIVE}"
    COMMAND ${CMAKE_COMMAND} -E env "CARGO_TARGET_DIR=${TAIJI_FUZZY_CRATE_DIR}/target"
            ${CARGO_EXECUTABLE} build --release --manifest-path "${TAIJI_FUZZY_CRATE_DIR}/Cargo.toml"
    WORKING_DIRECTORY "${TAIJI_FUZZY_CRATE_DIR}"
    COMMENT "Building taiji-fuzzy (skim staticlib)"
    VERBATIM
)

add_custom_target(taiji_fuzzy_build DEPENDS "${TAIJI_FUZZY_ARCHIVE}")

set(TAIJI_TOOLS_CRATE_DIR "${TAIJI_AI_DIR}/taiji-tools")
set(TAIJI_TOOLS_BIN "${TAIJI_TOOLS_CRATE_DIR}/target/release/taiji-tools")

add_custom_command(
    OUTPUT "${TAIJI_TOOLS_BIN}"
    COMMAND ${CMAKE_COMMAND} -E env "CARGO_TARGET_DIR=${TAIJI_TOOLS_CRATE_DIR}/target"
            ${CARGO_EXECUTABLE} build --release --manifest-path "${TAIJI_TOOLS_CRATE_DIR}/Cargo.toml"
    WORKING_DIRECTORY "${TAIJI_TOOLS_CRATE_DIR}"
    COMMENT "Building taiji-tools (ignore, tree-sitter, git2, tantivy)"
    VERBATIM
)

add_custom_target(taiji_tools_build DEPENDS "${TAIJI_TOOLS_BIN}")

add_library(taiji_fuzzy STATIC IMPORTED GLOBAL)
add_dependencies(taiji_fuzzy taiji_fuzzy_build)
set_target_properties(taiji_fuzzy PROPERTIES
    IMPORTED_LOCATION "${TAIJI_FUZZY_ARCHIVE}"
    INTERFACE_INCLUDE_DIRECTORIES "${TAIJI_FUZZY_CRATE_DIR}/include"
)

file(GLOB_RECURSE TAIJI_UI_LOCALE_JSON CONFIGURE_DEPENDS
    "${CMAKE_CURRENT_SOURCE_DIR}/locales/*.json"
)
set(TAIJI_UI_LOCALES_EMBED_CPP "${CMAKE_CURRENT_BINARY_DIR}/ui_locales_embed.cpp")
set(TAIJI_UI_LOCALES_CATALOG_GO "${TAIJI_AI_DIR}/internal/uilocale/catalog_gen.go")
add_custom_target(
    taiji_tui_locales_check
    COMMAND ${CMAKE_COMMAND} -E env PYTHONUTF8=1
            python3 "${CMAKE_CURRENT_SOURCE_DIR}/scripts/check_locales.py"
            "${CMAKE_CURRENT_SOURCE_DIR}/locales"
            --require-complete
    COMMAND ${CMAKE_COMMAND} -E env PYTHONUTF8=1
            python3 "${CMAKE_CURRENT_SOURCE_DIR}/scripts/check_locale_usage.py"
            "${CMAKE_CURRENT_SOURCE_DIR}/locales"
    WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}"
    COMMENT "Verify UI locale catalogs and ui_tr/ui_shell key usage in src"
    VERBATIM
)

add_custom_command(
    OUTPUT "${TAIJI_UI_LOCALES_EMBED_CPP}" "${TAIJI_UI_LOCALES_CATALOG_GO}"
    COMMAND ${CMAKE_COMMAND} -E env PYTHONUTF8=1
            python3 "${CMAKE_CURRENT_SOURCE_DIR}/scripts/check_locales.py"
            "${CMAKE_CURRENT_SOURCE_DIR}/locales"
            --require-complete
            --quiet
    COMMAND ${CMAKE_COMMAND} -E env PYTHONUTF8=1
            python3 "${CMAKE_CURRENT_SOURCE_DIR}/scripts/embed_locales.py"
            "${CMAKE_CURRENT_SOURCE_DIR}/locales"
            "${TAIJI_UI_LOCALES_EMBED_CPP}"
            --go "${TAIJI_UI_LOCALES_CATALOG_GO}"
    DEPENDS
        "${CMAKE_CURRENT_SOURCE_DIR}/scripts/embed_locales.py"
        "${CMAKE_CURRENT_SOURCE_DIR}/scripts/check_locales.py"
        "${CMAKE_CURRENT_SOURCE_DIR}/scripts/locale_lib.py"
        "${CMAKE_CURRENT_SOURCE_DIR}/locales/manifest.json"
        ${TAIJI_UI_LOCALE_JSON}
    COMMENT "Check and embed UI locale strings (C++ and Go)"
    VERBATIM
)
add_custom_target(taiji_tui_locales_embed DEPENDS "${TAIJI_UI_LOCALES_EMBED_CPP}" "${TAIJI_UI_LOCALES_CATALOG_GO}")

file(GLOB_RECURSE TAIJI_GO_SOURCES CONFIGURE_DEPENDS
    "${TAIJI_AI_DIR}/*.go"
)

add_custom_command(
    OUTPUT "${TAIJI_EMBED_ARCHIVE}" "${TAIJI_EMBED_HEADER}"
    COMMAND ${CMAKE_COMMAND} -E env CGO_ENABLED=1
            ${GO_EXECUTABLE} build -buildmode=c-archive
            -o "${TAIJI_EMBED_ARCHIVE}"
            ./embed
    WORKING_DIRECTORY "${TAIJI_AI_DIR}"
    DEPENDS
        ${TAIJI_GO_SOURCES}
        "${TAIJI_EMBED_DIR}/include/taiji_embed.h"
        "${TAIJI_UI_LOCALES_CATALOG_GO}"
    COMMENT "Building static Taiji embed archive"
    VERBATIM
)

add_custom_target(taiji_embed_build DEPENDS "${TAIJI_EMBED_ARCHIVE}" "${TAIJI_EMBED_HEADER}" "${TAIJI_UI_LOCALES_CATALOG_GO}")
add_dependencies(taiji_embed_build taiji_tui_locales_embed)

add_library(taiji_embed STATIC IMPORTED GLOBAL)
add_dependencies(taiji_embed taiji_embed_build)
set_target_properties(taiji_embed PROPERTIES
    IMPORTED_LOCATION "${TAIJI_EMBED_ARCHIVE}"
    INTERFACE_INCLUDE_DIRECTORIES "${CMAKE_CURRENT_BINARY_DIR}"
)

if(TAIJI_TUI_BUILD_TESTS)
    set(TUINATOR_ENABLE_TEST_IDS ON CACHE BOOL "Store widget test IDs for AppDriver/JSON agent" FORCE)
endif()

if(TAIJI_TUI_TUINATOR_DIR AND EXISTS "${TAIJI_TUI_TUINATOR_DIR}/CMakeLists.txt")
    message(STATUS "taiji-tui: using local Tuinator at ${TAIJI_TUI_TUINATOR_DIR}")
    set(TUINATOR_BUILD_EXAMPLES OFF CACHE BOOL "" FORCE)
    set(TUINATOR_BUILD_TESTS OFF CACHE BOOL "" FORCE)
    add_subdirectory("${TAIJI_TUI_TUINATOR_DIR}" "${CMAKE_CURRENT_BINARY_DIR}/tuinator-local")
elseif(TAIJI_TUI_FETCH_TUINATOR)
    include(FetchContent)
    FetchContent_Declare(
        tuinator
        GIT_REPOSITORY https://github.com/Coditary/Tuinator.git
        GIT_TAG main
    )
    set(TUINATOR_BUILD_EXAMPLES OFF CACHE BOOL "" FORCE)
    set(TUINATOR_BUILD_TESTS OFF CACHE BOOL "" FORCE)
    FetchContent_MakeAvailable(tuinator)
else()
    find_package(Tuinator CONFIG REQUIRED)
endif()

set(TAIJI_UI_THEMES_SOURCE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/themes")
set(TAIJI_UI_THEMES_BUILD_DIR "${CMAKE_CURRENT_BINARY_DIR}/themes")

set(TAIJI_UI_BUNDLED_THEME_IDS
    dark
    high_contrast
    gruvbox_dark
    gruvbox_light
    catppuccin_mocha
    catppuccin_latte
    nord
    dracula
    rose_pine
    solarized_dark
    everforest_dark
    one_dark
)

set(TAIJI_UI_BUNDLED_THEME_FILES "")
foreach(theme_id IN LISTS TAIJI_UI_BUNDLED_THEME_IDS)
    list(APPEND TAIJI_UI_BUNDLED_THEME_FILES "${TAIJI_UI_THEMES_SOURCE_DIR}/${theme_id}.theme")
endforeach()

add_custom_command(
    OUTPUT "${TAIJI_UI_THEMES_BUILD_DIR}/dark.theme"
    COMMAND ${CMAKE_COMMAND} -E copy_directory "${TAIJI_UI_THEMES_SOURCE_DIR}" "${TAIJI_UI_THEMES_BUILD_DIR}"
    DEPENDS ${TAIJI_UI_BUNDLED_THEME_FILES}
    COMMENT "Copy Taiji UI theme stylesheets"
    VERBATIM
)

add_custom_target(taiji_tui_themes DEPENDS "${TAIJI_UI_THEMES_BUILD_DIR}/dark.theme")

set(TAIJI_UI_THEMES_EMBED_CPP "${CMAKE_CURRENT_BINARY_DIR}/ui_themes_bundled.cpp")
add_custom_command(
    OUTPUT "${TAIJI_UI_THEMES_EMBED_CPP}"
    COMMAND ${CMAKE_COMMAND} -E env PYTHONUTF8=1
            python3 "${CMAKE_CURRENT_SOURCE_DIR}/scripts/embed_ui_themes.py"
            "${TAIJI_UI_THEMES_SOURCE_DIR}"
            "${TAIJI_UI_THEMES_EMBED_CPP}"
            ${TAIJI_UI_BUNDLED_THEME_IDS}
    DEPENDS
        "${CMAKE_CURRENT_SOURCE_DIR}/scripts/embed_ui_themes.py"
        ${TAIJI_UI_BUNDLED_THEME_FILES}
    COMMENT "Embedding default UI themes into taiji-tui"
    VERBATIM
)
add_custom_target(taiji_tui_themes_embed DEPENDS "${TAIJI_UI_THEMES_EMBED_CPP}")

set(TAIJI_TUI_SOURCES
    src/chat_app.cpp
    src/chat_message.cpp
    src/chat_message_chrome.cpp
    src/assistant_markdown.cpp
    src/queue_status_bar.cpp
    src/left_label.cpp
    src/session_sidebar.cpp
    src/left_anchored_split.cpp
    src/right_anchored_split.cpp
    src/session_context_menu.cpp
    src/mcp_sidebar.cpp
    src/turn_metrics_note.cpp
    src/user_turn_actions_note.cpp
    src/activity_log.cpp
    src/activity_trace.cpp
    src/activity_diff_layout.cpp
    src/thinking_indicator.cpp
    src/chat_input.cpp
    src/composer_attachments.cpp
    src/composer_image_clipboard.cpp
    src/composer_image_modal.cpp
    src/composer_palette.cpp
    src/composer_workspace.cpp
    src/message_composer.cpp
    src/question_composer.cpp
    src/status_footer.cpp
    src/ui_layout.cpp
    src/ui_theme.cpp
    src/ui_locale.cpp
    "${TAIJI_UI_LOCALES_EMBED_CPP}"
    src/ui_keybindings.cpp
    src/ui_navigation.cpp
    src/composer_fuzzy_picker.cpp
    src/composer_theme_picker.cpp
    src/composer_actor_picker.cpp
    src/ui_themes_bundled_lookup.cpp
    "${TAIJI_UI_THEMES_EMBED_CPP}"
    src/ui_message_time.cpp
    src/ui_anim.cpp
    src/context_panel.cpp
    src/graph_panel.cpp
    src/sidebar_kv_row.cpp
    src/mcp_server_list.cpp
    src/session_todo_panel.cpp
    src/chat_session_header.cpp
    src/chat_empty_state.cpp
    src/taiji_block_logo.cpp
    src/ui_panel_box.cpp
    src/ui_chat_spacing.cpp
    src/ui_chat_column.cpp
    src/chat_middle_pane.cpp
    src/sidebar_footer.cpp
    src/sidebar_divider.cpp
    src/settings_modal.cpp
    src/settings_modal_ui.cpp
    src/settings_modal_extras.cpp
    src/skim_fuzzy.cpp
    src/settings_gear.cpp
    src/chat_input_bar.cpp
    src/speech_mic_button.cpp
)

add_executable(taiji-tui src/main.cpp ${TAIJI_TUI_SOURCES})

add_dependencies(taiji-tui taiji_embed_build taiji_fuzzy_build taiji_tools_build taiji_tui_themes taiji_tui_themes_embed taiji_tui_locales_check taiji_tui_locales_embed)

add_custom_command(
    TARGET taiji-tui POST_BUILD
    COMMAND ${CMAKE_COMMAND} -E copy_if_different
            "${TAIJI_TOOLS_BIN}"
            "$<TARGET_FILE_DIR:taiji-tui>/taiji-tools"
    COMMENT "Copy taiji-tools next to taiji-tui"
    VERBATIM
)
target_include_directories(taiji-tui PRIVATE
    "${CMAKE_CURRENT_SOURCE_DIR}/src"
    "${CMAKE_CURRENT_BINARY_DIR}"
    "${TAIJI_FUZZY_CRATE_DIR}/include"
)
target_compile_definitions(taiji-tui PRIVATE TAIJI_UI_THEMES_DIR="${TAIJI_UI_THEMES_BUILD_DIR}")
target_link_libraries(taiji-tui PRIVATE tuinator::tuinator taiji_embed taiji_fuzzy)

install(DIRECTORY "${TAIJI_UI_THEMES_SOURCE_DIR}/" DESTINATION share/taiji-tui/themes)

if(UNIX AND NOT APPLE)
    target_link_libraries(taiji-tui PRIVATE pthread dl m resolv)
elseif(APPLE)
    target_link_libraries(taiji-tui PRIVATE pthread resolv)
endif()

if(TAIJI_TUI_BUILD_TESTS)
    enable_testing()

    set(TAIJI_LOCALE_KEYS_USED_HPP "${CMAKE_CURRENT_BINARY_DIR}/locale_keys_used.gen.hpp")
    add_custom_command(
        OUTPUT "${TAIJI_LOCALE_KEYS_USED_HPP}"
        COMMAND ${CMAKE_COMMAND} -E env PYTHONUTF8=1
                python3 "${CMAKE_CURRENT_SOURCE_DIR}/scripts/generate_locale_keys_used.py"
                "${CMAKE_CURRENT_SOURCE_DIR}/src"
                "${TAIJI_LOCALE_KEYS_USED_HPP}"
        DEPENDS
            "${CMAKE_CURRENT_SOURCE_DIR}/scripts/generate_locale_keys_used.py"
            "${CMAKE_CURRENT_SOURCE_DIR}/scripts/locale_usage.py"
            ${TAIJI_TUI_SOURCES}
        COMMENT "Generate locale key list from ui_tr/ui_shell references"
        VERBATIM
    )
    add_custom_target(taiji_tui_locale_keys_gen DEPENDS "${TAIJI_LOCALE_KEYS_USED_HPP}")

    set(TAIJI_TUI_TEST_SOURCES
        tests/main_test_runner.cpp
        tests/components/test_message_composer.cpp
        tests/components/test_theme_picker_esc.cpp
        tests/components/test_theme_picker_prompt_stable.cpp
        tests/components/test_session_sidebar.cpp
        tests/components/test_session_sidebar_hover_background.cpp
        tests/components/test_activity_log_failed_tool_background.cpp
        tests/components/test_status_footer.cpp
        tests/integration/test_sidebar_column.cpp
        tests/integration/test_theme_picker_ctrl_t_prompt_layout.cpp
        tests/integration/test_theme_picker_chat_app.cpp
        tests/i18n/test_locale_catalog.cpp
        tests/i18n/test_locale_render.cpp
    )

    add_executable(taiji-tui-ui-tests ${TAIJI_TUI_TEST_SOURCES} ${TAIJI_TUI_SOURCES})
    add_dependencies(taiji-tui-ui-tests taiji_tui_locale_keys_gen)
    add_dependencies(taiji-tui-ui-tests taiji_embed_build taiji_fuzzy_build taiji_tui_themes taiji_tui_themes_embed taiji_tui_locales_check taiji_tui_locales_embed)
    target_include_directories(taiji-tui-ui-tests PRIVATE
        "${CMAKE_CURRENT_SOURCE_DIR}/src"
        "${CMAKE_CURRENT_SOURCE_DIR}/tests"
        "${CMAKE_CURRENT_BINARY_DIR}"
        "${TAIJI_FUZZY_CRATE_DIR}/include"
    )
    target_compile_definitions(taiji-tui-ui-tests PRIVATE
        TAIJI_UI_THEMES_DIR="${TAIJI_UI_THEMES_BUILD_DIR}"
        TUINATOR_ENABLE_TEST_IDS=1
    )
    target_link_libraries(taiji-tui-ui-tests PRIVATE tuinator::tuinator taiji_embed taiji_fuzzy)
    if(UNIX AND NOT APPLE)
        target_link_libraries(taiji-tui-ui-tests PRIVATE pthread dl m resolv)
    elseif(APPLE)
        target_link_libraries(taiji-tui-ui-tests PRIVATE pthread resolv)
    endif()
    add_test(NAME taiji-tui-ui-tests COMMAND taiji-tui-ui-tests)
    add_test(
        NAME taiji-tui-locale-usage
        COMMAND ${CMAKE_COMMAND} -E env PYTHONUTF8=1
                python3 "${CMAKE_CURRENT_SOURCE_DIR}/scripts/check_locale_usage.py"
                "${CMAKE_CURRENT_SOURCE_DIR}/locales"
        WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}"
    )

    add_executable(taiji-tui-test-agent tests/agent/main.cpp ${TAIJI_TUI_SOURCES})
    add_dependencies(taiji-tui-test-agent taiji_embed_build taiji_fuzzy_build taiji_tui_themes taiji_tui_themes_embed taiji_tui_locales_check taiji_tui_locales_embed)
    target_include_directories(taiji-tui-test-agent PRIVATE
        "${CMAKE_CURRENT_SOURCE_DIR}/src"
        "${CMAKE_CURRENT_SOURCE_DIR}/tests"
        "${CMAKE_CURRENT_BINARY_DIR}"
        "${TAIJI_FUZZY_CRATE_DIR}/include"
    )
    target_compile_definitions(taiji-tui-test-agent PRIVATE
        TAIJI_UI_THEMES_DIR="${TAIJI_UI_THEMES_BUILD_DIR}"
        TUINATOR_ENABLE_TEST_IDS=1
    )
    target_link_libraries(taiji-tui-test-agent PRIVATE tuinator::tuinator taiji_embed taiji_fuzzy)
    if(UNIX AND NOT APPLE)
        target_link_libraries(taiji-tui-test-agent PRIVATE pthread dl m resolv)
    elseif(APPLE)
        target_link_libraries(taiji-tui-test-agent PRIVATE pthread resolv)
    endif()
endif()
