use std::collections::HashMap;
use std::ffi::{c_char, CStr};
use std::ptr;
use std::sync::Mutex;

use tree_sitter_highlight::{HighlightConfiguration, HighlightEvent, Highlighter};

const HIGHLIGHT_NAMES: &[&str] = &[
    "attribute",
    "comment",
    "constant",
    "constant.builtin",
    "constructor",
    "embedded",
    "function",
    "function.builtin",
    "function.method",
    "keyword",
    "label",
    "module",
    "number",
    "operator",
    "property",
    "property.builtin",
    "punctuation",
    "punctuation.bracket",
    "punctuation.delimiter",
    "punctuation.special",
    "string",
    "string.special",
    "tag",
    "type",
    "type.builtin",
    "variable",
    "variable.builtin",
    "variable.parameter",
];

#[repr(C)]
pub struct TaijiHlSpan {
    pub start: u32,
    pub end: u32,
    pub kind: u8,
}

struct LanguageConfig {
    config: HighlightConfiguration,
}

struct HighlightEngine {
    highlighter: Highlighter,
    languages: HashMap<String, usize>,
    configs: Vec<LanguageConfig>,
}

impl HighlightEngine {
    fn new() -> Self {
        let mut engine = Self {
            highlighter: Highlighter::new(),
            languages: HashMap::new(),
            configs: Vec::new(),
        };

        engine.register_language("rust", tree_sitter_rust::LANGUAGE.into(), tree_sitter_rust::HIGHLIGHTS_QUERY, "", "");
        engine.register_language(
            "python",
            tree_sitter_python::LANGUAGE.into(),
            tree_sitter_python::HIGHLIGHTS_QUERY,
            "",
            "",
        );
        engine.register_language(
            "javascript",
            tree_sitter_javascript::LANGUAGE.into(),
            tree_sitter_javascript::HIGHLIGHT_QUERY,
            tree_sitter_javascript::INJECTIONS_QUERY,
            tree_sitter_javascript::LOCALS_QUERY,
        );
        engine.register_alias("js", "javascript");
        engine.register_alias("ts", "javascript");
        engine.register_alias("typescript", "javascript");
        engine.register_language("c", tree_sitter_c::LANGUAGE.into(), tree_sitter_c::HIGHLIGHT_QUERY, "", "");
        engine.register_language("cpp", tree_sitter_cpp::LANGUAGE.into(), tree_sitter_cpp::HIGHLIGHT_QUERY, "", "");
        engine.register_alias("c++", "cpp");
        engine.register_language("bash", tree_sitter_bash::LANGUAGE.into(), tree_sitter_bash::HIGHLIGHT_QUERY, "", "");
        engine.register_alias("sh", "bash");
        engine.register_language("json", tree_sitter_json::LANGUAGE.into(), tree_sitter_json::HIGHLIGHTS_QUERY, "", "");

        engine
    }

    fn register_language(
        &mut self,
        name: &'static str,
        language: tree_sitter::Language,
        highlights: &str,
        injections: &str,
        locals: &str,
    ) {
        if highlights.is_empty() {
            return;
        }

        let mut config = HighlightConfiguration::new(language, name, highlights, injections, locals)
            .expect("highlight configuration");
        config.configure(HIGHLIGHT_NAMES);

        let index = self.configs.len();
        self.configs.push(LanguageConfig { config });
        self.languages.insert(name.to_string(), index);
    }

    fn register_alias(&mut self, alias: &str, target: &str) {
        if let Some(&index) = self.languages.get(target) {
            self.languages.insert(alias.to_string(), index);
        }
    }

    fn resolve_language(&self, lang: Option<&str>, code: &str) -> Option<usize> {
        if let Some(lang) = lang {
            let normalized = normalize_lang(lang);
            if let Some(index) = self.languages.get(&normalized) {
                return Some(*index);
            }
        }

        detect_language(code).and_then(|name| self.languages.get(name).copied())
    }

    fn highlight(&mut self, lang: Option<&str>, code: &str) -> Vec<TaijiHlSpan> {
        let Some(config_index) = self.resolve_language(lang, code) else {
            return Vec::new();
        };

        let config = &self.configs[config_index].config;
        let events = match self.highlighter.highlight(config, code.as_bytes(), None, |_| None) {
            Ok(events) => events,
            Err(_) => return Vec::new(),
        };

        let mut spans = Vec::new();
        let mut offset = 0usize;
        let mut open: Option<(usize, u8)> = None;

        for event in events {
            match event {
                Ok(HighlightEvent::Source { start: _, end }) => {
                    if let Some((start, kind)) = open.take() {
                        if kind != 0 && start < end {
                            spans.push(TaijiHlSpan {
                                start: start as u32,
                                end: end as u32,
                                kind,
                            });
                        }
                    }
                    offset = end;
                }
                Ok(HighlightEvent::HighlightStart(highlight)) => {
                    let name = config.names()[highlight.0];
                    open = Some((offset, kind_for_name(name)));
                }
                Ok(HighlightEvent::HighlightEnd) => {}
                Err(_) => break,
            }
        }

        spans
    }
}

