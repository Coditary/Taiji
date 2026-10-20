use std::ffi::CStr;
use std::os::raw::{c_char, c_int};

use skim::fuzzy_matcher::skim::SkimMatcherV2;
use skim::fuzzy_matcher::FuzzyMatcher;

#[repr(C)]
pub struct TaijiFuzzyItem {
    pub line: *const c_char,
    pub meta: *const c_char,
    pub provider_id: *const c_char,
}

fn field_score(matcher: &SkimMatcherV2, ptr: *const c_char, needle: &str) -> i64 {
    if ptr.is_null() {
        return 0;
    }
    let haystack = unsafe { CStr::from_ptr(ptr) }.to_str().unwrap_or("");
    if haystack.is_empty() {
        return 0;
    }
    matcher.fuzzy_match(haystack, needle).unwrap_or(0)
}

fn item_score(matcher: &SkimMatcherV2, item: &TaijiFuzzyItem, needle: &str) -> i64 {
    field_score(matcher, item.line, needle)
        .max(field_score(matcher, item.meta, needle))
        .max(field_score(matcher, item.provider_id, needle))
}

/// C ABI: rank indices by skim fuzzy score (descending). Empty query → identity order.
#[no_mangle]
pub extern "C" fn taiji_fuzzy_filter(
    query: *const c_char,
    items: *const TaijiFuzzyItem,
    item_count: usize,
    out_indices: *mut c_int,
    out_capacity: usize,
) -> c_int {
    if out_indices.is_null() || out_capacity == 0 {
        return -1;
    }
    if item_count == 0 {
        return 0;
    }
    if items.is_null() {
        return -1;
    }

    let query_str = if query.is_null() {
        ""
    } else {
        match unsafe { CStr::from_ptr(query) }.to_str() {
            Ok(s) => s,
            Err(_) => return -1,
        }
    };

    let slice = unsafe { std::slice::from_raw_parts(items, item_count) };

    let mut ranked: Vec<(usize, i64)> = Vec::with_capacity(item_count);
    if query_str.is_empty() {
        for i in 0..item_count {
            ranked.push((i, 0));
        }
    } else {
        let matcher = SkimMatcherV2::default();
        for i in 0..item_count {
            let score = item_score(&matcher, &slice[i], query_str);
            if score > 0 {
                ranked.push((i, score));
            }
        }
        ranked.sort_by(|a, b| {
            b.1.cmp(&a.1).then_with(|| a.0.cmp(&b.0))
        });
    }

    let n = ranked.len().min(out_capacity);
    for j in 0..n {
        unsafe {
            *out_indices.add(j) = ranked[j].0 as c_int;
        }
    }
    n as c_int
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn empty_query_preserves_order() {
        let items = [
            TaijiFuzzyItem {
                line: c"alpha".as_ptr(),
                meta: c"".as_ptr(),
                provider_id: c"p1".as_ptr(),
            },
            TaijiFuzzyItem {
                line: c"beta".as_ptr(),
                meta: c"".as_ptr(),
                provider_id: c"p2".as_ptr(),
            },
        ];
        let mut out = [0i32, 0i32];
        let n = taiji_fuzzy_filter(c"".as_ptr(), items.as_ptr(), 2, out.as_mut_ptr(), 2);
        assert_eq!(n, 2);
        assert_eq!(out, [0, 1]);
    }

    #[test]
    fn fuzzy_ranks_match() {
        let items = [
            TaijiFuzzyItem {
                line: c"Ollama  gemma".as_ptr(),
                meta: c"".as_ptr(),
                provider_id: c"ollama".as_ptr(),
            },
            TaijiFuzzyItem {
                line: c"OpenAI  gpt-4".as_ptr(),
                meta: c"".as_ptr(),
                provider_id: c"openai".as_ptr(),
            },
        ];
        let mut out = [0i32, 0i32];
        let n = taiji_fuzzy_filter(c"gem".as_ptr(), items.as_ptr(), 2, out.as_mut_ptr(), 2);
        assert_eq!(n, 1);
        assert_eq!(out[0], 0);
    }
}
