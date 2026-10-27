use serde::Deserialize;
use std::fs;
use std::path::{Path, PathBuf};
use tantivy::collector::TopDocs;
use tantivy::query::QueryParser;
use tantivy::schema::*;
use tantivy::{doc, Index, IndexWriter, ReloadPolicy, TantivyDocument};

const SESSIONS_SUBDIR: &str = ".taiji/sessions";

fn index_dir(project_root: &Path) -> PathBuf {
    project_root.join(".taiji").join("index").join("sessions")
}

fn open_or_create_index(dir: &Path) -> Result<Index, String> {
    if dir.exists() {
        return Index::open_in_dir(dir).map_err(|e| format!("open index: {e}"));
    }
    fs::create_dir_all(dir).map_err(|e| format!("mkdir index: {e}"))?;
    let mut schema_builder = Schema::builder();
    let _session_id = schema_builder.add_text_field("session_id", STRING | STORED);
    let _session_name = schema_builder.add_text_field("session_name", TEXT | STORED);
    let _role = schema_builder.add_text_field("role", STRING | STORED);
    let _body = schema_builder.add_text_field("body", TEXT);
    let schema = schema_builder.build();
    Index::create_in_dir(dir, schema)
        .map_err(|e| format!("create index: {e}"))?;
    Index::open_in_dir(dir).map_err(|e| format!("reopen index: {e}"))
}

#[derive(Deserialize)]
struct SessionState {
    id: String,
    name: Option<String>,
    messages: Vec<SessionMessage>,
}

#[derive(Deserialize)]
struct SessionMessage {
    role: String,
    content: String,
}

pub fn reindex(project_root: &Path) -> Result<(), String> {
    let sessions_root = project_root.join(SESSIONS_SUBDIR);
    if !sessions_root.is_dir() {
        return Ok(());
    }
    let idx_path = index_dir(project_root);
    let index = open_or_create_index(&idx_path)?;
    let schema = index.schema();
    let f_session_id = schema.get_field("session_id").unwrap();
    let f_session_name = schema.get_field("session_name").unwrap();
    let f_role = schema.get_field("role").unwrap();
    let f_body = schema.get_field("body").unwrap();

    let mut writer: IndexWriter = index
        .writer(50_000_000)
        .map_err(|e| format!("writer: {e}"))?;
    writer
        .delete_all_documents()
        .map_err(|e| format!("delete_all: {e}"))?;

    for entry in fs::read_dir(&sessions_root).map_err(|e| format!("read_dir: {e}"))? {
        let entry = entry.map_err(|e| format!("entry: {e}"))?;
        if !entry.file_type().map_err(|e| format!("type: {e}"))?.is_dir() {
            continue;
        }
        let state_path = entry.path().join("state.json");
        let raw = fs::read_to_string(&state_path).map_err(|e| format!("read state: {e}"))?;
        let state: SessionState =
            serde_json::from_str(&raw).map_err(|e| format!("json: {e}"))?;
        let name = state.name.unwrap_or_else(|| state.id.clone());
        for msg in state.messages {
            if msg.role != "user" && msg.role != "assistant" {
                continue;
            }
            let content = msg.content.trim();
            if content.is_empty() {
                continue;
            }
            writer
                .add_document(doc!(
                    f_session_id => state.id.clone(),
                    f_session_name => name.clone(),
                    f_role => msg.role.clone(),
                    f_body => content,
                ))
                .map_err(|e| format!("add_doc: {e}"))?;
        }
    }
    writer.commit().map_err(|e| format!("commit: {e}"))?;
    Ok(())
}

pub fn search(project_root: &Path, query: &str, limit: usize) -> Result<Vec<String>, String> {
    reindex(project_root)?;
    let idx_path = index_dir(project_root);
    let index = open_or_create_index(&idx_path)?;
    let schema = index.schema();
    let body = schema.get_field("body").unwrap();
    let reader = index
        .reader_builder()
        .reload_policy(ReloadPolicy::OnCommitWithDelay)
        .try_into()
        .map_err(|e| format!("reader: {e}"))?;
    let searcher = reader.searcher();
    let parser = QueryParser::for_index(&index, vec![body]);
    let q = parser
        .parse_query(query)
        .map_err(|e| format!("parse query: {e}"))?;
    let top = TopDocs::with_limit(limit);
    let hits = searcher
        .search(&q, &top)
        .map_err(|e| format!("search: {e}"))?;

    let f_session_id = schema.get_field("session_id").unwrap();
    let f_session_name = schema.get_field("session_name").unwrap();
    let f_role = schema.get_field("role").unwrap();
    let f_body = schema.get_field("body").unwrap();

    let mut out = Vec::new();
    for (_score, addr) in hits {
        let doc: TantivyDocument = searcher.doc(addr).map_err(|e| format!("doc: {e}"))?;
        let sid = doc
            .get_first(f_session_id)
            .and_then(|v| v.as_str())
            .unwrap_or("");
        let name = doc
            .get_first(f_session_name)
            .and_then(|v| v.as_str())
            .unwrap_or("");
        let role = doc
            .get_first(f_role)
            .and_then(|v| v.as_str())
            .unwrap_or("");
        let snippet = doc
            .get_first(f_body)
            .and_then(|v| v.as_str())
            .unwrap_or("");
        let snip = if snippet.len() > 200 {
            format!("{}…", &snippet[..200])
        } else {
            snippet.to_string()
        };
        out.push(format!("- session {} ({}) [{}]\n  {}", sid, name, role, snip));
    }
    Ok(out)
}
