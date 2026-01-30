package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/fatih/color"
)

// ResponseWriter wrapper to capture status code and size
type statusWriter struct {
	http.ResponseWriter
	statusCode int
	length     int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	w.length += len(b)
	return w.ResponseWriter.Write(b)
}

var (
	// Method Colors
	cGet    = color.New(color.FgHiCyan, color.Bold).SprintFunc()    
	cPost   = color.New(color.FgHiGreen, color.Bold).SprintFunc()   
	cPut    = color.New(color.FgHiYellow, color.Bold).SprintFunc() 
	cDelete = color.New(color.FgHiRed, color.Bold).SprintFunc()     
	cPatch  = color.New(color.FgHiMagenta, color.Bold).SprintFunc()
	cDefault= color.New(color.FgWhite, color.Bold).SprintFunc()     

	
	c200 = color.New(color.FgGreen, color.Bold).SprintFunc()
	c400 = color.New(color.FgYellow, color.Bold).SprintFunc()
	c500 = color.New(color.FgRed, color.Bold).SprintFunc()

	
	cTime = color.New(color.FgHiBlack).SprintFunc() 
	cPath = color.New(color.FgWhite).SprintFunc()  
)

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		
		var statusStr string
		code := ww.statusCode
		switch {
		case code >= 500:
			statusStr = c500(fmt.Sprintf("%d", code))
		case code >= 400:
			statusStr = c400(fmt.Sprintf("%d", code))
		default:
			statusStr = c200(fmt.Sprintf("%d", code))
		}

		
		var methodStr string
		switch r.Method {
		case http.MethodGet:
			methodStr = cGet(fmt.Sprintf("%-7s", "["+r.Method+"]")) 
		case http.MethodPost:
			methodStr = cPost(fmt.Sprintf("%-7s", "["+r.Method+"]"))
		case http.MethodPut:
			methodStr = cPut(fmt.Sprintf("%-7s", "["+r.Method+"]"))
		case http.MethodDelete:
			methodStr = cDelete(fmt.Sprintf("%-7s", "["+r.Method+"]"))
		case http.MethodPatch:
			methodStr = cPatch(fmt.Sprintf("%-7s", "["+r.Method+"]"))
		default:
			methodStr = cDefault(fmt.Sprintf("%-7s", "["+r.Method+"]"))
		}

		
		timeStamp := cTime(start.Format("2006-01-02 15:04:05"))
		
		fmt.Printf("%s %s %s %s %s %s\n",
			timeStamp,
			methodStr,
			cPath(r.RequestURI),
			statusStr,
			cTime("|"),
			cTime(duration.String()),
		)
	})
}