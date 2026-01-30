package logger

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
)

var (
	cInf  = color.New(color.FgCyan, color.Bold).SprintFunc()
	cWarn = color.New(color.FgYellow, color.Bold).SprintFunc()
	cErr  = color.New(color.FgRed, color.Bold).SprintFunc()
	cSucc = color.New(color.FgGreen, color.Bold).SprintFunc()
	cFatl = color.New(color.BgRed, color.FgWhite, color.Bold).SprintFunc()
	cTime = color.New(color.FgHiBlack).SprintFunc()
)

func init() {
	log.SetFlags(0)
}

func timeStamp() string {
	// return cTime(time.Now().Format("15:04:05"))
	return cTime(time.Now().Format("2006-01-02 15:04"))
}

func LogInfo(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s %s %s\n", timeStamp(), cInf("[INFO]"), msg)
}

func LogSuccess(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s %s %s\n", timeStamp(), cSucc("[OK]"), msg)
}

func LogWarn(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s %s %s\n", timeStamp(), cWarn("[WARN]"), msg)
}

func LogError(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	fmt.Fprintf(os.Stderr, "%s %s %s\n", timeStamp(), cErr("[ERR]"), msg)
}

func LogFatal(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	fmt.Fprintf(os.Stderr, "%s %s %s\n", timeStamp(), cFatl("[FATAL]"), msg)
	os.Exit(1)
}

func LogServerStart(port int, baseURL string) {
	fmt.Println()
	fmt.Printf("   %s  %s\n", cSucc("⚡ Server is Active"), cTime("waiting for requests..."))
	fmt.Printf("   %s  %s\n", cInf("➜ Local:"), fmt.Sprintf("http://localhost:%d", port))
	fmt.Printf("   %s  %s\n", cInf("➜ Public:"), color.New(color.FgHiBlue, color.Underline).Sprint(baseURL))
	fmt.Println()
}