fn normalize_lang(lang: &str) -> String {
    match lang.to_ascii_lowercase().as_str() {
        "typescript" | "ts" => "javascript".to_string(),
        "c++" => "cpp".to_string(),
        other => other.to_string(),
    }
}

fn kind_for_name(name: &str) -> u8 {
    match name.split('.').next() {
        Some("keyword") => 1,
        Some("string") => 2,
        Some("number") | Some("constant") => 3,
        Some("comment") => 4,
        Some("function") | Some("constructor") => 5,
        Some("type") | Some("module") | Some("tag") => 6,
        Some("operator") => 7,
        Some("property") => 8,
        Some("punctuation") => 9,
        Some("attribute") => 10,
        Some("label") => 11,
        Some("variable") => 8,
        _ => 0,
    }
}

fn detect_language(code: &str) -> Option<&'static str> {
    let trimmed = code.trim();
    if trimmed.is_empty() {
        return None;
    }

    if trimmed.starts_with("#!/bin/bash") || trimmed.starts_with("#!/bin/sh") {
        return Some("bash");
    }

    let mut scores = [
        ("rust", 0i32),
        ("python", 0),
        ("javascript", 0),
        ("c", 0),
        ("cpp", 0),
        ("bash", 0),
        ("json", 0),
    ];

    let mut bump = |name: &str, points: i32| {
        if let Some(entry) = scores.iter_mut().find(|(n, _)| *n == name) {
            entry.1 += points;
        }
    };

    if trimmed.contains("fn ") || trimmed.contains("let ") || trimmed.contains("impl ") || trimmed.contains("pub ") {
        bump("rust", 3);
    }
    if trimmed.contains("def ") || trimmed.contains("import ") || trimmed.contains("elif ") {
        bump("python", 3);
    }
    if trimmed.contains("const ") || trimmed.contains("function ") || trimmed.contains("=>") {
        bump("javascript", 3);
    }
    if trimmed.contains("#include") || trimmed.contains("std::") || trimmed.contains("::") {
        bump("cpp", 2);
    }
    if trimmed.contains("int main") || trimmed.contains("printf(") {
        bump("c", 2);
    }
    if trimmed.starts_with('{') && trimmed.ends_with('}') && trimmed.contains('"') {
        bump("json", 2);
    }
    if trimmed.contains("echo ") || trimmed.contains("export ") {
        bump("bash", 2);
    }

    scores
        .iter()
        .filter(|(_, score)| *score > 0)
        .max_by_key(|(_, score)| *score)
        .map(|(name, _)| *name)
}

static ENGINE: Mutex<Option<HighlightEngine>> = Mutex::new(None);

fn engine() -> std::sync::MutexGuard<'static, Option<HighlightEngine>> {
    let mut guard = ENGINE.lock().expect("highlight engine lock");
    if guard.is_none() {
        *guard = Some(HighlightEngine::new());
    }
    guard
}

#[no_mangle]
pub extern "C" fn taiji_hl_highlight(
    lang: *const c_char,
    code: *const c_char,
    out_spans: *mut *mut TaijiHlSpan,
    out_count: *mut usize,
) -> i32 {
    if out_spans.is_null() || out_count.is_null() || code.is_null() {
        return 1;
    }

    let code = unsafe { CStr::from_ptr(code) };
    let code = match code.to_str() {
        Ok(value) => value,
        Err(_) => return 2,
    };

    let lang = if lang.is_null() {
        None
    } else {
        let lang = unsafe { CStr::from_ptr(lang) };
        match lang.to_str() {
            Ok(value) if !value.is_empty() => Some(normalize_lang(value)),
            _ => None,
        }
    };

    let spans = {
        let mut guard = engine();
        let engine = guard.as_mut().expect("highlight engine");
        engine.highlight(lang.as_deref(), code)
    };

    if spans.is_empty() {
        unsafe {
            *out_spans = ptr::null_mut();
            *out_count = 0;
        }
        return 0;
    }

    let count = spans.len();
    let ptr = unsafe {
        let layout = std::alloc::Layout::array::<TaijiHlSpan>(count).expect("layout");
        let mem = std::alloc::alloc(layout) as *mut TaijiHlSpan;
        if mem.is_null() {
            return 3;
        }
        ptr::copy_nonoverlapping(spans.as_ptr(), mem, count);
        mem
    };

    unsafe {
        *out_spans = ptr;
        *out_count = count;
    }

    0
}

#[no_mangle]
pub extern "C" fn taiji_hl_free(spans: *mut TaijiHlSpan, count: usize) {
    if spans.is_null() || count == 0 {
        return;
    }

    unsafe {
        let layout = std::alloc::Layout::array::<TaijiHlSpan>(count).expect("layout");
        std::alloc::dealloc(spans as *mut u8, layout);
    }
}
