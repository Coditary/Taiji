mod find;
mod git;
mod glob;
mod html;
mod list;
mod outline;
mod sessions;
mod walk;

use clap::{Parser, Subcommand};
use std::io::{self, Read, Write};
use std::path::PathBuf;
use std::process::ExitCode;

#[derive(Parser)]
#[command(name = "taiji-tools")]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Glob files (ignore + globset).
    Glob {
        #[arg(long)]
        root: PathBuf,
        #[arg(long)]
        pattern: String,
        #[arg(long, default_value_t = 100)]
        max: usize,
    },
    /// List entries under root (ignore walk).
    List {
        #[arg(long)]
        root: PathBuf,
        #[arg(long, default_value_t = 3)]
        max_depth: u32,
        #[arg(long, default_value_t = 500)]
        max: usize,
    },
    /// Fuzzy-find paths / project markers (nucleo).
    Find {
        #[arg(long)]
        root: PathBuf,
        #[arg(long)]
        query: String,
        #[arg(long, default_value_t = 50)]
        max: usize,
    },
    /// tree-sitter symbol outline.
    Outline {
        #[arg(long)]
        path: PathBuf,
    },
    /// HTML to plain text (html5ever via html2text).
    HtmlToText,
    /// git2 status.
    GitStatus {
        #[arg(long)]
        root: PathBuf,
    },
    /// git2 oneline log.
    GitLog {
        #[arg(long)]
        root: PathBuf,
        #[arg(long, default_value_t = 10)]
        limit: usize,
    },
    /// Tantivy session search (reindex then query).
    SessionsSearch {
        #[arg(long)]
        root: PathBuf,
        #[arg(long)]
        query: String,
        #[arg(long, default_value_t = 20)]
        max: usize,
    },
}

fn main() -> ExitCode {
    match run() {
        Ok(()) => ExitCode::SUCCESS,
        Err(msg) => {
            let _ = writeln!(io::stderr(), "{msg}");
            ExitCode::from(1)
        }
    }
}

fn run() -> Result<(), String> {
    let cli = Cli::parse();
    match cli.command {
        Commands::Glob { root, pattern, max } => {
            for line in glob::run(&root, &pattern, max)? {
                println!("{line}");
            }
        }
        Commands::List { root, max_depth, max } => {
            for line in list::run(&root, max_depth, max)? {
                println!("{line}");
            }
        }
        Commands::Find { root, query, max } => {
            for line in find::run(&root, &query, max)? {
                println!("{line}");
            }
        }
        Commands::Outline { path } => {
            print!("{}", outline::run(&path)?);
        }
        Commands::HtmlToText => {
            let mut buf = String::new();
            io::stdin()
                .read_to_string(&mut buf)
                .map_err(|e| format!("stdin: {e}"))?;
            print!("{}", html::html_to_text(&buf));
        }
        Commands::GitStatus { root } => {
            print!("{}", git::status(&root)?);
        }
        Commands::GitLog { root, limit } => {
            print!("{}", git::log_oneline(&root, limit)?);
        }
        Commands::SessionsSearch { root, query, max } => {
            for line in sessions::search(&root, &query, max)? {
                println!("{line}");
            }
        }
    }
    Ok(())
}
