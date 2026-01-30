package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
)


type BenchConfig struct {
	BaseURL       string `json:"base_url"`
	TotalRequests int    `json:"total_req"`
	Concurrency   int    `json:"worker"`
	UploadSecret  string `json:"upload_secret"`
}
var client *http.Client

// Reduce GC pressure by reusing buffers
var bufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

type Stats struct {
	Success     uint64
	Failed      uint64
	Latencies   []time.Duration
	StatusCodes map[int]int
	mu          sync.Mutex
}

func main() {
	pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("OCTA", pterm.NewStyle(pterm.FgCyan)),
		pterm.NewLettersFromStringWithStyle("BENCH", pterm.NewStyle(pterm.FgMagenta)),
	).Render()

	// 1. Load Configuration
	config := loadConfig()
	pterm.Info.Printf("Loaded Config: %s | Workers: %d | Requests: %d\n", config.BaseURL, config.Concurrency, config.TotalRequests)

	// 2. Initialize Client with Dynamic Config
	client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: config.Concurrency + 50, // Ensure enough connections
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  true,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}

	if !checkServerHealth(config.BaseURL) {
		return
	}

	// --- PHASE 1: READ TEST ---
	// Closure config'i capture eder (yakalar)
	runBenchmark("🔥 READ STRESS TEST (Avatar Gen)", config, func() int {
		return makeRequest("GET", fmt.Sprintf("%s/avatar/%s", config.BaseURL, uuid.New().String()), nil, "")
	})

	fmt.Println()

	// --- PHASE 2: WRITE TEST ---
	dummyImg := createDummyImage()

	runBenchmark("⚡ WRITE STRESS TEST (Upload Asset)", config, func() int {
		return uploadRequest(dummyImg, config)
	})
}

// --- HELPER FUNCTIONS ---

func loadConfig() BenchConfig {
	// Root dizinden veya bir üst dizinden bakabilir
	paths := []string{"bench.json", "../../bench.json"}
	
	for _, path := range paths {
		if content, err := os.ReadFile(path); err == nil {
			var config BenchConfig
			if err := json.Unmarshal(content, &config); err != nil {
				pterm.Fatal.Printf("Invalid JSON in %s: %v\n", path, err)
			}
			pterm.Success.Printf("Config loaded from: %s\n", path)
			return config
		}
	}
	
	pterm.Fatal.Println("bench.json not found! Please create it in the root directory.")
	return BenchConfig{} // Unreachable due to Fatal
}

func runBenchmark(name string, cfg BenchConfig, operation func() int) {
	bar, _ := pterm.DefaultProgressbar.WithTotal(cfg.TotalRequests).WithTitle(name).WithRemoveWhenDone(true).Start()

	stats := &Stats{
		StatusCodes: make(map[int]int),
		Latencies:   make([]time.Duration, 0, cfg.TotalRequests),
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.Concurrency) // Semaphore from config
	start := time.Now()

	for i := 0; i < cfg.TotalRequests; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			t0 := time.Now()
			code := operation()
			dur := time.Since(t0)

			stats.mu.Lock()
			stats.Latencies = append(stats.Latencies, dur)
			stats.StatusCodes[code]++
			stats.mu.Unlock()

			if code >= 200 && code < 300 {
				atomic.AddUint64(&stats.Success, 1)
			} else {
				atomic.AddUint64(&stats.Failed, 1)
			}

			bar.Increment()
		}()
	}

	wg.Wait()
	printReport(stats, time.Since(start), cfg.TotalRequests)
}

