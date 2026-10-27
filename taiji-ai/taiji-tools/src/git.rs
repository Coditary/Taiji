use git2::{Repository, StatusOptions};
use std::path::Path;

pub fn status(root: &Path) -> Result<String, String> {
    let repo = Repository::discover(root).map_err(|e| format!("git2 discover: {e}"))?;
    let head = repo.head().map_err(|e| format!("head: {e}"))?;
    let branch = head
        .shorthand()
        .map(|s| s.to_string())
        .unwrap_or_else(|| "HEAD".to_string());

    let mut opts = StatusOptions::new();
    opts.include_untracked(true)
        .recurse_untracked_dirs(true)
        .show(git2::StatusShow::IndexAndWorkdir);
    let statuses = repo
        .statuses(Some(&mut opts))
        .map_err(|e| format!("statuses: {e}"))?;

    let mut lines = vec![format!("branch: {branch}")];
    if statuses.is_empty() {
        lines.push("(clean)".to_string());
    } else {
        for entry in statuses.iter() {
            let path = entry.path().unwrap_or("?");
            let st = entry.status();
            lines.push(format!("{st:?} {path}"));
        }
    }
    Ok(lines.join("\n"))
}

pub fn log_oneline(root: &Path, limit: usize) -> Result<String, String> {
    let repo = Repository::discover(root).map_err(|e| format!("git2 discover: {e}"))?;
    let mut revwalk = repo.revwalk().map_err(|e| format!("revwalk: {e}"))?;
    revwalk.push_head().map_err(|e| format!("push_head: {e}"))?;
    let mut lines = Vec::new();
    for oid in revwalk {
        let oid = oid.map_err(|e| format!("oid: {e}"))?;
        let commit = repo.find_commit(oid).map_err(|e| format!("commit: {e}"))?;
        let summary = commit.summary().unwrap_or("(no message)");
        lines.push(format!("{oid} {summary}"));
        if lines.len() >= limit {
            break;
        }
    }
    Ok(lines.join("\n"))
}
