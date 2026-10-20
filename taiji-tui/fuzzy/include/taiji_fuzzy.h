#pragma once

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/** One row in the model/provider fuzzy list. */
typedef struct taiji_fuzzy_item {
    const char* line;
    const char* meta;
    const char* provider_id;
} taiji_fuzzy_item;

/**
 * Rank items with skim's SkimMatcherV2 (same matcher as skim/fzf-style fuzzy find).
 *
 * @param query       User filter string (UTF-8). Empty query returns all items in original order.
 * @param items       Array of item pointers (fields may be null).
 * @param item_count  Length of items.
 * @param out_indices Caller buffer: original item indices, best match first.
 * @param out_capacity Size of out_indices.
 * @return Number of indices written, or -1 on error.
 */
int taiji_fuzzy_filter(const char* query, const taiji_fuzzy_item* items, size_t item_count, int* out_indices,
                       size_t out_capacity);

#ifdef __cplusplus
}
#endif