func makeRequest(method, url string, body io.Reader, contentType string) int {
	req, _ := http.NewRequest(method, url, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

func uploadRequest(imgData []byte, cfg BenchConfig) int {
	body := bufferPool.Get().(*bytes.Buffer)
	body.Reset()
	defer bufferPool.Put(body)

	writer := multipart.NewWriter(body)
	// Go-Bench prefix ile ayırt edilebilir olsun
	writer.WriteField("keys", "go-bench/go-bench-"+uuid.New().String())
	writer.WriteField("mode", "square")
	writer.WriteField("size", "256")

	part, _ := writer.CreateFormFile("avatar", "bench.jpg")
	part.Write(imgData)
	writer.Close()

	req, _ := http.NewRequest("POST", cfg.BaseURL+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Secret-Key", cfg.UploadSecret) // Config'den gelen secret

	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

func createDummyImage() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100)) // 100x100 (Rust ile aynı)
	// Fill simple noise
	for i := 0; i < 100*100*4; i += 4 {
		img.Pix[i] = uint8(rand.Intn(255))
		img.Pix[i+3] = 255
	}
	buf := new(bytes.Buffer)
	jpeg.Encode(buf, img, nil)
	return buf.Bytes()
}

func checkServerHealth(baseURL string) bool {
	spinner, _ := pterm.DefaultSpinner.Start("Checking server...")
	if resp, err := http.Get(baseURL + "/"); err == nil {
		resp.Body.Close()
		spinner.Success("Server is UP! (" + baseURL + ")")
		return true
	}
	spinner.Fail("Server is DOWN! (" + baseURL + ")")
	return false
}

func printReport(s *Stats, totalTime time.Duration, totalReq int) {
	if len(s.Latencies) == 0 {
		return
	}

	sort.Slice(s.Latencies, func(i, j int) bool { return s.Latencies[i] < s.Latencies[j] })
	count := len(s.Latencies)

	data := [][]string{
		{"Metric", "Value"},
		{"Throughput", fmt.Sprintf("%.2f Req/sec", float64(totalReq)/totalTime.Seconds())},
		{"Success Rate", fmt.Sprintf("%.2f%%", float64(atomic.LoadUint64(&s.Success))/float64(totalReq)*100)},
		{"Avg Latency (P50)", fmt.Sprintf("%v", s.Latencies[count/2])},
		{"P95 Latency", fmt.Sprintf("%v", s.Latencies[int(float64(count)*0.95)])},
		{"P99 Latency", fmt.Sprintf("%v", s.Latencies[int(float64(count)*0.99)])},
	}

	pterm.DefaultTable.WithHasHeader().WithData(data).Render()

	if atomic.LoadUint64(&s.Failed) > 0 {
		pterm.Warning.Println("Status Code Breakdown (Errors):")
		for code, cnt := range s.StatusCodes {
			if code >= 400 || code == 0 {
				fmt.Printf("HTTP %d: %d\n", code, cnt)
			}
		}
	}
}

// package main

// import (
// 	"bytes"
// 	"fmt"
// 	"image"
// 	// "image/color"
// 	"image/jpeg"
// 	"io"
// 	"math/rand"
// 	"mime/multipart"
// 	"net/http"
// 	"sort"
// 	"sync"
// 	"sync/atomic"
// 	"time"

// 	"github.com/google/uuid"
// 	"github.com/pterm/pterm"
// )

// // --- CONFIGURATION ---
// const (
// 	BaseURL       = "http://127.0.0.1:9980" // Use IP to avoid DNS lookup overhead
// 	TotalRequests = 25_000
// 	Concurrency   = 200
// 	UploadSecret  = "secret" // Must match config.yaml
// )

// // High-performance HTTP client tuned for benchmarking (Keep-Alive enabled)
// var client = &http.Client{
// 	Timeout: 30 * time.Second,
// 	Transport: &http.Transport{
// 		MaxIdleConns:          1000,
// 		MaxIdleConnsPerHost:   Concurrency + 50, // Ensure enough connections for all workers
// 		IdleConnTimeout:       90 * time.Second,
// 		DisableCompression:    true, // Save CPU by skipping gzip
// 		ResponseHeaderTimeout: 30 * time.Second,
// 	},
// }

// // Reduce GC pressure by reusing buffers for upload payloads
// var bufferPool = sync.Pool{
// 	New: func() interface{} { return new(bytes.Buffer) },
// }

// type Stats struct {
// 	Success     uint64
// 	Failed      uint64
// 	Latencies   []time.Duration
// 	StatusCodes map[int]int
// 	mu          sync.Mutex // Protects map and slice only
// }

// func main() {
// 	pterm.DefaultBigText.WithLetters(
// 		pterm.NewLettersFromStringWithStyle("OCTA", pterm.NewStyle(pterm.FgCyan)),
// 		pterm.NewLettersFromStringWithStyle("BENCH", pterm.NewStyle(pterm.FgMagenta)),
// 	).Render()

// 	pterm.Info.Printf("Target: %s | Workers: %d | Requests: %d\n\n", BaseURL, Concurrency, TotalRequests)

// 	if !checkServerHealth() {
// 		return
// 	}

// 	// --- PHASE 1: READ TEST ---
// 	runBenchmark("🔥 READ STRESS TEST (Avatar Gen)", func() int {
// 		// Simulate cache-miss by generating random UUIDs
// 		return makeRequest("GET", fmt.Sprintf("%s/avatar/%s", BaseURL, uuid.New().String()), nil, "")
// 	})

// 	fmt.Println()

// 	// --- PHASE 2: WRITE TEST ---
// 	// Generate dummy image once in RAM to avoid IO bottlenecks
// 	dummyImg := createDummyImage()

// 	runBenchmark("⚡ WRITE STRESS TEST (Upload Asset)", func() int {
// 		return uploadRequest(dummyImg)
// 	})
// }

// func runBenchmark(name string, operation func() int) {
// 	bar, _ := pterm.DefaultProgressbar.WithTotal(TotalRequests).WithTitle(name).WithRemoveWhenDone(true).Start()

// 	// Pre-allocate slice to avoid resize overhead during runtime
// 	stats := &Stats{
// 		StatusCodes: make(map[int]int),
// 		Latencies:   make([]time.Duration, 0, TotalRequests),
// 	}

// 	var wg sync.WaitGroup
// 	sem := make(chan struct{}, Concurrency) // Semaphore for concurrency control
// 	start := time.Now()

// 	for i := 0; i < TotalRequests; i++ {
// 		wg.Add(1)
// 		sem <- struct{}{}

// 		go func() {
// 			defer wg.Done()
// 			defer func() { <-sem }()

// 			t0 := time.Now()
// 			code := operation()
// 			dur := time.Since(t0)

// 			// Update stats
// 			stats.mu.Lock()
// 			stats.Latencies = append(stats.Latencies, dur)
// 			stats.StatusCodes[code]++
// 			stats.mu.Unlock()

// 			if code >= 200 && code < 300 {
// 				atomic.AddUint64(&stats.Success, 1)
// 			} else {
// 				atomic.AddUint64(&stats.Failed, 1)
// 			}

// 			bar.Increment()
// 		}()
// 	}

// 	wg.Wait()
// 	printReport(stats, time.Since(start))
// }

// func makeRequest(method, url string, body io.Reader, contentType string) int {
// 	req, _ := http.NewRequest(method, url, body)
// 	if contentType != "" {
// 		req.Header.Set("Content-Type", contentType)
// 	}

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return 0 // Connection Error
// 	}
// 	defer resp.Body.Close()
// 	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse
// 	return resp.StatusCode
// }

// func uploadRequest(imgData []byte) int {
// 	// Get buffer from pool to minimize allocation
// 	body := bufferPool.Get().(*bytes.Buffer)
// 	body.Reset()
// 	defer bufferPool.Put(body)

// 	writer := multipart.NewWriter(body)
// 	writer.WriteField("keys", "go-bench/go-bench-"+uuid.New().String()) // Unique key to prevent DB conflicts
// 	writer.WriteField("mode", "square")
// 	writer.WriteField("size", "256")

// 	part, _ := writer.CreateFormFile("avatar", "bench.jpg")
// 	part.Write(imgData)
// 	writer.Close()

// 	req, _ := http.NewRequest("POST", BaseURL+"/upload", body)
// 	req.Header.Set("Content-Type", writer.FormDataContentType())
// 	req.Header.Set("X-Secret-Key", UploadSecret)

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return 0
// 	}
// 	defer resp.Body.Close()
// 	io.Copy(io.Discard, resp.Body)
// 	return resp.StatusCode
// }

// func createDummyImage() []byte {
// 	img := image.NewRGBA(image.Rect(0, 0, 500, 500))
// 	// Fill simple noise
// 	for i := 0; i < 500*500*4; i += 4 {
// 		img.Pix[i] = uint8(rand.Intn(255)) // R
// 		img.Pix[i+3] = 255                 // A
// 	}
// 	buf := new(bytes.Buffer)
// 	jpeg.Encode(buf, img, nil)
// 	return buf.Bytes()
// }

// func checkServerHealth() bool {
// 	spinner, _ := pterm.DefaultSpinner.Start("Checking server...")
// 	if resp, err := http.Get(BaseURL + "/"); err == nil {
// 		resp.Body.Close()
// 		spinner.Success("Server is UP!")
// 		return true
// 	}
// 	spinner.Fail("Server is DOWN! Run 'go run cmd/api/main.go'")
// 	return false
// }

// func printReport(s *Stats, totalTime time.Duration) {
// 	if len(s.Latencies) == 0 {
// 		return
// 	}

// 	// Sort for percentiles
// 	sort.Slice(s.Latencies, func(i, j int) bool { return s.Latencies[i] < s.Latencies[j] })
// 	count := len(s.Latencies)

// 	data := [][]string{
// 		{"Metric", "Value"},
// 		{"Throughput", fmt.Sprintf("%.2f Req/sec", float64(TotalRequests)/totalTime.Seconds())},
// 		{"Success Rate", fmt.Sprintf("%.2f%%", float64(atomic.LoadUint64(&s.Success))/float64(TotalRequests)*100)},
// 		{"Avg Latency (P50)", fmt.Sprintf("%v", s.Latencies[count/2])},
// 		{"P95 Latency", fmt.Sprintf("%v", s.Latencies[int(float64(count)*0.95)])},
// 		{"P99 Latency", fmt.Sprintf("%v", s.Latencies[int(float64(count)*0.99)])},
// 	}

// 	pterm.DefaultTable.WithHasHeader().WithData(data).Render()

// 	if atomic.LoadUint64(&s.Failed) > 0 {
// 		pterm.Warning.Println("Status Code Breakdown (Errors):")
// 		for code, cnt := range s.StatusCodes {
// 			if code >= 400 || code == 0 {
// 				fmt.Printf("HTTP %d: %d\n", code, cnt)
// 			}
// 		}
// 	}
// }
