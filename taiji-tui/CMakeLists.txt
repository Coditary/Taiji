cmake_minimum_required(VERSION 3.16)
project(taiji-tui VERSION 0.1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

option(TAIJI_TUI_FETCH_TUINATOR "Fetch Tuinator from GitHub when no local tree is set" ON)
set(TAIJI_TUI_TUINATOR_DIR "" CACHE PATH "Local Tuinator source tree (overrides fetch)")
if(TAIJI_TUI_TUINATOR_DIR STREQUAL "" AND EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/../../../Coditary/core/Tuinator/CMakeLists.txt")
    set(TAIJI_TUI_TUINATOR_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../../../Coditary/core/Tuinator")
endif()

set(TAIJI_AI_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../taiji-ai")
set(TAIJI_EMBED_DIR "${TAIJI_AI_DIR}/embed")
set(TAIJI_EMBED_ARCHIVE "${CMAKE_CURRENT_BINARY_DIR}/libtaiji.a")
set(TAIJI_EMBED_HEADER "${CMAKE_CURRENT_BINARY_DIR}/libtaiji.h")

find_program(GO_EXECUTABLE go REQUIRED)

add_custom_command(
    OUTPUT "${TAIJI_EMBED_ARCHIVE}" "${TAIJI_EMBED_HEADER}"
    COMMAND ${CMAKE_COMMAND} -E env CGO_ENABLED=1
            ${GO_EXECUTABLE} build -buildmode=c-archive
            -o "${TAIJI_EMBED_ARCHIVE}"
            ./embed
    WORKING_DIRECTORY "${TAIJI_AI_DIR}"
    DEPENDS
        "${TAIJI_EMBED_DIR}/embed.go"
        "${TAIJI_EMBED_DIR}/include/taiji_embed.h"
    COMMENT "Building static Taiji embed archive"
    VERBATIM
)

add_custom_target(taiji_embed_build DEPENDS "${TAIJI_EMBED_ARCHIVE}" "${TAIJI_EMBED_HEADER}")

add_library(taiji_embed STATIC IMPORTED GLOBAL)
add_dependencies(taiji_embed taiji_embed_build)
set_target_properties(taiji_embed PROPERTIES
    IMPORTED_LOCATION "${TAIJI_EMBED_ARCHIVE}"
    INTERFACE_INCLUDE_DIRECTORIES "${CMAKE_CURRENT_BINARY_DIR}"
)

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

add_executable(taiji-tui
    src/main.cpp
    src/chat_app.cpp
    src/chat_message.cpp
    src/assistant_markdown.cpp
    src/queue_status_bar.cpp
    src/left_label.cpp
    src/mcp_sidebar.cpp
    src/turn_metrics_note.cpp
    src/activity_log.cpp
    src/thinking_indicator.cpp
    src/chat_input.cpp
    src/status_footer.cpp
)

add_dependencies(taiji-tui taiji_embed_build)
target_link_libraries(taiji-tui PRIVATE tuinator::tuinator taiji_embed)

if(UNIX AND NOT APPLE)
    target_link_libraries(taiji-tui PRIVATE pthread dl m resolv)
elseif(APPLE)
    target_link_libraries(taiji-tui PRIVATE pthread resolv)
endif()
