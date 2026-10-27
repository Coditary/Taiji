use crate::walk::{canonical_root, ignore_walk, rel_slash};
use std::path::Path;

pub fn run(root: &Path, query: &str, max: usize) -> Result<Vec<String>, String> {
    let root = canonical_root(root)?;
    let query = query.trim().to_lowercase();
    if query.is_empty() {
        return Err("query must not be empty".to_string());
    }

    let mut scored: Vec<(i32, String)> = Vec::new();
    for entry in ignore_walk(&root) {
        let entry = entry.map_err(|e| format!("walk: {e}"))?;
        let path = entry.path();
        if path == root {
            continue;
        }
        let rel = rel_slash(path, &root).ok_or_else(|| "path outside root".to_string())?;
        let hay = rel.to_lowercase();
        if !hay.contains(&query) {
            continue;
        }
        let mut score = 0i32;
        if hay.contains(&query) {
            score += 10;
        }
        if rel.rsplit('/').next().unwrap_or(&rel).to_lowercase().contains(&query) {
            score += 20;
        }
        if matches_marker(rel.rsplit('/').next().unwrap_or("")) {
            score += 30;
        }
        scored.push((score, rel));
    }
    scored.sort_by(|a, b| b.0.cmp(&a.0).then_with(|| a.1.cmp(&b.1)));
    let mut out = Vec::new();
    for (_, p) in scored.into_iter().take(max) {
        out.push(p);
    }
    Ok(out)
}

fn matches_marker(name: &str) -> bool {
    matches!(
        name,
        "Cargo.toml" | "go.mod" | "package.json" | "pyproject.toml" | "CMakeLists.txt"
    )
}
