use clap::Parser;
use console::style;
use image::load_from_memory;
use rusqlite::{Connection, OpenFlags, Result};
use serde::Deserialize;
use std::fs;
use std::path::Path;
use std::time::Instant;

/*
OCTA-WARDEN: SQLite Integrity Auditor
=============================================
Mission: Audit SQLite BLOB assets without service interruption.
Safety:  Uses READ_ONLY mode and fail-safe iteration.
*/

#[derive(Parser, Debug)]
#[command(author, version, about = "Database Integrity Guard for Octa")]
struct Args {
    /// Path to the configuration file
    #[arg(short, long, default_value = "../../config.yaml")]
    config: String,
}

#[derive(Debug, Deserialize)]
struct Config {
    database: DatabaseConfig,
}

#[derive(Debug, Deserialize)]
struct DatabaseConfig {
    path: String,
}

struct AuditStats {
    total_scanned: u64,
    healthy: u64,
    corrupted_blob: u64,  // Image data is corrupted
    db_schema_error: u64, // Column type is incorrect (Text vs Blob)
}

fn main() -> Result<()> {
    let args = Args::parse();
    let start = Instant::now();

    print_banner();

    println!(
        "{} Loading configuration from: {}",
        style("→").cyan(),
        style(&args.config).yellow()
    );

    let config_content = match fs::read_to_string(&args.config) {
        Ok(c) => c,
        Err(_) => {
            println!(
                "{} Could not read config file at: {}",
                style("[FATAL]").red().bold(),
                args.config
            );
            return Ok(());
        }
    };

    let config: Config = match serde_yaml::from_str(&config_content) {
        Ok(c) => c,
        Err(_) => {
            println!(
                "{} Invalid YAML format in config file.",
                style("[FATAL]").red().bold()
            );
            return Ok(());
        }
    };

    let db_path = &config.database.path;

    if !Path::new(db_path).exists() {
        println!(
            "{} Database file not found at: {}",
            style("[FATAL]").red().bold(),
            db_path
        );
        return Ok(());
    }

    let conn = Connection::open_with_flags(
        db_path,
        OpenFlags::SQLITE_OPEN_READ_ONLY | OpenFlags::SQLITE_OPEN_NO_MUTEX,
    )?;

    println!(
        "{} Database connected. Integrity audit starting...\n",
        style("[OK]").green()
    );

    // Scanning is starting
    let mut stmt = conn.prepare("SELECT id, data FROM images")?;

    // Fail-Safe Iterator: We will catch erroneous lines during iteration.
    let image_iter = stmt.query_map([], |row| {
        let id_result = row.get::<_, String>(0);
        let blob_result = row.get::<_, Vec<u8>>(1);
        Ok((id_result, blob_result))
    })?;

    let mut stats = AuditStats {
        total_scanned: 0,
        healthy: 0,
        corrupted_blob: 0,
        db_schema_error: 0,
    };

    for item in image_iter {
        stats.total_scanned += 1;

        match item {
            // Iteration successful (SQLite row could be read)
            Ok((id_res, blob_res)) => {
                match (id_res, blob_res) {
                    (Ok(id), Ok(blob)) => {
                        // Deep Image Analysis (Deep Inspection)
                        if let Err(e) = load_from_memory(&blob) {
                            println!(
                                "{} {} ID: {} | Reason: {}",
                                style("[CORRUPT]").red(),
                                style("!").on_red(),
                                style(&id).bold(),
                                style(e).dim()
                            );
                            stats.corrupted_blob += 1;
                        } else {
                            stats.healthy += 1;
                        }
                    }
                    // Column types are incorrect (e.g., TEXT instead of BLOB)
                    (Err(e), _) | (_, Err(e)) => {
                        println!(
                            "{} {} Schema Mismatch | Reason: {}",
                            style("[DB-ERR]").magenta(),
                            style("X").on_magenta(),
                            style(e).dim()
                        );
                        stats.db_schema_error += 1;
                    }
                }
            }
            // The iteration itself failed (Very rare, disk error, etc.)
            Err(e) => {
                println!(
                    "{} Critical Row Failure: {}",
                    style("[FATAL]").red().bold(),
                    e
                );
            }
        }
    }

    render_report(&stats, start.elapsed());

    Ok(())
}

fn render_report(stats: &AuditStats, duration: std::time::Duration) {
    println!("\n{}", style("WARDEN AUDIT REPORT").bold().underlined());
    println!("Time Elapsed   : {:?}", duration);
    println!("Assets Scanned : {}", stats.total_scanned);
    println!("--------------------------------");
    println!("Healthy Assets : {}", style(stats.healthy).green());

    if stats.corrupted_blob > 0 {
        println!(
            "Corrupted Blobs: {}",
            style(stats.corrupted_blob).red().bold()
        );
    } else {
        println!("Corrupted Blobs: {}", style("0").dim());
    }

    if stats.db_schema_error > 0 {
        println!(
            "Schema Errors  : {}",
            style(stats.db_schema_error).magenta().bold()
        );
    } else {
        println!("Schema Errors  : {}", style("0").dim());
    }

    println!("--------------------------------");

    if stats.corrupted_blob == 0 && stats.db_schema_error == 0 {
        println!(
            "Status         : {}",
            style("SYSTEM HEALTHY").green().bold().on_black()
        );
    } else {
        println!(
            "Status         : {}",
            style("ATTENTION REQUIRED").yellow().bold().on_black()
        );
    }
}

fn print_banner() {
    println!("{}\n", style("Octa Warden - Database Health Check").dim());
}
