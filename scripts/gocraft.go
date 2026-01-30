package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

const (
	DefaultDistDir = "bin"
)

type BuildTarget struct {
	OS   string
	Arch string
}

type BuildResult struct {
	Platform string
	Status   string
	Duration time.Duration
	Artifact string
	Size     string
	ErrorMsg string
}

// GLOBAL FLAGS 
var (
	appName    string
	appVersion string
	entryPoint string
	outputDir  string
	versionPkg string
	buildAll   bool
	platforms  []string
	stripDebug bool
)

// Default targets for --all flag
var commonTargets = []BuildTarget{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"windows", "amd64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "gocraft",
		Short: "Professional Go Build Tool",
		Run:   runBuild,
	}

	// Configuration Flags
	rootCmd.Flags().StringVarP(&appName, "name", "n", "", "Binary name (Required)")
	rootCmd.Flags().StringVarP(&appVersion, "version", "v", "1.0.0", "App version")
	rootCmd.Flags().StringVarP(&entryPoint, "entry", "e", ".", "Main package path")
	rootCmd.Flags().StringVarP(&outputDir, "out", "o", DefaultDistDir, "Output directory")
	rootCmd.Flags().StringVar(&versionPkg, "ver-pkg", "", "Variable to inject version (e.g. main.Version)")

	// Build Strategy Flags
	rootCmd.Flags().BoolVar(&buildAll, "all", false, "Build for all common platforms")
	rootCmd.Flags().StringSliceVarP(&platforms, "platform", "p", []string{}, "Custom platforms (os/arch)")
	rootCmd.Flags().BoolVar(&stripDebug, "strip", true, "Strip debug symbols (-s -w)")

	rootCmd.MarkFlagRequired("name")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runBuild(cmd *cobra.Command, args []string) {
	startTime := time.Now()
	printBanner()

	targets := resolveTargets()
	prepareWorkspace()
	printInfo(targets)

	var results []BuildResult
	pterm.Println()
	
	// Start multi-line spinner
	multiSpinner, _ := pterm.DefaultMultiPrinter.Start()

	for _, t := range targets {
		res := executeBuild(t, multiSpinner)
		results = append(results, res)
	}

	multiSpinner.Stop()
	printSummary(results, time.Since(startTime))
}

// --- CORE LOGIC ---

func resolveTargets() []BuildTarget {
	// 1. Custom platforms via flag
	if len(platforms) > 0 {
		var customTargets []BuildTarget
		for _, p := range platforms {
			parts := strings.Split(p, "/")
			if len(parts) != 2 {
				pterm.Warning.Printf("Invalid platform: %s\n", p)
				continue
			}
			customTargets = append(customTargets, BuildTarget{parts[0], parts[1]})
		}
		return customTargets
	}

	// 2. Build all common targets
	if buildAll {
		return commonTargets
	}

	// 3. Default to current OS
	return []BuildTarget{
		{runtime.GOOS, runtime.GOARCH},
	}
}

func prepareWorkspace() {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		os.MkdirAll(outputDir, 0755)
	}
}

func executeBuild(t BuildTarget, printer *pterm.MultiPrinter) BuildResult {
	start := time.Now()

	// Determine filename (append .exe for windows)
	fileName := appName
	if buildAll || len(platforms) > 0 {
		fileName = fmt.Sprintf("%s-%s-%s", appName, t.OS, t.Arch)
	}
	if t.OS == "windows" {
		fileName += ".exe"
	}

	outPath := filepath.Join(outputDir, fileName)
	label := fmt.Sprintf("%s/%s", t.OS, t.Arch)

	spinner, _ := pterm.DefaultSpinner.WithWriter(printer.NewWriter()).Start("Building " + label + "...")

	// Prepare LDFLAGS
	var ldflags []string
	if stripDebug {
		ldflags = append(ldflags, "-s", "-w")
	}
	if versionPkg != "" {
		date := time.Now().Format(time.RFC3339)
		ldflags = append(ldflags, fmt.Sprintf("-X '%s=%s'", versionPkg, appVersion))
		ldflags = append(ldflags, fmt.Sprintf("-X '%s_Date=%s'", versionPkg, date))
	}

	// Prepare Command
	cmdArgs := []string{"build"}
	if len(ldflags) > 0 {
		cmdArgs = append(cmdArgs, "-ldflags", strings.Join(ldflags, " "))
	}
	cmdArgs = append(cmdArgs, "-o", outPath, entryPoint)

	cmd := exec.Command("go", cmdArgs...)
	cmd.Env = append(os.Environ(), "GOOS="+t.OS, "GOARCH="+t.Arch, "CGO_ENABLED=1")

	// Capture output to show compiler errors
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed: %s", label))
		return BuildResult{label, pterm.FgRed.Sprint("FAIL"), duration, "-", "-", string(output)}
	}

	// Get file size
	fi, _ := os.Stat(outPath)
	size := formatSize(fi.Size())

	spinner.Success(fmt.Sprintf("Built: %s (%s)", label, size))

	return BuildResult{
		Platform: label,
		Status:   pterm.FgGreen.Sprint("SUCCESS"),
		Duration: duration,
		Artifact: fileName,
		Size:     size,
	}
}

// Helper to format bytes (since pterm function might vary)
func formatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}


func printBanner() {
  
    fmt.Println()

    color.New(color.FgHiCyan, color.Bold).Print("GO")
    color.New(color.FgHiMagenta, color.Bold).Print("CRAFT")
    color.New(color.FgHiBlack).Printf(" v%s\n", "2.0") 

    color.New(color.FgHiBlack).Println("High-Performance Build Engine")
    
    fmt.Println()
}

func printInfo(targets []BuildTarget) {
	data := [][]string{
		{"App Name", pterm.FgCyan.Sprint(appName)},
		{"Version", pterm.FgCyan.Sprint(appVersion)},
		{"Entry Point", entryPoint},
		{"Output Dir", outputDir},
		{"Target Count", fmt.Sprintf("%d", len(targets))},
	}
	pterm.DefaultTable.WithData(data).WithBoxed().Render()
}

func printSummary(results []BuildResult, totalTime time.Duration) {
	pterm.Println()
	pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).Println("BUILD SUMMARY")
	pterm.Println()

	tableData := [][]string{
		{"PLATFORM", "STATUS", "SIZE", "DURATION", "ARTIFACT"},
	}

	for _, r := range results {
		durStr := fmt.Sprintf("%v", r.Duration.Round(time.Millisecond))
		tableData = append(tableData, []string{r.Platform, r.Status, r.Size, durStr, r.Artifact})
	}

	pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(tableData).Render()

	// If there were errors, print them at the bottom
	for _, r := range results {
		if r.ErrorMsg != "" {
			pterm.Println()
			pterm.Error.Printf("Compiler Error for [%s]:\n%s\n", r.Platform, r.ErrorMsg)
		}
	}

	pterm.Println()
	pterm.Info.Printf("Total time: %v\n", totalTime.Round(time.Millisecond))
}