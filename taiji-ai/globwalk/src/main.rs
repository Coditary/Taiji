//! File glob walker using the `ignore` crate (gitignore-aware), with `globset` matching.

use globset::{Glob, GlobSetBuilder};
use ignore::WalkBuilder;
use std::env;
use std::io::{self, Write};
use std::path::{Path, PathBuf};
use std::process::ExitCode;

fn usage() -> &'static str {
    "usage: taiji-globwalk --root <dir> --pattern <glob> [--max <n>]\n"
}

fn parse_args() -> Result<(PathBuf, String, usize), String> {
    let args: Vec<String> = env::args().skip(1).collect();
    let mut root: Option<PathBuf> = None;
    let mut pattern: Option<String> = None;
    let mut max = 100usize;
    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "--root" => {
                i += 1;
                root = Some(PathBuf::from(args.get(i).ok_or("missing --root value")?));
            }
            "--pattern" => {
                i += 1;
                pattern = Some(args.get(i).ok_or("missing --pattern value")?.clone());
            }
            "--max" => {
                i += 1;
                let n = args
                    .get(i)
                    .ok_or("missing --max value")?
                    .parse::<usize>()
                    .map_err(|_| "invalid --max")?;
                if n > 0 {
                    max = n;
                }
            }
            "-h" | "--help" => return Err(usage().to_string()),
            other => return Err(format!("unknown argument: {other}")),
        }
        i += 1;
    }
    let root = root.ok_or("missing --root")?;
    let pattern = pattern.ok_or("missing --pattern")?;
    if pattern.trim().is_empty() {
        return Err("pattern must not be empty".to_string());
    }
    Ok((root, pattern, max))
}

fn normalize_rel(path: &Path, root: &Path) -> Option<String> {
    let rel = path.strip_prefix(root).ok()?;
    let s = rel.to_string_lossy().replace('\\', "/");
    if s.is_empty() {
        return None;
    }
    Some(s)
}

fn walk_glob(root: &Path, pattern: &str, max: usize) -> Result<Vec<String>, String> {
    let root = root.canonicalize().map_err(|e| format!("root: {e}"))?;
    if !root.is_dir() {
        return Err(format!("{} is not a directory", root.display()));
    }

    let glob = Glob::new(pattern).map_err(|e| format!("pattern: {e}"))?;
    let mut builder = GlobSetBuilder::new();
    builder.add(glob);
    let set = builder.build().map_err(|e| format!("globset: {e}"))?;

    let mut matches = Vec::new();
    let walker = WalkBuilder::new(&root)
        .standard_filters(true)
        .hidden(true)
        .git_ignore(true)
        .git_global(true)
        .git_exclude(true)
        .ignore(true)
        .build();

    for entry in walker {
        let entry = entry.map_err(|e| format!("walk: {e}"))?;
        let ft = entry.file_type();
        if !ft.map(|t| t.is_file()).unwrap_or(false) {
            continue;
        }
        let path = entry.path();
        let rel = normalize_rel(path, &root).ok_or_else(|| "path outside root".to_string())?;
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

fn main() -> ExitCode {
    let result = match parse_args() {
        Ok((root, pattern, max)) => walk_glob(&root, &pattern, max),
        Err(msg) if msg.starts_with("usage:") => {
            let _ = writeln!(io::stderr(), "{msg}");
            return ExitCode::from(2);
        }
        Err(msg) => Err(msg),
    };

    match result {
        Ok(lines) => {
            let mut out = io::stdout().lock();
            for line in lines {
                let _ = writeln!(out, "{line}");
            }
            ExitCode::SUCCESS
        }
        Err(msg) => {
            let _ = writeln!(io::stderr(), "{msg}");
            ExitCode::from(1)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::process::Command;

    #[test]
    fn finds_go_files_respecting_gitignore() {
        let tmp = tempfile::tempdir().expect("tempdir");
        let root = tmp.path();
        fs::write(root.join("a.go"), "package a\n").unwrap();
        fs::write(root.join("b.txt"), "x").unwrap();
        fs::write(root.join(".gitignore"), "skip.go\n").unwrap();
        fs::write(root.join("skip.go"), "package skip\n").unwrap();
        Command::new("git")
            .args(["init"])
            .current_dir(root)
            .output()
            .expect("git init");

        let hits = walk_glob(root, "*.go", 50).expect("walk");
        let joined = hits.join(",");
        assert!(joined.contains("a.go"), "got {joined}");
        assert!(!joined.contains("skip.go"), "gitignore should skip skip.go");
    }
}
