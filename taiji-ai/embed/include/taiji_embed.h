#ifndef TAIJI_EMBED_H
#define TAIJI_EMBED_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct taiji_session_stats_t {
    int32_t context_tokens;
    int32_t context_window;
    double context_used_pct;
    int32_t input_tokens;
    int32_t output_tokens;
    double estimated_cost_usd;
    int32_t usage_system_tokens;
    int32_t usage_tools_tokens;
    int32_t usage_skills_tokens;
    int32_t usage_conversation_tokens;
} taiji_session_stats_t;

typedef struct taiji_turn_metrics_t {
    int32_t total_duration_ms;
    int32_t wait_duration_ms;
    int32_t thinking_duration_ms;
    int32_t output_tokens;
    int32_t generation_duration_ms;
    double tokens_per_sec;
    int32_t had_thinking;
    int32_t input_tokens;
    double estimated_cost_usd;
    int32_t tool_calls;
    int32_t tool_duration_ms;
    int32_t steps;
    int32_t context_tokens;
    int32_t context_window;
    double context_used_pct;
} taiji_turn_metrics_t;

typedef void (*taiji_delta_cb)(const char* delta, void* user_data);
typedef void (*taiji_done_cb)(int ok, const char* error, const char* answer, void* user_data);
typedef void (*taiji_event_cb)(const char* event_json, void* user_data);

uint64_t taiji_init(char* work_dir, int local_mode, char** error_out);
int taiji_runtime_ready(uint64_t handle);
char* taiji_runtime_error(uint64_t handle);
void taiji_close(uint64_t handle);

int taiji_chat(uint64_t handle, char* goal, char* provider, taiji_delta_cb on_delta, taiji_done_cb on_done,
               void* user_data);

void taiji_set_event_callback(uint64_t handle, taiji_event_cb on_event, void* user_data);

void taiji_cancel(uint64_t handle);

char* taiji_session_label(uint64_t handle);
char* taiji_session_name(uint64_t handle);
char* taiji_session_id(uint64_t handle);
char* taiji_work_dir(uint64_t handle);
char* taiji_model_name(uint64_t handle);
char* taiji_provider_label(uint64_t handle);
char* taiji_mcp_status(uint64_t handle);
char* taiji_mcp_servers(uint64_t handle);
char* taiji_skills_list(uint64_t handle);
char* taiji_decisions_list(uint64_t handle);
char* taiji_active_skills(uint64_t handle);
int taiji_session_stats(uint64_t handle, taiji_session_stats_t* out);
int taiji_last_turn_metrics(uint64_t handle, taiji_turn_metrics_t* out);
char* taiji_format_turn_metrics(uint64_t handle, taiji_turn_metrics_t* metrics, int32_t max_width);
int taiji_turn_metrics_show_actions(uint64_t handle);
char* taiji_tool_display_title(uint64_t handle, char* tool_name);
char* taiji_tool_scope_label(uint64_t handle, char* scope);
char* taiji_projects_list(uint64_t handle);
char* taiji_session_turns(uint64_t handle);
int taiji_session_compact(uint64_t handle, char** error_out);
int taiji_session_switch(uint64_t handle, char* project_root, char* session_id, char** error_out);
int taiji_session_new(uint64_t handle, char* project_root, char** error_out);
int taiji_session_delete(uint64_t handle, char* project_root, char* session_id, char** error_out);
char* taiji_init_error(void);

int taiji_question_reply(uint64_t handle, char* request_id, char* answers_json);
int taiji_question_reject(uint64_t handle, char* request_id);

char* taiji_settings_get(uint64_t handle, char* scope);
int taiji_settings_apply(uint64_t handle, char* json_payload);
char* taiji_stt_toggle(uint64_t handle);
char* taiji_provider_models(uint64_t handle, char* provider_id);
char* taiji_provider_model_details(uint64_t handle, char* provider_id, char* model_id);
int taiji_template_apply(uint64_t handle, char* template_id);
char* taiji_presentation_mode(uint64_t handle);

#ifdef __cplusplus
}
#endif

#endif
