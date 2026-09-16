#ifndef TAIJI_HIGHLIGHT_H
#define TAIJI_HIGHLIGHT_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

enum {
    TAIJI_HL_DEFAULT = 0,
    TAIJI_HL_KEYWORD = 1,
    TAIJI_HL_STRING = 2,
    TAIJI_HL_NUMBER = 3,
    TAIJI_HL_COMMENT = 4,
    TAIJI_HL_FUNCTION = 5,
    TAIJI_HL_TYPE = 6,
    TAIJI_HL_OPERATOR = 7,
    TAIJI_HL_PROPERTY = 8,
    TAIJI_HL_PUNCTUATION = 9,
    TAIJI_HL_ATTRIBUTE = 10,
    TAIJI_HL_LABEL = 11,
};

typedef struct {
    uint32_t start;
    uint32_t end;
    uint8_t kind;
} taiji_hl_span;

// lang may be NULL for auto-detection. Returns 0 on success, non-zero on failure.
int taiji_hl_highlight(const char* lang, const char* code, taiji_hl_span** out_spans, size_t* out_count);

void taiji_hl_free(taiji_hl_span* spans, size_t count);

#ifdef __cplusplus
}
#endif

#endif
