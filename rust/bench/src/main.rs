use console::style;
use indicatif::{ProgressBar, ProgressStyle};
use reqwest::{multipart, Client};
use serde::Deserialize;
use std::fs;
use std::io::Cursor;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::{Mutex, Semaphore};
use uuid::Uuid;
use comfy_table::Table;


#[derive(Debug, Deserialize, Clone)]
struct BenchConfig {
    base_url: String,
    total_req: usize,
    worker: usize,      // Concurrency
    // #[serde(rename = "UploadSecret")] 
    upload_secret: String,
}

struct BenchStats {
    success: AtomicU64,
    failed: AtomicU64,
    latencies: Mutex<Vec<Duration>>,
}

// generate fake image
fn generate_valid_jpeg() -> Vec<u8> {
    let img = image::RgbImage::new(100, 100);
    let mut bytes: Vec<u8> = Vec::new();
    img.write_to(&mut Cursor::new(&mut bytes), image::ImageOutputFormat::Jpeg(80))
        .expect("Failed to generate image");
    bytes
}

fn load_config() -> BenchConfig {
    let paths = ["../../bench.json", "bench.json"];
    
    for path in paths {
        if let Ok(content) = fs::read_to_string(path) {
            println!("{} Loaded config from: {}", style("[CONFIG]").green(), style(path).bold());
            return serde_json::from_str(&content).expect("JSON format error in bench.json");
        }
    }
    
    panic!("{} bench.json not found in root or current directory!", style("[ERR]").red());
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    print_banner();

    let config = load_config();

    let client = Client::builder()
        .pool_max_idle_per_host(config.worker + 50)
        .tcp_keepalive(Duration::from_secs(90))
        .build()?;

    if !check_health(&client, &config.base_url).await { return Ok(()); }

    //  PHASE 1: READ STRESS TEST
    println!("\n{}", style("PHASE 1: Starting Read Test...").yellow());
    
    let read_client = client.clone();
    let read_url = config.base_url.clone(); 
    
    let loop_config = config.clone(); 

    run_benchmark(&loop_config, "🔥 READ STRESS TEST", move || {
        let url_base = read_url.clone();
        let c = read_client.clone();
        async move {
            let url = format!("{}/avatar/{}", url_base, Uuid::new_v4());
            c.get(url).send().await.map(|r| r.status().as_u16())
        }
    }).await;

    // PHASE 2: WRITE STRESS TEST 
    println!("\n{}", style("PHASE 2: Starting Write Test...").yellow());
    
    println!("Generating valid JPEG asset for benchmark...");
    let valid_img_data = generate_valid_jpeg(); 
    
    let write_client = client.clone();
    let write_config = config.clone();

    run_benchmark(&config, "⚡ WRITE STRESS TEST", move || {
        let c = write_client.clone();
        let cfg = write_config.clone();
        let data = valid_img_data.clone();
        
        async move {
            let form = multipart::Form::new()
                .text("keys", format!("rust-bench/{}", Uuid::new_v4()))
                .text("mode", "square")
                .part("avatar", multipart::Part::bytes(data)
                    .file_name("bench.jpg")
                    .mime_str("image/jpeg")?);

            c.post(format!("{}/upload", cfg.base_url))
                .header("X-Secret-Key", cfg.upload_secret)
                .multipart(form)
                .send()
                .await
                .map(|r| r.status().as_u16())
        }
    }).await;

    Ok(())
}

// To run benchmark tests, run_benchmark should be used. What it does is simple:

// Based on the requests and worker values it gets from the config file,
// it executes the given operation function and logs it.
async fn run_benchmark<F, Fut>(config: &BenchConfig, name: &str, mut operation: F) 
where 
    F: FnMut() -> Fut,
    Fut: std::future::Future<Output = Result<u16, reqwest::Error>> + Send + 'static
{
    let stats = Arc::new(BenchStats {
        success: AtomicU64::new(0),
        failed: AtomicU64::new(0),
        latencies: Mutex::new(Vec::with_capacity(config.total_req)),
    });

    let pb = ProgressBar::new(config.total_req as u64);
    pb.set_style(ProgressStyle::default_bar()
        .template(&format!("{{spinner:.green}} {}: [{{elapsed_precise}}] [{{bar:40.cyan/blue}}] {{pos}}/{{len}}", name))
        .unwrap());

    // Get the number of workers from Config
    let semaphore = Arc::new(Semaphore::new(config.worker));
    let start_time = Instant::now();
    let mut workers = vec![];

    for _ in 0..config.total_req {
        let permit = semaphore.clone().acquire_owned().await.unwrap();
        let stats = stats.clone();
        let fut = operation();
        let pb = pb.clone();

        workers.push(tokio::spawn(async move {
            let _permit = permit;
            let start = Instant::now();
            let result = fut.await;
            let duration = start.elapsed();

            let mut lats = stats.latencies.lock().await;
            lats.push(duration);
            
            match result {
                Ok(code) if code >= 200 && code < 300 => {
                    stats.success.fetch_add(1, Ordering::Relaxed);
                }
                _ => {
                    stats.failed.fetch_add(1, Ordering::Relaxed);
                }
            }
            pb.inc(1);
        }));
    }

    for worker in workers { let _ = worker.await; }
    pb.finish_and_clear();

    print_report(&stats, start_time.elapsed()).await;
}

async fn print_report(stats: &Arc<BenchStats>, total_time: Duration) {
    let mut lats = stats.latencies.lock().await;
    if lats.is_empty() { return; }
    lats.sort();
    
    let success = stats.success.load(Ordering::Relaxed);
    let failed = stats.failed.load(Ordering::Relaxed);
    let total = success + failed;

    let mut table = Table::new();
    table.set_header(vec!["Metric", "Value"]);

    table.add_row(vec![
        "Throughput".to_string(), 
        format!("{:.2} Req/sec", total as f64 / total_time.as_secs_f64())
    ]);
    table.add_row(vec![
        "Success Rate".to_string(), 
        format!("{:.2}%", (success as f64 / total as f64) * 100.0)
    ]);
    table.add_row(vec![
        "Avg Latency (P50)".to_string(), 
        format!("{:?}", lats[lats.len() / 2])
    ]);
    table.add_row(vec![
        "P95 Latency".to_string(), 
        format!("{:?}", lats[(lats.len() as f64 * 0.95) as usize])
    ]);
    table.add_row(vec![
        "P99 Latency".to_string(), 
        format!("{:?}", lats[(lats.len() as f64 * 0.99) as usize])
    ]);

    println!("{}", table);
}

fn print_banner() {
    println!("{}", style("OCTA-PULSE BENCHMARK TOOL").bold().cyan());
    println!("{}\n", style("==========================").dim());
}

async fn check_health(client: &Client, base_url: &str) -> bool {
    match client.get(base_url).send().await {
        Ok(_) => { println!("{} Server is UP! ({})", style("[OK]").green(), base_url); true }
        Err(_) => { println!("{} Server is DOWN! ({})", style("[ERR]").red(), base_url); false }
    }
}