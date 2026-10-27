use crate::walk::{canonical_root, ignore_walk, rel_slash};
use std::path::Path;

pub fn run(root: &Path, max_depth: u32, max_entries: usize) -> Result<Vec<String>, String> {
    let root = canonical_root(root)?;
    let mut lines = Vec::new();
    for entry in ignore_walk(&root) {
        let entry = entry.map_err(|e| format!("walk: {e}"))?;
        let path = entry.path();
        if path == root {
            continue;
        }
        let rel = rel_slash(path, &root).ok_or_else(|| "path outside root".to_string())?;
        let depth = rel.chars().filter(|c| *c == '/').count() + 1;
        if depth > max_depth as usize {
            continue;
        }
        let kind = if entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
            "dir"
        } else {
            "file"
        };
        let size = if kind == "file" {
            entry
                .metadata()
                .map(|m| format!(", {} bytes", m.len()))
                .unwrap_or_default()
        } else {
            String::new()
        };
        let suffix = if kind == "dir" { "/" } else { "" };
        lines.push(format!("{}{} ({}{})", rel, suffix, kind, size));
        if lines.len() >= max_entries {
            break;
        }
    }
    lines.sort();
    Ok(lines)
}
