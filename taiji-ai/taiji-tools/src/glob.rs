use crate::walk::{canonical_root, ignore_walk, rel_slash};
use globset::{Glob, GlobSetBuilder};
use std::path::Path;

pub fn run(root: &Path, pattern: &str, max: usize) -> Result<Vec<String>, String> {
    let root = canonical_root(root)?;
    let glob = Glob::new(pattern).map_err(|e| format!("pattern: {e}"))?;
    let mut builder = GlobSetBuilder::new();
    builder.add(glob);
    let set = builder.build().map_err(|e| format!("globset: {e}"))?;

    let mut matches = Vec::new();
    for entry in ignore_walk(&root) {
        let entry = entry.map_err(|e| format!("walk: {e}"))?;
        if !entry.file_type().map(|t| t.is_file()).unwrap_or(false) {
            continue;
        }
        let path = entry.path();
        let rel = rel_slash(path, &root).ok_or_else(|| "path outside root".to_string())?;
        let base = path
            .file_name()
            .map(|n| n.to_string_lossy().replace('\\', "/"))
            .unwrap_or_default();
        if set.is_match(&rel) || (!base.is_empty() && set.is_match(&base)) {
            matches.push(rel);
            if matches.len() >= max {
                break;
            }
        }
    }
    matches.sort();
    Ok(matches)
}
