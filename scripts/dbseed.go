package main

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
)

// COnfig
const (
	ServerURL    = "http://localhost:9980/upload"
	UploadSecret = "secret"
	TotalImages  = 50
	WorkerCount  = 5
)

var (
	folders = []string{"nature", "space", "architecture", "users/avatars", "products", "wallpapers"}
	names   = []string{"mountain", "river", "nebula", "mars", "building", "office", "profile", "admin", "hero-banner", "footer-bg"}
)

type Result struct {
	Key     string
	Success bool
	Error   error
}

func main() {
	pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgLightMagenta)).WithTextStyle(pterm.NewStyle(pterm.FgBlack)).Println("OCTA ASSET SEEDER")
	pterm.Println()

	data := pterm.TableData{
		{"Target Server", color.New(color.FgCyan).Sprint(ServerURL)},
		{"Total Assets", color.New(color.FgYellow).Sprintf("%d images", TotalImages)},
		{"Concurrency", color.New(color.FgYellow).Sprintf("%d workers", WorkerCount)},
		{"Auth Secret", color.New(color.FgRed).Sprint("******")},
	}
	_ = pterm.DefaultTable.WithBoxed().WithData(data).Render()
	pterm.Println()

	bar, _ := pterm.DefaultProgressbar.
		WithTotal(TotalImages).
		WithTitle("Seeding Assets...").
		WithShowCount(true).
		WithShowElapsedTime(true).
		Start()

	var wg sync.WaitGroup
	jobs := make(chan int, TotalImages)
	results := make(chan Result, TotalImages)

	// Start Workers
	for w := 1; w <= WorkerCount; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg, bar)
	}

	for i := 1; i <= TotalImages; i++ {
		jobs <- i
	}
	close(jobs)

	//Wait to end workers
	wg.Wait()
	close(results)
	bar.Stop()

	// Results Analysis and Reporting
	successCount := 0
	failCount := 0
	var failures []Result

	for res := range results {
		if res.Success {
			successCount++
		} else {
			failCount++
			failures = append(failures, res)
		}
	}

	pterm.Println()
	if failCount == 0 {
		pterm.DefaultSection.WithStyle(pterm.NewStyle(pterm.FgGreen)).Println("SEEDING COMPLETED SUCCESSFULLY")
		pterm.Info.Printf("Uploaded %d assets to Octa engine.\n", successCount)
	} else {
		pterm.DefaultSection.WithStyle(pterm.NewStyle(pterm.FgYellow)).Println("COMPLETED WITH ERRORS")
		pterm.Info.Printf("Success: %d | Failed: %d\n", successCount, failCount)

		pterm.Println()
		pterm.Error.Println("Failure Report:")
		for _, f := range failures {
			fmt.Printf(" • %s: %v\n", color.RedString(f.Key), f.Error)
		}
	}

	pterm.Println()
}

func worker(id int, jobs <-chan int, results chan<- Result, wg *sync.WaitGroup, bar *pterm.ProgressbarPrinter) {
	defer wg.Done()

	for j := range jobs {
		// Download Image
		imgURL := fmt.Sprintf("https://picsum.photos/seed/%d/800/600", rand.Intn(10000)+j)
		imgData, err := downloadImage(imgURL)

		if err != nil {
			bar.Increment() // The process is considered complete (even if it is incorrect).
			results <- Result{Success: false, Error: fmt.Errorf("download failed: %w", err)}
			continue
		}


		var key string
		name := names[rand.Intn(len(names))]

		if rand.Intn(100) < 25 {
			// Root file: "hero-banner-12"
			key = fmt.Sprintf("%s-%d", name, j)
		} else {
			// Folder file: "nature/mountain-12"
			folder := folders[rand.Intn(len(folders))]
			key = fmt.Sprintf("%s/%s-%d", folder, name, j)
		}

		// Upload Server
		err = uploadToOcta(key, imgData)
		if err != nil {
			results <- Result{Key: key, Success: false, Error: err}
		} else {
			results <- Result{Key: key, Success: true}
		}

		bar.Increment()
	}
}

func downloadImage(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func uploadToOcta(key string, data []byte) error {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("avatar", "seed-image.jpg")
	if err != nil {
		return err
	}
	part.Write(data)
	_ = writer.WriteField("keys", key)
	_ = writer.WriteField("mode", "scale")
	_ = writer.WriteField("scale", "75")
	writer.Close()

	req, err := http.NewRequest("POST", ServerURL, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Secret-Key", UploadSecret)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Read and discard the response body (Memory leak prevention)
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("server rejected: %d", resp.StatusCode)
	}

	return nil
}
