use std::path::Path;
use streaming_iterator::StreamingIterator;
use tree_sitter::{Language, Parser, Query, QueryCursor};

fn language_for_path(path: &Path) -> Option<Language> {
    let ext = path.extension()?.to_string_lossy().to_lowercase();
    match ext.as_str() {
        "go" => Some(tree_sitter_go::LANGUAGE.into()),
        "rs" => Some(tree_sitter_rust::LANGUAGE.into()),
        "js" | "jsx" | "mjs" | "cjs" | "ts" | "tsx" => Some(tree_sitter_javascript::LANGUAGE.into()),
        "py" => Some(tree_sitter_python::LANGUAGE.into()),
        _ => None,
    }
}

fn query_for_lang(lang: &Language) -> Option<&'static str> {
    // Broad captures for top-level-ish named nodes.
    let _ = lang;
    Some(
        "(function_item name: (identifier) @name) \
         (struct_item name: (type_identifier) @name) \
         (enum_item name: (type_identifier) @name) \
         (function_declaration name: (identifier) @name) \
         (class_declaration name: (identifier) @name) \
         (method_definition name: (property_identifier) @name) \
         (function_definition name: (identifier) @name) \
         (class_definition name: (identifier) @name)",
    )
}

pub fn run(path: &Path) -> Result<String, String> {
    let lang = language_for_path(path).ok_or_else(|| "unsupported language for tree-sitter outline".to_string())?;
    let source = std::fs::read_to_string(path).map_err(|e| format!("read: {e}"))?;
    let mut parser = Parser::new();
    parser
        .set_language(&lang)
        .map_err(|e| format!("set_language: {e}"))?;
    let tree = parser
        .parse(&source, None)
        .ok_or_else(|| "parse failed".to_string())?;
    let q_src = query_for_lang(&lang).unwrap();
    let query = Query::new(&lang, q_src).map_err(|e| format!("query: {e}"))?;
    let mut cursor = QueryCursor::new();
    let mut matches = cursor.matches(&query, tree.root_node(), source.as_bytes());
    let mut lines = Vec::new();
    while let Some(m) = matches.next() {
        for cap in m.captures {
            let node = cap.node;
            let name = node.utf8_text(source.as_bytes()).unwrap_or("").trim();
            if name.is_empty() {
                continue;
            }
            let row = node.start_position().row + 1;
            lines.push(format!("L{row}: {name}"));
        }
    }
    if lines.is_empty() {
        return Ok("(no symbols captured)".to_string());
    }
    lines.sort();
    lines.dedup();
    Ok(lines.join("\n"))
}
